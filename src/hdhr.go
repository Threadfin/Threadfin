package src

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// --- helpers ---

// toString safely converts common JSON types to string.
func toString(v interface{}) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		// Preserve decimals like 2.1 while avoiding trailing .0
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case int:
		return strconv.Itoa(t), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case json.Number:
		return t.String(), true
	default:
		if v == nil {
			return "", false
		}
		// Fallback stringification
		s := fmt.Sprintf("%v", v)
		if s == "<nil>" {
			return "", false
		}
		return s, true
	}
}

// --- HDHR parsing ---

// Keep the original (typo) name for compatibility.
func makeInteraceFromHDHR(content []byte, playlistName, id string) (channels []interface{}, err error) {
	return makeInterfaceFromHDHR(content, playlistName, id)
}

// Preferred spelling; use this from new call sites.
func makeInterfaceFromHDHR(content []byte, playlistName, id string) (channels []interface{}, err error) {
	// Decode into a generic slice so we tolerate providers returning mixed types.
	var raw []map[string]interface{}
	if err = json.Unmarshal(content, &raw); err != nil {
		return nil, err
	}

	channels = make([]interface{}, 0, len(raw))

	uuidKey := "ID-" + id
	for _, item := range raw {
		guideName, okGN := toString(item["GuideName"])
		url, okURL := toString(item["URL"])
		guideNumber, okNum := toString(item["GuideNumber"])

		// Skip invalid entries, but don’t hard-fail the whole lineup.
		if !okGN || !okURL || !okNum || guideName == "" || url == "" || guideNumber == "" {
			// Optionally log:
			// showDebug("HDHR: skipping invalid item (missing GuideName/URL/GuideNumber)", 1)
			continue
		}

		ch := map[string]string{
			"group-title": playlistName,
			"name":        guideName,
			"tvg-id":      guideName,
			"url":         url,
			uuidKey:       guideNumber,                    // provider-unique number
			"_uuid.key":   uuidKey,                        // tells the filter/compat which field is the UUID
			"_values":     playlistName + " " + guideName, // used by filters/search
		}
		channels = append(channels, ch)
	}

	return channels, nil
}

// --- Device capability / discovery ---

func getCapability() (xmlContent []byte, err error) {
	var cap Capability
	var buf bytes.Buffer

	cap.Xmlns = "urn:schemas-upnp-org:device-1-0"
	cap.URLBase = System.ServerProtocol.WEB + "://" + System.Domain

	cap.SpecVersion.Major = 1
	cap.SpecVersion.Minor = 0

	cap.Device.DeviceType = "urn:schemas-upnp-org:device:MediaServer:1"
	cap.Device.FriendlyName = System.Name
	cap.Device.Manufacturer = "Silicondust"
	cap.Device.ModelName = "HDTC-2US"
	cap.Device.ModelNumber = "HDTC-2US"
	cap.Device.SerialNumber = ""
	cap.Device.UDN = "uuid:" + System.DeviceID

	output, mErr := xml.MarshalIndent(cap, " ", "  ")
	if mErr != nil {
		ShowError(mErr, 1003)
		return nil, mErr
	}

	buf.WriteString(xml.Header)
	buf.Write(output)
	return buf.Bytes(), nil
}

func getDiscover() (jsonContent []byte, err error) {
	var d Discover

	d.BaseURL = System.ServerProtocol.WEB + "://" + System.Domain
	d.DeviceAuth = System.AppName
	d.DeviceID = System.DeviceID
	d.FirmwareName = "bin_" + System.Version
	d.FirmwareVersion = System.Version
	d.FriendlyName = System.Name
	d.LineupURL = fmt.Sprintf("%s://%s/lineup.json", System.ServerProtocol.DVR, System.Domain)
	d.Manufacturer = "Golang"
	d.ModelNumber = System.Version
	d.TunerCount = Settings.Tuner

	return json.MarshalIndent(d, "", "  ")
}

func getLineupStatus() (jsonContent []byte, err error) {
	var ls LineupStatus
	ls.ScanInProgress = 0
	ls.ScanPossible = 0
	ls.Source = "Cable"
	ls.SourceList = []string{"Cable"}
	return json.MarshalIndent(ls, "", "  ")
}

// --- Lineup generation ---

func getLineup() (jsonContent []byte, err error) {
	var lineup Lineup

	switch Settings.EpgSource {

	case "PMS":
		// Build lineup directly from currently active streams.
		for _, dsa := range Data.Streams.Active {
			var ch M3UChannelStructXEPG
			if err = json.Unmarshal([]byte(mapToJSON(dsa)), &ch); err != nil {
				return nil, err
			}

			var stream LineupStream
			stream.GuideName = ch.Name

			// Stable per-name PMS ID (persisted in Data.Cache.PMS)
			guideNumber, gErr := getGuideNumberPMS(stream.GuideName)
			if gErr != nil || guideNumber == "" {
				// Provide a sensible fallback if PMS cache can’t be read
				guideNumber = "1000"
			}
			stream.GuideNumber = guideNumber

			stream.URL, err = createStreamingURL("DVR",
				ch.FileM3UID,
				stream.GuideNumber,
				ch.Name,
				ch.URL,
				nil, nil, nil,
			)
			if err != nil {
				ShowError(err, 1202)
				continue
			}
			lineup = append(lineup, stream)
		}

	case "XEPG":
		for _, dxc := range Data.XEPG.Channels {
			var xc XEPGChannelStruct
			if err = json.Unmarshal([]byte(mapToJSON(dxc)), &xc); err != nil {
				return nil, err
			}

			if !xc.XActive || xc.XHideChannel {
				continue
			}

			var stream LineupStream
			stream.GuideName = xc.XName
			stream.GuideNumber = xc.XChannelID

			stream.URL, err = createStreamingURL(
				"DVR",
				xc.FileM3UID,
				xc.XChannelID,
				xc.XName,
				xc.URL,
				xc.BackupChannel1,
				xc.BackupChannel2,
				xc.BackupChannel3,
			)
			if err != nil {
				ShowError(err, 1202)
				continue
			}
			lineup = append(lineup, stream)
		}
	}

	jsonContent, err = json.MarshalIndent(lineup, "", "  ")
	if err != nil {
		return nil, err
	}

	// Best effort cache write; don’t fail the HTTP response on error.
	if err := saveMapToJSONFile(System.File.URLS, Data.Cache.StreamingURLS); err != nil {
		ShowError(err, 0)
	}
	// Reset PMS cache in memory after lineup build
	Data.Cache.PMS = nil

	return jsonContent, nil
}

// --- PMS guide number mapping ---

func getGuideNumberPMS(channelName string) (pmsID string, err error) {
	// Lazily load PMS cache once
	if len(Data.Cache.PMS) == 0 {
		Data.Cache.PMS = make(map[string]string)

		pms, loadErr := loadJSONFileToMap(System.File.PMS)
		if loadErr != nil {
			// If the PMS file doesn’t exist yet, that’s OK; we’ll create it on write.
			// Return empty pmsID and the error so caller can decide on fallback.
			return "", loadErr
		}

		for k, v := range pms {
			if s, ok := v.(string); ok {
				Data.Cache.PMS[k] = s
			}
		}
	}

	// Generate a new unique id like id-0, id-1, ...
	nextID := func() string {
		// Build a set of current IDs for quick lookup
		used := make(map[string]struct{}, len(Data.Cache.PMS))
		for _, v := range Data.Cache.PMS {
			used[v] = struct{}{}
		}
		for i := 0; ; i++ {
			id := fmt.Sprintf("id-%d", i)
			if _, exists := used[id]; !exists {
				return id
			}
		}
	}

	// Normalize channel name key a bit to avoid accidental drift
	key := strings.TrimSpace(channelName)

	if existing, ok := Data.Cache.PMS[key]; ok && existing != "" {
		return existing, nil
	}

	newID := nextID()
	Data.Cache.PMS[key] = newID
	if err := saveMapToJSONFile(System.File.PMS, Data.Cache.PMS); err != nil {
		// Persisting failed; log but still return the new ID so the caller can proceed.
		ShowError(err, 0)
	}
	return newID, nil
}

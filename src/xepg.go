package src

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"bufio"
	"threadfin/src/internal/imgcache"
	_ "time/tzdata"
)

// Provider XMLTV Datei überprüfen
func checkXMLCompatibility(id string, body []byte) (err error) {

	var xmltv XMLTV
	var compatibility = make(map[string]int)

	err = xml.Unmarshal(body, &xmltv)
	if err != nil {
		return
	}

	compatibility["xmltv.channels"] = len(xmltv.Channel)
	compatibility["xmltv.programs"] = len(xmltv.Program)

	setProviderCompatibility(id, "xmltv", compatibility)

	return
}

var buildXEPGCount int

// XEPG Daten erstellen
func buildXEPG(background bool) {
	xepgMutex.Lock()
	defer func() {
		xepgMutex.Unlock()
	}()

	if System.ScanInProgress == 1 {
		return
	}

	System.ScanInProgress = 1
	// Enter maintenance during core steps

	// Clear streaming URL cache
	Data.Cache.StreamingURLS = make(map[string]StreamInfo)
	saveMapToJSONFile(System.File.URLS, Data.Cache.StreamingURLS)

	var err error

	Data.Cache.Images, err = imgcache.New(System.Folder.ImagesCache, fmt.Sprintf("%s://%s/images/", System.ServerProtocol.WEB, System.Domain), Settings.CacheImages)
	if err != nil {
		ShowError(err, 0)
	}

	if Settings.EpgSource == "XEPG" {

		switch background {

		case true:

			go func() {

				createXEPGMapping()
				createXEPGDatabase()
				mapping()
				cleanupXEPG()
				createXMLTVFile()
				createM3UFile()

				showInfo("XEPG:" + fmt.Sprintf("Ready to use"))

				if Settings.CacheImages && System.ImageCachingInProgress == 0 {

					go func() {

						systemMutex.Lock()
						System.ImageCachingInProgress = 1
						systemMutex.Unlock()

						showInfo(fmt.Sprintf("Image Caching:Images are cached (%d)", len(Data.Cache.Images.Queue)))

						Data.Cache.Images.Image.Caching()
						Data.Cache.Images.Image.Remove()
						showInfo("Image Caching:Done")

						createXMLTVFile()
						createM3UFile()

						systemMutex.Lock()
						System.ImageCachingInProgress = 0
						systemMutex.Unlock()

					}()

				}

				// Core work is done; exit maintenance
				systemMutex.Lock()
				System.ScanInProgress = 0
				systemMutex.Unlock()

				// Cache löschen
				Data.Cache.XMLTV = make(map[string]XMLTV)
				runtime.GC()

			}()

		case false:

			createXEPGMapping()
			createXEPGDatabase()
			mapping()
			cleanupXEPG()
			createXMLTVFile()
			createM3UFile()

			// Exit maintenance before long file generation to keep UI responsive
			System.ScanInProgress = 0

			go func() {

				if Settings.CacheImages && System.ImageCachingInProgress == 0 {

					go func() {

						systemMutex.Lock()
						System.ImageCachingInProgress = 1
						systemMutex.Unlock()

						showInfo(fmt.Sprintf("Image Caching:Images are cached (%d)", len(Data.Cache.Images.Queue)))

						Data.Cache.Images.Image.Caching()
						Data.Cache.Images.Image.Remove()
						showInfo("Image Caching:Done")

						createXMLTVFile()
						createM3UFile()

						systemMutex.Lock()
						System.ImageCachingInProgress = 0
						systemMutex.Unlock()

					}()

				}

				showInfo("XEPG:" + fmt.Sprintf("Ready to use"))

				// Cache löschen
				Data.Cache.XMLTV = make(map[string]XMLTV)
				runtime.GC()

			}()

		}

	} else {

		getLineup()
		System.ScanInProgress = 0

	}

}

// Update XEPG data
func updateXEPG(background bool) {

	if System.ScanInProgress == 1 {
		return
	}

	System.ScanInProgress = 1

	if Settings.EpgSource == "XEPG" {

		switch background {

		case false:

			createXEPGDatabase()
			mapping()
			cleanupXEPG()

			// Exit maintenance before long file generation to keep UI responsive
			System.ScanInProgress = 0

			go func() {

				createXMLTVFile()
				createM3UFile()
				showInfo("XEPG:" + fmt.Sprintf("Ready to use"))

			}()

		case true:
			System.ScanInProgress = 0

		}

	} else {

		System.ScanInProgress = 0

	}

	// Cache löschen
	Data.Cache.XMLTV = make(map[string]XMLTV)

	return
}

// Mapping Menü für die XMLTV Dateien erstellen
func createXEPGMapping() {
	Data.XMLTV.Files = getLocalProviderFiles("xmltv")
	Data.XMLTV.Mapping = make(map[string]interface{})

	var tmpMap = make(map[string]interface{})

	var friendlyDisplayName = func(channel Channel) (displayName string) {
		var dn = channel.DisplayName
		if len(dn) > 0 {
			switch len(dn) {
			case 1:
				displayName = dn[0].Value
			default:
				displayName = fmt.Sprintf("%s (%s)", dn[0].Value, dn[1].Value)
			}
		}

		return
	}

	if len(Data.XMLTV.Files) > 0 {

		// For multiple large files, process in parallel for better performance
		if len(Data.XMLTV.Files) > 1 {
			showInfo("XEPG:" + fmt.Sprintf("Processing %d XMLTV files in parallel", len(Data.XMLTV.Files)))
		}

		for i := len(Data.XMLTV.Files) - 1; i >= 0; i-- {

			var file = Data.XMLTV.Files[i]

			var err error
			var fileID = strings.TrimSuffix(getFilenameFromPath(file), path.Ext(getFilenameFromPath(file)))
			showInfo("XEPG:" + "Parse XMLTV file: " + getProviderParameter(fileID, "xmltv", "name"))

			//xmltv, err = getLocalXMLTV(file)
			var xmltv XMLTV
			err = getLocalXMLTV(file, &xmltv)
			if err != nil {
				Data.XMLTV.Files = append(Data.XMLTV.Files, Data.XMLTV.Files[i+1:]...)
				var errMsg = err.Error()
				err = errors.New(getProviderParameter(fileID, "xmltv", "name") + ": " + errMsg)
				ShowError(err, 000)
			}

			// XML Parsen (Provider Datei)
			if err == nil {
				var imgc = Data.Cache.Images
				// Daten aus der XML Datei in eine temporäre Map schreiben
				var xmltvMap = make(map[string]interface{}, len(xmltv.Channel)) // Pre-allocate

				for _, c := range xmltv.Channel {
					var channel = make(map[string]interface{}, 4) // Pre-allocate

					channel["id"] = c.ID
					channel["display-name"] = friendlyDisplayName(*c)
					channel["icon"] = imgc.Image.GetURL(c.Icon.Src, Settings.HttpThreadfinDomain, Settings.Port, Settings.ForceHttps, Settings.HttpsPort, Settings.HttpsThreadfinDomain)
					channel["active"] = c.Active

					xmltvMap[c.ID] = channel

				}

				tmpMap[getFilenameFromPath(file)] = xmltvMap
				Data.XMLTV.Mapping[getFilenameFromPath(file)] = xmltvMap

			}

		}

		Data.XMLTV.Mapping = tmpMap
		tmpMap = make(map[string]interface{})

	} else {

		if System.ConfigurationWizard == false {
			showWarning(1007)
		}

	}

	// Auswahl für den Dummy erstellen
	var dummy = make(map[string]interface{})
	var times = []string{"30", "60", "90", "120", "180", "240", "360", "PPV"}

	for _, i := range times {

		var dummyChannel = make(map[string]string)
		if i == "PPV" {
			dummyChannel["display-name"] = "PPV Event"
			dummyChannel["id"] = "PPV"
		} else {
			dummyChannel["display-name"] = i + " Minutes"
			dummyChannel["id"] = i + "_Minutes"
		}
		dummyChannel["icon"] = ""

		dummy[dummyChannel["id"]] = dummyChannel

	}

	Data.XMLTV.Mapping["Threadfin Dummy"] = dummy

	return
}

// XEPG Datenbank erstellen / aktualisieren
func createXEPGDatabase() (err error) {

	var allChannelNumbers = make([]float64, 0, System.UnfilteredChannelLimit)
	Data.Cache.Streams.Active = make([]string, 0, System.UnfilteredChannelLimit)
	Data.XEPG.Channels = make(map[string]interface{}, System.UnfilteredChannelLimit)

	// Clear streaming URL cache
	Data.Cache.StreamingURLS = make(map[string]StreamInfo)
	saveMapToJSONFile(System.File.URLS, Data.Cache.StreamingURLS)

	Data.Cache.Streams.Active = make([]string, 0, System.UnfilteredChannelLimit)
	Settings = SettingsStruct{}
	Data.XEPG.Channels, err = loadJSONFileToMap(System.File.XEPG)
	if err != nil {
		ShowError(err, 1004)
		return err
	}

	settings, err := loadJSONFileToMap(System.File.Settings)
	if err != nil || len(settings) == 0 {
		return
	}
	settings_json, _ := json.Marshal(settings)
	json.Unmarshal(settings_json, &Settings)

	// Remove duplicate channels from existing XEPG database based on new hash logic
	removeDuplicateChannels()

	// Get current M3U channels
	m3uChannels := make(map[string]M3UChannelStructXEPG)
	for _, dsa := range Data.Streams.Active {
		var m3uChannel M3UChannelStructXEPG
		err = json.Unmarshal([]byte(mapToJSON(dsa)), &m3uChannel)
		if err == nil {
			// Use tvg-id as the key for matching channels
			key := m3uChannel.TvgID
			if key == "" {
				key = m3uChannel.TvgName
			}
			m3uChannels[key] = m3uChannel
		}
	}

	// Update URLs in XEPG database
	for id, dxc := range Data.XEPG.Channels {
		var xepgChannel XEPGChannelStruct
		err = json.Unmarshal([]byte(mapToJSON(dxc)), &xepgChannel)
		if err == nil {
			// Find matching M3U channel using tvg-id or tvg-name
			key := xepgChannel.TvgID
			if key == "" {
				key = xepgChannel.TvgName
			}
			if m3uChannel, ok := m3uChannels[key]; ok {
				// Always update URL if it's different
				if xepgChannel.URL != m3uChannel.URL {
					xepgChannel.URL = m3uChannel.URL
					Data.XEPG.Channels[id] = xepgChannel
				}
			}
		}
	}

	// Save updated XEPG database
	err = saveMapToJSONFile(System.File.XEPG, Data.XEPG.Channels)
	if err != nil {
		ShowError(err, 000)
		return err
	}

	var createNewID = func() (xepg string) {

		var firstID = 0 //len(Data.XEPG.Channels)

	newXEPGID:

		if _, ok := Data.XEPG.Channels["x-ID."+strconv.FormatInt(int64(firstID), 10)]; ok {
			firstID++
			goto newXEPGID
		}

		xepg = "x-ID." + strconv.FormatInt(int64(firstID), 10)
		return
	}

	var getFreeChannelNumber = func(startingNumber float64) (xChannelID string) {

		sort.Float64s(allChannelNumbers)

		for {

			if indexOfFloat64(startingNumber, allChannelNumbers) == -1 {
				xChannelID = fmt.Sprintf("%g", startingNumber)
				allChannelNumbers = append(allChannelNumbers, startingNumber)
				return
			}

			startingNumber++

		}
	}

	showInfo("XEPG:" + "Update database")

	// Kanal mit fehlenden Kanalnummern löschen.  Delete channel with missing channel numbers
	for id, dxc := range Data.XEPG.Channels {

		var xepgChannel XEPGChannelStruct
		err = json.Unmarshal([]byte(mapToJSON(dxc)), &xepgChannel)
		if err != nil {
			return
		}

		if len(xepgChannel.XChannelID) == 0 {
			delete(Data.XEPG.Channels, id)
		}

		if xChannelID, err := strconv.ParseFloat(xepgChannel.XChannelID, 64); err == nil {
			allChannelNumbers = append(allChannelNumbers, xChannelID)
		}

	}

	// Make a map of the db channels based on their previously downloaded attributes -- filename, group, title, etc
	var xepgChannelsValuesMap = make(map[string]XEPGChannelStruct, System.UnfilteredChannelLimit)
	for _, v := range Data.XEPG.Channels {
		var channel XEPGChannelStruct
		err = json.Unmarshal([]byte(mapToJSON(v)), &channel)
		if err != nil {
			return
		}

		if channel.TvgName == "" {
			channel.TvgName = channel.Name
		}

		// Create consistent channel hash using URL as primary identifier
		// Each unique URL should create a separate channel, even if tvg-id/name are similar (backup channels)
		hashInput := channel.URL + channel.TvgName + channel.FileM3UID
		hash := md5.Sum([]byte(hashInput))
		channelHash := hex.EncodeToString(hash[:])
		xepgChannelsValuesMap[channelHash] = channel
	}

	for _, dsa := range Data.Streams.Active {
		var channelExists = false  // Entscheidet ob ein Kanal neu zu Datenbank hinzugefügt werden soll.  Decides whether a channel should be added to the database
		var channelHasUUID = false // Überprüft, ob der Kanal (Stream) eindeutige ID's besitzt.  Checks whether the channel (stream) has unique IDs
		var currentXEPGID string   // Aktuelle Datenbank ID (XEPG). Wird verwendet, um den Kanal in der Datenbank mit dem Stream der M3u zu aktualisieren. Current database ID (XEPG) Used to update the channel in the database with the stream of the M3u
		var currentChannelNumber string

		var m3uChannel M3UChannelStructXEPG

		err = json.Unmarshal([]byte(mapToJSON(dsa)), &m3uChannel)
		if err != nil {
			return
		}

		if m3uChannel.TvgName == "" {
			m3uChannel.TvgName = m3uChannel.Name
		}

		// Try to find the channel based on matching all known values.  If that fails, then move to full channel scan
		// Create consistent channel hash using URL as primary identifier
		// Each unique URL should create a separate channel, even if tvg-id/name are similar (backup channels)
		hashInput := m3uChannel.URL + m3uChannel.TvgName + m3uChannel.FileM3UID
		hash := md5.Sum([]byte(hashInput))
		m3uChannelHash := hex.EncodeToString(hash[:])

		Data.Cache.Streams.Active = append(Data.Cache.Streams.Active, m3uChannelHash)

		if val, ok := xepgChannelsValuesMap[m3uChannelHash]; ok {
			channelExists = true
			currentXEPGID = val.XEPG
			currentChannelNumber = val.TvgChno
			if len(m3uChannel.UUIDValue) > 0 {
				channelHasUUID = true
			}
		} else {
			// XEPG Datenbank durchlaufen um nach dem Kanal zu suchen.  Run through the XEPG database to search for the channel (full scan)
			for _, dxc := range xepgChannelsValuesMap {
				if m3uChannel.FileM3UID == dxc.FileM3UID && !isInInactiveList(dxc.URL) {

					dxc.FileM3UID = m3uChannel.FileM3UID
					dxc.FileM3UName = m3uChannel.FileM3UName

					// Vergleichen des Streams anhand einer UUID in der M3U mit dem Kanal in der Databank.  Compare the stream using a UUID in the M3U with the channel in the database
					if len(dxc.UUIDValue) > 0 && len(m3uChannel.UUIDValue) > 0 {
						if dxc.UUIDValue == m3uChannel.UUIDValue {

							channelExists = true
							channelHasUUID = true
							currentXEPGID = dxc.XEPG
							currentChannelNumber = dxc.TvgChno
							break

						}
					}
				}

			}
		}

		switch channelExists {

		case true:
			// Bereits vorhandener Kanal
			var xepgChannel XEPGChannelStruct
			err = json.Unmarshal([]byte(mapToJSON(Data.XEPG.Channels[currentXEPGID])), &xepgChannel)
			if err != nil {
				return
			}

			// Update existing channel - since we found it via hash, it's the same logical channel
			if xepgChannel.TvgName == "" {
				xepgChannel.TvgName = xepgChannel.Name
			}

			xepgChannel.XChannelID = currentChannelNumber
			xepgChannel.TvgChno = currentChannelNumber

			// Always update streaming URL
			xepgChannel.URL = m3uChannel.URL

			// Update Live Event status
			if m3uChannel.LiveEvent == "true" {
				xepgChannel.Live = true
			}

			// Update the ChannelUniqueID to new hash value
			xepgChannel.ChannelUniqueID = m3uChannelHash

			// Update channel name - for Live Events, allow name updates even without UUID
			if channelHasUUID {
				programData, _ := getProgramData(xepgChannel)
				if xepgChannel.XUpdateChannelName || strings.Contains(xepgChannel.TvgID, "threadfin-") || (m3uChannel.LiveEvent == "true" && len(programData.Program) <= 3) {
					xepgChannel.XName = m3uChannel.Name
					xepgChannel.TvgName = m3uChannel.TvgName // Also update TvgName for Live Events
				}
			} else if m3uChannel.LiveEvent == "true" {
				// For Live Events without UUID, still allow name updates since they change frequently
				xepgChannel.XName = m3uChannel.Name
				xepgChannel.TvgName = m3uChannel.TvgName
			}

			// Update channel logo
			if xepgChannel.XUpdateChannelIcon {
				var imgc = Data.Cache.Images
				xepgChannel.TvgLogo = imgc.Image.GetURL(m3uChannel.TvgLogo, Settings.HttpThreadfinDomain, Settings.Port, Settings.ForceHttps, Settings.HttpsPort, Settings.HttpsThreadfinDomain)
			}

			Data.XEPG.Channels[currentXEPGID] = xepgChannel

		case false:
			// Neuer Kanal
			var firstFreeNumber float64 = Settings.MappingFirstChannel
			// Check channel start number from Group Filter
			filters := []FilterStruct{}
			for _, filter := range Settings.Filter {
				filter_json, _ := json.Marshal(filter)
				f := FilterStruct{}
				json.Unmarshal(filter_json, &f)
				filters = append(filters, f)
			}

			for _, filter := range filters {
				if m3uChannel.GroupTitle == filter.Filter {
					start_num, _ := strconv.ParseFloat(filter.StartingNumber, 64)
					firstFreeNumber = start_num
				}
			}

			var xepg = createNewID()
			var xChannelID string

			if m3uChannel.TvgChno == "" {
				xChannelID = getFreeChannelNumber(firstFreeNumber)
			} else {
				xChannelID = m3uChannel.TvgChno
			}

			var newChannel XEPGChannelStruct
			newChannel.FileM3UID = m3uChannel.FileM3UID
			newChannel.FileM3UName = m3uChannel.FileM3UName
			newChannel.FileM3UPath = m3uChannel.FileM3UPath
			newChannel.Values = m3uChannel.Values
			newChannel.GroupTitle = m3uChannel.GroupTitle
			newChannel.Name = m3uChannel.Name
			newChannel.TvgID = m3uChannel.TvgID
			newChannel.TvgLogo = m3uChannel.TvgLogo
			newChannel.TvgName = m3uChannel.TvgName
			newChannel.URL = m3uChannel.URL
			newChannel.Live, _ = strconv.ParseBool(m3uChannel.LiveEvent)

			for file, xmltvChannels := range Data.XMLTV.Mapping {
				channelsMap, ok := xmltvChannels.(map[string]interface{})
				if !ok {
					continue
				}
				if channel, ok := channelsMap[m3uChannel.TvgID]; ok {
					filters := []FilterStruct{}
					for _, filter := range Settings.Filter {
						filter_json, _ := json.Marshal(filter)
						f := FilterStruct{}
						json.Unmarshal(filter_json, &f)
						filters = append(filters, f)
					}
					for _, filter := range filters {
						if newChannel.GroupTitle == filter.Filter {
							category := &Category{}
							category.Value = filter.Category
							category.Lang = "en"
							newChannel.XCategory = filter.Category
						}
					}

					chmap, okk := channel.(map[string]interface{})
					if !okk {
						continue
					}

					if channelID, ok := chmap["id"].(string); ok {
						newChannel.XmltvFile = file
						newChannel.XMapping = channelID
						newChannel.XActive = true

						// Falls in der XMLTV Datei ein Logo existiert, wird dieses verwendet. Falls nicht, dann das Logo aus der M3U Datei
						/*if icon, ok := chmap["icon"].(string); ok {
							if len(icon) > 0 {
								newChannel.TvgLogo = icon
							}
						}*/

						break

					}

				}

			}

			programData, _ := getProgramData(newChannel)

			if newChannel.Live && len(programData.Program) <= 3 {
				newChannel.XmltvFile = "Threadfin Dummy"
				newChannel.XMapping = "PPV"
				newChannel.XActive = true
			}

			if len(m3uChannel.UUIDKey) > 0 {
				newChannel.UUIDKey = m3uChannel.UUIDKey
				newChannel.UUIDValue = m3uChannel.UUIDValue
			} else {
				newChannel.UUIDKey = ""
				newChannel.UUIDValue = ""
			}

			newChannel.XName = m3uChannel.Name
			newChannel.XGroupTitle = m3uChannel.GroupTitle
			newChannel.XEPG = xepg
			newChannel.TvgChno = xChannelID
			newChannel.XChannelID = xChannelID
			newChannel.ChannelUniqueID = m3uChannelHash
			Data.XEPG.Channels[xepg] = newChannel
			xepgChannelsValuesMap[m3uChannelHash] = newChannel

		}
	}

	showInfo("XEPG:" + "Save DB file")

	err = saveMapToJSONFile(System.File.XEPG, Data.XEPG.Channels)
	if err != nil {
		return
	}

	return
}

// Kanäle automatisch zuordnen und das Mapping überprüfen
func mapping() (err error) {
	showInfo("XEPG:" + "Map channels")

	for xepg, dxc := range Data.XEPG.Channels {

		var xepgChannel XEPGChannelStruct
		err = json.Unmarshal([]byte(mapToJSON(dxc)), &xepgChannel)
		if err != nil {
			return
		}

		if xepgChannel.TvgName == "" {
			xepgChannel.TvgName = xepgChannel.Name
		}

		if (xepgChannel.XBackupChannel1 != "" && xepgChannel.XBackupChannel1 != "-") || (xepgChannel.XBackupChannel2 != "" && xepgChannel.XBackupChannel2 != "-") || (xepgChannel.XBackupChannel3 != "" && xepgChannel.XBackupChannel3 != "-") {
			for _, stream := range Data.Streams.Active {
				var m3uChannel M3UChannelStructXEPG

				err = json.Unmarshal([]byte(mapToJSON(stream)), &m3uChannel)
				if err != nil {
					return err
				}

				if m3uChannel.TvgName == "" {
					m3uChannel.TvgName = m3uChannel.Name
				}

				backup_channel1 := strings.Trim(xepgChannel.XBackupChannel1, " ")
				if m3uChannel.TvgName == backup_channel1 {
					xepgChannel.BackupChannel1 = &BackupStream{PlaylistID: m3uChannel.FileM3UID, URL: m3uChannel.URL}
				}

				backup_channel2 := strings.Trim(xepgChannel.XBackupChannel2, " ")
				if m3uChannel.TvgName == backup_channel2 {
					xepgChannel.BackupChannel2 = &BackupStream{PlaylistID: m3uChannel.FileM3UID, URL: m3uChannel.URL}
				}

				backup_channel3 := strings.Trim(xepgChannel.XBackupChannel3, " ")
				if m3uChannel.TvgName == backup_channel3 {
					xepgChannel.BackupChannel3 = &BackupStream{PlaylistID: m3uChannel.FileM3UID, URL: m3uChannel.URL}
				}
			}
		}

		// Automatische Mapping für neue Kanäle. Wird nur ausgeführt, wenn der Kanal deaktiviert ist und keine XMLTV Datei und kein XMLTV Kanal zugeordnet ist.
		if !xepgChannel.XActive {
			// Werte kann "-" sein, deswegen len < 1
			if len(xepgChannel.XmltvFile) < 1 {

				var tvgID = xepgChannel.TvgID

				xepgChannel.XmltvFile = "-"
				xepgChannel.XMapping = "-"

				Data.XEPG.Channels[xepg] = xepgChannel
				for file, xmltvChannels := range Data.XMLTV.Mapping {
					channelsMap, ok := xmltvChannels.(map[string]interface{})
					if !ok {
						continue
					}
					if channel, ok := channelsMap[tvgID]; ok {

						filters := []FilterStruct{}
						for _, filter := range Settings.Filter {
							filter_json, _ := json.Marshal(filter)
							f := FilterStruct{}
							json.Unmarshal(filter_json, &f)
							filters = append(filters, f)
						}
						for _, filter := range filters {
							if xepgChannel.GroupTitle == filter.Filter {
								category := &Category{}
								category.Value = filter.Category
								category.Lang = "en"
								xepgChannel.XCategory = filter.Category
							}
						}

						chmap, okk := channel.(map[string]interface{})
						if !okk {
							continue
						}

						if channelID, ok := chmap["id"].(string); ok {
							xepgChannel.XmltvFile = file
							xepgChannel.XMapping = channelID
							xepgChannel.XActive = true

							// Falls in der XMLTV Datei ein Logo existiert, wird dieses verwendet. Falls nicht, dann das Logo aus der M3U Datei
							/*if icon, ok := chmap["icon"].(string); ok {
								if len(icon) > 0 {
									xepgChannel.TvgLogo = icon
								}
							}*/

							Data.XEPG.Channels[xepg] = xepgChannel
							break

						}

					}

				}
			}
		}

		// Überprüfen, ob die zugeordneten XMLTV Dateien und Kanäle noch existieren.
		if xepgChannel.XActive && !xepgChannel.XHideChannel {

			var mapping = xepgChannel.XMapping
			var file = xepgChannel.XmltvFile

			if file != "Threadfin Dummy" && !xepgChannel.Live {

				if value, ok := Data.XMLTV.Mapping[file].(map[string]interface{}); ok {

					if channel, ok := value[mapping].(map[string]interface{}); ok {

						filters := []FilterStruct{}
						for _, filter := range Settings.Filter {
							filter_json, _ := json.Marshal(filter)
							f := FilterStruct{}
							json.Unmarshal(filter_json, &f)
							filters = append(filters, f)
						}
						for _, filter := range filters {
							if xepgChannel.GroupTitle == filter.Filter {
								category := &Category{}
								category.Value = filter.Category
								category.Lang = "en"
								if xepgChannel.XCategory == "" {
									xepgChannel.XCategory = filter.Category
								}
							}
						}

						// Kanallogo aktualisieren
						if logo, ok := channel["icon"].(string); ok {

							if xepgChannel.XUpdateChannelIcon && len(logo) > 0 {
								/*var imgc = Data.Cache.Images
								xepgChannel.TvgLogo = imgc.Image.GetURL(logo, Settings.HttpThreadfinDomain, Settings.Port, Settings.ForceHttps, Settings.HttpsPort, Settings.HttpsThreadfinDomain)*/
							}

						}

					}

				}

			} else {
				// Loop through dummy channels and assign the filter info
				filters := []FilterStruct{}
				for _, filter := range Settings.Filter {
					filter_json, _ := json.Marshal(filter)
					f := FilterStruct{}
					json.Unmarshal(filter_json, &f)
					filters = append(filters, f)
				}
				for _, filter := range filters {
					if xepgChannel.GroupTitle == filter.Filter {
						category := &Category{}
						category.Value = filter.Category
						category.Lang = "en"
						if xepgChannel.XCategory == "" {
							xepgChannel.XCategory = filter.Category
						}
					}
				}
			}
			if len(xepgChannel.XmltvFile) == 0 {
				xepgChannel.XmltvFile = "-"
				xepgChannel.XActive = true
			}

			if len(xepgChannel.XMapping) == 0 {
				xepgChannel.XMapping = "-"
				xepgChannel.XActive = true
			}

			Data.XEPG.Channels[xepg] = xepgChannel

		}

	}

	err = saveMapToJSONFile(System.File.XEPG, Data.XEPG.Channels)
	if err != nil {
		return
	}

	return
}

// XMLTV Datei erstellen
func createXMLTVFile() (err error) {

	// Image Cache
	// 4edd81ab7c368208cc6448b615051b37.jpg
	var imgc = Data.Cache.Images

	Data.Cache.ImagesFiles = []string{}
	Data.Cache.ImagesURLS = []string{}
	Data.Cache.ImagesCache = []string{}

	files, err := os.ReadDir(System.Folder.ImagesCache)
	if err == nil {

		for _, file := range files {

			if indexOfString(file.Name(), Data.Cache.ImagesCache) == -1 {
				Data.Cache.ImagesCache = append(Data.Cache.ImagesCache, file.Name())
			}

		}

	}

	if len(Data.XMLTV.Files) == 0 && len(Data.Streams.Active) == 0 {
		Data.XEPG.Channels = make(map[string]interface{})
		return
	}

	showInfo("XEPG:" + fmt.Sprintf("Create XMLTV file (%s)", System.File.XML))

	// Stream XML to disk to avoid huge memory usage
	xmlFile, err := os.Create(System.File.XML)
	if err != nil {
		return err
	}
	defer xmlFile.Close()
	writer := bufio.NewWriterSize(xmlFile, 1<<20) // 1MB buffer
	defer writer.Flush()

	var xepgXML XMLTV

	xepgXML.Generator = System.Name

	if System.Branch == "main" {
		xepgXML.Source = fmt.Sprintf("%s - %s", System.Name, System.Version)
	} else {
		xepgXML.Source = fmt.Sprintf("%s - %s.%s", System.Name, System.Version, System.Build)
	}

	var tmpProgram = &XMLTV{}

	// Start writing XML header and open tags
	if _, err = writer.WriteString(xml.Header); err != nil {
		return err
	}
	if _, err = writer.WriteString("<tv>\n"); err != nil {
		return err
	}

	// Write generator/source
	if _, err = writer.WriteString(fmt.Sprintf("  <generator>%s</generator>\n", xepgXML.Generator)); err != nil {
		return err
	}
	if _, err = writer.WriteString(fmt.Sprintf("  <source>%s</source>\n", xepgXML.Source)); err != nil {
		return err
	}

	// Channels and programs
	for _, dxc := range Data.XEPG.Channels {
		var xepgChannel XEPGChannelStruct
		err := json.Unmarshal([]byte(mapToJSON(dxc)), &xepgChannel)
		if err == nil {
			if xepgChannel.TvgName == "" {
				xepgChannel.TvgName = xepgChannel.Name
			}
			if xepgChannel.XName == "" {
				xepgChannel.XName = xepgChannel.TvgName
			}

			if xepgChannel.XActive && !xepgChannel.XHideChannel {
				if (Settings.XepgReplaceChannelTitle && xepgChannel.XMapping == "PPV") || xepgChannel.XName != "" {
					// Write channel entry
					channel := Channel{ID: xepgChannel.XChannelID, Icon: Icon{Src: imgc.Image.GetURL(xepgChannel.TvgLogo, Settings.HttpThreadfinDomain, Settings.Port, Settings.ForceHttps, Settings.HttpsPort, Settings.HttpsThreadfinDomain)}, DisplayName: []DisplayName{{Value: xepgChannel.XName}}, Active: xepgChannel.XActive, Live: xepgChannel.Live}
					bytes, _ := xml.MarshalIndent(channel, "  ", "    ")
					if _, err = writer.Write(bytes); err != nil {
						return err
					}
					if _, err = writer.WriteString("\n"); err != nil {
						return err
					}
				}

				// Programme
				*tmpProgram, err = getProgramData(xepgChannel)
				if err == nil {
					for _, p := range tmpProgram.Program {
						bytes, _ := xml.MarshalIndent(p, "  ", "    ")
						if _, err = writer.Write(bytes); err != nil {
							return err
						}
						if _, err = writer.WriteString("\n"); err != nil {
							return err
						}
					}
				}
			}
		} else {
			showDebug("XEPG:"+fmt.Sprintf("Error: %s", err), 3)
		}
	}

	// Close tv root
	if _, err = writer.WriteString("</tv>\n"); err != nil {
		return err
	}

	showInfo("XEPG:" + fmt.Sprintf("Compress XMLTV file (%s)", System.Compressed.GZxml))
	// Streaming file compression
	if err = compressGZIPFile(System.File.XML, System.Compressed.GZxml); err != nil {
		return err
	}

	return
}

// Programmdaten erstellen (createXMLTVFile)
func getProgramData(xepgChannel XEPGChannelStruct) (xepgXML XMLTV, err error) {
	var xmltvFile = System.Folder.Data + xepgChannel.XmltvFile
	var channelID = xepgChannel.XMapping

	var xmltv XMLTV

	if strings.Contains(xmltvFile, "Threadfin Dummy") {
		xmltv = createDummyProgram(xepgChannel)
	} else {
		if xepgChannel.XmltvFile != "" {
			err = getLocalXMLTV(xmltvFile, &xmltv)
			if err != nil {
				return
			}
		}
	}

	for _, xmltvProgram := range xmltv.Program {
		if xmltvProgram.Channel == channelID {
			var program = &Program{}

			// Channel ID
			program.Channel = xepgChannel.XChannelID
			program.Start = xmltvProgram.Start
			program.Stop = xmltvProgram.Stop

			// Title
			if len(xmltvProgram.Title) > 0 {
				if !Settings.EnableNonAscii {
					xmltvProgram.Title[0].Value = strings.TrimSpace(strings.Map(func(r rune) rune {
						if r > unicode.MaxASCII {
							return -1
						}
						return r
					}, xmltvProgram.Title[0].Value))
				}
				program.Title = xmltvProgram.Title
			}

			filters := []FilterStruct{}
			for _, filter := range Settings.Filter {
				filter_json, _ := json.Marshal(filter)
				f := FilterStruct{}
				json.Unmarshal(filter_json, &f)
				filters = append(filters, f)
			}

			// Category (Kategorie)
			getCategory(program, xmltvProgram, xepgChannel, filters)

			// Sub-Title
			program.SubTitle = xmltvProgram.SubTitle

			// Description
			program.Desc = xmltvProgram.Desc

			// Credits : (Credits)
			program.Credits = xmltvProgram.Credits

			// Rating (Bewertung)
			program.Rating = xmltvProgram.Rating

			// StarRating (Bewertung / Kritiken)
			program.StarRating = xmltvProgram.StarRating

			// Country (Länder)
			program.Country = xmltvProgram.Country

			// Program icon (Poster / Cover)
			getPoster(program, xmltvProgram, xepgChannel, Settings.ForceHttps)

			// Language (Sprache)
			program.Language = xmltvProgram.Language

			// Episodes numbers (Episodennummern)
			getEpisodeNum(program, xmltvProgram, xepgChannel)

			// Video (Videoparameter)
			getVideo(program, xmltvProgram, xepgChannel)

			// Date (Datum)
			program.Date = xmltvProgram.Date

			// Previously shown (Wiederholung)
			program.PreviouslyShown = xmltvProgram.PreviouslyShown

			// New (Neu)
			program.New = xmltvProgram.New

			// Live
			program.Live = xmltvProgram.Live

			// Premiere
			program.Premiere = xmltvProgram.Premiere

			xepgXML.Program = append(xepgXML.Program, program)

		}

	}

	return
}

func createLiveProgram(xepgChannel XEPGChannelStruct, channelId string) []*Program {
	var programs []*Program

	var currentTime = time.Now()
	localLocation := currentTime.Location() // Central Time (CT)

	startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, currentTime.Nanosecond(), localLocation)
	stopTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, currentTime.Nanosecond(), localLocation)

	name := ""
	if xepgChannel.XName != "" {
		name = xepgChannel.XName
	} else {
		name = xepgChannel.TvgName
	}

	// Search for Datetime or Time
	// Datetime examples: '12/31-11:59 PM', '1.1 6:30 AM', '09/15-10:00PM', '7/4 12:00 PM', '3.21 3:45 AM', '6/30-8:00 AM', '4/15 3AM'
	// Time examples: '11:59 PM', '6:30 AM', '11:59PM', '1PM'
	re := regexp.MustCompile(`((\d{1,2}[./]\d{1,2})[-\s])*(\d{1,2}(:\d{2})*\s*(AM|PM)?(?:\s*(ET|CT|MT|PT|EST|CST|MST|PST))?)`)
	matches := re.FindStringSubmatch(name)
	layout := "2006.1.2 3:04 PM"
	if len(matches) > 0 {
		timePart := matches[len(matches)-2]
		if timePart == "" {
			timePart = matches[len(matches)-1]
		}

		timeString := strings.TrimSpace(timePart)
		timeString = strings.ReplaceAll(timeString, "  ", " ")

		// Handle timezone if present
		var location *time.Location
		if strings.Contains(timeString, "ET") || strings.Contains(timeString, "EST") {
			location, _ = time.LoadLocation("America/New_York")
		} else if strings.Contains(timeString, "CT") || strings.Contains(timeString, "CST") {
			location, _ = time.LoadLocation("America/Chicago")
		} else if strings.Contains(timeString, "MT") || strings.Contains(timeString, "MST") {
			location, _ = time.LoadLocation("America/Denver")
		} else if strings.Contains(timeString, "PT") || strings.Contains(timeString, "PST") {
			location, _ = time.LoadLocation("America/Los_Angeles")
		} else {
			location = currentTime.Location()
		}

		// Remove timezone from timeString
		timeString = strings.ReplaceAll(timeString, "ET", "")
		timeString = strings.ReplaceAll(timeString, "CT", "")
		timeString = strings.ReplaceAll(timeString, "MT", "")
		timeString = strings.ReplaceAll(timeString, "PT", "")
		timeString = strings.ReplaceAll(timeString, "EST", "")
		timeString = strings.ReplaceAll(timeString, "CST", "")
		timeString = strings.ReplaceAll(timeString, "MST", "")
		timeString = strings.ReplaceAll(timeString, "PST", "")
		timeString = strings.TrimSpace(timeString)

		// Handle different date formats
		var datePart string
		if len(matches) > 3 && matches[2] != "" {
			datePart = matches[2]
			// Convert slashes to dots for consistency
			datePart = strings.ReplaceAll(datePart, "/", ".")
		}

		// Build the full time string
		var fullTimeString string
		if datePart != "" {
			// If we have a date part, use it
			parts := strings.Split(datePart, ".")
			if len(parts) == 2 {
				month := parts[0]
				day := parts[1]
				fullTimeString = fmt.Sprintf("%d.%s.%s %s", currentTime.Year(), month, day, timeString)
			}
		} else {
			// If no date part, use current date
			fullTimeString = fmt.Sprintf("%d.%d.%d %s", currentTime.Year(), currentTime.Month(), currentTime.Day(), timeString)
		}

		// Determine layout based on time format
		if strings.Contains(timeString, ":") {
			if strings.Contains(timeString, "AM") || strings.Contains(timeString, "PM") {
				layout = "2006.1.2 3:04 PM"
			} else {
				layout = "2006.1.2 15:04"
			}
		} else {
			if strings.Contains(timeString, "AM") || strings.Contains(timeString, "PM") {
				layout = "2006.1.2 3PM"
			} else {
				layout = "2006.1.2 15"
			}
		}

		startTimeParsed, err := time.ParseInLocation(layout, fullTimeString, location)
		if err != nil {
			startTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 6, 0, 0, 0, location)
		} else {
			localTime := startTimeParsed.In(localLocation)
			startTime = localTime
		}
	}

	// Add "CHANNEL OFFLINE" program for the time before the event
	beginningOfDay := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, localLocation)

	// Handle non-ASCII characters in offline text
	var offlineText = "CHANNEL OFFLINE"
	if !Settings.EnableNonAscii {
		offlineText = strings.TrimSpace(strings.Map(func(r rune) rune {
			if r > unicode.MaxASCII {
				return -1
			}
			return r
		}, offlineText))
	}

	programBefore := &Program{
		Channel: channelId,
		Start:   beginningOfDay.Format("20060102150405 -0700"),
		Stop:    startTime.Format("20060102150405 -0700"),
		Title:   []*Title{{Lang: "en", Value: offlineText}},
		Desc:    []*Desc{{Lang: "en", Value: offlineText}},
	}
	programs = append(programs, programBefore)

	// Add the main program
	mainProgram := &Program{
		Channel: channelId,
		Start:   startTime.Format("20060102150405 -0700"),
		Stop:    stopTime.Format("20060102150405 -0700"),
	}

	if Settings.XepgReplaceChannelTitle && xepgChannel.XMapping == "PPV" {
		title := []*Title{}
		title_parsed := fmt.Sprintf("%s %s", name, xepgChannel.XPpvExtra)

		// Handle non-ASCII characters in title
		if !Settings.EnableNonAscii {
			title_parsed = strings.TrimSpace(strings.Map(func(r rune) rune {
				if r > unicode.MaxASCII {
					return -1
				}
				return r
			}, title_parsed))
		}

		t := &Title{Lang: "en", Value: title_parsed}
		title = append(title, t)
		mainProgram.Title = title

		desc := []*Desc{}
		d := &Desc{Lang: "en", Value: title_parsed}
		desc = append(desc, d)
		mainProgram.Desc = desc
	}
	programs = append(programs, mainProgram)

	// Add "CHANNEL OFFLINE" program for the time after the event
	midnightNextDayStart := time.Date(stopTime.Year(), stopTime.Month(), stopTime.Day()+1, 0, 0, 0, currentTime.Nanosecond(), localLocation)
	midnightNextDayStop := time.Date(stopTime.Year(), stopTime.Month(), stopTime.Day()+1, 23, 59, 59, currentTime.Nanosecond(), localLocation)
	programAfter := &Program{
		Channel: channelId,
		Start:   midnightNextDayStart.Format("20060102150405 -0700"),
		Stop:    midnightNextDayStop.Format("20060102150405 -0700"),
		Title:   []*Title{{Lang: "en", Value: offlineText}},
		Desc:    []*Desc{{Lang: "en", Value: offlineText}},
	}
	programs = append(programs, programAfter)

	return programs
}

// Dummy Daten erstellen (createXMLTVFile)
func createDummyProgram(xepgChannel XEPGChannelStruct) (dummyXMLTV XMLTV) {
	if xepgChannel.XMapping == "PPV" {
		var channelID = xepgChannel.XMapping
		programs := createLiveProgram(xepgChannel, channelID)
		dummyXMLTV.Program = programs
		return
	}

	var imgc = Data.Cache.Images
	var currentTime = time.Now()
	var dateArray = strings.Fields(currentTime.String())
	var offset = " " + dateArray[2]
	var currentDay = currentTime.Format("20060102")
	var startTime, _ = time.Parse("20060102150405", currentDay+"000000")

	showInfo("Create Dummy Guide:" + "Time offset" + offset + " - " + xepgChannel.XName)

	var dummyLength int = 30 // Default to 30 minutes if parsing fails
	var err error
	var dl = strings.Split(xepgChannel.XMapping, "_")
	if dl[0] != "" {
		// Check if the first part is a valid integer
		if match, _ := regexp.MatchString(`^\d+$`, dl[0]); match {
			dummyLength, err = strconv.Atoi(dl[0])
			if err != nil {
				ShowError(err, 000)
				// Continue with default value instead of returning
			}
		} else {
			// For non-numeric formats that aren't "PPV" (which is handled above),
			// use the default value
			showInfo(fmt.Sprintf("Non-numeric format for XMapping: %s, using default duration of 30 minutes", xepgChannel.XMapping))
		}
	}

	for d := 0; d < 4; d++ {

		var epgStartTime = startTime.Add(time.Hour * time.Duration(d*24))

		for t := dummyLength; t <= 1440; t = t + dummyLength {

			var epgStopTime = epgStartTime.Add(time.Minute * time.Duration(dummyLength))

			var epg Program
			poster := Poster{}

			epg.Channel = xepgChannel.XMapping
			epg.Start = epgStartTime.Format("20060102150405") + offset
			epg.Stop = epgStopTime.Format("20060102150405") + offset

			// Create title with proper handling of non-ASCII characters
			var titleValue = xepgChannel.XName + " (" + epgStartTime.Weekday().String()[0:2] + ". " + epgStartTime.Format("15:04") + " - " + epgStopTime.Format("15:04") + ")"
			if !Settings.EnableNonAscii {
				titleValue = strings.TrimSpace(strings.Map(func(r rune) rune {
					if r > unicode.MaxASCII {
						return -1
					}
					return r
				}, titleValue))
			}
			epg.Title = append(epg.Title, &Title{Value: titleValue, Lang: "en"})

			if len(xepgChannel.XDescription) == 0 {
				var descValue = "Threadfin: (" + strconv.Itoa(dummyLength) + " Minutes) " + epgStartTime.Weekday().String() + " " + epgStartTime.Format("15:04") + " - " + epgStopTime.Format("15:04")
				if !Settings.EnableNonAscii {
					descValue = strings.TrimSpace(strings.Map(func(r rune) rune {
						if r > unicode.MaxASCII {
							return -1
						}
						return r
					}, descValue))
				}
				epg.Desc = append(epg.Desc, &Desc{Value: descValue, Lang: "en"})
			} else {
				var descValue = xepgChannel.XDescription
				if !Settings.EnableNonAscii {
					descValue = strings.TrimSpace(strings.Map(func(r rune) rune {
						if r > unicode.MaxASCII {
							return -1
						}
						return r
					}, descValue))
				}
				epg.Desc = append(epg.Desc, &Desc{Value: descValue, Lang: "en"})
			}

			if Settings.XepgReplaceMissingImages {
				poster.Src = imgc.Image.GetURL(xepgChannel.TvgLogo, Settings.HttpThreadfinDomain, Settings.Port, Settings.ForceHttps, Settings.HttpsPort, Settings.HttpsThreadfinDomain)
				epg.Poster = append(epg.Poster, poster)
			}

			if xepgChannel.XCategory != "Movie" {
				epg.EpisodeNum = append(epg.EpisodeNum, &EpisodeNum{Value: epgStartTime.Format("2006-01-02 15:04:05"), System: "original-air-date"})
			}

			epg.New = &New{Value: ""}

			dummyXMLTV.Program = append(dummyXMLTV.Program, &epg)
			epgStartTime = epgStopTime

		}

	}

	return
}

// Kategorien erweitern (createXMLTVFile)
func getCategory(program *Program, xmltvProgram *Program, xepgChannel XEPGChannelStruct, filters []FilterStruct) {

	for _, i := range xmltvProgram.Category {

		category := &Category{}
		category.Value = i.Value
		category.Lang = i.Lang
		program.Category = append(program.Category, category)

	}

	if len(xepgChannel.XCategory) > 0 {

		category := &Category{}
		category.Value = strings.ToLower(xepgChannel.XCategory)
		category.Lang = "en"
		program.Category = append(program.Category, category)

	}
}

// Programm Poster Cover aus der XMLTV Datei laden
func getPoster(program *Program, xmltvProgram *Program, xepgChannel XEPGChannelStruct, forceHttps bool) {

	var imgc = Data.Cache.Images

	for _, poster := range xmltvProgram.Poster {
		poster.Src = imgc.Image.GetURL(poster.Src, Settings.HttpThreadfinDomain, Settings.Port, Settings.ForceHttps, Settings.HttpsPort, Settings.HttpsThreadfinDomain)
		program.Poster = append(program.Poster, poster)
	}

	if Settings.XepgReplaceMissingImages {

		if len(xmltvProgram.Poster) == 0 {
			var poster Poster
			poster.Src = imgc.Image.GetURL(xepgChannel.TvgLogo, Settings.HttpThreadfinDomain, Settings.Port, Settings.ForceHttps, Settings.HttpsPort, Settings.HttpsThreadfinDomain)
			program.Poster = append(program.Poster, poster)
		}

	}

}

// Episodensystem übernehmen, falls keins vorhanden ist und eine Kategorie im Mapping eingestellt wurden, wird eine Episode erstellt
func getEpisodeNum(program *Program, xmltvProgram *Program, xepgChannel XEPGChannelStruct) {

	program.EpisodeNum = xmltvProgram.EpisodeNum

	if len(xepgChannel.XCategory) > 0 && xepgChannel.XCategory != "Movie" {

		if len(xmltvProgram.EpisodeNum) == 0 {

			var timeLayout = "20060102150405"

			t, err := time.Parse(timeLayout, strings.Split(xmltvProgram.Start, " ")[0])
			if err == nil {
				program.EpisodeNum = append(program.EpisodeNum, &EpisodeNum{Value: t.Format("2006-01-02 15:04:05"), System: "original-air-date"})
			} else {
				ShowError(err, 0)
			}

		}

	}

	return
}

// Videoparameter erstellen (createXMLTVFile)
func getVideo(program *Program, xmltvProgram *Program, xepgChannel XEPGChannelStruct) {

	var video Video
	video.Present = xmltvProgram.Video.Present
	video.Colour = xmltvProgram.Video.Colour
	video.Aspect = xmltvProgram.Video.Aspect
	video.Quality = xmltvProgram.Video.Quality

	if len(xmltvProgram.Video.Quality) == 0 {

		if strings.Contains(strings.ToUpper(xepgChannel.XName), " HD") || strings.Contains(strings.ToUpper(xepgChannel.XName), " FHD") {
			video.Quality = "HDTV"
		}

		if strings.Contains(strings.ToUpper(xepgChannel.XName), " UHD") || strings.Contains(strings.ToUpper(xepgChannel.XName), " 4K") {
			video.Quality = "UHDTV"
		}

	}

	program.Video = video

	return
}

// Lokale Provider XMLTV Datei laden
func getLocalXMLTV(file string, xmltv *XMLTV) (err error) {

	if _, ok := Data.Cache.XMLTV[file]; !ok {

		// Cache initialisieren
		if len(Data.Cache.XMLTV) == 0 {
			Data.Cache.XMLTV = make(map[string]XMLTV)
		}

		// Check file size to determine parsing strategy
		fileInfo, err := os.Stat(file)
		if err != nil {
			err = errors.New("Local copy of the file no longer exists")
			return err
		}

		// For large files (>50MB), use streaming parser
		if fileInfo.Size() > 50*1024*1024 {
			showInfo("XEPG:" + "Using streaming parser for large XMLTV file: " + file)
			err = parseXMLTVStream(file, xmltv)
		} else {
			// Use original method for smaller files
			content, err := readByteFromFile(file)
			if err != nil {
				err = errors.New("Local copy of the file no longer exists")
				return err
			}

			// XML Datei parsen
			err = xml.Unmarshal(content, &xmltv)
		}

		if err != nil {
			return err
		}

		Data.Cache.XMLTV[file] = *xmltv

	} else {
		*xmltv = Data.Cache.XMLTV[file]
	}

	return
}

// parseXMLTVStream : Streaming XML parser for large XMLTV files
func parseXMLTVStream(file string, xmltv *XMLTV) error {
	xmlFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer xmlFile.Close()

	decoder := xml.NewDecoder(xmlFile)

	// Pre-allocate slices with reasonable capacity
	xmltv.Channel = make([]*Channel, 0, 10000)   // Expect ~10k channels
	xmltv.Program = make([]*Program, 0, 100000)  // Expect ~100k programs

	var currentElement string
	var channelCount, programCount int

	for {
		token, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}

		switch se := token.(type) {
		case xml.StartElement:
			currentElement = se.Name.Local

			switch currentElement {
			case "tv":
				// Parse TV attributes if needed
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "generator-info-name":
						xmltv.Generator = attr.Value
					case "source-info-name":
						xmltv.Source = attr.Value
					}
				}

			case "channel":
				// Parse channel element
				var channel Channel
				if err := decoder.DecodeElement(&channel, &se); err != nil {
					showDebug("XMLTV Stream:Error parsing channel: "+err.Error(), 2)
					continue
				}
				xmltv.Channel = append(xmltv.Channel, &channel)
				channelCount++

				// Log progress for large channel counts
				if channelCount%1000 == 0 {
					showInfo(fmt.Sprintf("XMLTV Stream:Parsed %d channels", channelCount))
				}

			case "programme":
				// Parse program element
				var program Program
				if err := decoder.DecodeElement(&program, &se); err != nil {
					showDebug("XMLTV Stream:Error parsing program: "+err.Error(), 3)
					continue
				}
				xmltv.Program = append(xmltv.Program, &program)
				programCount++

				// Log progress for large program counts
				if programCount%10000 == 0 {
					showInfo(fmt.Sprintf("XMLTV Stream:Parsed %d programs", programCount))
				}
			}
		}
	}

	showInfo(fmt.Sprintf("XMLTV Stream:Completed - %d channels, %d programs", channelCount, programCount))
	return nil
}

func isInInactiveList(channelURL string) bool {
	for _, channel := range Data.Streams.Inactive {
		// Type assert channel to map[string]interface{}
		chMap, ok := channel.(map[string]interface{})
		if !ok {
			continue
		}

		urlValue, exists := chMap["url"]
		if !exists {
			continue
		}

		urlStr, ok := urlValue.(string)
		if !ok {
			continue
		}

		if urlStr == channelURL {
			return true
		}
	}
	return false
}

// M3U Datei erstellen
func createM3UFile() {

	showInfo("XEPG:" + fmt.Sprintf("Create M3U file (%s)", System.File.M3U))
	_, err := buildM3U([]string{})
	if err != nil {
		ShowError(err, 000)
	}

	saveMapToJSONFile(System.File.URLS, Data.Cache.StreamingURLS)

	return
}

// XEPG Datenbank bereinigen
func cleanupXEPG() {

	var sourceIDs []string

	for source := range Settings.Files.M3U {
		sourceIDs = append(sourceIDs, source)
	}

	for source := range Settings.Files.HDHR {
		sourceIDs = append(sourceIDs, source)
	}

	showInfo("XEPG:" + fmt.Sprintf("Cleanup database"))
	Data.XEPG.XEPGCount = 0

	for id, dxc := range Data.XEPG.Channels {

		var xepgChannel XEPGChannelStruct
		err := json.Unmarshal([]byte(mapToJSON(dxc)), &xepgChannel)
		if err == nil {

			if xepgChannel.TvgName == "" {
				xepgChannel.TvgName = xepgChannel.Name
			}

			// Create consistent channel hash using URL as primary identifier
			// Each unique URL should create a separate channel, even if tvg-id/name are similar (backup channels)
			hashInput := xepgChannel.URL + xepgChannel.TvgName + xepgChannel.FileM3UID
			hash := md5.Sum([]byte(hashInput))
			m3uChannelHash := hex.EncodeToString(hash[:])

			if indexOfString(m3uChannelHash, Data.Cache.Streams.Active) == -1 {
				delete(Data.XEPG.Channels, id)
			} else {
				if xepgChannel.XActive && !xepgChannel.XHideChannel {
					Data.XEPG.XEPGCount++
				}
			}

			if indexOfString(xepgChannel.FileM3UID, sourceIDs) == -1 {
				delete(Data.XEPG.Channels, id)
			}

		}

	}

	err := saveMapToJSONFile(System.File.XEPG, Data.XEPG.Channels)
	if err != nil {
		ShowError(err, 000)
		return
	}

	showInfo("XEPG Channels:" + fmt.Sprintf("%d", Data.XEPG.XEPGCount))

	if len(Data.Streams.Active) > 0 && Data.XEPG.XEPGCount == 0 {
		showWarning(2005)
	}

	return
}

// Remove duplicate channels from XEPG database using consistent hash logic
func removeDuplicateChannels() {
	showInfo("XEPG:" + "Remove duplicate channels")

	// Track channels by hash to identify exact duplicates (same URL + name + source)
	hashToChannelID := make(map[string]string)
	var channelsToRemove []string
	var duplicatesFound int

	for id, dxc := range Data.XEPG.Channels {
		var xepgChannel XEPGChannelStruct
		err := json.Unmarshal([]byte(mapToJSON(dxc)), &xepgChannel)
		if err != nil {
			continue
		}

		if xepgChannel.TvgName == "" {
			xepgChannel.TvgName = xepgChannel.Name
		}

		// Create consistent channel hash using URL as primary identifier
		// Each unique URL should create a separate channel, even if tvg-id/name are similar (backup channels)
		hashInput := xepgChannel.URL + xepgChannel.TvgName + xepgChannel.FileM3UID
		hash := md5.Sum([]byte(hashInput))
		channelHash := hex.EncodeToString(hash[:])

		// Check for hash-based duplicates (exact same content)
		if existingChannelID, exists := hashToChannelID[channelHash]; exists {
			channelsToRemove = append(channelsToRemove, handleDuplicate(id, existingChannelID, "hash"))
			duplicatesFound++
		} else {
			hashToChannelID[channelHash] = id
		}

		// DISABLED: Name-based duplicate removal - backup channels with different URLs are legitimate
		// Only remove true duplicates (exact same URL + tvg-id + source) via hash-based detection
		//
		// Note: Channels like "NFL RedZone (1)", "NFL RedZone (2)", "NFL RedZone (3)" with same tvg-id
		// but different URLs are backup channels and should be preserved, not treated as duplicates
	}

	// Remove duplicate channels
	for _, channelID := range channelsToRemove {
		delete(Data.XEPG.Channels, channelID)
	}

	if duplicatesFound > 0 {
		showInfo(fmt.Sprintf("XEPG:Removed %d duplicate channels", duplicatesFound))
		// Save the cleaned database
		err := saveMapToJSONFile(System.File.XEPG, Data.XEPG.Channels)
		if err != nil {
			ShowError(err, 000)
		}
	} else {
		showInfo("XEPG:No duplicate channels found")
	}
}

// Helper function to clean channel names for duplicate detection
func cleanChannelNameForDuplicateDetection(name string) string {
	// Remove backup indicators like (1), (2), etc.
	re := regexp.MustCompile(`\s*\([0-9]+\)\s*$`)
	cleaned := re.ReplaceAllString(name, "")

	// Remove extra whitespace
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// Helper function to determine if a channel should be removed as name duplicate
func shouldRemoveAsNameDuplicate(currentID, existingID string) bool {
	currentChannel := getChannelByID(currentID)
	existingChannel := getChannelByID(existingID)

	if currentChannel == nil || existingChannel == nil {
		return false
	}

	// Don't remove if they're in different groups
	if currentChannel.XGroupTitle != existingChannel.XGroupTitle {
		return false
	}

	// Prefer active channels
	if currentChannel.XActive && !existingChannel.XActive {
		return false // Keep current, remove existing (handled elsewhere)
	}
	if !currentChannel.XActive && existingChannel.XActive {
		return true // Remove current, keep existing
	}

	// Prefer channels with XMLTV mapping
	currentHasMapping := currentChannel.XmltvFile != "" && currentChannel.XmltvFile != "-"
	existingHasMapping := existingChannel.XmltvFile != "" && existingChannel.XmltvFile != "-"

	if currentHasMapping && !existingHasMapping {
		return false // Keep current
	}
	if !currentHasMapping && existingHasMapping {
		return true // Remove current
	}

	// If everything else is equal, keep the one with lower channel number
	currentChno, err1 := strconv.ParseFloat(currentChannel.TvgChno, 64)
	existingChno, err2 := strconv.ParseFloat(existingChannel.TvgChno, 64)

	if err1 == nil && err2 == nil {
		return currentChno > existingChno // Remove current if it has higher channel number
	}

	// Default: remove current (keep existing)
	return true
}

// Helper function to get channel by ID
func getChannelByID(id string) *XEPGChannelStruct {
	if dxc, exists := Data.XEPG.Channels[id]; exists {
		var channel XEPGChannelStruct
		if err := json.Unmarshal([]byte(mapToJSON(dxc)), &channel); err == nil {
			return &channel
		}
	}
	return nil
}

// Helper function to handle duplicate channel removal
func handleDuplicate(currentID, existingID, duplicateType string) string {
	currentChannel := getChannelByID(currentID)
	existingChannel := getChannelByID(existingID)

	if currentChannel == nil || existingChannel == nil {
		return currentID // Default to removing current
	}

	// Prefer active channels over inactive ones
	if currentChannel.XActive && !existingChannel.XActive {
		showInfo(fmt.Sprintf("XEPG:Removing %s duplicate %s (%s), keeping %s (%s)",
			duplicateType, existingID, existingChannel.XName, currentID, currentChannel.XName))
		return existingID
	} else if !currentChannel.XActive && existingChannel.XActive {
		showInfo(fmt.Sprintf("XEPG:Removing %s duplicate %s (%s), keeping %s (%s)",
			duplicateType, currentID, currentChannel.XName, existingID, existingChannel.XName))
		return currentID
	}

	// Both have same active status, prefer one with XMLTV mapping
	currentHasMapping := currentChannel.XmltvFile != "" && currentChannel.XmltvFile != "-"
	existingHasMapping := existingChannel.XmltvFile != "" && existingChannel.XmltvFile != "-"

	if currentHasMapping && !existingHasMapping {
		showInfo(fmt.Sprintf("XEPG:Removing %s duplicate %s (%s), keeping %s (%s)",
			duplicateType, existingID, existingChannel.XName, currentID, currentChannel.XName))
		return existingID
	} else if !currentHasMapping && existingHasMapping {
		showInfo(fmt.Sprintf("XEPG:Removing %s duplicate %s (%s), keeping %s (%s)",
			duplicateType, currentID, currentChannel.XName, existingID, existingChannel.XName))
		return currentID
	}

	// Prefer the one created later (larger XEPG ID suggests later creation)
	if currentID > existingID {
		showInfo(fmt.Sprintf("XEPG:Removing %s duplicate %s (%s), keeping %s (%s)",
			duplicateType, existingID, existingChannel.XName, currentID, currentChannel.XName))
		return existingID
	} else {
		showInfo(fmt.Sprintf("XEPG:Removing %s duplicate %s (%s), keeping %s (%s)",
			duplicateType, currentID, currentChannel.XName, existingID, existingChannel.XName))
		return currentID
	}
}

// Streaming URL für die Channels App generieren
func getStreamByChannelID(channelID string) (playlistID, streamURL string, err error) {

	err = errors.New("Channel not found")

	for _, dxc := range Data.XEPG.Channels {

		var xepgChannel XEPGChannelStruct
		err := json.Unmarshal([]byte(mapToJSON(dxc)), &xepgChannel)

		fmt.Println(xepgChannel.XChannelID)

		if err == nil {

			if xepgChannel.TvgName == "" {
				xepgChannel.TvgName = xepgChannel.Name
			}

			if channelID == xepgChannel.XChannelID {

				playlistID = xepgChannel.FileM3UID
				streamURL = xepgChannel.URL

				return playlistID, streamURL, nil
			}

		}

	}

	return
}

package src

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	m3u "threadfin/src/internal/m3u-parser"
)

var (
	reInclude = regexp.MustCompile(`\{([^}]+)\}`)  // {foo}
	reExclude = regexp.MustCompile(`!\{([^}]+)\}`) // !{bar}
)

func buildStreamSearchValue(s map[string]string, caseSensitive bool) string {
	gt := s["group-title"]
	name := s["name"]
	tvg := s["tvg-id"]
	val := strings.TrimSpace(strings.Join([]string{gt, name, tvg}, " "))
	if !caseSensitive {
		val = strings.ToLower(val)
	}
	return val
}

// Playlisten parsen
func parsePlaylist(filename, fileType string) (channels []interface{}, err error) {
	content, err := readByteFromFile(filename)
	if err != nil {
		return nil, err
	}

	id := strings.TrimSuffix(getFilenameFromPath(filename), path.Ext(getFilenameFromPath(filename)))
	playlistName := getProviderParameter(id, fileType, "name")

	switch fileType {
	case "m3u":
		channels, err = m3u.MakeInterfaceFromM3U(content)
	case "hdhr":
		channels, err = makeInteraceFromHDHR(content, playlistName, id)
	}
	return
}

// Streams filtern (immutable: does not mutate FilterStruct.Rule)
func filterThisStream(s interface{}) (status bool, liveEvent bool) {
	stream := s.(map[string]string)

	// Gather searchable text
	values := strings.Replace(stream["_values"], "\r", "", -1)

	group := ""
	if v, ok := stream["group-title"]; ok {
		group = v
	}
	name := ""
	if v, ok := stream["name"]; ok {
		name = v
	}

	reYES := regexp.MustCompile(`\{[^}]+\}`) // e.g. {DEU}
	reNO := regexp.MustCompile(`!\{[^}]+\}`) // e.g. !{DEU}

	for _, f := range Data.Filter {
		ruleText := f.Rule
		if ruleText == "" {
			continue
		}

		liveEvent = f.LiveEvent

		// Extract include/exclude without mutating f.Rule
		var include, exclude string
		if m := reNO.FindString(ruleText); m != "" {
			exclude = m[2 : len(m)-1] // strip "!{ }"
			ruleText = strings.Replace(ruleText, m, "", -1)
			ruleText = strings.TrimSpace(ruleText)
		}
		if m := reYES.FindString(ruleText); m != "" {
			include = m[1 : len(m)-1] // strip "{ }"
			ruleText = strings.Replace(ruleText, m, "", -1)
			ruleText = strings.TrimSpace(ruleText)
		}
		if ruleText == "" {
			continue
		}

		// Case sensitivity
		searchValues := values
		searchGroup := group
		searchName := name
		searchRule := ruleText
		searchInc := include
		searchExc := exclude
		if !f.CaseSensitive {
			searchValues = strings.ToLower(searchValues)
			searchGroup = strings.ToLower(searchGroup)
			searchName = strings.ToLower(searchName)
			searchRule = strings.ToLower(searchRule)
			searchInc = strings.ToLower(searchInc)
			searchExc = strings.ToLower(searchExc)
		}

		// Match by type
		var match bool
		switch f.Type {
		case "group-title":
			// When type is group-title, we match the RULE against the GROUP,
			// but we still show the channel NAME in UI.
			match = (searchGroup == searchRule)
		case "custom-filter":
			match = strings.Contains(searchValues, searchRule)
		default:
			match = false
		}

		if !match {
			continue
		}

		// Exclude/Include conditions (comma-separated)
		if searchExc != "" && strings.Contains(searchValues, searchExc) {
			continue
		}
		if searchInc != "" && !strings.Contains(searchValues, searchInc) {
			continue
		}

		return true, liveEvent
	}

	return false, false
}

// Bedingungen für den Filter (kept for compatibility; not used by new filter)
func checkConditions(streamValues, conditions, coType string) (status bool) {
	switch coType {
	case "exclude":
		status = true
	case "include":
		status = false
	}

	conditions = strings.Replace(conditions, ", ", ",", -1)
	conditions = strings.Replace(conditions, " ,", ",", -1)
	keys := strings.Split(conditions, ",")

	for _, key := range keys {
		if strings.Contains(streamValues, key) {
			switch coType {
			case "exclude":
				return false
			case "include":
				return true
			}
		}
	}
	return
}

// Threadfin M3U Datei erstellen
func buildM3U(groups []string) (m3u string, err error) {
	imgc := Data.Cache.Images
	m3uChannels := make(map[float64]XEPGChannelStruct)
	var channelNumbers []float64

	for _, dxc := range Data.XEPG.Channels {
		var xepgChannel XEPGChannelStruct
		if err := json.Unmarshal([]byte(mapToJSON(dxc)), &xepgChannel); err == nil {
			channelNumber, e := strconv.ParseFloat(strings.TrimSpace(xepgChannel.XChannelID), 64)

			if xepgChannel.TvgName == "" {
				xepgChannel.TvgName = xepgChannel.Name
			}
			if xepgChannel.XActive && !xepgChannel.XHideChannel {
				if len(groups) > 0 && indexOfString(xepgChannel.XGroupTitle, groups) == -1 {
					continue
				}
				if e == nil {
					m3uChannels[channelNumber] = xepgChannel
					channelNumbers = append(channelNumbers, channelNumber)
				}
			}
		}
	}

	// Sort channel numbers
	sort.Float64s(channelNumbers)

	// Header
	xmltvURL := fmt.Sprintf("%s://%s/xmltv/threadfin.xml", System.ServerProtocol.XML, System.Domain)
	if Settings.ForceHttps && Settings.HttpsThreadfinDomain != "" {
		xmltvURL = fmt.Sprintf("https://%s/xmltv/threadfin.xml", Settings.HttpsThreadfinDomain)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `#EXTM3U url-tvg="%s" x-tvg-url="%s"`+"\n", xmltvURL, xmltvURL)

	// Deduplicate with a stable key instead of O(n^2) string scanning
	seen := make(map[string]struct{})

	for _, channelNumber := range channelNumbers {
		ch := m3uChannels[channelNumber]

		group := ch.XGroupTitle
		if ch.XCategory != "" {
			group = ch.XCategory
		}

		if Settings.ForceHttps && Settings.HttpsThreadfinDomain != "" {
			if u, err := url.Parse(ch.URL); err == nil {
				u.Scheme = "https"
				hostSplit := strings.Split(u.Host, ":")
				if len(hostSplit) > 0 {
					u.Host = hostSplit[0]
				}
				if u.RawQuery != "" {
					ch.URL = fmt.Sprintf("https://%s:%d%s?%s", u.Host, Settings.HttpsPort, u.Path, u.RawQuery)
				} else {
					ch.URL = fmt.Sprintf("https://%s:%d%s", u.Host, Settings.HttpsPort, u.Path)
				}
			}
		}

		logo := ""
		if ch.TvgLogo != "" && imgc != nil {
			logo = imgc.Image.GetURL(
				ch.TvgLogo,
				Settings.HttpThreadfinDomain,
				Settings.Port,
				Settings.ForceHttps,
				Settings.HttpsPort,
				Settings.HttpsThreadfinDomain,
			)
		}

		custom := ""
		if ch.CustomTags != "" {
			custom = " " + ch.CustomTags
		}

		param := fmt.Sprintf(
			`#EXTINF:0 channelID="%s" tvg-chno="%s" tvg-name="%s" tvg-id="%s" tvg-logo="%s" group-title="%s"%s,%s`+"\n",
			ch.XEPG, ch.XChannelID, ch.XName, ch.XChannelID, logo, group, custom, ch.XName,
		)

		stream, err := createStreamingURL("M3U", ch.FileM3UID, ch.XChannelID, ch.XName, ch.URL, ch.BackupChannel1, ch.BackupChannel2, ch.BackupChannel3)
		if err != nil {
			continue
		}

		key := ch.FileM3UID + "|" + ch.XChannelID
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		sb.WriteString(param)
		sb.WriteString(stream)
		sb.WriteByte('\n')
	}

	m3u = sb.String()

	if len(groups) == 0 {
		filename := System.Folder.Data + "threadfin.m3u"
		if werr := writeByteToFile(filename, []byte(m3u)); werr != nil {
			return m3u, werr
		}
	}

	return
}

func probeChannel(request RequestStruct) (string, string, string, error) {
	ffmpegPath := Settings.FFmpegPath
	ffprobePath := strings.Replace(ffmpegPath, "ffmpeg", "ffprobe", 1)

	cmd := exec.Command(ffprobePath, "-v", "error", "-show_streams", "-of", "json", request.ProbeURL)
	output, err := cmd.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to execute ffprobe: %v", err)
	}

	var ffprobeOutput FFProbeOutput
	if err := json.Unmarshal(output, &ffprobeOutput); err != nil {
		return "", "", "", fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	var resolution, frameRate, audioChannels string
	for _, stream := range ffprobeOutput.Streams {
		if stream.CodecType == "video" {
			resolution = fmt.Sprintf("%dp", stream.Height)
			parts := strings.Split(stream.RFrameRate, "/")
			if len(parts) == 2 {
				frameRate = fmt.Sprintf("%d", parseFrameRate(parts))
			} else {
				frameRate = stream.RFrameRate
			}
		}
		if stream.CodecType == "audio" {
			audioChannels = stream.ChannelLayout
			if audioChannels == "" {
				switch stream.Channels {
				case 1:
					audioChannels = "Mono"
				case 2:
					audioChannels = "Stereo"
				case 6:
					audioChannels = "5.1"
				case 8:
					audioChannels = "7.1"
				default:
					audioChannels = fmt.Sprintf("%d channels", stream.Channels)
				}
			}
		}
	}
	return resolution, frameRate, audioChannels, nil
}

func parseFrameRate(parts []string) int {
	numerator, _ := strconv.Atoi(parts[0])
	denom, _ := strconv.Atoi(parts[1])
	if denom == 0 {
		return 0
	}
	return int(math.Round(float64(numerator) / float64(denom)))
}

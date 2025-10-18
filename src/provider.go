package src

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	m3u "threadfin/src/internal/m3u-parser"
)

// fileType: Welcher Dateityp soll aktualisiert werden (m3u, hdhr, xml) | fileID: Update einer bestimmten Datei (Provider ID)
func getProviderData(fileType, fileID string) (err error) {

	var fileExtension, serverFileName string
	var body = make([]byte, 0)
	var newProvider = false
	var dataMap = make(map[string]interface{})

	var saveDateFromProvider = func(fileSource, serverFileName, id string, body []byte) (err error) {

		var data = make(map[string]interface{})

		if value, ok := dataMap[id].(map[string]interface{}); ok {
			data = value
		} else {
			data["id.provider"] = id
			dataMap[id] = data
		}

		// Default keys für die Providerdaten
		var keys = []string{"name", "description", "type", "file." + System.AppName, "file.source", "tuner", "http_proxy.ip", "http_proxy.port", "last.update", "compatibility", "counter.error", "counter.download", "provider.availability"}

		for _, key := range keys {

			if _, ok := data[key]; !ok {

				switch key {

				case "name":
					data[key] = serverFileName

				case "description":
					data[key] = ""

				case "type":
					data[key] = fileType

				case "file." + System.AppName:
					data[key] = id + fileExtension

				case "file.source":
					data[key] = fileSource

				case "http_proxy.ip":
					data[key] = ""

				case "http_proxy.port":
					data[key] = ""

				case "last.update":
					data[key] = time.Now().Format("2006-01-02 15:04:05")

				case "tuner":
					if fileType == "m3u" || fileType == "hdhr" {
						if _, ok := data[key].(float64); !ok {
							data[key] = 1
						}
					}

				case "compatibility":
					data[key] = make(map[string]interface{})

				case "counter.download":
					data[key] = 0.0

				case "counter.error":
					data[key] = 0.0

				case "provider.availability":
					data[key] = 100
				}

			}

		}

		if _, ok := data["id.provider"]; !ok {
			data["id.provider"] = id
		}

		// Datei extrahieren
		body, err = extractGZIP(body, fileSource)
		if err != nil {
			ShowError(err, 000)
			return
		}

		// Daten überprüfen
		showInfo("Check File:" + fileSource)

		switch fileType {

		case "m3u":
			if providerValue, ok := data["type"].(string); ok && strings.ToLower(providerValue) == "xtream" {
				showInfo("Xtream playlist normalization skipped (raw playlist preserved)")
				break
			}

			newM3u, err := m3u.MakeInterfaceFromM3U(body)
			if err != nil {
				return err
			}

			var m3uContent strings.Builder
			m3uContent.WriteString("#EXTM3U\n")

			for _, channel := range newM3u {
				channelMap := channel.(map[string]string)

				extinf := fmt.Sprintf(`#EXTINF:-1 tvg-id="%s" tvg-name="%s" tvg-chno="%s" tvg-logo="%s" group-title="%s",%s`,
					channelMap["tvg-id"],
					channelMap["tvg-name"],
					channelMap["tvg-chno"],
					channelMap["tvg-logo"],
					channelMap["group-title"],
					channelMap["name"],
				)

				m3uContent.WriteString(extinf + "\n" + channelMap["url"] + "\n")
			}

			body = []byte(m3uContent.String())

		case "hdhr":
			_, err = jsonToInterface(string(body))

		case "xmltv":
			err = checkXMLCompatibility(id, body)

		}

		if err != nil {
			return
		}

		var filePath = System.Folder.Data + data["file."+System.AppName].(string)

		err = writeByteToFile(filePath, body)

		if err == nil {
			data["last.update"] = time.Now().Format("2006-01-02 15:04:05")
			data["counter.download"] = data["counter.download"].(float64) + 1

			if fileType == "m3u" {
				if providerTypeValue, ok := data["type"].(string); ok && strings.ToLower(providerTypeValue) == "xtream" {
					if err := ensureXtreamXMLTV(data, id); err != nil {
						showInfo(fmt.Sprintf("Xtream XMLTV sync failed:%v", err))
					}
				}
			}
		}

		return

	}

	switch fileType {

	case "m3u":
		dataMap = Settings.Files.M3U
		fileExtension = ".m3u"

	case "hdhr":
		dataMap = Settings.Files.HDHR
		fileExtension = ".json"

	case "xmltv":
		dataMap = Settings.Files.XMLTV
		fileExtension = ".xml"

	}

	for dataID, d := range dataMap {

		var data = d.(map[string]interface{})
		var fileSource = data["file.source"].(string)
		var providerType string
		if providerTypeValue, ok := data["type"].(string); ok {
			providerType = strings.ToLower(providerTypeValue)
		}
		var httpProxyIp = ""
		if data["http_proxy.ip"] != nil {
			httpProxyIp = data["http_proxy.ip"].(string)
		}
		var httpProxyPort = ""
		if data["http_proxy.port"] != nil {
			httpProxyPort = data["http_proxy.port"].(string)
		}
		var httpProxyUrl = ""
		if httpProxyIp != "" && httpProxyPort != "" {
			httpProxyUrl = fmt.Sprintf("http://%s:%s", httpProxyIp, httpProxyPort)
		}

		newProvider = false

		if _, ok := data["new"]; ok {
			newProvider = true
			delete(data, "new")
		}

		// Wenn eine ID vorhanden ist und nicht mit der aus der Datanbank übereinstimmt, wird die Aktualisierung übersprungen (goto)
		if len(fileID) > 0 && newProvider == false {
			if dataID != fileID {
				goto Done
			}
		}

		if fileType == "m3u" && providerType == "xtream" {

			serverFileName, body, err = fetchXtreamPlaylist(data, httpProxyUrl)

		} else {

			switch fileType {

			case "hdhr":

				// Laden vom HDHomeRun Tuner
				showInfo("Tuner:" + fileSource)
				var tunerURL = "http://" + fileSource + "/lineup.json"
				serverFileName, body, err = downloadFileFromServer(tunerURL, httpProxyUrl)

			default:

				if strings.Contains(fileSource, "http://") || strings.Contains(fileSource, "https://") {

					// Laden vom Remote Server
					showInfo("Download:" + fileSource)
					serverFileName, body, err = downloadFileFromServer(fileSource, httpProxyUrl)

				} else {

					// Laden einer lokalen Datei
					showInfo("Open:" + fileSource)

					err = checkFile(fileSource)
					if err == nil {
						body, err = readByteFromFile(fileSource)
						serverFileName = getFilenameFromPath(fileSource)
					}

				}

			}

		}

		if err == nil {

			err = saveDateFromProvider(fileSource, serverFileName, dataID, body)
			if err == nil {
				showInfo("Save File:" + fileSource + " [ID: " + dataID + "]")
			}

		}

		if err != nil {

			ShowError(err, 000)
			var downloadErr = err

			if newProvider == false {

				// Prüfen ob ältere Datei vorhanden ist
				var file = System.Folder.Data + dataID + fileExtension

				err = checkFile(file)
				if err == nil {

					if len(fileID) == 0 {
						showWarning(1011)
					}

					err = downloadErr
				}

				// Fehler Counter um 1 erhöhen
				var data = make(map[string]interface{})
				if value, ok := dataMap[dataID].(map[string]interface{}); ok {

					data = value
					data["counter.error"] = data["counter.error"].(float64) + 1
					data["counter.download"] = data["counter.download"].(float64) + 1

				}

			} else {
				return downloadErr
			}

		}

		// Berechnen der Fehlerquote
		if newProvider == false {

			if value, ok := dataMap[dataID].(map[string]interface{}); ok {

				var data = make(map[string]interface{})
				data = value

				if data["counter.error"].(float64) == 0 {
					data["provider.availability"] = 100
				} else {
					data["provider.availability"] = int(data["counter.error"].(float64)*100/data["counter.download"].(float64)*-1 + 100)
				}

			}

		}

		switch fileType {

		case "m3u":
			Settings.Files.M3U = dataMap

		case "hdhr":
			Settings.Files.HDHR = dataMap

		case "xmltv":
			Settings.Files.XMLTV = dataMap
			delete(Data.Cache.XMLTV, System.Folder.Data+dataID+fileExtension)

		}

		saveSettings(Settings)

	Done:
	}

	return
}

func fetchXtreamPlaylist(data map[string]interface{}, proxyUrl string) (filename string, body []byte, err error) {
	baseURL, _ := data["xtream.url"].(string)
	username, _ := data["xtream.username"].(string)
	password, _ := data["xtream.password"].(string)
	output := "mpegts"
	if v, ok := data["xtream.output"].(string); ok && len(strings.TrimSpace(v)) > 0 {
		output = strings.TrimSpace(v)
	}

	baseURL = strings.TrimSpace(baseURL)
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if len(baseURL) == 0 || len(username) == 0 || len(password) == 0 {
		err = fmt.Errorf("Xtream Codes credentials incomplete")
		return
	}

	parsedBase, parseErr := url.Parse(baseURL)
	if parseErr != nil {
		err = fmt.Errorf("invalid Xtream Codes URL: %w", parseErr)
		return
	}

	apiBase := strings.TrimRight(baseURL, "/")
	playlistURL := fmt.Sprintf("%s/player_api.php?username=%s&password=%s&type=m3u_plus&output=%s",
		apiBase,
		url.QueryEscape(username),
		url.QueryEscape(password),
		url.QueryEscape(output),
	)

	showInfo("Xtream API:" + apiBase)

	filename, body, err = downloadFileFromServer(playlistURL, proxyUrl)
	if err != nil {
		return
	}

	trimmedBody := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmedBody, "#EXTM3U") {

		var loginResp xtreamLoginResponse
		if jsonErr := json.Unmarshal(body, &loginResp); jsonErr == nil && (loginResp.UserInfo.Username != "" || loginResp.UserInfo.Status != "") {

			var categoryMap map[string]string
			if cats, catErr := fetchXtreamCategories(apiBase, username, password, proxyUrl); catErr == nil {
				categoryMap = cats
			} else {
				showInfo(fmt.Sprintf("Xtream categories unavailable:%v", catErr))
			}

			var builderBody []byte
			builderBody, err = buildXtreamPlaylistFromAPI(parsedBase, apiBase, username, password, output, proxyUrl, loginResp, categoryMap, data)
			if err != nil {
				return "", nil, err
			}

			body = builderBody

		} else {

			var apiErr map[string]interface{}
			if jsonErr := json.Unmarshal(body, &apiErr); jsonErr == nil {
				if msg := getXtreamErrorMessage(apiErr); len(msg) > 0 {
					err = fmt.Errorf("Xtream Codes API error: %s", msg)
					return "", nil, err
				}
			}

			err = fmt.Errorf("unexpected Xtream Codes API response from %s", apiBase)
			return "", nil, err
		}
	}

	if len(filename) == 0 {
		if providerID, ok := data["id.provider"].(string); ok && len(providerID) > 0 {
			filename = providerID + ".m3u"
		} else if name, ok := data["name"].(string); ok && len(name) > 0 {
			safeName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
			filename = safeName + ".m3u"
		} else {
			filename = System.AppName + ".m3u"
		}
	}

	return
}

func fetchXtreamCategories(apiBase, username, password, proxyUrl string) (map[string]string, error) {
	categoryMap := make(map[string]string)

	categoriesURL := fmt.Sprintf("%s/player_api.php?username=%s&password=%s&action=get_live_categories",
		apiBase,
		url.QueryEscape(username),
		url.QueryEscape(password),
	)

	_, body, err := downloadFileFromServer(categoriesURL, proxyUrl)
	if err != nil {
		return categoryMap, err
	}

	var categories []xtreamCategory
	if err := json.Unmarshal(body, &categories); err != nil {
		return categoryMap, err
	}

	for _, category := range categories {
		name := strings.TrimSpace(category.Name)
		if name == "" {
			name = category.ID.String()
		}
		categoryMap[strings.TrimSpace(category.ID.String())] = name
	}

	counter := 0
	for id, name := range categoryMap {
		showInfo(fmt.Sprintf("Xtream category %s -> %s", id, name))
		counter++
		if counter >= 3 {
			break
		}
	}

	return categoryMap, nil
}

type xtreamLoginResponse struct {
	UserInfo   xtreamUserInfo   `json:"user_info"`
	ServerInfo xtreamServerInfo `json:"server_info"`
}

type xtreamUserInfo struct {
	Username             string      `json:"username"`
	Password             string      `json:"password"`
	Message              string      `json:"message"`
	Auth                 json.Number `json:"auth"`
	Status               string      `json:"status"`
	AllowedOutputFormats []string    `json:"allowed_output_formats"`
	MaxConnections       string      `json:"max_connections"`
	ActiveConnections    json.Number `json:"active_cons"`
	Expiry               json.Number `json:"exp_date"`
	IsTrial              json.Number `json:"is_trial"`
}

type xtreamServerInfo struct {
	URL            string `json:"url"`
	Port           string `json:"port"`
	HTTPSPort      string `json:"https_port"`
	ServerProtocol string `json:"server_protocol"`
}

type xtreamCategory struct {
	ID   xtreamString `json:"category_id"`
	Name string       `json:"category_name"`
}

type xtreamLiveStream struct {
	StreamID       xtreamString `json:"stream_id"`
	Name           string       `json:"name"`
	StreamIcon     string       `json:"stream_icon"`
	CategoryName   string       `json:"category_name"`
	CategoryID     xtreamString `json:"category_id"`
	EPGChannelID   xtreamString `json:"epg_channel_id"`
	ChannelID      xtreamString `json:"channel_id"`
	StreamType     string       `json:"stream_type"`
	TVArchive      xtreamString `json:"tv_archive"`
	TVArchiveStart xtreamString `json:"tv_archive_start"`
	TVArchiveEnd   xtreamString `json:"tv_archive_end"`
}

func (info xtreamUserInfo) isAuthorized() bool {
	authStr := strings.TrimSpace(info.Auth.String())
	if authStr != "" {
		if authVal, err := info.Auth.Int64(); err == nil {
			if authVal != 1 {
				return false
			}
		} else if authStr != "1" {
			return false
		}
	}

	status := strings.ToLower(strings.TrimSpace(info.Status))
	if status != "" && status != "active" {
		return false
	}

	return true
}

func buildXtreamPlaylistFromAPI(parsedBase *url.URL, apiBase, username, password, requestedOutput, proxyUrl string, loginResp xtreamLoginResponse, categories map[string]string, data map[string]interface{}) ([]byte, error) {
	if !loginResp.UserInfo.isAuthorized() {
		message := strings.TrimSpace(loginResp.UserInfo.Message)
		if len(message) == 0 {
			message = "authentication rejected"
		}
		return nil, fmt.Errorf("Xtream Codes authentication failed: %s", message)
	}

	if len(strings.TrimSpace(loginResp.UserInfo.Message)) > 0 {
		showInfo("Xtream API message:" + strings.TrimSpace(loginResp.UserInfo.Message))
	}

	selectedFormat, extension, streamFolder := chooseXtreamOutput(requestedOutput, loginResp.UserInfo.AllowedOutputFormats)
	if !strings.EqualFold(strings.TrimSpace(requestedOutput), selectedFormat) {
		showInfo(fmt.Sprintf("Xtream API: output %s not permitted, using %s", strings.TrimSpace(requestedOutput), selectedFormat))
	}
	if data != nil {
		data["xtream.output"] = selectedFormat
	}

	streamBase := determineXtreamStreamBase(parsedBase, loginResp.ServerInfo)

	liveURL := fmt.Sprintf("%s/player_api.php?username=%s&password=%s&action=get_live_streams",
		apiBase,
		url.QueryEscape(username),
		url.QueryEscape(password),
	)

	_, liveBody, err := downloadFileFromServer(liveURL, proxyUrl)
	if err != nil {
		return nil, err
	}

	var liveStreams []xtreamLiveStream
	if err := json.Unmarshal(liveBody, &liveStreams); err != nil {
		return nil, fmt.Errorf("failed to parse Xtream live stream list: %w", err)
	}

	if len(liveStreams) == 0 {
		return nil, fmt.Errorf("Xtream Codes live stream list is empty")
	}

	showInfo(fmt.Sprintf("Xtream API streams:%d", len(liveStreams)))

	var builder strings.Builder
	builder.WriteString("#EXTM3U\n")

	for idx, stream := range liveStreams {
		// Skip entries without a valid stream ID or name
		streamID := sanitizeForM3U(stream.StreamID.String())
		channelName := strings.TrimSpace(stream.Name)
		if len(streamID) == 0 || len(channelName) == 0 {
			showInfo(fmt.Sprintf("Xtream stream skipped [%d]: id=\"%s\" name=\"%s\"", idx, streamID, channelName))
			continue
		}

		tvgID := firstNonEmpty(stream.EPGChannelID.String(), stream.ChannelID.String(), streamID)
		tvgName := sanitizeForM3U(channelName)
		tvgLogo := sanitizeForM3U(stream.StreamIcon)
		categoryName := strings.TrimSpace(stream.CategoryName)
		if len(categoryName) == 0 && categories != nil {
			if mapped, ok := categories[strings.TrimSpace(stream.CategoryID.String())]; ok {
				categoryName = mapped
			}
		}
		if len(categoryName) == 0 {
			categoryName = stream.CategoryID.String()
		}
		groupTitle := sanitizeForM3U(categoryName)
		if idx < 3 {
			showInfo(fmt.Sprintf("Xtream category resolved [%d]: id=%s name=%s title=%s", idx, stream.CategoryID.String(), categoryName, groupTitle))
		}

		extinf := fmt.Sprintf(`#EXTINF:-1 tvg-id="%s" tvg-name="%s" tvg-logo="%s" group-title="%s",%s`,
			sanitizeForM3U(tvgID),
			tvgName,
			tvgLogo,
			groupTitle,
			tvgName,
		)

		builder.WriteString(extinf + "\n")

		streamURL := fmt.Sprintf("%s/%s/%s/%s/%s.%s",
			streamBase,
			streamFolder,
			url.PathEscape(username),
			url.PathEscape(password),
			url.PathEscape(streamID),
			extension,
		)

		builder.WriteString(streamURL + "\n")

		if idx < 3 {
			showInfo(fmt.Sprintf("Xtream stream [%d]: id=%s name=%s url=%s", idx, streamID, tvgName, streamURL))
		}
	}

	finalPlaylist := builder.String()
	showInfo(fmt.Sprintf("Xtream playlist final bytes:%d", len(finalPlaylist)))
	return []byte(finalPlaylist), nil
}

func chooseXtreamOutput(requested string, allowed []string) (selected string, extension string, folder string) {
	requested = strings.ToLower(strings.TrimSpace(requested))
	if requested == "" {
		requested = "mpegts"
	}

	allowedLower := make([]string, 0, len(allowed))
	for _, entry := range allowed {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry != "" {
			allowedLower = append(allowedLower, entry)
		}
	}

	selected = requested
	if len(allowedLower) > 0 && !stringSliceContains(allowedLower, selected) {
		selected = allowedLower[0]
	}

	switch selected {
	case "mpegts":
		selected = "ts"
		extension = "ts"
		folder = "live"
	case "ts":
		extension = "ts"
		folder = "live"
	case "m3u8":
		extension = "m3u8"
		folder = "hls"
	case "hls":
		selected = "m3u8"
		extension = "m3u8"
		folder = "hls"
	case "rtmp":
		extension = "ts"
		folder = "live"
	default:
		selected = "ts"
		extension = "ts"
		folder = "live"
	}

	return selected, extension, folder
}

func determineXtreamStreamBase(parsedBase *url.URL, serverInfo xtreamServerInfo) string {
	scheme := strings.TrimSpace(serverInfo.ServerProtocol)
	if scheme == "" && parsedBase != nil && parsedBase.Scheme != "" {
		scheme = parsedBase.Scheme
	}
	if scheme == "" {
		scheme = "http"
	}

	host := strings.TrimSpace(serverInfo.URL)
	if host == "" && parsedBase != nil {
		if parsedBase.Hostname() != "" {
			host = parsedBase.Hostname()
		} else {
			host = parsedBase.Host
		}
	}

	if host == "" {
		host = "localhost"
	}

	port := strings.TrimSpace(serverInfo.Port)
	if scheme == "https" && strings.TrimSpace(serverInfo.HTTPSPort) != "" {
		port = strings.TrimSpace(serverInfo.HTTPSPort)
	}
	if port == "" && parsedBase != nil && parsedBase.Port() != "" {
		port = parsedBase.Port()
	}

	addPort := false
	if port != "" {
		if scheme == "http" && port != "80" {
			addPort = true
		}
		if scheme == "https" && port != "443" {
			addPort = true
		}
		if scheme != "http" && scheme != "https" {
			addPort = true
		}
	}

	if strings.Contains(host, ":") {
		addPort = false
	}

	if addPort {
		return fmt.Sprintf("%s://%s:%s", scheme, host, port)
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}

func sanitizeForM3U(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `"`, "'")
	return value
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func stringSliceContains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func getXtreamErrorMessage(apiErr map[string]interface{}) string {
	if msg, ok := apiErr["message"].(string); ok && len(strings.TrimSpace(msg)) > 0 {
		return strings.TrimSpace(msg)
	}
	if msg, ok := apiErr["error"].(string); ok && len(strings.TrimSpace(msg)) > 0 {
		return strings.TrimSpace(msg)
	}
	if userInfo, ok := apiErr["user_info"].(map[string]interface{}); ok {
		if status, ok := userInfo["status"].(string); ok && len(strings.TrimSpace(status)) > 0 {
			if strings.ToLower(strings.TrimSpace(status)) != "active" {
				return strings.TrimSpace(status)
			}
		}
		if msg, ok := userInfo["message"].(string); ok && len(strings.TrimSpace(msg)) > 0 {
			return strings.TrimSpace(msg)
		}
	}
	if faultCode, ok := apiErr["faultCode"]; ok {
		return fmt.Sprintf("fault code: %v", faultCode)
	}
	if faultString, ok := apiErr["faultString"]; ok {
		return fmt.Sprintf("fault: %v", faultString)
	}
	return ""
}

func boolValue(val interface{}) bool {
	switch v := val.(type) {
	case bool:
		return v
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		return lower == "1" || lower == "true" || lower == "yes" || lower == "on"
	case float64:
		return v != 0
	default:
		return false
	}
}

func stringValue(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func ensureXtreamXMLTV(m3uEntry map[string]interface{}, m3uID string) error {
	wantXMLTV := boolValue(m3uEntry["xtream.xmltv"])
	currentID := strings.TrimSpace(stringValue(m3uEntry["xtream.xmltv.id"]))

	if !wantXMLTV {
		if currentID != "" {
			deleteLocalProviderFiles(currentID, "xmltv")
			if Settings.Files.XMLTV != nil {
				delete(Settings.Files.XMLTV, currentID)
			}
			delete(m3uEntry, "xtream.xmltv.id")
		}
		return nil
	}

	baseURL := strings.TrimRight(strings.TrimSpace(stringValue(m3uEntry["xtream.url"])), "/")
	username := strings.TrimSpace(stringValue(m3uEntry["xtream.username"]))
	password := strings.TrimSpace(stringValue(m3uEntry["xtream.password"]))
	if baseURL == "" || username == "" || password == "" {
		return fmt.Errorf("missing Xtream credentials for XMLTV")
	}

	xmlTVURL := fmt.Sprintf("%s/xmltv.php?username=%s&password=%s", baseURL, url.QueryEscape(username), url.QueryEscape(password))

	if Settings.Files.XMLTV == nil {
		Settings.Files.XMLTV = make(map[string]interface{})
	}

	var xmlData map[string]interface{}
	if currentID != "" {
		if existing, ok := Settings.Files.XMLTV[currentID].(map[string]interface{}); ok {
			xmlData = existing
		}
	}

	if xmlData == nil {
		currentID = "X" + randomString(19)
		xmlData = make(map[string]interface{})
		xmlData["id.provider"] = currentID
		xmlData["counter.download"] = 0.0
		xmlData["counter.error"] = 0.0
		xmlData["provider.availability"] = 100
		xmlData["new"] = true
	}

	name := stringValue(m3uEntry["name"])
	if name == "" {
		name = "Xtream XMLTV"
	} else {
		name = fmt.Sprintf("%s (XMLTV)", name)
	}

	xmlData["name"] = name
	xmlData["description"] = stringValue(xmlData["description"])
	xmlData["type"] = "xmltv"
	xmlData["file."+System.AppName] = currentID + ".xml"
	xmlData["file.source"] = xmlTVURL
	xmlData["xtream.parent"] = m3uID

	Settings.Files.XMLTV[currentID] = xmlData
	m3uEntry["xtream.xmltv.id"] = currentID

	if err := getProviderData("xmltv", currentID); err != nil {
		return err
	}

	return nil
}

func downloadFileFromServer(providerURL string, proxyUrl string) (filename string, body []byte, err error) {
	_, err = url.ParseRequestURI(providerURL)
	if err != nil {
		return
	}

	// Derive a timeout: prefer configured buffer timeout if provided, else default to 30s
	requestTimeout := 30 * time.Second
	if Settings.BufferTimeout > 0 {
		requestTimeout = time.Duration(Settings.BufferTimeout*1000) * time.Millisecond
	}

	httpClient := &http.Client{Timeout: requestTimeout}

	if proxyUrl != "" {
		proxyURL, err := url.Parse(proxyUrl)
		if err != nil {
			return "", nil, err
		}

		httpClient = &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}

	req, err := http.NewRequest("GET", providerURL, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", Settings.UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	resp.Header.Set("User-Agent", Settings.UserAgent)

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("%d: %s %s", resp.StatusCode, providerURL, http.StatusText(resp.StatusCode))
		return
	}

	// Get filename from the header
	var index = strings.Index(resp.Header.Get("Content-Disposition"), "filename")

	if index > -1 {
		var headerFilename = resp.Header.Get("Content-Disposition")[index:]
		var value = strings.Split(headerFilename, `=`)
		var f = strings.Replace(value[1], `"`, "", -1)
		f = strings.Replace(f, `;`, "", -1)
		filename = f
		showInfo("Header filename:" + filename)
	} else {
		var cleanFilename = strings.SplitN(getFilenameFromPath(providerURL), "?", 2)
		filename = cleanFilename[0]
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return
}

type xtreamString string

func (s *xtreamString) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.EqualFold(data, []byte("null")) {
		*s = ""
		return nil
	}

	if data[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*s = xtreamString(str)
		return nil
	}

	*s = xtreamString(string(data))
	return nil
}

func (s xtreamString) String() string {
	return string(s)
}

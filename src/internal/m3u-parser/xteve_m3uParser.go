package m3u

import (
	"crypto/md5"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// MakeInterfaceFromM3U :
func MakeInterfaceFromM3U(byteStream []byte) (allChannels []interface{}, err error) {

	var content = string(byteStream)

	var parseMetaData = func(channel string) (stream map[string]string) {
		var channelName string // Declare inside the function
		var uuids []string

		stream = make(map[string]string)
		var exceptForParameter = `[a-z-A-Z&=]*(".*?")`
		var exceptForChannelName = `,([^\n]*|,[^\r]*)`
		var lines = strings.Split(strings.Replace(channel, "\r\n", "\n", -1), "\n")

		// Remove lines starting with # and empty lines
		for i := len(lines) - 1; i >= 0; i-- {
			if len(lines[i]) == 0 || lines[i][0:1] == "#" {
				lines = append(lines[:i], lines[i+1:]...)
			}
		}

		// URL is always on the second line after #EXTINF
		if len(lines) >= 2 {
			stream["url"] = strings.Trim(lines[1], "\r\n")

			// Parse the first line (#EXTINF line) for metadata
			var value string
			var p = regexp.MustCompile(exceptForParameter)
			var streamParameter = p.FindAllString(lines[0], -1)
			for _, p := range streamParameter {
				lines[0] = strings.Replace(lines[0], p, "", 1)
				p = strings.Replace(p, `"`, "", -1)
				var parameter = strings.SplitN(p, "=", 2)
				if len(parameter) == 2 {
					// Save TVG Key in lowercase
					if strings.Contains(parameter[0], "tvg") {
						stream[strings.ToLower(parameter[0])] = parameter[1]
					} else {
						stream[parameter[0]] = parameter[1]
					}

					// Do not pass URLs to the filter function
					if !strings.Contains(parameter[1], "://") && len(parameter[1]) > 0 {
						value = value + parameter[1] + " "
					}
				}
			}

			// Parse channel name
			n := regexp.MustCompile(exceptForChannelName)
			var name = n.FindAllString(lines[0], 1)

			if len(name) > 0 {
				channelName = name[0]
				channelName = strings.Replace(channelName, `,`, "", 1)
				channelName = strings.TrimRight(channelName, "\r\n")
				channelName = strings.TrimRight(channelName, " ")
			}

			if len(channelName) == 0 {
				if v, ok := stream["tvg-name"]; ok {
					channelName = v
				}
			}

			// Only generate a new tvg-id if it's missing, blank, or (no tvg-id)
			if stream["tvg-id"] == "" || stream["tvg-id"] == "(no tvg-id)" {
				hash := md5.Sum([]byte(stream["url"]))
				stream["tvg-id"] = fmt.Sprintf("threadfin-%x", hash)
			}

			channelName = strings.TrimRight(channelName, " ")

			// Skip channels without a name
			if len(channelName) == 0 {
				return
			}

			stream["name"] = channelName
			value = value + channelName
			stream["_values"] = value
		}

		// Assign a unique ID to the stream
		for key, value := range stream {
			if strings.Contains(strings.ToLower(key), "tvg-name") {
				if indexOfString(value, uuids) != -1 {
					break
				}
				uuids = append(uuids, value)
				stream["_uuid.key"] = key
				stream["_uuid.value"] = value
				break
			}
		}

		return
	}

	// Check if the content is a valid M3U file
	if strings.Contains(content, "#EXT-X-TARGETDURATION") || strings.Contains(content, "#EXT-X-MEDIA-SEQUENCE") {
		err = errors.New("Invalid M3U file, an extended M3U file is required.")
		return
	}

	if strings.Contains(content, "#EXTM3U") {
		content = strings.Replace(content, ":-1", "", -1)
		content = strings.Replace(content, "'", "\"", -1)
		var channels = strings.Split(content, "#EXTINF")

		channels = append(channels[:0], channels[1:]...)

		for _, channel := range channels {
			var stream = parseMetaData(channel)
			if len(stream) > 0 && stream != nil {
				allChannels = append(allChannels, stream)
			}
		}

	} else {
		err = errors.New("Invalid M3U file, an extended M3U file is required.")
	}

	return
}

func indexOfString(element string, data []string) int {

	for k, v := range data {
		if element == v {
			return k
		}
	}

	return -1
}

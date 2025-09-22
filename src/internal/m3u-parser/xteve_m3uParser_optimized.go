package m3u

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Precompiled regex patterns for better performance
var (
	parameterRegex    = regexp.MustCompile(`[a-z-A-Z&=]*(".*?")`)
	channelNameRegex  = regexp.MustCompile(`,([^\n]*|,[^\r]*)`)
	crlfReplacer      = strings.NewReplacer("\r\n", "\n")
	quoteReplacer     = strings.NewReplacer(`"`, "")
	commaReplacer     = strings.NewReplacer(`,`, "")
	oldM3UReplacer    = strings.NewReplacer(":-1", "", "'", `"`)
)

// MakeInterfaceFromM3UOptimized : Optimized version for large M3U files
func MakeInterfaceFromM3UOptimized(byteStream []byte) (allChannels []interface{}, err error) {
	// Use bytes.Contains for faster validation
	if bytes.Contains(byteStream, []byte("#EXT-X-TARGETDURATION")) || bytes.Contains(byteStream, []byte("#EXT-X-MEDIA-SEQUENCE")) {
		err = errors.New("Invalid M3U file, an extended M3U file is required.")
		return
	}

	if !bytes.Contains(byteStream, []byte("#EXTM3U")) {
		err = errors.New("Invalid M3U file, an extended M3U file is required.")
		return
	}

	// Use scanner for line-by-line processing instead of loading full content
	scanner := bufio.NewScanner(bytes.NewReader(byteStream))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line length

	var currentChannel strings.Builder
	var isInChannel bool
	var lineCount int

	// Pre-allocate channels slice with estimated capacity
	estimatedChannels := bytes.Count(byteStream, []byte("#EXTINF"))
	allChannels = make([]interface{}, 0, estimatedChannels)

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Process #EXTINF lines
		if strings.HasPrefix(line, "#EXTINF") {
			if isInChannel && currentChannel.Len() > 0 {
				// Process previous channel
				if stream := parseMetaDataOptimized(currentChannel.String()); len(stream) > 0 {
					allChannels = append(allChannels, stream)
				}
				currentChannel.Reset()
			}
			currentChannel.WriteString(line)
			currentChannel.WriteByte('\n')
			isInChannel = true
			continue
		}

		// Skip other # lines
		if strings.HasPrefix(line, "#") {
			continue
		}

		// This is a URL line
		if isInChannel {
			currentChannel.WriteString(line)
			// Process complete channel
			if stream := parseMetaDataOptimized(currentChannel.String()); len(stream) > 0 {
				allChannels = append(allChannels, stream)
			}
			currentChannel.Reset()
			isInChannel = false
		}
	}

	// Process last channel if exists
	if isInChannel && currentChannel.Len() > 0 {
		if stream := parseMetaDataOptimized(currentChannel.String()); len(stream) > 0 {
			allChannels = append(allChannels, stream)
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return allChannels, nil
}

// parseMetaDataOptimized : Optimized metadata parsing
func parseMetaDataOptimized(channel string) map[string]string {
	stream := make(map[string]string, 12) // Pre-allocate with typical size

	// Use bufio.Scanner for line splitting
	scanner := bufio.NewScanner(strings.NewReader(channel))
	lines := make([]string, 0, 3) // Most channels have 2-3 lines

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	if len(lines) < 2 {
		return nil // Invalid channel format
	}

	// URL is always the last non-# line
	stream["url"] = strings.TrimSpace(lines[len(lines)-1])

	// Parse the #EXTINF line (first line in channel string)
	extinfLine := ""
	scanner = bufio.NewScanner(strings.NewReader(channel))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#EXTINF") {
			extinfLine = line
			break
		}
	}

	if extinfLine == "" {
		return nil
	}

	// Extract parameters using pre-compiled regex
	var value strings.Builder
	streamParameter := parameterRegex.FindAllString(extinfLine, -1)

	for _, p := range streamParameter {
		extinfLine = strings.Replace(extinfLine, p, "", 1)
		cleanParam := quoteReplacer.Replace(p)

		if paramParts := strings.SplitN(cleanParam, "=", 2); len(paramParts) == 2 {
			key, val := paramParts[0], paramParts[1]

			// Save TVG Key in lowercase
			if strings.Contains(key, "tvg") {
				stream[strings.ToLower(key)] = val
			} else {
				stream[key] = val
			}

			// Build value string (skip URLs)
			if !strings.Contains(val, "://") && len(val) > 0 {
				value.WriteString(val)
				value.WriteByte(' ')
			}
		}
	}

	// Parse channel name using pre-compiled regex
	var channelName string
	if nameMatches := channelNameRegex.FindStringSubmatch(extinfLine); len(nameMatches) > 1 {
		channelName = nameMatches[1]
		channelName = commaReplacer.Replace(channelName)
		channelName = strings.TrimSpace(channelName)
	}

	// Fallback to tvg-name if no channel name found
	if len(channelName) == 0 {
		if tvgName, ok := stream["tvg-name"]; ok {
			channelName = tvgName
		}
	}

	// Skip channels without a name
	if len(channelName) == 0 {
		return nil
	}

	// Generate tvg-id if missing
	if tvgID := stream["tvg-id"]; tvgID == "" || tvgID == "(no tvg-id)" {
		hash := md5.Sum([]byte(stream["url"]))
		stream["tvg-id"] = fmt.Sprintf("threadfin-%x", hash)
	}

	stream["name"] = strings.TrimSpace(channelName)
	value.WriteString(channelName)
	stream["_values"] = value.String()

	// Set UUID for tvg-name (simplified)
	if tvgName, ok := stream["tvg-name"]; ok && len(tvgName) > 0 {
		stream["_uuid.key"] = "tvg-name"
		stream["_uuid.value"] = tvgName
	}

	return stream
}

// Wrapper to maintain backward compatibility
func MakeInterfaceFromM3U(byteStream []byte) (allChannels []interface{}, err error) {
	// For files larger than 10MB, use optimized version
	if len(byteStream) > 10*1024*1024 {
		return MakeInterfaceFromM3UOptimized(byteStream)
	}

	// Use original implementation for smaller files to maintain compatibility
	return makeInterfaceFromM3UOriginal(byteStream)
}
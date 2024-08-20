package src

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
)

var activeClientCount int
var activePlaylistCount int

func getActiveClientCount() (count int) {
	return activeClientCount
}

func getActivePlaylistCount() (count int) {
	return activePlaylistCount
}

func createStreamID(stream map[int]*ThisStream) (streamID int) {
	var debug string

	streamID = 0
	for i := 0; i <= len(stream); i++ {
		if _, ok := stream[i]; !ok {
			streamID = i
			break
		}
	}

	debug = fmt.Sprintf("Streaming Status:Stream ID = %d", streamID)
	showDebug(debug, 1)

	return
}

func bufferingStream(playlistID, streamingURL, backupStreamingURL1, backupStreamingURL2, backupStreamingURL3, channelName string, w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Duration(Settings.BufferTimeout) * time.Millisecond)

	playlist := &Playlist{}
	client := &ThisClient{}
	stream := &ThisStream{}
	var streamID int
	var debug string
	var newStream = true
	var timeOut = 0
	var streaming = false

	w.Header().Set("Connection", "close")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Check whether the playlist is already in use
	if storedPlaylist, exists := BufferInformation.Load(playlistID); !exists {
		var playlistType string

		// Playlist not in use, create default values for the playlist
		playlist.Folder = System.Folder.Temp + playlistID + string(os.PathSeparator)
		playlist.PlaylistID = playlistID
		playlist.Streams = make(map[int]*ThisStream)
		playlist.Clients = make(map[int]*ThisClient)

		err := checkVFSFolder(playlist.Folder, bufferVFS)
		if err != nil {
			ShowError(err, 000)
			httpStatusError(w, r, 404)
			return
		}

		switch playlist.PlaylistID[0:1] {
		case "M":
			playlistType = "m3u"
		case "H":
			playlistType = "hdhr"
		}

		playlist.Tuner = getTuner(playlistID, playlistType)
		playlist.PlaylistName = getProviderParameter(playlist.PlaylistID, playlistType, "name")
		playlist.HttpProxyIP = getProviderParameter(playlist.PlaylistID, playlistType, "http_proxy.ip")
		playlist.HttpProxyPort = getProviderParameter(playlist.PlaylistID, playlistType, "http_proxy.port")

		// Create default values for the stream
		streamID = createStreamID(playlist.Streams)
		activeClientCount = 1
		if activePlaylistCount == 0 {
			activePlaylistCount = 1
		}
		client.Connection = activeClientCount
		stream.URL = streamingURL
		stream.BackupChannel1URL = backupStreamingURL1
		stream.BackupChannel2URL = backupStreamingURL2
		stream.BackupChannel3URL = backupStreamingURL3
		stream.ChannelName = channelName
		stream.Status = false

		playlist.Streams[streamID] = stream
		playlist.Clients[streamID] = client

		BufferInformation.Store(playlistID, playlist)
		BufferInformation.ClientCount++
		BufferInformation.PlaylistCount++

	} else {
		// Playlist is already used for streaming
		playlist = storedPlaylist
		for id := range playlist.Streams {
			stream = playlist.Streams[id]
			client = playlist.Clients[id]

			if streamingURL == stream.URL {
				streamID = id
				newStream = false
				activeClientCount++
				client.Connection = activeClientCount

				playlist.Streams[streamID] = stream
				playlist.Clients[streamID] = client

				BufferInformation.Store(playlistID, playlist)

				debug = fmt.Sprintf("Restream Status:Playlist: %s - Channel: %s - Connections: %d", playlist.PlaylistName, stream.ChannelName, client.Connection)
				showDebug(debug, 1)

				if playlist.Clients[streamID].Connection > 0 {
					var clients = playlist.Clients[streamID]
					clients.Connection = activeClientCount

					showInfo(fmt.Sprintf("Streaming Status:Channel: %s (Clients: %d)", stream.ChannelName, client.Connection))
					playlist.Clients[streamID] = clients
					BufferInformation.Store(playlistID, playlist)
				}

				break
			}

		}

		// New stream for an already active playlist
		if newStream {
			// Check whether the playlist allows another stream (tuner)
			if len(playlist.Streams) >= playlist.Tuner {
				showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - No new connections available. Tuner = %d", playlist.PlaylistName, playlist.Tuner))

				if value, ok := webUI["html/video/stream-limit.ts"]; ok {
					content := GetHTMLString(value.(string))
					w.WriteHeader(200)
					w.Header().Set("Content-type", "video/mpeg")
					w.Header().Set("Content-Length:", "0")

					for i := 1; i < 60; i++ {
						_ = i
						w.Write([]byte(content))
						time.Sleep(time.Duration(500) * time.Millisecond)
					}
					return
				}
				return
			}

			// Playlist allows another stream (the tuner's limit has not yet been reached)
			// Create default values for the stream
			stream = &ThisStream{}
			client = &ThisClient{}
			streamID = createStreamID(playlist.Streams)
			activePlaylistCount++
			activeClientCount++
			stream.URL = streamingURL
			stream.ChannelName = channelName
			stream.Status = false
			stream.BackupChannel1URL = backupStreamingURL1
			stream.BackupChannel2URL = backupStreamingURL2
			stream.BackupChannel3URL = backupStreamingURL3

			playlist.Streams[streamID] = stream
			playlist.Clients[streamID] = client

			BufferInformation.Store(playlistID, playlist)
		}
	}

	// Check whether the stream is already being played by another client
	if !playlist.Streams[streamID].Status && newStream {
		// New buffer is needed
		stream = playlist.Streams[streamID]
		stream.MD5 = getMD5(streamingURL)
		stream.Folder = playlist.Folder + stream.MD5 + string(os.PathSeparator)
		stream.PlaylistID = playlistID
		stream.PlaylistName = playlist.PlaylistName

		playlist.Streams[streamID] = stream
		BufferInformation.Store(playlistID, playlist)

		switch Settings.Buffer {
		case "ffmpeg", "vlc":
			go thirdPartyBuffer(streamID, playlistID, false, 0)
		default:
			break
		}

		showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))

		clients := &ThisClient{}
		activeClientCount = 1
		clients.Connection = activeClientCount
		playlist.Clients[streamID] = clients
		BufferInformation.Store(playlistID, playlist)
	}

	w.WriteHeader(200)

	for { // Loop 1: Wait until the first segment has been downloaded through the buffer
		if playlist, exists := BufferInformation.Load(playlistID); exists {
			if stream, ok := playlist.Streams[streamID]; ok {
				if !stream.Status {
					timeOut++
					time.Sleep(time.Duration(100) * time.Millisecond)

					if playlist.Clients[streamID].Connection > 0 {
						var clients = playlist.Clients[streamID]

						if clients.Error != nil || (timeOut > 200 && (stream.BackupChannel1URL == "" && stream.BackupChannel2URL == "" && stream.BackupChannel3URL == "")) {
							killClientConnection(streamID, stream.PlaylistID, false)
							return
						}
					}
					continue
				}

				var oldSegments []string
				for { // Loop 2: Temporary files are present, data can be sent to the client
					// Monitor HTTP client connection
					ctx := r.Context()
					if ok {
						select {
						case <-ctx.Done():
							killClientConnection(streamID, playlistID, false)
							clientConnection(stream)
							return
						default:
							if playlist, exists := BufferInformation.Load(playlistID); exists {
								if playlist.Clients[streamID] != nil {
									var clients = playlist.Clients[streamID]
									if clients.Error != nil {
										ShowError(clients.Error, 0)
										killClientConnection(streamID, playlistID, false)
										clientConnection(stream)
										return
									}
								} else {
									return
								}
							} else {
								return
							}
						}
					}

					if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
						killClientConnection(streamID, playlistID, false)
						return
					}

					var tmpFiles = getBufTmpFiles(stream)
					for _, f := range tmpFiles {
						if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
							killClientConnection(streamID, playlistID, false)
							return
						}

						oldSegments = append(oldSegments, f)

						var fileName = stream.Folder + f

						file, err := bufferVFS.Open(fileName)
						if err != nil {
							debug = fmt.Sprintf("Buffer Open (%s)", fileName)
							showDebug(debug, 2)
							return
						}
						defer file.Close()

						l, err := file.Stat()
						if err == nil {
							debug = fmt.Sprintf("Buffer Status:Send to client (%s)", fileName)
							showDebug(debug, 2)

							var buffer = make([]byte, int(l.Size()))
							_, err = file.Read(buffer)

							if err == nil {
								file.Seek(0, 0)

								if !streaming {
									contentType := http.DetectContentType(buffer)
									_ = contentType
									w.Header().Set("Content-type", contentType)
									w.Header().Set("Content-Length", "0")
									w.Header().Set("Connection", "close")
								}

								_, err := w.Write(buffer)
								if err != nil {
									file.Close()
									killClientConnection(streamID, playlistID, false)
									return
								}

								file.Close()
								streaming = true
							}
							file.Close()
						}

						var n = indexOfString(f, oldSegments)

						if n > 20 {
							var fileToRemove = stream.Folder + oldSegments[0]
							if err = bufferVFS.RemoveAll(getPlatformFile(fileToRemove)); err != nil {
								ShowError(err, 4007)
							}
							oldSegments = append(oldSegments[:0], oldSegments[0+1:]...)
						}

						file.Close()
					}

					if len(tmpFiles) == 0 {
						time.Sleep(time.Duration(100) * time.Millisecond)
					}
				} // End Loop 2
			} else {
				// Stream not found
				if stream != nil {
					killClientConnection(streamID, stream.PlaylistID, false)
					showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
				}
				return
			}
		} // End BufferInformation
	} // End Loop 1
}

func killClientConnection(streamID int, playlistID string, force bool) {
	BufferInformation.Lock()
	defer BufferInformation.Unlock()

	if playlist, exists := BufferInformation.Load(playlistID); exists {
		if force {
			delete(playlist.Streams, streamID)
			showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
			return
		}

		if stream, ok := playlist.Streams[streamID]; ok {
			if playlist.Clients[streamID].Connection > 0 {
				if activeClientCount > 0 {
					activeClientCount--
				} else {
					activeClientCount = 0
				}

				var clients = playlist.Clients[streamID]
				clients.Connection = activeClientCount
				playlist.Clients[streamID] = clients
				BufferInformation.Store(playlistID, playlist)

				showInfo("Streaming Status:Client has terminated the connection")
				showInfo(fmt.Sprintf("Streaming Status:Channel: %s (Clients: %d)", stream.ChannelName, clients.Connection))

				if clients.Connection <= 0 {
					if activePlaylistCount > 0 {
						activePlaylistCount--
					} else {
						activePlaylistCount = 0
					}

					BufferInformation.Delete(playlistID)
				}
			}

			if len(playlist.Streams) > 0 {
				showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
			}
		}
	}
}

func clientConnection(stream *ThisStream) (status bool) {
	status = true
	BufferInformation.Lock()
	defer BufferInformation.Unlock()

	playlist, exists := BufferInformation.Load(stream.PlaylistID)
	if !exists {
		return false
	}

	// If no clients are connected
	if len(playlist.Clients) == 0 || playlist.Clients == nil {
		status = false
	}

	// Additional checks to ensure the state is consistent
	if activeClientCount <= 0 {
		status = false
	}

	if !status {
		if err := bufferVFS.RemoveAll(stream.Folder); err != nil {
			ShowError(err, 4005)
		}
		BufferInformation.Delete(stream.PlaylistID)
	}

	return status
}

func getBufTmpFiles(stream *ThisStream) (tmpFiles []string) {
	var tmpFolder = stream.Folder
	var fileIDs []float64

	if _, err := bufferVFS.Stat(tmpFolder); !fsIsNotExistErr(err) {
		files, err := bufferVFS.ReadDir(getPlatformPath(tmpFolder))
		if err != nil {
			ShowError(err, 000)
			return
		}

		if len(files) > 2 {
			for _, file := range files {
				var fileID = strings.Replace(file.Name(), ".ts", "", -1)
				var f, err = strconv.ParseFloat(fileID, 64)

				if err == nil {
					fileIDs = append(fileIDs, f)
				}
			}

			sort.Float64s(fileIDs)
			fileIDs = fileIDs[:len(fileIDs)-1]

			for _, file := range fileIDs {
				var fileName = fmt.Sprintf("%d.ts", int64(file))

				if indexOfString(fileName, stream.OldSegments) == -1 {
					tmpFiles = append(tmpFiles, fileName)
					stream.OldSegments = append(stream.OldSegments, fileName)
				}
			}
		}
	}

	return tmpFiles
}

func parseM3U8(stream *ThisStream) (err error) {
	var debug string
	var noNewSegment = false
	var lastSegmentDuration float64
	var segment Segment
	var m3u8Segments []Segment
	var sequence int64

	stream.DynamicBandwidth = false

	debug = fmt.Sprintf(`M3U8 Playlist:`+"\n"+`%s`, stream.Body)
	showDebug(debug, 3)

	var getBandwidth = func(line string) int {
		var infos = strings.Split(line, ",")

		for _, info := range infos {
			if strings.Contains(info, "BANDWIDTH=") {
				var bandwidth = strings.Replace(info, "BANDWIDTH=", "", -1)
				n, err := strconv.Atoi(bandwidth)
				if err == nil {
					return n
				}
			}
		}
		return 0
	}

	var parseParameter = func(line string, segment *Segment) (err error) {
		line = strings.Trim(line, "\r\n")

		var parameters = []string{"#EXT-X-VERSION:", "#EXT-X-PLAYLIST-TYPE:", "#EXT-X-MEDIA-SEQUENCE:", "#EXT-X-STREAM-INF:", "#EXTINF:"}

		for _, parameter := range parameters {
			if strings.Contains(line, parameter) {
				var value = strings.Replace(line, parameter, "", -1)

				switch parameter {
				case "#EXT-X-VERSION:":
					version, err := strconv.Atoi(value)
					if err == nil {
						segment.Version = version
					}
				case "#EXT-X-PLAYLIST-TYPE:":
					segment.PlaylistType = value
				case "#EXT-X-MEDIA-SEQUENCE:":
					n, err := strconv.ParseInt(value, 10, 64)
					if err == nil {
						stream.Sequence = n
						sequence = n
					}
				case "#EXT-X-STREAM-INF:":
					segment.Info = true
					segment.StreamInf.Bandwidth = getBandwidth(value)
				case "#EXTINF:":
					var d = strings.Split(value, ",")
					if len(d) > 0 {
						value = strings.Replace(d[0], ",", "", -1)
						duration, err := strconv.ParseFloat(value, 64)
						if err == nil {
							segment.Duration = duration
						} else {
							ShowError(err, 1050)
							return err
						}
					}
				}
			}
		}
		return
	}

	var parseURL = func(line string, segment *Segment) {
		_, err := url.ParseRequestURI(line)
		if err == nil {
			u, _ := url.Parse(line)

			if len(u.Host) == 0 {
				segment.URL = stream.URLStreamingServer + line
			} else {
				segment.URL = line
			}
		} else {
			var serverURLPath = strings.Replace(stream.M3U8URL, path.Base(stream.M3U8URL), line, -1)
			segment.URL = serverURLPath
		}
	}

	if strings.Contains(stream.Body, "#EXTM3U") {
		var lines = strings.Split(strings.Replace(stream.Body, "\r\n", "\n", -1), "\n")

		if !stream.DynamicBandwidth {
			stream.DynamicStream = make(map[int]DynamicStream)
		}

		for i, line := range lines {
			_ = i
			if len(line) > 0 {
				if line[0:1] == "#" {
					err := parseParameter(line, &segment)
					if err != nil {
						return err
					}
					lastSegmentDuration = segment.Duration
				}

				if segment.Info && len(line) > 0 && line[0:1] != "#" {
					var dynamicStream DynamicStream
					segment.Duration = 0
					noNewSegment = false
					stream.DynamicBandwidth = true
					parseURL(line, &segment)
					dynamicStream.Bandwidth = segment.StreamInf.Bandwidth
					dynamicStream.URL = segment.URL
					stream.DynamicStream[dynamicStream.Bandwidth] = dynamicStream
				}

				if segment.Duration > 0 && line[0:1] != "#" {
					parseURL(line, &segment)

					if len(segment.URL) > 0 {
						segment.Sequence = sequence
						m3u8Segments = append(m3u8Segments, segment)
						sequence++
					}
				}
			}
		}

	} else {
		err = errors.New(getErrMsg(4051))
		return
	}

	if len(m3u8Segments) > 0 {
		noNewSegment = true

		if !stream.Status {
			if len(m3u8Segments) >= 2 {
				m3u8Segments = m3u8Segments[0 : len(m3u8Segments)-1]
			}
		}

		for _, s := range m3u8Segments {
			segment = s

			if !stream.Status {
				noNewSegment = false
				stream.LastSequence = segment.Sequence

				if strings.ToUpper(segment.PlaylistType) == "VOD" {
					break
				}
			} else {
				if segment.Sequence > stream.LastSequence {
					stream.LastSequence = segment.Sequence
					noNewSegment = false
					break
				}
			}
		}
	}

	if !noNewSegment {
		if stream.DynamicBandwidth {
			switchBandwidth(stream)
		} else {
			stream.Segment = append(stream.Segment, segment)
		}
	}

	if noNewSegment {
		var sleep = lastSegmentDuration * 0.5
		for i := 0.0; i < sleep*1000; i = i + 100 {
			_ = i
			time.Sleep(time.Duration(100) * time.Millisecond)

			if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
				break
			}
		}
	}

	return
}

func switchBandwidth(stream *ThisStream) (err error) {
	var bandwidth []int
	var dynamicStream DynamicStream
	var segment Segment

	for key := range stream.DynamicStream {
		bandwidth = append(bandwidth, key)
	}

	sort.Ints(bandwidth)

	if len(bandwidth) > 0 {
		for i := range bandwidth {
			segment.StreamInf.Bandwidth = stream.DynamicStream[bandwidth[i]].Bandwidth
			dynamicStream = stream.DynamicStream[bandwidth[0]]

			if stream.NetworkBandwidth == 0 {
				dynamicStream = stream.DynamicStream[bandwidth[0]]
				break
			} else {
				if bandwidth[i] > stream.NetworkBandwidth {
					break
				}
				dynamicStream = stream.DynamicStream[bandwidth[i]]
			}
		}
	} else {
		err = errors.New("M3U8 does not contain streaming URLs")
		return
	}

	segment.URL = dynamicStream.URL
	segment.Duration = 0
	stream.Segment = append(stream.Segment, segment)

	return
}

// Buffer with FFMPEG/VLC
func thirdPartyBuffer(streamID int, playlistID string, useBackup bool, backupNumber int) {
	if playlist, exists := BufferInformation.Load(playlistID); exists {
		var debug, path, options, bufferType string
		var tmpSegment = 1
		var bufferSize = Settings.BufferSize * 1024
		var stream = playlist.Streams[streamID]
		var buf bytes.Buffer
		var fileSize = 0
		var streamStatus = make(chan bool)

		var tmpFolder = stream.Folder
		var url = stream.URL
		if useBackup {
			if backupNumber >= 1 && backupNumber <= 3 {
				switch backupNumber {
				case 1:
					url = stream.BackupChannel1URL
					showHighlight("START OF BACKUP 1 STREAM")
					showInfo("Backup Channel 1 URL: " + url)
				case 2:
					url = stream.BackupChannel2URL
					showHighlight("START OF BACKUP 2 STREAM")
					showInfo("Backup Channel 2 URL: " + url)
				case 3:
					url = stream.BackupChannel3URL
					showHighlight("START OF BACKUP 3 STREAM")
					showInfo("Backup Channel 3 URL: " + url)
				}
			}
		}

		stream.Status = false

		bufferType = strings.ToUpper(Settings.Buffer)

		switch Settings.Buffer {
		case "ffmpeg":
			path = Settings.FFmpegPath
			options = Settings.FFmpegOptions
		case "vlc":
			path = Settings.VLCPath
			options = Settings.VLCOptions
		default:
			return
		}

		var addErrorToStream = func(err error) {
			showDebug("ERROR ADDED TO STREAM", 3)

			if !useBackup || (useBackup && backupNumber >= 0 && backupNumber <= 3) {
				backupNumber++
				if playlist.Streams != nil && len(playlist.Streams) > 0 {
					if stream.BackupChannel1URL != "" || stream.BackupChannel2URL != "" || stream.BackupChannel3URL != "" {
						thirdPartyBuffer(streamID, playlistID, true, backupNumber)
					}
				}
				return
			}

			if playlist.Clients[streamID].Connection > 0 {
				var clients = playlist.Clients[streamID]
				clients.Error = err
				playlist.Clients[streamID] = clients
				BufferInformation.Store(playlistID, playlist)
			}
		}

		if err := bufferVFS.RemoveAll(getPlatformPath(tmpFolder)); err != nil {
			ShowError(err, 4005)
		}

		err := checkVFSFolder(tmpFolder, bufferVFS)
		if err != nil {
			ShowError(err, 0)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		err = checkFile(path)
		if err != nil {
			ShowError(err, 0)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		showInfo(fmt.Sprintf("%s path:%s", bufferType, path))
		showInfo("Streaming URL:" + url)

		var tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)

		f, err := bufferVFS.Create(tmpFile)
		f.Close()
		if err != nil {
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		var args []string

		for i, a := range strings.Split(options, " ") {
			switch bufferType {
			case "FFMPEG":
				a = strings.Replace(a, "[URL]", url, -1)
				if i == 0 {
					if len(Settings.UserAgent) != 0 {
						args = []string{"-user_agent", Settings.UserAgent}
					}
					if playlist.HttpProxyIP != "" && playlist.HttpProxyPort != "" {
						args = append(args, "-http_proxy", fmt.Sprintf("http://%s:%s", playlist.HttpProxyIP, playlist.HttpProxyPort))
					}
				}
				args = append(args, a)

			case "VLC":
				if a == "[URL]" {
					a = strings.Replace(a, "[URL]", url, -1)
					args = append(args, a)

					if len(Settings.UserAgent) != 0 {
						args = append(args, fmt.Sprintf(":http-user-agent=%s", Settings.UserAgent))
					}
					if playlist.HttpProxyIP != "" && playlist.HttpProxyPort != "" {
						args = append(args, fmt.Sprintf(":http-proxy=%s:%s", playlist.HttpProxyIP, playlist.HttpProxyPort))
					}
				} else {
					args = append(args, a)
				}
			}
		}

		var cmd = exec.Command(path, args...)

		debug = fmt.Sprintf("%s:%s %s", bufferType, path, args)
		showDebug(debug, 1)

		// Byte data from process
		stdOut, err := cmd.StdoutPipe()
		if err != nil {
			ShowError(err, 0)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		// Log data from the process
		logOut, err := cmd.StderrPipe()
		if err != nil {
			ShowError(err, 0)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		if len(buf.Bytes()) == 0 && !stream.Status {
			showInfo(bufferType + ":Processing data")
		}

		cmd.Start()
		defer cmd.Wait()

		go func() {
			// Show log data from the process in Debug Mode 1.
			scanner := bufio.NewScanner(logOut)
			scanner.Split(bufio.ScanLines)

			for scanner.Scan() {
				serviceText := strings.TrimSpace(scanner.Text())

				if strings.Contains(serviceText, "access stream error") {
					err = errors.New(serviceText)
					cmd.Process.Kill()
					killClientConnection(streamID, playlistID, false)
					addErrorToStream(err)
					cmd.Wait()
					return
				}

				debug = fmt.Sprintf("%s log:%s", bufferType, serviceText)

				select {
				case <-streamStatus:
					showDebug(debug, 1)
				default:
					showInfo(debug)
				}

				time.Sleep(time.Duration(10) * time.Millisecond)
			}
		}()

		f, err = bufferVFS.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		buffer := make([]byte, 1024*4)
		reader := bufio.NewReader(stdOut)

		t := make(chan int)

		go func() {
			var timeout = 0
			for {
				time.Sleep(time.Duration(1000) * time.Millisecond)
				timeout++

				select {
				case <-t:
					return
				default:
					t <- timeout
				}
			}
		}()

		for {
			select {
			case timeout := <-t:
				if timeout >= 5 && tmpSegment == 1 {
					cmd.Process.Kill()
					err = errors.New("Timout")
					ShowError(err, 4006)
					killClientConnection(streamID, playlistID, false)
					addErrorToStream(err)
					cmd.Wait()
					f.Close()
					return
				}

			default:
			}

			if fileSize == 0 && !stream.Status {
				showInfo("Streaming Status:Receive data from " + bufferType)
			}

			if !clientConnection(stream) {
				cmd.Process.Kill()
				killClientConnection(streamID, playlistID, false)
				addErrorToStream(err)
				f.Close()
				cmd.Wait()
				return
			}

			n, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}

			fileSize = fileSize + len(buffer[:n])

			if _, err := f.Write(buffer[:n]); err != nil {
				cmd.Process.Kill()
				ShowError(err, 0)
				killClientConnection(streamID, playlistID, false)
				addErrorToStream(err)
				cmd.Wait()
				return
			}

			if fileSize >= bufferSize/2 {
				if tmpSegment == 1 && !stream.Status {
					close(t)
					close(streamStatus)
					showInfo(fmt.Sprintf("Streaming Status:Buffering data from %s", bufferType))
				}

				f.Close()
				tmpSegment++

				if !stream.Status {
					BufferInformation.Lock()
					stream.Status = true
					playlist.Streams[streamID] = stream
					BufferInformation.Store(playlistID, playlist)
					BufferInformation.Unlock()
				}

				tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)

				fileSize = 0

				var errCreate, errOpen error
				_, errCreate = bufferVFS.Create(tmpFile)
				f, errOpen = bufferVFS.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
				if errCreate != nil || errOpen != nil {
					cmd.Process.Kill()
					ShowError(err, 0)
					killClientConnection(streamID, playlistID, false)
					addErrorToStream(err)
					cmd.Wait()
					return
				}
			}
		}

		cmd.Process.Kill()
		cmd.Wait()

		err = errors.New(bufferType + " error")
		killClientConnection(streamID, playlistID, false)
		addErrorToStream(err)
		ShowError(err, 1204)

		time.Sleep(time.Duration(500) * time.Millisecond)
		clientConnection(stream)

		return
	}
}

func getTuner(id, playlistType string) (tuner int) {
	switch Settings.Buffer {
	case "-":
		tuner = Settings.Tuner
	case "threadfin", "ffmpeg", "vlc":
		i, err := strconv.Atoi(getProviderParameter(id, playlistType, "tuner"))
		if err == nil {
			tuner = i
		} else {
			ShowError(err, 0)
			tuner = 1
		}
	}
	return
}

func initBufferVFS(virtual bool) {
	if virtual {
		bufferVFS = memfs.New()
	} else {
		bufferVFS = osfs.New()
	}
}

func debugRequest(req *http.Request) {
	var debugLevel = 3
	if System.Flag.Debug < debugLevel {
		return
	}

	var debug string

	fmt.Println()
	debug = "Request:* * * * * * BEGIN HTTP(S) REQUEST * * * * * * "
	showDebug(debug, debugLevel)

	debug = fmt.Sprintf("Method:%s", req.Method)
	showDebug(debug, debugLevel)

	debug = fmt.Sprintf("Proto:%s", req.Proto)
	showDebug(debug, debugLevel)

	debug = fmt.Sprintf("URL:%s", req.URL)
	showDebug(debug, debugLevel)

	for name, headers := range req.Header {
		name = strings.ToLower(name)

		for _, h := range headers {
			debug = fmt.Sprintf("Header:%v: %v", name, h)
			showDebug(debug, debugLevel)
		}
	}

	debug = "Request:* * * * * * END HTTP(S) REQUEST * * * * * *"
	showDebug(debug, debugLevel)

	return
}

func debugResponse(resp *http.Response) {
	var debugLevel = 3
	if System.Flag.Debug < debugLevel {
		return
	}

	var debug string

	fmt.Println()

	debug = "Response:* * * * * * BEGIN RESPONSE * * * * * * "
	showDebug(debug, debugLevel)

	debug = fmt.Sprintf("Proto:%s", resp.Proto)
	showDebug(debug, debugLevel)

	debug = fmt.Sprintf("Status Code:%d", resp.StatusCode)
	showDebug(debug, debugLevel)

	debug = fmt.Sprintf("Status Text:%s", http.StatusText(resp.StatusCode))
	showDebug(debug, debugLevel)

	for key, value := range resp.Header {
		switch fmt.Sprintf("%T", value) {
		case "[]string":
			debug = fmt.Sprintf("Header:%v: %s", key, strings.Join(value, " "))
		default:
			debug = fmt.Sprintf("Header:%v: %v", key, value)
		}

		showDebug(debug, debugLevel)
	}

	debug = "Pesponse:* * * * * * END RESPONSE * * * * * * "
	showDebug(debug, debugLevel)

	return
}

func terminateProcessGracefully(cmd *exec.Cmd) {
	if cmd.Process != nil {
		// Send a SIGTERM to the process
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// If an error occurred while trying to send the SIGTERM, you might resort to a SIGKILL.
			ShowError(err, 0)
			cmd.Process.Kill()
		}

		// Optionally, you can wait for the process to finish too
		cmd.Wait()
	}
}

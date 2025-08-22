package src

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// --- WS timing constants ---
const (
	wsWriteWait  = 10 * time.Second    // time to write a message
	wsPongWait   = 60 * time.Second    // allowed time to read the next pong
	wsPingPeriod = wsPongWait * 9 / 10 // send pings ~10% sooner than pong deadline
)

// Client represents one WS connection.
// IMPORTANT: only Client.writePump writes to the socket.
type Client struct {
	conn *websocket.Conn
	send chan []byte // outbound messages (JSON bytes)
	// carry the auth token from the URL so we can reissue a fresh one in replies
	token string
}

// Hub holds all connected clients and handles fan-out/broadcast.
type Hub struct {
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	clients    map[*Client]bool
}

var wsHub = &Hub{
	register:   make(chan *Client),
	unregister: make(chan *Client),
	broadcast:  make(chan []byte, 1024),
	clients:    make(map[*Client]bool),
}

// Run the hub forever.
func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
				_ = c.conn.Close()
			}

		case msg := <-h.broadcast:
			// Push to everyone; drop slow clients.
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					h.unregister <- c
				}
			}
		}
	}
}

// helper: enqueue a JSON-able message to all clients
func wsBroadcast(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	wsHub.broadcast <- b
}

// helper: enqueue a JSON-able message to one client
func replyJSON(c *Client, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.send <- b:
	default:
		// slow client: drop it
		wsHub.unregister <- c
	}
}

// Only goroutine that WRITES to the websocket.
func (c *Client) writePump() {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		wsHub.unregister <- c
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				// hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(msg); err != nil {
				_ = w.Close()
				return
			}
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Reads messages and dispatches commands.
// Never writes to the socket directly—use replyJSON.
func (c *Client) readPump() {
	defer func() { wsHub.unregister <- c }()
	c.conn.SetReadLimit(1 << 20)
	_ = c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var req RequestStruct
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}
		handleWSCommand(c, req)
	}
}

// finalize common fields and send
func finalizeAndSend(c *Client, resp *ResponseStruct, newToken string) {
	if newToken != "" {
		resp.Token = newToken
	}
	*resp = setDefaultResponseData(*resp, true)
	if System.ConfigurationWizard == true {
		resp.ConfigurationWizard = System.ConfigurationWizard
	}
	replyJSON(c, *resp)
}

// Core of your previous switch(request.Cmd). Rewritten to use replyJSON/finalizeAndSend.
func handleWSCommand(c *Client, request RequestStruct) {
	var response ResponseStruct
	response.Status = true

	// --- auth/token handling (as before) ---
	var newToken string
	systemMutex.Lock()
	authWeb := Settings.AuthenticationWEB
	confWizard := System.ConfigurationWizard
	systemMutex.Unlock()

	if !confWizard && authWeb {
		// validate token from URL once per message (same as before)
		var err error
		newToken, err = tokenAuthentication(c.token)
		if err != nil {
			response.Status = false
			response.Reload = true
			response.Error = err.Error()
			finalizeAndSend(c, &response, "")
			return
		}
	}

	switch request.Cmd {

	case "getServerConfig":
		response.Settings = Settings
		response.Cmd = "getServerConfig"
		finalizeAndSend(c, &response, newToken)
		return

	case "updateLog":
		response.Cmd = "updateLog"
		finalizeAndSend(c, &response, newToken)
		return

	case "ping":
		replyJSON(c, ResponseStruct{Status: true, Cmd: "pong"})
		return

	// ---- Data write commands ----
	case "saveSettings":
		var authenticationUpdate = Settings.AuthenticationWEB
		var err error
		response.Settings, err = updateServerSettings(request)
		if err == nil {
			response.OpenMenu = strconv.Itoa(indexOfString("settings", System.WEB.Menu))
			response.Alert = "Settings saved"
			if Settings.AuthenticationWEB == true && authenticationUpdate == false {
				response.Reload = true
			}
			initBufferVFS()
		} else {
			response.Status = false
			response.Error = err.Error()
		}
		finalizeAndSend(c, &response, newToken)
		return

	case "saveFilesM3U":
		if err := saveFiles(request, "m3u"); err == nil {
			response.OpenMenu = strconv.Itoa(indexOfString("m3u", System.WEB.Menu))
		} else {
			response.Status = false
			response.Error = err.Error()
		}
		finalizeAndSend(c, &response, newToken)
		// background processing
		go func() { _ = processStartupWorkflow("save", "m3u") }()
		return

	case "updateFileM3U":
		finalizeAndSend(c, &response, newToken)
		go func() { _ = processStartupWorkflow("update", "m3u") }()
		return

	case "saveFilesHDHR":
		finalizeAndSend(c, &response, newToken)
		go func() { _ = processStartupWorkflow("save", "hdhr") }()
		return

	case "updateFileHDHR":
		go func() { _ = processStartupWorkflow("update", "hdhr") }()
		finalizeAndSend(c, &response, newToken)
		return

	case "saveFilesXMLTV":
		if err := saveFiles(request, "xmltv"); err == nil {
			response.OpenMenu = strconv.Itoa(indexOfString("xmltv", System.WEB.Menu))
			finalizeAndSend(c, &response, newToken)
			go func() { _ = processStartupWorkflow("save", "xmltv") }()
		} else {
			response.Status = false
			response.Error = err.Error()
			finalizeAndSend(c, &response, newToken)
		}
		return

	case "updateFileXMLTV":
		finalizeAndSend(c, &response, newToken)
		go func() { _ = processStartupWorkflow("update", "xmltv") }()
		return

	case "saveFilter":
		var err error
		response.Settings, err = saveFilter(request)
		if err == nil {
			response.OpenMenu = strconv.Itoa(indexOfString("filter", System.WEB.Menu))
		} else {
			response.Status = false
			response.Error = err.Error()
		}
		finalizeAndSend(c, &response, newToken)
		return

	case "saveEpgMapping":
		if err := saveXEpgMapping(request); err != nil {
			response.Status = false
			response.Error = err.Error()
		}
		finalizeAndSend(c, &response, newToken)
		return

	case "saveUserData":
		if err := saveUserData(request); err == nil {
			response.OpenMenu = strconv.Itoa(indexOfString("users", System.WEB.Menu))
		} else {
			response.Status = false
			response.Error = err.Error()
		}
		finalizeAndSend(c, &response, newToken)
		return

	case "saveNewUser":
		if err := saveNewUser(request); err == nil {
			response.OpenMenu = strconv.Itoa(indexOfString("users", System.WEB.Menu))
		} else {
			response.Status = false
			response.Error = err.Error()
		}
		finalizeAndSend(c, &response, newToken)
		return

	case "resetLogs":
		logMutex.Lock()
		WebScreenLog.Log = make([]string, 0)
		WebScreenLog.Errors = 0
		WebScreenLog.Warnings = 0
		logMutex.Unlock()
		response.OpenMenu = strconv.Itoa(indexOfString("log", System.WEB.Menu))
		finalizeAndSend(c, &response, newToken)
		return

	case "ThreadfinBackup":
		if file, err := ThreadfinBackup(); err == nil {
			response.OpenLink = fmt.Sprintf("%s://%s/download/%s", System.ServerProtocol.WEB, System.Domain, file)
		} else {
			response.Status = false
			response.Error = err.Error()
		}
		finalizeAndSend(c, &response, newToken)
		return

	case "ThreadfinRestore":
		logMutex.Lock()
		WebScreenLog.Log = make([]string, 0)
		WebScreenLog.Errors = 0
		WebScreenLog.Warnings = 0
		logMutex.Unlock()

		if len(request.Base64) > 0 {
			newWebURL, err := ThreadfinRestoreFromWeb(request.Base64)
			if err != nil {
				response.Alert = err.Error()
				response.Status = false
				response.Error = err.Error()
			} else {
				if len(newWebURL) > 0 {
					response.Alert = "Backup was successfully restored.\nThe port of the sTeVe URL has changed, you have to restart Threadfin.\nAfter a restart, Threadfin can be reached again at the following URL:\n" + newWebURL
				} else {
					response.Alert = "Backup was successfully restored."
					response.Reload = true
				}
				showInfo("Threadfin:" + "Backup successfully restored.")
			}
		}
		finalizeAndSend(c, &response, newToken)
		return

	case "uploadLogo":
		if len(request.Base64) > 0 {
			if url, err := uploadLogo(request.Base64, request.Filename); err == nil {
				response.LogoURL = url
				replyJSON(c, response) // early return style kept
				return
			} else {
				response.Status = false
				response.Error = err.Error()
				finalizeAndSend(c, &response, newToken)
				return
			}
		}
		finalizeAndSend(c, &response, newToken)
		return

	case "saveWizard":
		nextStep, err := saveWizard(request)
		if err == nil {
			if nextStep == 10 {
				System.ConfigurationWizard = false
				response.Reload = true
			} else {
				response.Wizard = nextStep
			}
		} else {
			response.Status = false
			response.Error = err.Error()
		}
		finalizeAndSend(c, &response, newToken)
		return

	case "probeChannel":
		resolution, frameRate, audioChannels, _ := probeChannel(request)
		response.ProbeInfo = ProbeInfoStruct{Resolution: resolution, FrameRate: frameRate, AudioChannel: audioChannels}
		finalizeAndSend(c, &response, newToken)
		return

	default:
		// unknown command
		response.Status = false
		response.Error = "unknown command"
		finalizeAndSend(c, &response, newToken)
		return
	}
}

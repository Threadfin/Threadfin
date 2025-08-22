// Global WebSocket connection
var globalWS = null;
var WS_AVAILABLE = false;
var SERVER_CONNECTION = false;
var pendingRequests = [];
var isConnecting = false;
// Initialize persistent WebSocket connection
function initWebSocket() {
    // Prevent multiple simultaneous connection attempts
    if (isConnecting || globalWS) {
        return; // Already connecting or a connection exists
    }
    isConnecting = true;
    var protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    var url = protocol + "//" + window.location.hostname + ":" + window.location.port + "/data/?Token=" + getCookie("Token");
    globalWS = new WebSocket(url);
    globalWS.onopen = function () {
        isConnecting = false;
        WS_AVAILABLE = true;
        SERVER_CONNECTION = false;
        console.log("WebSocket connected");
        // Send any pending requests
        while (pendingRequests.length > 0) {
            const request = pendingRequests.shift();
            if (request) {
                if (globalWS.readyState === WebSocket.OPEN) {
                    globalWS.send(JSON.stringify(request));
                }
                else {
                    // Put it back in the queue if not ready
                    pendingRequests.unshift(request);
                    break;
                }
            }
        }
        // Automatically send updateLog to get server data and create menu
        setTimeout(() => {
            if (globalWS && globalWS.readyState === WebSocket.OPEN) {
                const updateLogData = { cmd: "updateLog" };
                globalWS.send(JSON.stringify(updateLogData));
            }
        }, 100); // Small delay to ensure connection is fully ready
    };
    globalWS.onerror = function (e) {
        console.log("WebSocket error:", e);
        SERVER_CONNECTION = false;
        WS_AVAILABLE = false;
        // Don't close the connection on error - let it try to recover
    };
    globalWS.onclose = function () {
        isConnecting = false;
        WS_AVAILABLE = false;
        SERVER_CONNECTION = false;
        console.log("WebSocket disconnected");
        // Clear the globalWS reference
        globalWS = null;
        // Only reconnect if we're not already trying to connect and no other connection exists
        setTimeout(() => {
            if (!globalWS && !isConnecting && document.readyState === 'complete') {
                initWebSocket();
            }
        }, 5000);
    };
    globalWS.onmessage = function (e) {
        SERVER_CONNECTION = false;
        console.log("RESPONSE:");
        var response = JSON.parse(e.data);
        console.log(response);
        if (response.hasOwnProperty("token")) {
            document.cookie = "Token=" + response["token"];
        }
        if (response["status"] == false) {
            alert(response["err"]);
            if (response.hasOwnProperty("reload")) {
                location.reload();
            }
            return;
        }
        if (response.hasOwnProperty("probeInfo")) {
            if (document.getElementById("probeDetails")) {
                if (response["probeInfo"]["resolution"] !== undefined) {
                    document.getElementById("probeDetails").innerHTML = "<p>Resolution: <span class='text-primary'>" + response["probeInfo"]["resolution"] + "</span></p><p>Frame Rate: <span class='text-primary'>" + response["probeInfo"]["frameRate"] + " FPS</span></p><p>Audio: <span class='text-primary'>" + response["probeInfo"]["audioChannel"] + "</span></p>";
                }
            }
        }
        if (response.hasOwnProperty("logoURL")) {
            var div = document.getElementById("channel-icon");
            div.value = response["logoURL"];
            div.className = "changed";
            return;
        }
        // Handle different command types
        if (response.hasOwnProperty("cmd")) {
            switch (response["cmd"]) {
                case "updateLog":
                    // Merge all response data into SERVER object
                    if (!SERVER) {
                        SERVER = new Object();
                    }
                    // Polyfill for Object.assign if not available
                    if (typeof Object.assign === 'function') {
                        Object.assign(SERVER, response);
                    }
                    else {
                        // Manual merge for older browsers
                        for (var key in response) {
                            if (response.hasOwnProperty(key)) {
                                SERVER[key] = response[key];
                            }
                        }
                    }
                    console.log("updateLog: SERVER object after merge:", SERVER);
                    console.log("updateLog: SERVER.clientInfo:", SERVER["clientInfo"]);
                    if (document.getElementById("content_log")) {
                        showLogs(false);
                    }
                    if (document.getElementById("playlist-connection-information")) {
                        let activeClass = "text-primary";
                        if (response["clientInfo"]["activePlaylist"] / response["clientInfo"]["totalPlaylist"] >= 0.6 && response["clientInfo"]["activePlaylist"] / response["clientInfo"]["totalPlaylist"] < 0.8) {
                            activeClass = "text-warning";
                        }
                        else if (response["clientInfo"]["activePlaylist"] / response["clientInfo"]["activePlaylist"] >= 0.8) {
                            activeClass = "text-danger";
                        }
                        document.getElementById("playlist-connection-information").innerHTML = "Playlist Connections: <span class='" + activeClass + "'>" + response["clientInfo"]["activePlaylist"] + " / " + response["clientInfo"]["totalPlaylist"] + "</span>";
                    }
                    if (document.getElementById("client-connection-information")) {
                        let activeClass = "text-primary";
                        if (response["clientInfo"]["activeClients"] / response["clientInfo"]["totalClients"] >= 0.6 && response["clientInfo"]["activeClients"] / response["clientInfo"]["totalClients"] < 0.8) {
                            activeClass = "text-warning";
                        }
                        else if (response["clientInfo"]["activeClients"] / response["clientInfo"]["totalClients"] >= 0.8) {
                            activeClass = "text-danger";
                        }
                        document.getElementById("client-connection-information").innerHTML = "Client Connections: <span class='" + activeClass + "'>" + response["clientInfo"]["activeClients"] + " / " + response["clientInfo"]["activeClients"] + "</span>";
                    }
                    // Don't handle progress updates in updateLog responses - they're handled separately
                    // Progress updates should only come through the "progress" case
                    // ALWAYS call createLayout for updateLog responses
                    console.log("updateLog: About to call createLayout");
                    try {
                        if (typeof createLayout === 'function') {
                            console.log("updateLog: createLayout function exists, calling it");
                            createLayout();
                            console.log("updateLog: createLayout completed");
                        }
                        else {
                            console.error("updateLog: createLayout function does not exist!");
                        }
                    }
                    catch (error) {
                        console.error("updateLog: Error in createLayout:", error);
                    }
                    break;
                case "progress":
                    // Handle progress updates ONLY
                    if (response.hasOwnProperty("progress")) {
                        updateHeaderProgress(response["progress"]);
                        // Close modal when operation completes (100% and not processing)
                        if (response["progress"]["percentage"] >= 100 && !response["progress"]["isProcessing"]) {
                            showElement("popup", false);
                        }
                    }
                    break;
                default:
                    // Merge response data into SERVER object instead of replacing it
                    if (!SERVER) {
                        SERVER = new Object();
                    }
                    // Polyfill for Object.assign if not available
                    if (typeof Object.assign === 'function') {
                        Object.assign(SERVER, response);
                    }
                    else {
                        // Manual merge for older browsers
                        for (var key in response) {
                            if (response.hasOwnProperty(key)) {
                                SERVER[key] = response[key];
                            }
                        }
                    }
                    break;
            }
        }
        if (response.hasOwnProperty("openMenu")) {
            var menu = document.getElementById(response["openMenu"]);
            menu.click();
            showElement("popup", false);
        }
        if (response.hasOwnProperty("openLink")) {
            window.location = response["openLink"];
        }
        if (response.hasOwnProperty("alert")) {
            showToast("", response["alert"], "warning");
        }
        if (response.hasOwnProperty("reload")) {
            location.reload();
        }
        if (response.hasOwnProperty("wizard")) {
            configurationWizard[response["wizard"]].createWizard();
            return;
        }
    };
}
class Server {
    constructor(cmd) {
        this.cmd = cmd;
    }
    request(data) {
        if (SERVER_CONNECTION)
            return;
        SERVER_CONNECTION = true;
        if (this.cmd !== "updateLog") {
            UNDO = new Object();
        }
        // ✅ Flattened payload (no "data" wrapper)
        const payload = Object.assign({ cmd: this.cmd }, data);
        if (globalWS && globalWS.readyState === WebSocket.OPEN) {
            console.log("WS SEND:", payload);
            globalWS.send(JSON.stringify(payload));
        }
        else {
            initWebSocket();
            // ✅ Queue the final payload as-is
            pendingRequests.push(payload);
        }
    }
}
// Initialize WebSocket connection when the page loads
document.addEventListener('DOMContentLoaded', function () {
    console.log("DOM loaded, initializing WebSocket");
    initWebSocket();
});
// Also initialize for updateLog command
function updateLog() {
    console.log("updateLog called");
    if (!globalWS || globalWS.readyState !== WebSocket.OPEN) {
        console.log("updateLog: WebSocket not ready, attempting connection");
        // Only try to connect if no connection exists
        if (!globalWS) {
            initWebSocket();
        }
        setTimeout(updateLog, 100);
        return;
    }
    var data = { cmd: "updateLog" };
    globalWS.send(JSON.stringify(data));
}
function getCookie(name) {
    var value = "; " + document.cookie;
    var parts = value.split("; " + name + "=");
    if (parts.length == 2)
        return parts.pop().split(";").shift();
}

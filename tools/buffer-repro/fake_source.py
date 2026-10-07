"""Fake IPTV source for reproducing Threadfin stream-lifecycle bugs.

GET /live/<name>.ts   loops test.ts forever at about real time (2 Mbps).
GET /hiccup/<name>.ts like /live, but pauses for HICCUP_SECS after
                      STALL_AFTER bytes, then continues.
GET /stall/<name>.ts  sends STALL_AFTER bytes at real time, then sends
                      nothing more but keeps the connection open, like a
                      provider that hangs mid-stream.
Every request is logged with open/close, so the log shows how many
upstream connections Threadfin holds at once.
"""
import http.server
import socketserver
import threading
import time

DATA = open("/src/test.ts", "rb").read()
RATE = 2_000_000 // 8          # bytes per second
CHUNK = 188 * 70               # whole TS packets
STALL_AFTER = 3_000_000
HICCUP_SECS = 8

lock = threading.Lock()
open_conns = {}


def log(msg):
    with lock:
        print(f"{time.strftime('%H:%M:%S')} {msg} | open upstream: {len(open_conns)} {sorted(open_conns.values())}", flush=True)


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.0"

    def log_message(self, *a):
        pass

    def do_GET(self):
        stall = self.path.startswith("/stall/")
        hiccup = self.path.startswith("/hiccup/")
        if not (stall or hiccup or self.path.startswith("/live/")):
            self.send_error(404)
            return
        key = id(self)
        with lock:
            open_conns[key] = self.path
        log(f"OPEN  {self.path}")
        try:
            self.send_response(200)
            self.send_header("Content-Type", "video/mp2t")
            self.end_headers()
            sent, pos = 0, 0
            while True:
                if stall and sent >= STALL_AFTER:
                    time.sleep(1)          # hold the connection, send nothing
                    continue
                if hiccup and STALL_AFTER <= sent < STALL_AFTER + CHUNK:
                    log(f"PAUSE {self.path} for {HICCUP_SECS}s")
                    time.sleep(HICCUP_SECS)
                piece = DATA[pos:pos + CHUNK]
                pos = (pos + CHUNK) % len(DATA)
                self.wfile.write(piece)
                sent += len(piece)
                time.sleep(len(piece) / RATE)
        except (BrokenPipeError, ConnectionResetError):
            pass
        finally:
            with lock:
                open_conns.pop(key, None)
            log(f"CLOSE {self.path}")


class Server(socketserver.ThreadingMixIn, http.server.HTTPServer):
    daemon_threads = True


Server(("0.0.0.0", 8000), Handler).serve_forever()

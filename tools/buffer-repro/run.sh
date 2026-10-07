#!/bin/bash
# Reproduce Threadfin buffer-lifecycle problems with the ffmpeg buffer:
# orphaned ffmpeg processes (and their upstream connections) after clients
# leave a stalled stream, and duplicate processes after a reconnect.
#
# Needs only Docker. Everything runs on a private Docker network, against a
# local fake source (fake_source.py). No IPTV provider is involved.
#
# Usage: ./run.sh <threadfin image> [scenario ...]
#   e.g. ./run.sh fyb3roptik/threadfin:latest
#        ./run.sh threadfin:dev s2 s6
# Scenarios: s1 s2 s3 s4 s5 s6 (default: all)
set -u
IMAGE=${1:?usage: run.sh <threadfin image> [scenario ...]}; shift
HERE=$(cd "$(dirname "$0")" && pwd)
WORK=$(mktemp -d)
NET=tfrepro
PY=python:3.12-alpine
CURL=curlimages/curl

cleanup() {
  docker rm -f tfrepro-src tfrepro-tf $(docker ps -aq --filter name=tfrepro-cl-) >/dev/null 2>&1
  docker network rm $NET >/dev/null 2>&1
  docker run --rm -v "$WORK":/w $PY sh -c 'rm -rf /w/*' >/dev/null 2>&1; rmdir "$WORK" 2>/dev/null
}
trap cleanup EXIT

wait_ready() { for _ in $(seq 1 90); do docker logs --since "$1" tfrepro-tf 2>&1 | grep -q "Ready to use" && return 0; sleep 1; done; echo "Threadfin did not start. Last log lines:"; docker logs tfrepro-tf 2>&1 | tail -15; exit 1; }

# ---- environment -----------------------------------------------------------
mkdir -p "$WORK/src" "$WORK/conf"
cp "$HERE/fake_source.py" "$WORK/src/"
docker network create --internal $NET >/dev/null

echo "Generating a 20 s test clip with the image's own ffmpeg..."
docker run --rm -v "$WORK/src":/out --entrypoint ffmpeg "$IMAGE" -v error \
  -f lavfi -i testsrc=size=640x360:rate=30 -f lavfi -i sine=frequency=440 -t 20 \
  -c:v mpeg2video -b:v 1700k -g 30 -c:a mp2 -f mpegts /out/test.ts

docker run -d --name tfrepro-src --network $NET -v "$WORK/src":/src $PY python -u /src/fake_source.py >/dev/null

cat > "$WORK/conf/repro.m3u" <<'EOF'
#EXTM3U
#EXTINF:-1 tvg-id="a" group-title="Repro",Live A
http://tfrepro-src:8000/live/a.ts
#EXTINF:-1 tvg-id="b" group-title="Repro",Live B
http://tfrepro-src:8000/live/b.ts
#EXTINF:-1 tvg-id="c" group-title="Repro",Stall C
http://tfrepro-src:8000/stall/c.ts
#EXTINF:-1 tvg-id="d" group-title="Repro",Hiccup D
http://tfrepro-src:8000/hiccup/d.ts
EOF
{ echo '<?xml version="1.0" encoding="UTF-8"?>'; echo '<tv>'
  for c in a b c d; do echo "  <channel id=\"$c\"><display-name>$c</display-name></channel>"
    echo "  <programme start=\"20200101000000 +0000\" stop=\"20991231000000 +0000\" channel=\"$c\"><title>$c</title></programme>"; done
  echo '</tv>'; } > "$WORK/conf/repro.xml"

# First start writes Threadfin's default settings; then point them at the test files.
docker run -d --name tfrepro-tf --network $NET -v "$WORK/conf":/home/threadfin/conf "$IMAGE" >/dev/null
for _ in $(seq 1 60); do [ -s "$WORK/conf/settings.json" ] && break; sleep 1; done; sleep 2
docker stop tfrepro-tf >/dev/null
docker run --rm -i -v "$WORK/conf":/c $PY python - <<'EOF'
import json
p = "/c/settings.json"; s = json.load(open(p))
base = {"counter.download": 0, "counter.error": 0, "http_proxy.ip": "", "http_proxy.port": "",
        "last.update": "", "provider.availability": 100, "compatibility": {}}
s.update({"epgSource": "XEPG", "buffer": "ffmpeg", "buffer.size.kb": 512, "tuner": 2, "files.update": True,
          "ignoreFilters": True, "dummy": True, "ssdp": False, "bindIpAddress": "0.0.0.0"})
s["files"] = {"hdhr": {},
  "m3u": {"MREPRO00000000000001": dict(base, **{"buffer": "ffmpeg", "description": "repro",
      "file.source": "/home/threadfin/conf/repro.m3u", "file.threadfin": "MREPRO00000000000001.m3u",
      "http_headers.origin": "", "http_headers.referer": "", "id.provider": "MREPRO00000000000001",
      "name": "Repro", "tuner": "2", "type": "m3u"})},
  "xmltv": {"XREPRO00000000000001": dict(base, **{"description": "repro",
      "file.source": "/home/threadfin/conf/repro.xml", "file.threadfin": "XREPRO00000000000001.xml",
      "id.provider": "XREPRO00000000000001", "name": "Repro", "type": "xmltv"})}}
json.dump(s, open(p, "w"), indent=2)
EOF
T0=$(date +%s); docker start tfrepro-tf >/dev/null; wait_ready "$T0"
echo "Threadfin under test: $(docker logs tfrepro-tf 2>&1 | grep -oE 'Version: +[^ ].*' | head -1 | tr -s ' ')"

M3U=$(docker exec tfrepro-src wget -qO- http://tfrepro-tf:34400/m3u/threadfin.m3u)
url() { echo "$M3U" | grep -A1 ",$1\$" | tail -1 | sed -E 's#http://[^/]+#http://tfrepro-tf:34400#'; }
A=$(url "Live A"); B=$(url "Live B"); C=$(url "Stall C"); D=$(url "Hiccup D")
[ -n "$A" ] && [ -n "$D" ] || { echo "Threadfin did not serve the test channels"; exit 1; }

# ---- helpers -------------------------------------------------------------
# client <name> <seconds> <url>: watch a channel for N seconds, then disconnect
client() { docker run -d --rm --name "tfrepro-cl-$1" --network $NET $CURL -s -o /dev/null --max-time "$2" "$3" >/dev/null; }
nffmpeg() { docker exec tfrepro-tf pgrep ffmpeg | wc -l | tr -d ' '; }
# Established TCP connections from Threadfin to the fake source (port 8000 = 1F40).
nupstream() { docker exec tfrepro-tf sh -c 'cat /proc/net/tcp /proc/net/tcp6 2>/dev/null' | awk '$3 ~ /:1F40$/ && $4 == "01"' | wc -l | tr -d ' '; }
state() { echo "  $(printf '%-28s' "$1") ffmpeg: $(nffmpeg)  upstream connections: $(nupstream)"; }
reset() {
  docker rm -f $(docker ps -aq --filter name=tfrepro-cl-) >/dev/null 2>&1
  local t; t=$(date +%s); docker restart tfrepro-tf >/dev/null; wait_ready "$t"; sleep 2; SINCE=$(date +%s)
}
summary() {
  local L; L=$(docker logs --since "$SINCE" tfrepro-tf 2>&1)
  echo "  panics: $(echo "$L" | grep -c '^panic')  'FFMPEG error' (1204): $(echo "$L" | grep -c 'EC: 1204')  tuner-limit hits: $(echo "$L" | grep -c 'No new connections')"
}

# ---- scenarios -----------------------------------------------------------
s1() { echo "S1 baseline: watch Live A for 10 s, leave"
  client a1 10 "$A"; sleep 6; state "while watching"; sleep 12; state "8 s after leaving"; }

s2() { echo "S2 source stalls (data stops after ~12 s, connection stays open); viewer leaves at 20 s"
  client c1 20 "$C"; sleep 15; state "stalled, still watching"; sleep 15; state "10 s after leaving"; sleep 20; state "30 s after leaving"; }

s3() { echo "S3 source stalls, viewer reconnects to the same channel"
  client c1 18 "$C"; sleep 19; client c2 15 "$C"; sleep 5; state "after reconnect"; sleep 20; state "10 s after both left"; }

s4() { echo "S4 channel change during startup: A plays, B opens, A leaves before B's first segment, then C opens"
  client a1 4 "$A"; sleep 3; client b1 30 "$B"; sleep 4; state "A left, B playing"
  client c1 8 "$C"; sleep 3; state "C opened as 2nd stream"; sleep 10; state "C gone"; }

s5() { echo "S5 source pauses 8 s mid-stream; viewer reconnects during the pause; data resumes"
  client d1 14 "$D"; sleep 14.5; client d2 25 "$D"; sleep 12; state "after resume"; sleep 16; state "10 s after both left"; }

s6() { echo "S6 like S5, but the new connection opens 0.3 s before the old one closes (player retry)"
  client d1 14 "$D"; sleep 13.7
  docker run --rm --name tfrepro-cl-d2 --network $NET $CURL -s -o /dev/null --max-time 30 -w "%{size_download}" "$D" > "$WORK/d2.bytes" &
  sleep 14; state "after resume"; wait
  echo "  second viewer received $(( $(cat "$WORK/d2.bytes") / 1024 )) KB"; sleep 3; state "after both left"; }

for s in ${@:-s1 s2 s3 s4 s5 s6}; do reset; $s; summary; done

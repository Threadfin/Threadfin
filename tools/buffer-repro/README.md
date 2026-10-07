# Buffer lifecycle repro

Reproduces stuck ffmpeg processes with the `ffmpeg` buffer. After all
clients leave a stream, the process, and its upstream connection, can
stay open. Threadfin no longer counts that connection against the
tuner limit, so the provider sees more connections than the limit
allows.

Everything runs in Docker, on a private network, against a local fake
source. No IPTV provider is needed.

```
./run.sh fyb3roptik/threadfin:latest      # stock image
./run.sh threadfin:dev s2 s6              # your build, selected scenarios
```

The script needs only Docker. It pulls `python:3.12-alpine` and
`curlimages/curl`, and it makes its test clip with the ffmpeg inside
the Threadfin image under test. It removes its containers, network,
and temp files when it exits.

## What it does

`fake_source.py` serves four channels:

| Channel | Path | Behaviour |
|---|---|---|
| Live A, Live B | `/live/` | Loops a test clip at about 2 Mbps, forever. |
| Stall C | `/stall/` | Sends about 3 MB, then sends nothing, but keeps the connection open. This is like a provider that hangs. |
| Hiccup D | `/hiccup/` | Sends about 3 MB, pauses for 8 s, then continues. |

Threadfin is set up with the `ffmpeg` buffer, `tuner` 2, and
`buffer.size.kb` 512. Each scenario restarts Threadfin. Clients are
`curl` containers that watch a channel for a set time, then
disconnect. After each step, the script counts `ffmpeg` processes in
the Threadfin container, and established TCP connections from
Threadfin to the fake source.

| Scenario | Steps |
|---|---|
| S1 | Watch a live channel, then leave. This is the baseline. |
| S2 | Watch a channel that stalls, then leave. |
| S3 | Watch a channel that stalls, then reconnect to the same channel. |
| S4 | Change channel while the new channel is still starting. |
| S5 | The source pauses. The viewer reconnects during the pause, then data resumes. |
| S6 | Like S5, but the new connection opens 0.3 s before the old one closes, as some players do on a retry. |

## Results on 1.2.37

Counts after every client has left (`ffmpeg` processes / upstream
connections):

| Scenario | Stock 1.2.37 | With the fix |
|---|---|---|
| S1 | 0 / 0 | 0 / 0 |
| S2 | **1 / 1** (still open 30 s later) | 0 / 0 |
| S3 | **2 / 2** | 0 / 0 |
| S4 | correct (only the client still watching) | correct |
| S5 | **1 / 1** (and 2 / 2 for one viewer while watching) | 0 / 0 (1 / 1 while watching) |
| S6 | **1 / 1** (and 2 / 2 for one viewer while watching) | 0 / 0 (1 / 1 while watching) |

In S6, the second viewer receives the same amount of data with and
without the fix.

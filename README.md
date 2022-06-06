<div align="center" style="background-color: #111; padding: 100;">
    <a href="https://github.com/Threadfin/Threadfin"><img width="285" height="80" src="html/img/threadfin.png" alt="Threadfin" /></a>
</div>
<br>

# Threadfin
## M3U Proxy for Plex DVR and Emby/Jellyfin Live TV. Based on xTeVe.

Documentation for setup and configuration is [here](https://github.com/xteve-project/xTeVe-Documentation/blob/master/en/configuration.md).

#### Donation
* **Bitcoin:** 3AyZ16uGWGJ4HdvZ3wx4deE5HsKZ5QxFBL  
![Bitcoin](html/img/BC-QR.png "Bitcoin - Threadfin")

## Requirements
### Plex
* Plex Media Server (1.11.1.4730 or newer)
* Plex Client with DVR support
* Plex Pass

### Emby
* Emby Server (3.5.3.0 or newer)
* Emby Client with Live-TV support
* Emby Premiere

--- 

## Features

#### Files
* Merge external M3U files
* Merge external XMLTV files
* Automatic M3U and XMLTV update
* M3U and XMLTV export

#### Channel management
* Filtering streams
* Channel mapping
* Channel order
* Channel logos
* Channel categories

#### Streaming
* Buffer with HLS / M3U8 support
* Re-streaming
* Number of tuners adjustable
* Compatible with Plex / Emby EPG

---

#### Recommended Docker Image (Linux 64 Bit)
Thanks to @alturismo and @LeeD for creating the Docker Images.

**Created by alturismo:**  
[xTeVe](https://hub.docker.com/r/alturismo/xteve)  
[xTeVe / Guide2go](https://hub.docker.com/r/alturismo/xteve_guide2go)  
[xTeVe / Guide2go / owi2plex](https://hub.docker.com/r/alturismo/xteve_g2g_owi)

Including:  
- Guide2go: XMLTV grabber for Schedules Direct  
- owi2plex: XMLTV file grabber for Enigma receivers

**Created by LeeD:**  
[xTeVe / Guide2go / Zap2XML](https://hub.docker.com/r/dnsforge/xteve)  

Including:  
- Guide2go: XMLTV grabber for Schedules Direct  
- Zap2XML: Perl based zap2it XMLTV grabber  
- Bash: A Unix / Linux shell  
- Crond: Daemon to execute scheduled commands  
- Perl: Programming language   

---

### Threadfin Beta branch
New features and bug fixes are only available in beta branch. Only after successful testing are they are merged into the master branch.

**It is not recommended to use the beta version in a production system.**  

With the command line argument `branch` the Git Branch can be changed. xTeVe must be started via the terminal.  

#### Switch from master to beta branch:
```
xteve -branch beta

...
[Threadfin] GitHub:                https://github.com/Threadfin
[Threadfin] Git Branch:            beta [Threadfin]
...
```

#### Switch from beta to master branch:
```
threadfin -branch master

...
[Threadfin] GitHub:                https://github.com/Threadfin
[Threadfin] Git Branch:            master [Threadfin]
...
```

When the branch is changed, an update is only performed if there is a new version and the update function is activated in the settings.  

---

## Build from source code [Go / Golang]

#### Requirements
* [Go](https://golang.org) (go1.16.2 or newer)

#### Dependencies
* [go-ssdp](https://github.com/koron/go-ssdp)
* [websocket](https://github.com/gorilla/websocket)
* [osext](https://github.com/kardianos/osext)

#### Build
1. Download source code
2. Install dependencies
```
go mod tidy
```
3. Build Threadfin
```
go build threadfin.go
```

4. Update web files (optional)

If TypeScript files were changed, run:

```sh
tsc -p ./ts/tsconfig.json
```

Then, to embed updated JavaScript files into the source code (src/webUI.go), run it in development mode at least once:

```sh
go build threadfin.go
threadfin -dev
```

---

## Fork without pull request :mega:
When creating a fork, the Threadfin GitHub account must be changed from the source code or the update function disabled.
Future updates of Threadfin would update your fork. :wink:

threadfin.go - Line: 29
```Go
var GitHub = GitHubStruct{Branch: "main", User: "Threadfin", Repo: "Threadfin", Update: true}

/*
  Branch: GitHub Branch
  User:   GitHub Username
  Repo:   GitHub Repository
  Update: Automatic updates from the GitHub repository [true|false]
*/

```



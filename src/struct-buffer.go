package src

import (
	"sync"
	"time"
)

// Playlist : Contains all playlist information required by the buffer
type Playlist struct {
	Folder        string
	PlaylistID    string
	PlaylistName  string
	Tuner         int
	HttpProxyIP   string
	HttpProxyPort string

	Clients map[int]*ThisClient
	Streams map[int]*ThisStream
}

// ThisClient : Client information
type ThisClient struct {
	Connection int
	Error      error
}

// ThisStream : Contains information about the stream to be played for a playlist
type ThisStream struct {
	ChannelName       string
	Error             string
	Folder            string
	MD5               string
	NetworkBandwidth  int
	PlaylistID        string
	PlaylistName      string
	Status            bool
	URL               string
	BackupChannel1URL string
	BackupChannel2URL string
	BackupChannel3URL string

	Segment []Segment

	// Server information
	Location           string
	URLFile            string
	URLHost            string
	URLPath            string
	URLRedirect        string
	URLScheme          string
	URLStreamingServer string

	// Used only for HLS / M3U8
	Body             string
	Difference       float64
	Duration         float64
	DynamicBandwidth bool
	FirstSequence    int64
	HLS              bool
	LastSequence     int64
	M3U8URL          string
	NewSegCount      int
	OldSegCount      int
	Sequence         int64
	TimeDiff         float64
	TimeEnd          time.Time
	TimeSegDuration  float64
	TimeStart        time.Time
	Version          int
	Wait             float64

	DynamicStream map[int]DynamicStream

	// Local temp files
	OldSegments []string
}

// Segment : URL segments (HLS / M3U8)
type Segment struct {
	Duration     float64
	Info         bool
	PlaylistType string
	Sequence     int64
	URL          string
	Version      int
	Wait         float64

	StreamInf struct {
		AverageBandwidth int
		Bandwidth        int
		Framerate        float64
		Resolution       string
		SegmentURL       string
	}
}

// DynamicStream : Stream information for dynamic bandwidth
type DynamicStream struct {
	AverageBandwidth int
	Bandwidth        int
	Framerate        float64
	Resolution       string
	URL              string
}

// ClientConnection : Client connections
type ClientConnection struct {
	Connection int
	Error      error
}

// BandwidthCalculation : Bandwidth calculation for the stream
type BandwidthCalculation struct {
	NetworkBandwidth int
	Size             int
	Start            time.Time
	Stop             time.Time
	TimeDiff         float64
}

// BufferDetails holds information about the buffer
type BufferDetails struct {
	sync.RWMutex
	Playlist      map[string]*Playlist
	ClientCount   int
	PlaylistCount int
}

// NewBufferDetails initializes a new BufferDetails instance
func NewBufferDetails() *BufferDetails {
	return &BufferDetails{
		Playlist: make(map[string]*Playlist),
	}
}

// Load retrieves a playlist from the buffer
func (bd *BufferDetails) Load(key string) (*Playlist, bool) {
	bd.RLock()
	defer bd.RUnlock()
	playlist, ok := bd.Playlist[key]
	return playlist, ok
}

// Store stores a playlist in the buffer
func (bd *BufferDetails) Store(key string, playlist *Playlist) {
	bd.Lock()
	defer bd.Unlock()
	bd.Playlist[key] = playlist
}

// Delete removes a playlist from the buffer
func (bd *BufferDetails) Delete(key string) {
	bd.Lock()
	defer bd.Unlock()
	delete(bd.Playlist, key)
}

// Range iterates over the buffer's playlists
func (bd *BufferDetails) Range(f func(key string, playlist *Playlist) bool) {
	bd.RLock()
	defer bd.RUnlock()
	for k, v := range bd.Playlist {
		if !f(k, v) {
			break
		}
	}
}

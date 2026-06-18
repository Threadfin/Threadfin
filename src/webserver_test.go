package src

import "testing"

func TestRewriteMulticastURLWithUDPxy(t *testing.T) {
	tests := []struct {
		name      string
		streamURL string
		udpxy     string
		want      string
	}{
		{
			name:      "udp multicast",
			streamURL: "udp://@239.3.1.54:4120",
			udpxy:     "127.0.0.1:4022",
			want:      "http://127.0.0.1:4022/udp/239.3.1.54:4120/",
		},
		{
			name:      "rtp multicast",
			streamURL: "rtp://239.3.1.54:4120",
			udpxy:     "127.0.0.1:4022",
			want:      "http://127.0.0.1:4022/rtp/239.3.1.54:4120/",
		},
		{
			name:      "rtp multicast with http udpxy url",
			streamURL: "rtp://239.3.1.54:4120",
			udpxy:     "http://127.0.0.1:4022",
			want:      "http://127.0.0.1:4022/rtp/239.3.1.54:4120/",
		},
		{
			name:      "rtp multicast with https udpxy url",
			streamURL: "rtp://239.3.1.54:4120",
			udpxy:     "https://127.0.0.1:4022",
			want:      "https://127.0.0.1:4022/rtp/239.3.1.54:4120/",
		},
		{
			name:      "rtp multicast with trailing slash udpxy url",
			streamURL: "rtp://239.3.1.54:4120",
			udpxy:     "https://127.0.0.1:4022/",
			want:      "https://127.0.0.1:4022/rtp/239.3.1.54:4120/",
		},
		{
			name:      "rtp multicast with at sign",
			streamURL: "rtp://@239.3.1.54:4120",
			udpxy:     "127.0.0.1:4022",
			want:      "http://127.0.0.1:4022/rtp/239.3.1.54:4120/",
		},
		{
			name:      "rtp ipv6 multicast",
			streamURL: "rtp://[ff15::1]:4120",
			udpxy:     "127.0.0.1:4022",
			want:      "http://127.0.0.1:4022/rtp/[ff15::1]:4120/",
		},
		{
			name:      "rtp ipv6 multicast with at sign",
			streamURL: "rtp://@[ff15::1]:4120",
			udpxy:     "127.0.0.1:4022",
			want:      "http://127.0.0.1:4022/rtp/[ff15::1]:4120/",
		},
		{
			name:      "http stream",
			streamURL: "http://example.com/live.ts",
			udpxy:     "127.0.0.1:4022",
			want:      "http://example.com/live.ts",
		},
		{
			name:      "udp unicast",
			streamURL: "udp://192.168.1.10:4120",
			udpxy:     "127.0.0.1:4022",
			want:      "udp://192.168.1.10:4120",
		},
		{
			name:      "udp hostname",
			streamURL: "udp://example.com:4120",
			udpxy:     "127.0.0.1:4022",
			want:      "udp://example.com:4120",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteMulticastURLWithUDPxy(tt.streamURL, tt.udpxy)
			if got != tt.want {
				t.Fatalf("rewriteMulticastURLWithUDPxy(%q, %q) = %q, want %q", tt.streamURL, tt.udpxy, got, tt.want)
			}
		})
	}
}

package security_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dennys-bd/gonext/core/security"
)

func TestIPExtractor(t *testing.T) {
	tests := []struct {
		name                  string
		trusted               []string
		remoteAddr, xff, want string
	}{
		{
			name:       "no proxies ignores forwarding headers",
			trusted:    nil,
			remoteAddr: "203.0.113.5:1234",
			xff:        "198.51.100.9",
			want:       "203.0.113.5",
		},
		{
			name:       "peer inside a listed range is walked through",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.1.2.3:1234",
			xff:        "198.51.100.9",
			want:       "198.51.100.9",
		},
		{
			name:       "peer outside every range is the client",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "203.0.113.5:1234",
			xff:        "198.51.100.9",
			want:       "203.0.113.5",
		},
		{
			name:       "loopback is not trusted unless listed",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "127.0.0.1:1234",
			xff:        "198.51.100.9",
			want:       "127.0.0.1",
		},
		{
			name:       "chain stops at the first untrusted hop",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.1.2.3:1234",
			xff:        "1.2.3.4, 198.51.100.9, 10.9.9.9",
			want:       "198.51.100.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := security.IPExtractor(tt.trusted)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set("X-Forwarded-For", tt.xff)

			if got := extractor(req); got != tt.want {
				t.Errorf("IPExtractor(%v)(req) = %q, want %q", tt.trusted, got, tt.want)
			}
		})
	}
}

func TestIPExtractor_RejectsBadCIDR(t *testing.T) {
	extractor, err := security.IPExtractor([]string{"10.0.0.0/8", "nope"})
	if extractor != nil {
		t.Errorf("expected a nil extractor, got %v", extractor)
	}
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected an error naming %q, got %v", "nope", err)
	}
}

func TestIPExtractor_RejectsCatchAllRange(t *testing.T) {
	for _, cidr := range []string{"0.0.0.0/0", "::/0"} {
		extractor, err := security.IPExtractor([]string{cidr})
		if extractor != nil || err == nil || !strings.Contains(err.Error(), cidr) {
			t.Errorf("%s: expected a nil extractor and an error naming it, got %v, %v", cidr, extractor, err)
		}
	}
}

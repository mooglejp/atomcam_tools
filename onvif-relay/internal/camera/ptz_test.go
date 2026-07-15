package camera

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
)

func newPTZTestClient(t *testing.T, handler http.HandlerFunc) (*Client, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	host, portString, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		server.Close()
		t.Fatalf("failed to split test server address: %v", err)
	}

	var port int
	if _, err := fmt.Sscanf(portString, "%d", &port); err != nil {
		server.Close()
		t.Fatalf("failed to parse test server port: %v", err)
	}

	client := NewClient(&config.CameraConfig{Host: host, HTTPPort: port})
	return client, func() {
		client.Close()
		server.Close()
	}
}

func TestPTZGetPosition(t *testing.T) {
	client, closeClient := newPTZTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/cmd.cgi" || r.URL.Query().Get("name") != "status" {
			t.Fatalf("unexpected request URL: %s", r.URL.String())
		}
		fmt.Fprintln(w, "TIMESTAMP=2026/07/15 12:00:00")
		fmt.Fprintln(w, "MOTORPOS=123.6 87.4 0 0 1")
	})
	defer closeClient()

	pan, tilt, err := client.PTZGetPosition()
	if err != nil {
		t.Fatalf("PTZGetPosition returned an error: %v", err)
	}
	if pan != 124 || tilt != 87 {
		t.Fatalf("PTZGetPosition = (%d, %d), want (124, 87)", pan, tilt)
	}
}

func TestPTZGetPositionMissingMotorPosition(t *testing.T) {
	client, closeClient := newPTZTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "TIMESTAMP=2026/07/15 12:00:00")
	})
	defer closeClient()

	if _, _, err := client.PTZGetPosition(); err == nil {
		t.Fatal("PTZGetPosition returned nil error without MOTORPOS")
	}
}

func TestPTZGetPositionRejectsOutOfRangePosition(t *testing.T) {
	client, closeClient := newPTZTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "MOTORPOS=400 90 0 0 1")
	})
	defer closeClient()

	if _, _, err := client.PTZGetPosition(); err == nil {
		t.Fatal("PTZGetPosition returned nil error for an out-of-range position")
	}
}

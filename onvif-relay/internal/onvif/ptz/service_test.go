package ptz

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/camera"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
)

func newTrackingTestService(t *testing.T, presets []config.PTZPreset) (*Service, <-chan string, func()) {
	t.Helper()

	commands := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request camera.CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		commands <- request.Exec
	}))

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

	cfg := &config.Config{Cameras: []config.CameraConfig{
		{
			Name:     "swing",
			Host:     host,
			HTTPPort: port,
			Capabilities: config.CapabilitiesConfig{
				PTZ: true,
			},
			Streams: []config.StreamConfig{
				{ProfileName: "Main"},
			},
			PTZ: config.PTZConfig{Presets: presets},
		},
	}}
	registry, err := camera.NewRegistry(cfg)
	if err != nil {
		server.Close()
		t.Fatalf("failed to create registry: %v", err)
	}

	return NewService(registry), commands, func() {
		registry.Close()
		server.Close()
	}
}

func TestGotoPresetTrackingAction(t *testing.T) {
	service, commands, closeService := newTrackingTestService(t, []config.PTZPreset{
		{Name: "Tracking On", Token: "tracking-on", Tracking: "on"},
	})
	defer closeService()

	if err := service.GotoPreset("Main", "tracking-on", nil); err != nil {
		t.Fatalf("GotoPreset returned an error: %v", err)
	}
	if command := <-commands; command != "property tracking on" {
		t.Fatalf("command = %q, want property tracking on", command)
	}
}

func TestSendAuxiliaryTrackingOff(t *testing.T) {
	service, commands, closeService := newTrackingTestService(t, nil)
	defer closeService()

	response, err := service.SendAuxiliaryCommand("Main", trackingAuxiliaryOff)
	if err != nil {
		t.Fatalf("SendAuxiliaryCommand returned an error: %v", err)
	}
	if response != trackingAuxiliaryOff {
		t.Fatalf("response = %q, want %q", response, trackingAuxiliaryOff)
	}
	if command := <-commands; command != "property tracking off" {
		t.Fatalf("command = %q, want property tracking off", command)
	}
}

func TestMoveAndStartTracking(t *testing.T) {
	service, commands, closeService := newTrackingTestService(t, nil)
	defer closeService()

	if err := service.MoveAndStartTracking(MoveAndStartTrackingRequest{ProfileToken: "Main"}); err != nil {
		t.Fatalf("MoveAndStartTracking returned an error: %v", err)
	}
	if command := <-commands; command != "property tracking on" {
		t.Fatalf("command = %q, want property tracking on", command)
	}
}

func TestGetNodesAdvertisesTrackingAndPresetCount(t *testing.T) {
	service, _, closeService := newTrackingTestService(t, []config.PTZPreset{
		{Name: "Tracking On", Tracking: "on"},
		{Name: "Tracking Off", Tracking: "off"},
	})
	defer closeService()

	nodes := service.GetNodes().PTZNode
	if len(nodes) != 1 {
		t.Fatalf("node count = %d, want 1", len(nodes))
	}
	if nodes[0].MaximumNumberOfPresets != 2 {
		t.Fatalf("MaximumNumberOfPresets = %d, want 2", nodes[0].MaximumNumberOfPresets)
	}
	if len(nodes[0].AuxiliaryCommands) != 2 {
		t.Fatalf("AuxiliaryCommands count = %d, want 2", len(nodes[0].AuxiliaryCommands))
	}
}

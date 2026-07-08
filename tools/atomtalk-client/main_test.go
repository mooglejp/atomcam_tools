package main

import (
	"context"
	"fmt"
	"testing"
)

func TestParseFirstDShowAudioDevice(t *testing.T) {
	output := `[dshow @ 000001] DirectShow video devices (some may be both video and audio devices)
[dshow @ 000001]  "Integrated Camera"
[dshow @ 000001]     Alternative name "@device_pnp_foo"
[dshow @ 000001] DirectShow audio devices
[dshow @ 000001]  "Microphone (USB Audio Device)"
[dshow @ 000001]     Alternative name "@device_cm_bar"
[dshow @ 000001]  "Stereo Mix"
`
	got, ok := parseFirstDShowAudioDevice(output)
	if !ok {
		t.Fatal("expected audio device")
	}
	if got != "Microphone (USB Audio Device)" {
		t.Fatalf("unexpected device: %q", got)
	}
}

func TestParseFirstDShowAudioDeviceMissing(t *testing.T) {
	if got, ok := parseFirstDShowAudioDevice("DirectShow audio devices\nAlternative name \"x\""); ok {
		t.Fatalf("unexpected device: %q", got)
	}
}

func TestIsExpectedShutdownErrorRequiresCanceledContext(t *testing.T) {
	err := fmt.Errorf("Post %q: write tcp 192.0.2.1:55250->192.0.2.2:8080: use of closed network connection", "http://relay/talk/camera")
	if isExpectedShutdownError(context.Background(), err) {
		t.Fatal("unexpected shutdown error match without cancellation")
	}
}

func TestIsExpectedShutdownErrorMatchesClosedNetworkConnectionAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := fmt.Errorf("Post %q: write tcp 192.0.2.1:55250->192.0.2.2:8080: use of closed network connection", "http://relay/talk/camera")
	if !isExpectedShutdownError(ctx, err) {
		t.Fatal("expected closed network connection to match after cancellation")
	}
}

func TestIsExpectedShutdownErrorMatchesContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if !isExpectedShutdownError(ctx, context.Canceled) {
		t.Fatal("expected context.Canceled to match after cancellation")
	}
}

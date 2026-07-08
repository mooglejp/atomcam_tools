package main

import "testing"

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

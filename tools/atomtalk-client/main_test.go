package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
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

func TestBuildFileFFmpegArgs(t *testing.T) {
	got := buildFileFFmpegArgs(`D:\Git\Irodori-TTS\outputs\no-leave.wav`, "")
	want := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-re",
		"-i", `D:\Git\Irodori-TTS\outputs\no-leave.wav`,
		"-vn",
		"-ac", "1",
		"-ar", "8000",
		"-acodec", "pcm_s16le",
		"-f", "s16le",
		"-",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args:\ngot  %#v\nwant %#v", got, want)
	}
}

func TestBuildCaptureFFmpegArgsKeepsExplicitFormat(t *testing.T) {
	got := buildFFmpegArgs("wav", `D:\Git\Irodori-TTS\outputs\no-leave.wav`, "-re")
	want := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-fflags", "nobuffer",
		"-re",
		"-f", "wav",
		"-i", `D:\Git\Irodori-TTS\outputs\no-leave.wav`,
		"-vn",
		"-ac", "1",
		"-ar", "8000",
		"-acodec", "pcm_s16le",
		"-f", "s16le",
		"-",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args:\ngot  %#v\nwant %#v", got, want)
	}
}

func TestCountingReaderCountsBytes(t *testing.T) {
	reader := &countingReader{r: strings.NewReader("abc")}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "abc" {
		t.Fatalf("unexpected body: %q", string(got))
	}
	if reader.n != 3 {
		t.Fatalf("unexpected byte count: %d", reader.n)
	}
}

func TestTailPaddingReaderAppendsSilenceAfterData(t *testing.T) {
	reader := &tailPaddingReader{
		r:         strings.NewReader("abc"),
		remaining: 4,
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{'a', 'b', 'c', 0, 0, 0, 0}
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected body: %#v", got)
	}
}

func TestTailPaddingReaderDoesNotPadEmptyInput(t *testing.T) {
	reader := &tailPaddingReader{
		r:         strings.NewReader(""),
		remaining: 4,
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("unexpected body: %#v", got)
	}
}

func TestResolveTailMSDefaultsToFileOnly(t *testing.T) {
	if got, err := resolveTailMS("", -1); err != nil || got != 0 {
		t.Fatalf("capture default tail = %d, %v", got, err)
	}
	if got, err := resolveTailMS("voice.wav", -1); err != nil || got != 1000 {
		t.Fatalf("file default tail = %d, %v", got, err)
	}
	if got, err := resolveTailMS("voice.wav", 1500); err != nil || got != 1500 {
		t.Fatalf("configured tail = %d, %v", got, err)
	}
	if _, err := resolveTailMS("voice.wav", 10001); err == nil {
		t.Fatal("expected out-of-range tail-ms error")
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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"time"
)

const (
	sampleRate     = 8000
	channels       = 1
	bytesPerSample = 2
)

func defaultFormat() string {
	switch runtime.GOOS {
	case "windows":
		return "dshow"
	case "darwin":
		return "avfoundation"
	default:
		return "pulse"
	}
}

func defaultInput() string {
	switch runtime.GOOS {
	case "windows":
		return "default"
	case "darwin":
		return ":0"
	default:
		return "default"
	}
}

func splitExtraArgs(s string) []string {
	fields := strings.Fields(s)
	if fields == nil {
		return []string{}
	}
	return fields
}

func buildFFmpegArgs(format, input, extra string) []string {
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-fflags", "nobuffer",
	}
	args = append(args, splitExtraArgs(extra)...)
	args = append(args,
		"-f", format,
		"-i", input,
		"-vn",
		"-ac", fmt.Sprint(channels),
		"-ar", fmt.Sprint(sampleRate),
		"-acodec", "pcm_s16le",
		"-f", "s16le",
		"-",
	)
	return args
}

func resolveCaptureInput(ffmpegPath, format, input string) (string, string, error) {
	if runtime.GOOS != "windows" {
		return format, input, nil
	}

	if strings.EqualFold(format, "auto") {
		format = "dshow"
		input = "default"
	}
	if !strings.EqualFold(format, "dshow") || input != "default" {
		return format, input, nil
	}

	device, err := firstDShowAudioDevice(ffmpegPath)
	if err != nil {
		return "", "", err
	}
	return "dshow", "audio=" + device, nil
}

func firstDShowAudioDevice(ffmpegPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffmpegPath, "-hide_banner", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	output, _ := cmd.CombinedOutput()
	if device, ok := parseFirstDShowAudioDevice(string(output)); ok {
		return device, nil
	}
	if ctx.Err() != nil {
		return "", fmt.Errorf("ffmpeg DirectShow device listing timed out")
	}
	return "", fmt.Errorf("failed to find a DirectShow audio device; run `ffmpeg -list_devices true -f dshow -i dummy` and pass -input \"audio=<device name>\"")
}

func parseFirstDShowAudioDevice(output string) (string, bool) {
	inAudioDevices := false
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "DirectShow audio devices") {
			inAudioDevices = true
			continue
		}
		if strings.Contains(line, "DirectShow video devices") {
			inAudioDevices = false
			continue
		}
		if !inAudioDevices || strings.Contains(line, "Alternative name") {
			continue
		}
		first := strings.IndexByte(line, '"')
		if first < 0 {
			continue
		}
		rest := line[first+1:]
		last := strings.IndexByte(rest, '"')
		if last <= 0 {
			continue
		}
		return rest[:last], true
	}
	return "", false
}

func sendControl(conn *net.UDPConn, token, command string) {
	var line string
	if token != "" {
		line = "ATOMTALK " + token
		if command != "" {
			line += " " + command
		}
	} else {
		line = "ATOMTALK"
		if command != "" {
			line += " " + command
		}
	}
	_, _ = conn.Write([]byte(line + "\n"))
}

func readOptionalReply(conn *net.UDPConn) {
	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 128)
	n, _, err := conn.ReadFromUDP(buf)
	_ = conn.SetReadDeadline(time.Time{})
	if err == nil && n > 0 {
		_, _ = os.Stderr.Write(buf[:n])
	}
}

func run() error {
	var (
		host       = flag.String("host", "", "camera IP address or host name")
		port       = flag.Int("port", 4010, "camera atomtalkd UDP port")
		token      = flag.String("token", "", "optional atomtalkd token")
		relayURL   = flag.String("relay-url", "", "optional ONVIF relay talk URL, e.g. http://relay:8080/talk/camera1")
		relayUser  = flag.String("relay-user", "", "ONVIF relay basic auth username")
		relayPass  = flag.String("relay-pass", "", "ONVIF relay basic auth password")
		ffmpegPath = flag.String("ffmpeg", "ffmpeg", "ffmpeg executable path")
		format     = flag.String("format", defaultFormat(), "ffmpeg input format")
		input      = flag.String("input", defaultInput(), "ffmpeg input device")
		frameMS    = flag.Int("frame-ms", 40, "UDP audio frame size in milliseconds")
		extraArgs  = flag.String("ffmpeg-args", "", "extra ffmpeg arguments inserted before -f/-i")
	)
	flag.Parse()

	if *host == "" && *relayURL == "" {
		return fmt.Errorf("either -host or -relay-url is required")
	}
	if *port <= 0 || *port > 65535 {
		return fmt.Errorf("-port must be between 1 and 65535")
	}
	if *frameMS < 10 || *frameMS > 80 {
		return fmt.Errorf("-frame-ms must be between 10 and 80")
	}

	frameBytes := sampleRate * channels * bytesPerSample * *frameMS / 1000
	if frameBytes%2 != 0 {
		frameBytes++
	}
	if frameBytes <= 0 || frameBytes > 1400 {
		return fmt.Errorf("computed frame size %d is outside safe UDP payload size", frameBytes)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	captureFormat, captureInput, err := resolveCaptureInput(*ffmpegPath, *format, *input)
	if err != nil {
		return err
	}

	args := buildFFmpegArgs(captureFormat, captureInput, *extraArgs)
	cmd := exec.CommandContext(ctx, *ffmpegPath, args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var streamErr error
	if *relayURL != "" {
		streamErr = streamHTTP(ctx, stdout, *relayURL, *relayUser, *relayPass, frameBytes)
	} else {
		streamErr = streamUDP(ctx, stdout, *host, *port, *token, frameBytes)
	}
	if streamErr != nil {
		_ = cmd.Process.Kill()
		if isExpectedShutdownError(ctx, streamErr) {
			_ = cmd.Wait()
			return nil
		}
		return streamErr
	}

	if err := cmd.Wait(); err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}

func streamUDP(ctx context.Context, r io.Reader, host string, port int, token string, frameBytes int) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if token != "" {
		sendControl(conn, token, "")
		readOptionalReply(conn)
	}
	defer sendControl(conn, token, "STOP")

	frame := make([]byte, frameBytes)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := io.ReadFull(r, frame)
		if n > 0 {
			if _, err := conn.Write(frame[:n]); err != nil {
				return err
			}
		}
		if readErr == nil {
			continue
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			return nil
		}
		return readErr
	}
}

func streamHTTP(ctx context.Context, r io.Reader, relayURL, username, password string, frameBytes int) error {
	pr, pw := io.Pipe()
	copyDone := make(chan error, 1)
	go func() {
		_, err := io.CopyBuffer(pw, r, make([]byte, frameBytes))
		if closeErr := pw.CloseWithError(err); err == nil {
			err = closeErr
		}
		copyDone <- err
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, relayURL, pr)
	if err != nil {
		_ = pw.Close()
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_ = pr.Close()
		_ = pw.Close()
		if isExpectedShutdownError(ctx, err) {
			return nil
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = pr.Close()
		_ = pw.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("relay returned %s: %s", resp.Status, string(body))
	}

	if err := <-copyDone; err != nil && !isExpectedShutdownError(ctx, err) {
		return err
	}
	return nil
}

func isExpectedShutdownError(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() == nil {
		return false
	}
	if errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.ErrClosedPipe) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "read/write on closed pipe")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "atomtalk-client:", err)
		os.Exit(1)
	}
}

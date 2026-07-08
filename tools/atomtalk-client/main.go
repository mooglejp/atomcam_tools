package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
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
		return "wasapi"
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
		ffmpegPath = flag.String("ffmpeg", "ffmpeg", "ffmpeg executable path")
		format     = flag.String("format", defaultFormat(), "ffmpeg input format")
		input      = flag.String("input", defaultInput(), "ffmpeg input device")
		frameMS    = flag.Int("frame-ms", 40, "UDP audio frame size in milliseconds")
		extraArgs  = flag.String("ffmpeg-args", "", "extra ffmpeg arguments inserted before -f/-i")
	)
	flag.Parse()

	if *host == "" {
		return fmt.Errorf("-host is required")
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

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if *token != "" {
		sendControl(conn, *token, "")
		readOptionalReply(conn)
	}
	defer sendControl(conn, *token, "STOP")

	args := buildFFmpegArgs(*format, *input, *extraArgs)
	cmd := exec.CommandContext(ctx, *ffmpegPath, args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	frame := make([]byte, frameBytes)
	for {
		n, readErr := io.ReadFull(stdout, frame)
		if n > 0 {
			if _, err := conn.Write(frame[:n]); err != nil {
				_ = cmd.Process.Kill()
				return err
			}
		}
		if readErr == nil {
			continue
		}
		if ctx.Err() != nil || readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		_ = cmd.Process.Kill()
		return readErr
	}

	if err := cmd.Wait(); err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "atomtalk-client:", err)
		os.Exit(1)
	}
}

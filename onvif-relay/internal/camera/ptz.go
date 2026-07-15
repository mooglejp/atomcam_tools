package camera

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// PTZMove moves the camera to the specified pan/tilt position with given speed
// pan: 0-355 degrees (0 = center, rotates clockwise)
// tilt: 0-180 degrees (0 = looking up, 90 = horizontal, 180 = looking down)
// speed: 1-9 (1 = slowest, 9 = fastest)
func (c *Client) PTZMove(pan, tilt, speed int) error {
	// Validate ranges
	if pan < 0 || pan > 355 {
		return fmt.Errorf("pan out of range: %d (must be 0-355)", pan)
	}
	if tilt < 0 || tilt > 180 {
		return fmt.Errorf("tilt out of range: %d (must be 0-180)", tilt)
	}
	if speed < 1 || speed > 9 {
		return fmt.Errorf("speed out of range: %d (must be 1-9)", speed)
	}

	// Send move command
	// Format: move <pan> <tilt> <speed>
	command := fmt.Sprintf("move %d %d %d", pan, tilt, speed)
	return c.SendCommand(command)
}

// PTZStop stops PTZ movement
func (c *Client) PTZStop() error {
	// AtomCam doesn't need explicit stop command
	// ContinuousMove sends absolute positions, camera stops automatically when it reaches the target
	// Sending "move 0 0 0" would cause the camera to move to position (0,0) which is wrong
	return nil
}

// PTZGetPosition gets the current PTZ position from AtomCam's status endpoint.
func (c *Client) PTZGetPosition() (pan, tilt int, err error) {
	url := fmt.Sprintf("http://%s:%d/cgi-bin/cmd.cgi?name=status", c.cfg.Host, c.cfg.HTTPPort)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get PTZ position: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("unexpected status code while getting PTZ position: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read PTZ position: %w", err)
	}

	for _, line := range strings.Split(string(body), "\n") {
		if !strings.HasPrefix(line, "MOTORPOS=") {
			continue
		}

		fields := strings.Fields(strings.TrimPrefix(line, "MOTORPOS="))
		if len(fields) < 2 {
			return 0, 0, fmt.Errorf("invalid MOTORPOS response: %q", line)
		}

		panValue, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid MOTORPOS pan value %q: %w", fields[0], err)
		}
		tiltValue, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid MOTORPOS tilt value %q: %w", fields[1], err)
		}

		pan = int(math.Round(panValue))
		tilt = int(math.Round(tiltValue))
		if pan < 0 || pan > 355 || tilt < 0 || tilt > 180 {
			return 0, 0, fmt.Errorf("MOTORPOS out of range: pan=%d tilt=%d", pan, tilt)
		}
		return pan, tilt, nil
	}

	return 0, 0, fmt.Errorf("MOTORPOS not found in status response")
}

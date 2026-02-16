package camera

import (
	"fmt"
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
	// Stop command - move to current position with speed 0
	// Some implementations use a specific stop command, but AtomCam uses move with speed 0
	command := "move 0 0 0"
	return c.SendCommand(command)
}

// PTZGetPosition gets current PTZ position (stub - AtomCam doesn't provide position feedback)
func (c *Client) PTZGetPosition() (pan, tilt int, err error) {
	// Note: AtomCam's cmd.cgi doesn't provide a command to query current position
	// This is a limitation of the camera firmware
	// Return error indicating not supported
	return 0, 0, fmt.Errorf("PTZ position query not supported by AtomCam")
}

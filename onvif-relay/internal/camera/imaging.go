package camera

import (
	"fmt"
)

// SetBrightness sets the brightness level
// ONVIF value: 0.0-1.0 → AtomCam value: 0-255 (center: 128)
func (c *Client) SetBrightness(value float64) error {
	atomcamValue := onvifToAtomCamValue(value)
	command := fmt.Sprintf("video bri %d", atomcamValue)
	return c.SendCommand(command)
}

// SetContrast sets the contrast level
// ONVIF value: 0.0-1.0 → AtomCam value: 0-255 (center: 128)
func (c *Client) SetContrast(value float64) error {
	atomcamValue := onvifToAtomCamValue(value)
	command := fmt.Sprintf("video cont %d", atomcamValue)
	return c.SendCommand(command)
}

// SetSaturation sets the saturation level
// ONVIF value: 0.0-1.0 → AtomCam value: 0-255 (center: 128)
func (c *Client) SetSaturation(value float64) error {
	atomcamValue := onvifToAtomCamValue(value)
	command := fmt.Sprintf("video sat %d", atomcamValue)
	return c.SendCommand(command)
}

// SetSharpness sets the sharpness level
// ONVIF value: 0.0-1.0 → AtomCam value: 0-255 (center: 128)
func (c *Client) SetSharpness(value float64) error {
	atomcamValue := onvifToAtomCamValue(value)
	command := fmt.Sprintf("video sharp %d", atomcamValue)
	return c.SendCommand(command)
}

// SetIRCutFilter sets the IR cut filter mode
// Modes: "ON" (day mode, IR filter enabled), "OFF" (night mode, IR filter disabled), "AUTO"
func (c *Client) SetIRCutFilter(mode string) error {
	switch mode {
	case "ON":
		// Day mode: IR LED off, night vision off
		if err := c.SendCommand("property IrLED off"); err != nil {
			return fmt.Errorf("failed to set IR LED: %w", err)
		}
		return c.SendCommand("property nightVision off")
	case "OFF":
		// Night mode: IR LED on, night vision on
		if err := c.SendCommand("property IrLED on"); err != nil {
			return fmt.Errorf("failed to set IR LED: %w", err)
		}
		return c.SendCommand("property nightVision on")
	case "AUTO":
		// Auto mode: let camera decide based on ambient light
		return c.SendCommand("property nightVision auto")
	default:
		return fmt.Errorf("invalid IR cut filter mode: %s (must be ON, OFF, or AUTO)", mode)
	}
}

// SetExposureMode sets the exposure mode
// Modes: "AUTO" or "MANUAL"
func (c *Client) SetExposureMode(mode string, minTime, maxTime int) error {
	switch mode {
	case "AUTO":
		command := fmt.Sprintf("video expr %d %d", minTime, maxTime)
		return c.SendCommand(command)
	case "MANUAL":
		// For manual mode, set both min and max to the same value
		command := fmt.Sprintf("video expr %d %d", maxTime, maxTime)
		return c.SendCommand(command)
	default:
		return fmt.Errorf("invalid exposure mode: %s (must be AUTO or MANUAL)", mode)
	}
}

// onvifToAtomCamValue converts ONVIF value (0.0-1.0) to AtomCam value (0-255)
func onvifToAtomCamValue(value float64) int {
	// Clamp to valid range
	if value < 0.0 {
		value = 0.0
	}
	if value > 1.0 {
		value = 1.0
	}

	// Convert to 0-255 range
	atomcamValue := int(value * 255.0)
	if atomcamValue > 255 {
		atomcamValue = 255
	}

	return atomcamValue
}

// atomCamToONVIFValue converts AtomCam value (0-255) to ONVIF value (0.0-1.0)
func atomCamToONVIFValue(value int) float64 {
	// Clamp to valid range
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}

	// Convert to 0.0-1.0 range
	return float64(value) / 255.0
}

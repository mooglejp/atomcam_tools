package ptz

import (
	"math"
)

// ONVIFToAtomCam converts ONVIF coordinates to AtomCam coordinates
// ONVIF: x, y ∈ [-1.0, 1.0], velocity ∈ [0.0, 1.0]
// AtomCam: pan ∈ [0, 355], tilt ∈ [0, 180], speed ∈ [1, 9]
func ONVIFToAtomCam(x, y, velocity float64) (pan, tilt, speed int) {
	// Clamp ONVIF values to valid range
	x = clamp(x, -1.0, 1.0)
	y = clamp(y, -1.0, 1.0)
	velocity = clamp(velocity, 0.0, 1.0)

	// Convert X to pan (0-355 degrees)
	// ONVIF X: -1.0 = left, 0.0 = center, 1.0 = right
	// AtomCam pan: 0 = left, 177.5 = center, 355 = right
	pan = int(math.Round((x + 1.0) * 177.5))

	// Convert Y to tilt (0-180 degrees)
	// ONVIF Y: 1.0 = up, 0.0 = center, -1.0 = down
	// AtomCam tilt: 0 = up, 90 = center, 180 = down
	// Note: Y axis is inverted
	tilt = int(math.Round((1.0 - y) * 90.0))

	// Convert velocity to speed (1-9)
	// ONVIF velocity: 0.0 = slowest, 1.0 = fastest
	// AtomCam speed: 1 = slowest, 9 = fastest
	// Ensure speed is at least 1 (0 would mean stop)
	speed = int(math.Round(velocity*8.0)) + 1
	if speed < 1 {
		speed = 1
	}
	if speed > 9 {
		speed = 9
	}

	return pan, tilt, speed
}

// AtomCamToONVIF converts AtomCam coordinates to ONVIF coordinates
// AtomCam: pan ∈ [0, 355], tilt ∈ [0, 180]
// ONVIF: x, y ∈ [-1.0, 1.0]
func AtomCamToONVIF(pan, tilt int) (x, y float64) {
	// Convert pan to X
	x = (float64(pan) / 177.5) - 1.0

	// Convert tilt to Y (inverted)
	y = 1.0 - (float64(tilt) / 90.0)

	// Clamp to valid range
	x = clamp(x, -1.0, 1.0)
	y = clamp(y, -1.0, 1.0)

	return x, y
}

// clamp restricts a value to a given range
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// VelocityToAtomCam converts ONVIF velocity to AtomCam movement parameters
// This is used for ContinuousMove where velocity indicates direction and speed
// ONVIF velocity: x, y ∈ [-1.0, 1.0] (negative = left/down, positive = right/up)
// Returns: pan, tilt (target position in the direction of movement), speed
func VelocityToAtomCam(velocityX, velocityY, velocityMagnitude float64) (pan, tilt, speed int) {
	// Convert velocity magnitude to speed (1-9)
	speed = int(math.Round(velocityMagnitude*8.0)) + 1
	if speed < 1 {
		speed = 1
	}
	if speed > 9 {
		speed = 9
	}

	// Determine target position based on velocity direction
	// Move towards the edge in the direction of the velocity

	// Pan calculation (X axis)
	if math.Abs(velocityX) > 0.01 {
		if velocityX > 0 {
			// Moving right: target is far right
			pan = 355
		} else {
			// Moving left: target is far left
			pan = 0
		}
	} else {
		// No X movement: stay centered
		pan = 177
	}

	// Tilt calculation (Y axis)
	if math.Abs(velocityY) > 0.01 {
		if velocityY > 0 {
			// Moving up: target is top (ONVIF Y positive = up)
			tilt = 0
		} else {
			// Moving down: target is bottom
			tilt = 180
		}
	} else {
		// No Y movement: stay centered
		tilt = 90
	}

	return pan, tilt, speed
}

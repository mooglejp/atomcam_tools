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

// CalculateAbsolutePosition calculates absolute position from relative velocity
// This is used for ContinuousMove where velocity indicates direction/speed
func CalculateAbsolutePosition(velocityX, velocityY float64, durationSeconds float64) (x, y float64) {
	// For continuous move, velocity indicates direction and speed
	// We need to calculate target position based on velocity and duration

	// Simple approach: velocity * duration gives displacement
	// Assuming full range movement takes about 5 seconds at max speed
	const fullRangeSeconds = 5.0

	x = velocityX * (durationSeconds / fullRangeSeconds) * 2.0  // *2 because range is -1 to 1
	y = velocityY * (durationSeconds / fullRangeSeconds) * 2.0

	// Clamp to valid range
	x = clamp(x, -1.0, 1.0)
	y = clamp(y, -1.0, 1.0)

	return x, y
}

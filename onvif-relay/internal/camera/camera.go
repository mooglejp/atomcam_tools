package camera

import (
	"log"
	"sync"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
)

// Camera represents a camera instance
type Camera struct {
	Config   *config.CameraConfig
	Client   *Client
	healthMu sync.RWMutex
	health   bool
	ptzMu    sync.RWMutex
	ptzPan   int // Current pan position (0-355)
	ptzTilt  int // Current tilt position (0-180)
}

// NewCamera creates a new camera instance
func NewCamera(cfg *config.CameraConfig) *Camera {
	// Initialize PTZ position to home position if available, otherwise center
	pan := 177  // Default center
	tilt := 90  // Default center
	if cfg.PTZ.Home != nil {
		pan = cfg.PTZ.Home.Pan
		tilt = cfg.PTZ.Home.Tilt
	}

	cam := &Camera{
		Config:  cfg,
		Client:  NewClient(cfg),
		health:  true,
		ptzPan:  pan,
		ptzTilt: tilt,
	}

	// Debug: Log initialization
	log.Printf("NewCamera: %s initialized with PTZ position (%d, %d)", cfg.Name, pan, tilt)

	return cam
}

// GetHealth returns the camera health status (thread-safe)
func (c *Camera) GetHealth() bool {
	c.healthMu.RLock()
	defer c.healthMu.RUnlock()
	return c.health
}

// SetHealth sets the camera health status (thread-safe)
func (c *Camera) SetHealth(healthy bool) {
	c.healthMu.Lock()
	defer c.healthMu.Unlock()
	c.health = healthy
}

// GetStreamByPath finds a stream configuration by path
func (c *Camera) GetStreamByPath(path string) *config.StreamConfig {
	for i := range c.Config.Streams {
		if c.Config.Streams[i].Path == path {
			return &c.Config.Streams[i]
		}
	}
	return nil
}

// GetStreamByProfileName finds a stream configuration by profile name
func (c *Camera) GetStreamByProfileName(profileName string) *config.StreamConfig {
	for i := range c.Config.Streams {
		if c.Config.Streams[i].ProfileName == profileName {
			return &c.Config.Streams[i]
		}
	}
	return nil
}

// GetPTZPosition returns the current PTZ position (thread-safe)
func (c *Camera) GetPTZPosition() (pan, tilt int) {
	c.ptzMu.RLock()
	defer c.ptzMu.RUnlock()
	return c.ptzPan, c.ptzTilt
}

// SetPTZPosition sets the current PTZ position (thread-safe)
func (c *Camera) SetPTZPosition(pan, tilt int) {
	c.ptzMu.Lock()
	defer c.ptzMu.Unlock()
	c.ptzPan = pan
	c.ptzTilt = tilt
}

// Close stops the camera client's background goroutines
func (c *Camera) Close() {
	c.Client.Close()
}

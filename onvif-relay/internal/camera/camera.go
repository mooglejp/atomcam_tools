package camera

import (
	"sync"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
)

// Camera represents a camera instance
type Camera struct {
	Config   *config.CameraConfig
	Client   *Client
	healthMu sync.RWMutex
	health   bool
}

// NewCamera creates a new camera instance
func NewCamera(cfg *config.CameraConfig) *Camera {
	return &Camera{
		Config: cfg,
		Client: NewClient(cfg),
		health: true,
	}
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

// Close stops the camera client's background goroutines
func (c *Camera) Close() {
	c.Client.Close()
}

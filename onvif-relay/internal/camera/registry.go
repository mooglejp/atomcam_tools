package camera

import (
	"fmt"
	"sync"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
)

// Registry represents a camera registry
type Registry struct {
	cameras map[string]*Camera
	mu      sync.RWMutex
}

// NewRegistry creates a new camera registry
func NewRegistry(cfg *config.Config) (*Registry, error) {
	r := &Registry{
		cameras: make(map[string]*Camera),
	}

	for i := range cfg.Cameras {
		cam := NewCamera(&cfg.Cameras[i])
		r.cameras[cfg.Cameras[i].Name] = cam
	}

	return r, nil
}

// Get retrieves a camera by name
func (r *Registry) Get(name string) (*Camera, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cam, ok := r.cameras[name]
	if !ok {
		return nil, fmt.Errorf("camera not found: %s", name)
	}

	return cam, nil
}

// List returns all cameras
func (r *Registry) List() []*Camera {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cameras := make([]*Camera, 0, len(r.cameras))
	for _, cam := range r.cameras {
		cameras = append(cameras, cam)
	}

	return cameras
}

// GetAllProfiles returns all stream profiles across all cameras
func (r *Registry) GetAllProfiles() []Profile {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var profiles []Profile
	for _, cam := range r.cameras {
		for i := range cam.Config.Streams {
			stream := &cam.Config.Streams[i]
			profiles = append(profiles, Profile{
				Camera: cam,
				Stream: stream,
			})
		}
	}

	return profiles
}

// Profile represents a camera stream profile
type Profile struct {
	Camera *Camera
	Stream *config.StreamConfig
}

// GetProfileByToken retrieves a profile by token
func (r *Registry) GetProfileByToken(token string) (*Profile, error) {
	profiles := r.GetAllProfiles()
	for i := range profiles {
		if profiles[i].Stream.ProfileName == token {
			return &profiles[i], nil
		}
	}

	return nil, fmt.Errorf("profile not found: %s", token)
}

// Close stops all cameras' background goroutines
func (r *Registry) Close() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, cam := range r.cameras {
		cam.Close()
	}
}

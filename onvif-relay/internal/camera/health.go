package camera

import (
	"context"
	"log"
	"sync"
	"time"
)

// HealthChecker monitors camera health
type HealthChecker struct {
	registry *Registry
	interval time.Duration
	failures map[string]int // Track consecutive failures per camera
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(registry *Registry, interval time.Duration) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		registry: registry,
		interval: interval,
		failures: make(map[string]int),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the health checker
func (h *HealthChecker) Start() {
	go h.run()
}

// Stop stops the health checker
func (h *HealthChecker) Stop() {
	h.cancel()
}

// run performs periodic health checks
func (h *HealthChecker) run() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	log.Printf("Health checker started (interval: %v)", h.interval)

	for {
		select {
		case <-h.ctx.Done():
			log.Printf("Health checker stopped")
			return
		case <-ticker.C:
			h.checkAllCameras()
		}
	}
}

// checkAllCameras checks all cameras in the registry
func (h *HealthChecker) checkAllCameras() {
	cameras := h.registry.List()

	for _, cam := range cameras {
		h.checkCamera(cam)
	}
}

// checkCamera checks a single camera
func (h *HealthChecker) checkCamera(cam *Camera) {
	err := cam.Client.Ping()

	h.mu.Lock()
	defer h.mu.Unlock()

	if err != nil {
		// Increment failure count
		h.failures[cam.Config.Name]++
		consecutiveFailures := h.failures[cam.Config.Name]

		log.Printf("Camera %s health check failed (%d consecutive failures): %v",
			cam.Config.Name, consecutiveFailures, err)

		// Mark as unhealthy after 3 consecutive failures
		if consecutiveFailures >= 3 && cam.GetHealth() {
			cam.SetHealth(false)
			log.Printf("Camera %s marked as UNHEALTHY", cam.Config.Name)
		}
	} else {
		// Reset failure count on success
		if h.failures[cam.Config.Name] > 0 {
			log.Printf("Camera %s health check succeeded (recovered)", cam.Config.Name)
		}
		h.failures[cam.Config.Name] = 0

		// Mark as healthy
		if !cam.GetHealth() {
			cam.SetHealth(true)
			log.Printf("Camera %s marked as HEALTHY", cam.Config.Name)
		}
	}
}

// GetCameraHealth returns the health status of a camera
func (h *HealthChecker) GetCameraHealth(cameraName string) bool {
	cam, err := h.registry.Get(cameraName)
	if err != nil {
		return false
	}
	return cam.GetHealth()
}

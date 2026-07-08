package snapshot

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/camera"
)

// Proxy represents a snapshot proxy server
type Proxy struct {
	registry *camera.Registry
	username string
	password string
}

// NewProxy creates a new snapshot proxy
func NewProxy(registry *camera.Registry, username, password string) *Proxy {
	return &Proxy{
		registry: registry,
		username: username,
		password: password,
	}
}

// Handler returns an HTTP handler for snapshot requests
func (p *Proxy) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// HTTP Basic authentication
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="ONVIF Relay Snapshot"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Use constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(p.username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(p.password)) == 1
		if !usernameMatch || !passwordMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="ONVIF Relay Snapshot"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract camera name from path: /snapshot/{camera}
		path := strings.TrimPrefix(r.URL.Path, "/snapshot/")
		cameraName := strings.TrimSuffix(path, "/")

		if cameraName == "" {
			http.Error(w, "camera name required", http.StatusBadRequest)
			return
		}

		// Validate camera name (prevent path traversal)
		if strings.Contains(cameraName, "/") || strings.Contains(cameraName, "..") || strings.Contains(cameraName, "\\") {
			log.Printf("Invalid camera name attempted: %s", cameraName)
			http.Error(w, "invalid camera name", http.StatusBadRequest)
			return
		}

		// Get camera
		cam, err := p.registry.Get(cameraName)
		if err != nil {
			log.Printf("Camera not found: %s", cameraName)
			http.Error(w, "camera not found", http.StatusNotFound)
			return
		}

		// Check camera health before attempting snapshot
		if !cam.GetHealth() {
			log.Printf("Camera %s is unhealthy, refusing snapshot request", cameraName)
			http.Error(w, "camera unavailable", http.StatusServiceUnavailable)
			return
		}

		// Get snapshot
		data, err := cam.Client.GetSnapshot()
		if err != nil {
			log.Printf("Failed to get snapshot from %s: %v", cameraName, err)
			http.Error(w, "failed to get snapshot", http.StatusInternalServerError)
			return
		}

		// Return JPEG image
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

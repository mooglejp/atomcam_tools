package talk

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/camera"
)

// Proxy accepts raw PCM over HTTP and forwards it to camera atomtalkd.
type Proxy struct {
	registry *camera.Registry
	username string
	password string
}

// NewProxy creates a talk proxy.
func NewProxy(registry *camera.Registry, username, password string) *Proxy {
	return &Proxy{
		registry: registry,
		username: username,
		password: password,
	}
}

// Handler returns an HTTP handler for POST /talk/{camera}.
func (p *Proxy) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		if !p.authorized(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="ONVIF Relay Talk"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		cameraName := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/talk/"), "/")
		if cameraName == "" {
			http.Error(w, "camera name required", http.StatusBadRequest)
			return
		}
		if strings.Contains(cameraName, "/") || strings.Contains(cameraName, "..") || strings.Contains(cameraName, "\\") {
			log.Printf("Invalid talk camera name attempted: %s", cameraName)
			http.Error(w, "invalid camera name", http.StatusBadRequest)
			return
		}

		cam, err := p.registry.Get(cameraName)
		if err != nil {
			log.Printf("Talk camera not found: %s", cameraName)
			http.Error(w, "camera not found", http.StatusNotFound)
			return
		}
		if !cam.Config.Talk.Enabled {
			http.Error(w, "talk disabled", http.StatusNotFound)
			return
		}
		if !cam.GetHealth() {
			log.Printf("Camera %s is unhealthy, refusing talk request", cameraName)
			http.Error(w, "camera unavailable", http.StatusServiceUnavailable)
			return
		}

		client := NewClient(cam.Config.Host, cam.Config.Talk.Port, cam.Config.Talk.Token)
		if err := client.Stream(r.Context(), r.Body); err != nil {
			log.Printf("Talk stream failed for %s: %v", cameraName, err)
			http.Error(w, "talk stream failed", http.StatusBadGateway)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (p *Proxy) authorized(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(p.username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(p.password)) == 1
	return usernameMatch && passwordMatch
}

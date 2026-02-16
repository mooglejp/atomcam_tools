package mediamtx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
)

// Client represents a mediamtx REST API client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// PathConfig represents mediamtx path configuration
type PathConfig struct {
	RunOnDemand           string `json:"runOnDemand,omitempty"`
	RunOnDemandRestart    bool   `json:"runOnDemandRestart,omitempty"`
	RunOnDemandCloseAfter string `json:"runOnDemandCloseAfter,omitempty"`
	Source                string `json:"source,omitempty"`
	SourceOnDemand        bool   `json:"sourceOnDemand,omitempty"`
	RTSPTransport         string `json:"rtspTransport,omitempty"`
}

// ErrorResponse represents mediamtx API error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// NewClient creates a new mediamtx API client
func NewClient(apiURL string) *Client {
	return &Client{
		baseURL: apiURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WaitReady waits for mediamtx API to become ready
func (c *Client) WaitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			resp, err := c.httpClient.Get(c.baseURL + "/v3/config/global/get")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("mediamtx API not ready after %v: %w", timeout, err)
			}
		}
	}
}

// ConfigurePath configures a stream path with idempotent create-or-replace logic
func (c *Client) ConfigurePath(name string, cfg PathConfig) error {
	// Try to create the path
	err := c.addPath(name, cfg)
	if err == nil {
		return nil
	}

	// If path already exists, delete and recreate
	if isPathExistsError(err) {
		if err := c.deletePath(name); err != nil {
			return fmt.Errorf("failed to delete existing path %s: %w", name, err)
		}

		// Retry add
		if err := c.addPath(name, cfg); err != nil {
			return fmt.Errorf("failed to recreate path %s: %w", name, err)
		}

		return nil
	}

	return err
}

// addPath adds a new path
func (c *Client) addPath(name string, cfg PathConfig) error {
	url := fmt.Sprintf("%s/v3/config/paths/add/%s", c.baseURL, name)

	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal path config: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add path: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error != "" {
			return &apiError{statusCode: resp.StatusCode, message: errResp.Error}
		}
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// deletePath deletes a path
func (c *Client) deletePath(name string) error {
	url := fmt.Sprintf("%s/v3/config/paths/delete/%s", c.baseURL, name)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete path: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// apiError represents a mediamtx API error
type apiError struct {
	statusCode int
	message    string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("mediamtx API error (status %d): %s", e.statusCode, e.message)
}

// isPathExistsError checks if the error indicates that the path already exists
func isPathExistsError(err error) bool {
	apiErr, ok := err.(*apiError)
	if !ok {
		return false
	}
	return apiErr.statusCode == http.StatusBadRequest && apiErr.message == "path already exists"
}

// BuildFFmpegCommand builds the ffmpeg command for a stream
func BuildFFmpegCommand(camera *config.CameraConfig, stream *config.StreamConfig, mtxConfig *config.MediamtxConfig) string {
	// Source RTSP URL (without embedded credentials)
	sourceURL := fmt.Sprintf("rtsp://%s:%d/%s", camera.Host, camera.RTSPPort, stream.Path)

	// Base ffmpeg options with credentials via -user/-password (not embedded in URL)
	cmd := "ffmpeg -fflags +genpts -rtsp_transport tcp"
	if camera.Username != "" && camera.Password != "" {
		// Use -rtsp_user and -rtsp_password options instead of embedding in URL
		cmd += fmt.Sprintf(" -rtsp_user %s -rtsp_password %s", camera.Username, camera.Password)
	}
	cmd += fmt.Sprintf(" -i %s -map 0:v:0 -map 0:a:0? -c:v copy", sourceURL)

	// Audio transcoding settings
	audioCodec := camera.AudioTranscode
	if audioCodec == "" {
		audioCodec = "pcm_mulaw" // default
	}

	audioVolume := camera.AudioVolume
	if audioVolume == 0 {
		audioVolume = 1.0 // default
	}

	// H.265/HEVC uses SRT publish (MPEG-TS container)
	// MPEG-TS doesn't support pcm_mulaw/pcm_alaw, so convert to AAC
	isHEVC := stream.Codec == "h265" || stream.Codec == "hevc"
	if isHEVC {
		if audioCodec == "pcm_mulaw" || audioCodec == "pcm_alaw" {
			audioCodec = "aac"
		}
		// AAC requires higher sample rate
		cmd += fmt.Sprintf(" -c:a %s -ar 48000 -ac 1 -async 1 -af volume=%.1f", audioCodec, audioVolume)
		cmd += fmt.Sprintf(" -f mpegts \"srt://localhost:8890?streamid=publish:$MTX_PATH&pkt_size=1316\"")
	} else {
		// H.264 uses RTSP publish
		sampleRate := "8000"
		if audioCodec == "aac" {
			sampleRate = "48000"
		}
		cmd += fmt.Sprintf(" -c:a %s -ar %s -ac 1 -async 1 -af volume=%.1f", audioCodec, sampleRate, audioVolume)
		cmd += fmt.Sprintf(" -rtsp_transport tcp -f rtsp rtsp://localhost:%d/$MTX_PATH", mtxConfig.RTSPPort)
	}

	return cmd
}

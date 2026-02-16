package camera

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
	"github.com/mooglejp/atomcam_tools/onvif-relay/pkg/digest"
)

// Client represents an HTTP client for AtomCam cmd.cgi interface
type Client struct {
	cfg             *config.CameraConfig
	httpClient      *http.Client
	digestTransport *digest.Transport // for cleanup
}

// CommandRequest represents a cmd.cgi command request
type CommandRequest struct {
	Exec string `json:"exec"`
}

// NewClient creates a new AtomCam HTTP client
func NewClient(cfg *config.CameraConfig) *Client {
	// Create HTTP client with Digest auth if credentials are provided
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	var digestTransport *digest.Transport
	if cfg.Username != "" && cfg.Password != "" {
		digestTransport = digest.NewTransport(cfg.Username, cfg.Password)
		httpClient.Transport = digestTransport
	}

	return &Client{
		cfg:             cfg,
		httpClient:      httpClient,
		digestTransport: digestTransport,
	}
}

// Close stops the digest transport cleanup goroutine
func (c *Client) Close() {
	if c.digestTransport != nil {
		c.digestTransport.Close()
	}
}

// SendCommand sends a command to the camera via cmd.cgi
func (c *Client) SendCommand(command string) error {
	// Whitelist validation: only allow specific command prefixes
	allowedPrefixes := []string{"move ", "video ", "property ", "alarm "}
	allowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(command, prefix) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("command not allowed (must start with move/video/property/alarm)")
	}

	url := fmt.Sprintf("http://%s:%d/cgi-bin/cmd.cgi?port=socket", c.cfg.Host, c.cfg.HTTPPort)

	req := CommandRequest{
		Exec: command,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Digest auth is handled by the http.Client.Transport
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Ping checks if the camera is reachable
func (c *Client) Ping() error {
	url := fmt.Sprintf("http://%s:%d/", c.cfg.Host, c.cfg.HTTPPort)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

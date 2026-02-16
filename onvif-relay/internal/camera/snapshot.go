package camera

import (
	"fmt"
	"io"
	"net/http"
)

const (
	// maxSnapshotSize limits snapshot size to prevent memory exhaustion
	maxSnapshotSize = 10 * 1024 * 1024 // 10MB
)

// GetSnapshot retrieves a JPEG snapshot from the camera
func (c *Client) GetSnapshot() ([]byte, error) {
	url := fmt.Sprintf("http://%s:%d/cgi-bin/get_jpeg.cgi", c.cfg.Host, c.cfg.HTTPPort)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read JPEG data with size limit
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSnapshotSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot data: %w", err)
	}

	return data, nil
}

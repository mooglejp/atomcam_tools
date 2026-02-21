package config

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// validNamePattern restricts names to alphanumeric, hyphen, and underscore
	validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	// validHostPattern allows hostname/IP (alphanumeric, dots, hyphens)
	validHostPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	// validCredentialPattern disallows shell metacharacters in credentials
	validCredentialPattern = regexp.MustCompile(`^[a-zA-Z0-9@._-]+$`)
)

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	if len(c.Cameras) == 0 {
		return fmt.Errorf("at least one camera must be configured")
	}

	cameraNames := make(map[string]bool)
	for i, cam := range c.Cameras {
		if err := cam.Validate(); err != nil {
			return fmt.Errorf("camera[%d] (%s): %w", i, cam.Name, err)
		}

		// Check for duplicate camera names
		if cameraNames[cam.Name] {
			return fmt.Errorf("duplicate camera name: %s", cam.Name)
		}
		cameraNames[cam.Name] = true
	}

	return nil
}

// reservedPaths are paths used internally by the ONVIF server
var reservedPaths = []string{"/onvif/", "/snapshot/"}

// Validate validates server configuration
func (s *ServerConfig) Validate() error {
	if s.OnvifPort <= 0 || s.OnvifPort > 65535 {
		return fmt.Errorf("invalid onvif_port: %d (must be 1-65535)", s.OnvifPort)
	}

	if s.DeviceName == "" {
		return fmt.Errorf("device_name is required")
	}

	if s.Auth.Username == "" {
		return fmt.Errorf("auth.username is required")
	}

	// Validate auth username format (prevent shell injection, consistent with camera validation)
	if !validCredentialPattern.MatchString(s.Auth.Username) {
		return fmt.Errorf("invalid auth.username: contains shell metacharacters")
	}

	if s.Auth.Password == "" {
		return fmt.Errorf("auth.password is required")
	}

	// Validate auth password format (prevent shell injection, consistent with camera validation)
	if !validCredentialPattern.MatchString(s.Auth.Password) {
		return fmt.Errorf("invalid auth.password: contains shell metacharacters")
	}

	if err := s.Mediamtx.Validate(); err != nil {
		return fmt.Errorf("mediamtx: %w", err)
	}

	proxyPaths := make(map[string]bool)
	for i, p := range s.Proxies {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("proxies[%d]: %w", i, err)
		}
		if proxyPaths[p.Path] {
			return fmt.Errorf("proxies[%d]: duplicate path: %s", i, p.Path)
		}
		proxyPaths[p.Path] = true
	}

	return nil
}

// Validate validates a single proxy rule
func (p *ProxyConfig) Validate() error {
	if p.Path == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(p.Path, "/") {
		return fmt.Errorf("path must start with /: %s", p.Path)
	}
	for _, reserved := range reservedPaths {
		if strings.HasPrefix(p.Path, reserved) {
			return fmt.Errorf("path conflicts with reserved ONVIF path %s", reserved)
		}
	}
	if p.Target == "" {
		return fmt.Errorf("target is required")
	}
	if !strings.HasPrefix(p.Target, "http://") && !strings.HasPrefix(p.Target, "https://") {
		return fmt.Errorf("target must start with http:// or https://: %s", p.Target)
	}
	return nil
}

// Validate validates mediamtx configuration
func (m *MediamtxConfig) Validate() error {
	// Empty API means mediamtx is disabled; skip all mediamtx validation
	if m.API == "" {
		return nil
	}

	if !strings.HasPrefix(m.API, "http://") && !strings.HasPrefix(m.API, "https://") {
		return fmt.Errorf("api must start with http:// or https://")
	}

	if m.RTSPPort <= 0 || m.RTSPPort > 65535 {
		return fmt.Errorf("invalid rtsp_port: %d (must be 1-65535)", m.RTSPPort)
	}

	return nil
}

// Validate validates camera configuration
func (c *CameraConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate camera name format (prevent path traversal)
	if !validNamePattern.MatchString(c.Name) {
		return fmt.Errorf("invalid camera name: %s (only alphanumeric, hyphen, and underscore allowed)", c.Name)
	}

	if c.Host == "" {
		return fmt.Errorf("host is required")
	}

	// Validate host format (prevent shell injection via ffmpeg command)
	if !validHostPattern.MatchString(c.Host) {
		return fmt.Errorf("invalid host: %s (contains shell metacharacters)", c.Host)
	}

	// Validate username/password format if provided (prevent shell injection)
	if c.Username != "" && !validCredentialPattern.MatchString(c.Username) {
		return fmt.Errorf("invalid username: contains shell metacharacters")
	}
	if c.Password != "" && !validCredentialPattern.MatchString(c.Password) {
		return fmt.Errorf("invalid password: contains shell metacharacters")
	}

	if c.RTSPPort <= 0 || c.RTSPPort > 65535 {
		return fmt.Errorf("invalid rtsp_port: %d (must be 1-65535)", c.RTSPPort)
	}

	if c.HTTPPort <= 0 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid http_port: %d (must be 1-65535)", c.HTTPPort)
	}

	if len(c.Streams) == 0 {
		return fmt.Errorf("at least one stream must be configured")
	}

	streamPaths := make(map[string]bool)
	profileNames := make(map[string]bool)
	for i, stream := range c.Streams {
		if err := stream.Validate(); err != nil {
			return fmt.Errorf("stream[%d]: %w", i, err)
		}

		// Check for duplicate stream paths
		if streamPaths[stream.Path] {
			return fmt.Errorf("duplicate stream path: %s", stream.Path)
		}
		streamPaths[stream.Path] = true

		// Check for duplicate profile names
		if profileNames[stream.ProfileName] {
			return fmt.Errorf("duplicate profile name: %s", stream.ProfileName)
		}
		profileNames[stream.ProfileName] = true
	}

	// Validate audio settings
	if c.AudioTranscode != "" {
		validCodecs := map[string]bool{
			"pcm_mulaw": true,
			"pcm_alaw":  true,
			"aac":       true,
		}
		if !validCodecs[c.AudioTranscode] {
			return fmt.Errorf("invalid audio_transcode: %s (must be pcm_mulaw, pcm_alaw, or aac)", c.AudioTranscode)
		}
	}

	if c.AudioVolume < 0 {
		return fmt.Errorf("audio_volume must be >= 0")
	}

	return nil
}

// Validate validates stream configuration
func (s *StreamConfig) Validate() error {
	if s.Path == "" {
		return fmt.Errorf("path is required")
	}

	// Validate stream path format (prevent path traversal)
	if !validNamePattern.MatchString(s.Path) {
		return fmt.Errorf("invalid stream path: %s (only alphanumeric, hyphen, and underscore allowed)", s.Path)
	}

	if s.ProfileName == "" {
		return fmt.Errorf("profile_name is required")
	}

	// Validate profile name format
	if !validNamePattern.MatchString(s.ProfileName) {
		return fmt.Errorf("invalid profile name: %s (only alphanumeric, hyphen, and underscore allowed)", s.ProfileName)
	}

	// Validate codec
	validCodecs := map[string]bool{
		"h264": true,
		"h265": true,
		"hevc": true, // alias for h265
	}
	if !validCodecs[strings.ToLower(s.Codec)] {
		return fmt.Errorf("invalid codec: %s (must be h264 or h265)", s.Codec)
	}

	return nil
}

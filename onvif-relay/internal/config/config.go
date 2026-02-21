package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration
type Config struct {
	Server  ServerConfig   `yaml:"server"`
	Cameras []CameraConfig `yaml:"cameras"`
}

// ServerConfig represents ONVIF relay server configuration
type ServerConfig struct {
	OnvifPort  int            `yaml:"onvif_port"`
	DeviceName string         `yaml:"device_name"`
	Discovery  bool           `yaml:"discovery"`
	Auth       AuthConfig     `yaml:"auth"`
	Mediamtx   MediamtxConfig `yaml:"mediamtx"`
	Proxies    []ProxyConfig  `yaml:"proxies,omitempty"`
}

// ProxyConfig represents a single reverse proxy rule
type ProxyConfig struct {
	Path        string `yaml:"path"`         // URL path prefix handled by this proxy (e.g. "/api/")
	Target      string `yaml:"target"`       // Backend base URL (e.g. "http://192.168.1.100:9000")
	StripPrefix bool   `yaml:"strip_prefix"` // Strip path prefix before forwarding (default: false)
}

// AuthConfig represents authentication credentials
type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// MediamtxConfig represents mediamtx integration settings
type MediamtxConfig struct {
	API      string `yaml:"api"`
	RTSPHost string `yaml:"rtsp_host,omitempty"`
	RTSPPort int    `yaml:"rtsp_port"`
}

// CameraConfig represents a single camera configuration
type CameraConfig struct {
	Name            string            `yaml:"name"`
	Host            string            `yaml:"host"`
	RTSPPort        int               `yaml:"rtsp_port"`
	HTTPPort        int               `yaml:"http_port"`
	Username        string            `yaml:"username,omitempty"`
	Password        string            `yaml:"password,omitempty"`
	AudioTranscode  string            `yaml:"audio_transcode,omitempty"`
	AudioVolume     float64           `yaml:"audio_volume,omitempty"`
	Capabilities    CapabilitiesConfig `yaml:"capabilities"`
	Streams         []StreamConfig    `yaml:"streams"`
	PTZ             PTZConfig         `yaml:"ptz,omitempty"`
}

// PTZConfig represents PTZ-specific configuration
type PTZConfig struct {
	Home         *PTZPreset `yaml:"home,omitempty"`          // Home position
	Presets      []PTZPreset `yaml:"presets,omitempty"`      // Presets 1-9
	HorizontalFOV float64    `yaml:"horizontal_fov,omitempty"` // Horizontal field of view in degrees (e.g., 120.0)
	VerticalFOV   float64    `yaml:"vertical_fov,omitempty"`   // Vertical field of view in degrees (e.g., 67.5)
}

// CapabilitiesConfig represents camera capabilities
type CapabilitiesConfig struct {
	PTZ bool `yaml:"ptz"`
	IR  bool `yaml:"ir"`
}

// PTZPreset represents a PTZ preset position or MQTT action
type PTZPreset struct {
	Name        string `yaml:"name"`
	Pan         int    `yaml:"pan,omitempty"`   // 0-355 degrees (omit if using MQTT)
	Tilt        int    `yaml:"tilt,omitempty"`  // 0-180 degrees (omit if using MQTT)
	Token       string `yaml:"token,omitempty"` // Optional preset token (e.g., "1", "2", etc.)
	MQTTBroker  string `yaml:"mqtt_broker,omitempty"`  // MQTT broker URL (e.g., "tcp://localhost:1883")
	MQTTTopic   string `yaml:"mqtt_topic,omitempty"`   // MQTT topic (e.g., "home/light/livingroom")
	MQTTMessage string `yaml:"mqtt_message,omitempty"` // MQTT message payload (e.g., "ON")
}

// StreamConfig represents a single stream configuration
type StreamConfig struct {
	Path        string `yaml:"path"`
	Resolution  string `yaml:"resolution"`
	Codec       string `yaml:"codec"`
	ProfileName string `yaml:"profile_name"`
	RTSPURL     string `yaml:"rtsp_url,omitempty"` // Optional: override RTSP URL (if not set, use mediamtx)
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

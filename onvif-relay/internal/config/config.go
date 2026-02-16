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
}

// CapabilitiesConfig represents camera capabilities
type CapabilitiesConfig struct {
	PTZ bool `yaml:"ptz"`
	IR  bool `yaml:"ir"`
}

// StreamConfig represents a single stream configuration
type StreamConfig struct {
	Path        string `yaml:"path"`
	Resolution  string `yaml:"resolution"`
	Codec       string `yaml:"codec"`
	ProfileName string `yaml:"profile_name"`
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

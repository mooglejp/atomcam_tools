package media

import (
	"encoding/xml"
	"fmt"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/camera"
)

// GetProfilesRequest represents GetProfiles request
type GetProfilesRequest struct {
	XMLName xml.Name `xml:"GetProfiles"`
}

// GetProfilesResponse represents GetProfiles response
type GetProfilesResponse struct {
	XMLName  xml.Name  `xml:"trt:GetProfilesResponse"`
	Profiles []Profile `xml:"Profiles"`
}

// Profile represents a media profile
type Profile struct {
	Token              string              `xml:"token,attr"`
	Fixed              bool                `xml:"fixed,attr"`
	Name               string              `xml:"Name"`
	VideoSourceConfiguration *VideoSourceConfiguration `xml:"VideoSourceConfiguration,omitempty"`
	VideoEncoderConfiguration *VideoEncoderConfiguration `xml:"VideoEncoderConfiguration,omitempty"`
	PTZConfiguration   *PTZConfiguration   `xml:"PTZConfiguration,omitempty"`
}

// VideoSourceConfiguration represents video source configuration
type VideoSourceConfiguration struct {
	Token       string `xml:"token,attr"`
	Name        string `xml:"Name"`
	SourceToken string `xml:"SourceToken"`
	Bounds      Bounds `xml:"Bounds"`
}

// Bounds represents bounds
type Bounds struct {
	X      int `xml:"x,attr"`
	Y      int `xml:"y,attr"`
	Width  int `xml:"width,attr"`
	Height int `xml:"height,attr"`
}

// VideoEncoderConfiguration represents video encoder configuration
type VideoEncoderConfiguration struct {
	Token      string     `xml:"token,attr"`
	Name       string     `xml:"Name"`
	Encoding   string     `xml:"Encoding"`
	Resolution Resolution `xml:"Resolution"`
	Quality    float64    `xml:"Quality"`
	RateControl RateControl `xml:"RateControl,omitempty"`
}

// Resolution represents resolution
type Resolution struct {
	Width  int `xml:"Width"`
	Height int `xml:"Height"`
}

// RateControl represents rate control
type RateControl struct {
	FrameRateLimit      int `xml:"FrameRateLimit"`
	EncodingInterval    int `xml:"EncodingInterval"`
	BitrateLimit        int `xml:"BitrateLimit"`
}

// PTZConfiguration represents PTZ configuration
type PTZConfiguration struct {
	Token    string `xml:"token,attr"`
	Name     string `xml:"Name"`
	NodeToken string `xml:"NodeToken"`
}

// GetStreamUriRequest represents GetStreamUri request
type GetStreamUriRequest struct {
	XMLName       xml.Name      `xml:"GetStreamUri"`
	StreamSetup   StreamSetup   `xml:"StreamSetup"`
	ProfileToken  string        `xml:"ProfileToken"`
}

// StreamSetup represents stream setup
type StreamSetup struct {
	Stream    string    `xml:"Stream"`
	Transport Transport `xml:"Transport"`
}

// Transport represents transport
type Transport struct {
	Protocol string `xml:"Protocol"`
}

// GetStreamUriResponse represents GetStreamUri response
type GetStreamUriResponse struct {
	XMLName   xml.Name  `xml:"trt:GetStreamUriResponse"`
	MediaUri  MediaUri  `xml:"MediaUri"`
}

// MediaUri represents media URI
type MediaUri struct {
	Uri               string `xml:"Uri"`
	InvalidAfterConnect bool `xml:"InvalidAfterConnect"`
	InvalidAfterReboot  bool `xml:"InvalidAfterReboot"`
	Timeout           string `xml:"Timeout"`
}

// GetSnapshotUriRequest represents GetSnapshotUri request
type GetSnapshotUriRequest struct {
	XMLName      xml.Name `xml:"GetSnapshotUri"`
	ProfileToken string   `xml:"ProfileToken"`
}

// GetSnapshotUriResponse represents GetSnapshotUri response
type GetSnapshotUriResponse struct {
	XMLName   xml.Name  `xml:"trt:GetSnapshotUriResponse"`
	MediaUri  MediaUri  `xml:"MediaUri"`
}

// Service represents the Media service
type Service struct {
	registry      *camera.Registry
	mediamtxHost  string
	mediamtxPort  int
	snapshotHost  string
	snapshotPort  int
}

// NewService creates a new Media service
func NewService(registry *camera.Registry, mediamtxHost string, mediamtxPort int, snapshotHost string, snapshotPort int) *Service {
	return &Service{
		registry:     registry,
		mediamtxHost: mediamtxHost,
		mediamtxPort: mediamtxPort,
		snapshotHost: snapshotHost,
		snapshotPort: snapshotPort,
	}
}

// GetProfiles handles GetProfiles request
func (s *Service) GetProfiles() *GetProfilesResponse {
	profiles := s.registry.GetAllProfiles()
	resp := &GetProfilesResponse{
		Profiles: make([]Profile, 0, len(profiles)),
	}

	for _, p := range profiles {
		profile := s.buildProfile(&p)
		resp.Profiles = append(resp.Profiles, *profile)
	}

	return resp
}

// GetStreamUri handles GetStreamUri request
func (s *Service) GetStreamUri(profileToken string) (*GetStreamUriResponse, error) {
	profile, err := s.registry.GetProfileByToken(profileToken)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %s", profileToken)
	}

	// Build RTSP URL: rtsp://{mediamtx_host}:{port}/{camera}/{stream}
	rtspPath := fmt.Sprintf("%s/%s", profile.Camera.Config.Name, profile.Stream.Path)
	rtspURL := fmt.Sprintf("rtsp://%s:%d/%s", s.mediamtxHost, s.mediamtxPort, rtspPath)

	return &GetStreamUriResponse{
		MediaUri: MediaUri{
			Uri:                 rtspURL,
			InvalidAfterConnect: false,
			InvalidAfterReboot:  false,
			Timeout:             "PT1H",
		},
	}, nil
}

// buildProfile builds a Profile from camera.Profile
func (s *Service) buildProfile(p *camera.Profile) *Profile {
	width, height := parseResolution(p.Stream.Resolution)
	encoding := "H264"
	if p.Stream.Codec == "h265" || p.Stream.Codec == "hevc" {
		encoding = "H265"
	}

	profile := &Profile{
		Token: p.Stream.ProfileName,
		Fixed: true,
		Name:  p.Stream.ProfileName,
		VideoSourceConfiguration: &VideoSourceConfiguration{
			Token:       p.Stream.ProfileName + "_VSC",
			Name:        p.Stream.ProfileName + " Video Source",
			SourceToken: "VideoSource_1",
			Bounds: Bounds{
				X:      0,
				Y:      0,
				Width:  width,
				Height: height,
			},
		},
		VideoEncoderConfiguration: &VideoEncoderConfiguration{
			Token:    p.Stream.ProfileName + "_VEC",
			Name:     p.Stream.ProfileName + " Video Encoder",
			Encoding: encoding,
			Resolution: Resolution{
				Width:  width,
				Height: height,
			},
			Quality: 4.0,
			RateControl: RateControl{
				FrameRateLimit:   30,
				EncodingInterval: 1,
				BitrateLimit:     4096,
			},
		},
	}

	// Add PTZ configuration if supported
	if p.Camera.Config.Capabilities.PTZ {
		profile.PTZConfiguration = &PTZConfiguration{
			Token:     p.Stream.ProfileName + "_PTZ",
			Name:      p.Stream.ProfileName + " PTZ",
			NodeToken: "PTZNode_1",
		}
	}

	return profile
}

// GetSnapshotUri handles GetSnapshotUri request
func (s *Service) GetSnapshotUri(profileToken string) (*GetSnapshotUriResponse, error) {
	profile, err := s.registry.GetProfileByToken(profileToken)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %s", profileToken)
	}

	// Build snapshot URL: http://{relay_host}:{port}/snapshot/{camera}
	snapshotURL := fmt.Sprintf("http://%s:%d/snapshot/%s", s.snapshotHost, s.snapshotPort, profile.Camera.Config.Name)

	return &GetSnapshotUriResponse{
		MediaUri: MediaUri{
			Uri:                 snapshotURL,
			InvalidAfterConnect: false,
			InvalidAfterReboot:  false,
			Timeout:             "PT1H",
		},
	}, nil
}

// parseResolution parses resolution string (e.g., "1920x1080") to width and height
func parseResolution(res string) (int, int) {
	var width, height int
	fmt.Sscanf(res, "%dx%d", &width, &height)
	if width == 0 || height == 0 {
		// Default to 1080p if parsing fails
		return 1920, 1080
	}
	return width, height
}

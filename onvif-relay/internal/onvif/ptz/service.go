package ptz

import (
	"encoding/xml"
	"fmt"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/camera"
)

// GetNodesRequest represents GetNodes request
type GetNodesRequest struct {
	XMLName xml.Name `xml:"GetNodes"`
}

// GetNodesResponse represents GetNodes response
type GetNodesResponse struct {
	XMLName xml.Name `xml:"tptz:GetNodesResponse"`
	PTZNode []PTZNode `xml:"PTZNode"`
}

// PTZNode represents a PTZ node
type PTZNode struct {
	Token            string           `xml:"token,attr"`
	Name             string           `xml:"Name"`
	SupportedPTZSpaces PTZSpaces      `xml:"SupportedPTZSpaces"`
	MaximumNumberOfPresets int        `xml:"MaximumNumberOfPresets"`
	HomeSupported    bool             `xml:"HomeSupported"`
	FixedHomePosition bool            `xml:"FixedHomePosition,omitempty"`
}

// PTZSpaces represents PTZ coordinate spaces
type PTZSpaces struct {
	AbsolutePanTiltPositionSpace   []Space2D `xml:"AbsolutePanTiltPositionSpace,omitempty"`
	AbsoluteZoomPositionSpace      []Space1D `xml:"AbsoluteZoomPositionSpace,omitempty"`
	RelativePanTiltTranslationSpace []Space2D `xml:"RelativePanTiltTranslationSpace,omitempty"`
	RelativeZoomTranslationSpace   []Space1D `xml:"RelativeZoomTranslationSpace,omitempty"`
	ContinuousPanTiltVelocitySpace []Space2D `xml:"ContinuousPanTiltVelocitySpace,omitempty"`
	ContinuousZoomVelocitySpace    []Space1D `xml:"ContinuousZoomVelocitySpace,omitempty"`
	PanTiltSpeedSpace              []Space1D `xml:"PanTiltSpeedSpace,omitempty"`
	ZoomSpeedSpace                 []Space1D `xml:"ZoomSpeedSpace,omitempty"`
}

// Space2D represents a 2D coordinate space
type Space2D struct {
	URI    string  `xml:"URI"`
	XRange Range   `xml:"XRange"`
	YRange Range   `xml:"YRange"`
}

// Space1D represents a 1D coordinate space
type Space1D struct {
	URI   string `xml:"URI"`
	XRange Range  `xml:"XRange"`
}

// Range represents a value range
type Range struct {
	Min float64 `xml:"Min"`
	Max float64 `xml:"Max"`
}

// GetConfigurationsRequest represents GetConfigurations request
type GetConfigurationsRequest struct {
	XMLName xml.Name `xml:"GetConfigurations"`
}

// GetConfigurationsResponse represents GetConfigurations response
type GetConfigurationsResponse struct {
	XMLName        xml.Name        `xml:"tptz:GetConfigurationsResponse"`
	PTZConfiguration []PTZConfiguration `xml:"PTZConfiguration"`
}

// PTZConfiguration represents PTZ configuration
type PTZConfiguration struct {
	Token      string     `xml:"token,attr"`
	Name       string     `xml:"Name"`
	NodeToken  string     `xml:"NodeToken"`
	DefaultPTZSpeed *PTZSpeed `xml:"DefaultPTZSpeed,omitempty"`
	DefaultPTZTimeout string  `xml:"DefaultPTZTimeout,omitempty"`
	PanTiltLimits  *PanTiltLimits `xml:"PanTiltLimits,omitempty"`
	ZoomLimits     *ZoomLimits    `xml:"ZoomLimits,omitempty"`
}

// PTZSpeed represents PTZ speed
type PTZSpeed struct {
	PanTilt *Vector2D `xml:"PanTilt,omitempty"`
	Zoom    *Vector1D `xml:"Zoom,omitempty"`
}

// Vector2D represents a 2D vector
type Vector2D struct {
	Space string  `xml:"space,attr,omitempty"`
	X     float64 `xml:"x,attr"`
	Y     float64 `xml:"y,attr"`
}

// Vector1D represents a 1D vector
type Vector1D struct {
	Space string  `xml:"space,attr,omitempty"`
	X     float64 `xml:"x,attr"`
}

// PanTiltLimits represents pan/tilt limits
type PanTiltLimits struct {
	Range Space2D `xml:"Range"`
}

// ZoomLimits represents zoom limits
type ZoomLimits struct {
	Range Space1D `xml:"Range"`
}

// ContinuousMoveRequest represents ContinuousMove request
type ContinuousMoveRequest struct {
	XMLName      xml.Name `xml:"ContinuousMove"`
	ProfileToken string   `xml:"ProfileToken"`
	Velocity     PTZSpeed `xml:"Velocity"`
	Timeout      string   `xml:"Timeout,omitempty"`
}

// ContinuousMoveResponse represents ContinuousMove response
type ContinuousMoveResponse struct {
	XMLName xml.Name `xml:"tptz:ContinuousMoveResponse"`
}

// StopRequest represents Stop request
type StopRequest struct {
	XMLName      xml.Name `xml:"Stop"`
	ProfileToken string   `xml:"ProfileToken"`
	PanTilt      bool     `xml:"PanTilt,omitempty"`
	Zoom         bool     `xml:"Zoom,omitempty"`
}

// StopResponse represents Stop response
type StopResponse struct {
	XMLName xml.Name `xml:"tptz:StopResponse"`
}

// Service represents the PTZ service
type Service struct {
	registry *camera.Registry
}

// NewService creates a new PTZ service
func NewService(registry *camera.Registry) *Service {
	return &Service{
		registry: registry,
	}
}

// GetNodes handles GetNodes request
func (s *Service) GetNodes() *GetNodesResponse {
	return &GetNodesResponse{
		PTZNode: []PTZNode{
			{
				Token: "PTZNode_1",
				Name:  "PTZ Node 1",
				SupportedPTZSpaces: PTZSpaces{
					AbsolutePanTiltPositionSpace: []Space2D{
						{
							URI: "http://www.onvif.org/ver10/tptz/PanTiltSpaces/PositionGenericSpace",
							XRange: Range{Min: -1.0, Max: 1.0},
							YRange: Range{Min: -1.0, Max: 1.0},
						},
					},
					ContinuousPanTiltVelocitySpace: []Space2D{
						{
							URI: "http://www.onvif.org/ver10/tptz/PanTiltSpaces/VelocityGenericSpace",
							XRange: Range{Min: -1.0, Max: 1.0},
							YRange: Range{Min: -1.0, Max: 1.0},
						},
					},
					PanTiltSpeedSpace: []Space1D{
						{
							URI:    "http://www.onvif.org/ver10/tptz/PanTiltSpaces/GenericSpeedSpace",
							XRange: Range{Min: 0.0, Max: 1.0},
						},
					},
				},
				MaximumNumberOfPresets: 0,
				HomeSupported:          false,
			},
		},
	}
}

// GetConfigurations handles GetConfigurations request
func (s *Service) GetConfigurations() *GetConfigurationsResponse {
	return &GetConfigurationsResponse{
		PTZConfiguration: []PTZConfiguration{
			{
				Token:     "PTZConfig_1",
				Name:      "PTZ Configuration 1",
				NodeToken: "PTZNode_1",
				DefaultPTZSpeed: &PTZSpeed{
					PanTilt: &Vector2D{
						Space: "http://www.onvif.org/ver10/tptz/PanTiltSpaces/VelocityGenericSpace",
						X:     0.5,
						Y:     0.5,
					},
				},
				DefaultPTZTimeout: "PT10S",
				PanTiltLimits: &PanTiltLimits{
					Range: Space2D{
						URI:    "http://www.onvif.org/ver10/tptz/PanTiltSpaces/PositionGenericSpace",
						XRange: Range{Min: -1.0, Max: 1.0},
						YRange: Range{Min: -1.0, Max: 1.0},
					},
				},
			},
		},
	}
}

// ContinuousMove handles ContinuousMove request
func (s *Service) ContinuousMove(profileToken string, velocity PTZSpeed) error {
	// Get camera from profile token
	profile, err := s.registry.GetProfileByToken(profileToken)
	if err != nil {
		return fmt.Errorf("profile not found: %s", profileToken)
	}

	// Check if camera supports PTZ
	if !profile.Camera.Config.Capabilities.PTZ {
		return fmt.Errorf("camera does not support PTZ: %s", profile.Camera.Config.Name)
	}

	// Extract velocity
	var velocityX, velocityY float64
	if velocity.PanTilt != nil {
		velocityX = velocity.PanTilt.X
		velocityY = velocity.PanTilt.Y
	}

	// Convert ONVIF coordinates to AtomCam coordinates
	// Use absolute value for velocity magnitude
	velocityMag := velocityX*velocityX + velocityY*velocityY
	if velocityMag < 0.01 {
		// Velocity too small, treat as stop
		return profile.Camera.Client.PTZStop()
	}

	pan, tilt, speed := ONVIFToAtomCam(velocityX, velocityY, 0.5)

	// Send PTZ move command
	return profile.Camera.Client.PTZMove(pan, tilt, speed)
}

// Stop handles Stop request
func (s *Service) Stop(profileToken string) error {
	// Get camera from profile token
	profile, err := s.registry.GetProfileByToken(profileToken)
	if err != nil {
		return fmt.Errorf("profile not found: %s", profileToken)
	}

	// Check if camera supports PTZ
	if !profile.Camera.Config.Capabilities.PTZ {
		return fmt.Errorf("camera does not support PTZ: %s", profile.Camera.Config.Name)
	}

	// Send PTZ stop command
	return profile.Camera.Client.PTZStop()
}

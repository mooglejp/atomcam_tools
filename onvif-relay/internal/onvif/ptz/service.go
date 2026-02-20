package ptz

import (
	"encoding/xml"
	"fmt"
	"log"
	"math"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/camera"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/mqtt"
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

// GotoHomePositionRequest represents GotoHomePosition request
type GotoHomePositionRequest struct {
	XMLName      xml.Name `xml:"GotoHomePosition"`
	ProfileToken string   `xml:"ProfileToken"`
	Speed        *PTZSpeed `xml:"Speed,omitempty"`
}

// GotoHomePositionResponse represents GotoHomePosition response
type GotoHomePositionResponse struct {
	XMLName xml.Name `xml:"tptz:GotoHomePositionResponse"`
}

// GetPresetsRequest represents GetPresets request
type GetPresetsRequest struct {
	XMLName      xml.Name `xml:"GetPresets"`
	ProfileToken string   `xml:"ProfileToken"`
}

// GetPresetsResponse represents GetPresets response
type GetPresetsResponse struct {
	XMLName xml.Name `xml:"tptz:GetPresetsResponse"`
	Preset  []Preset `xml:"tptz:Preset"`
}

// Preset represents a PTZ preset
type Preset struct {
	Token    string        `xml:"token,attr"`
	Name     string        `xml:"tt:Name"`
	PTZPosition *PTZPosition `xml:"tt:PTZPosition,omitempty"`
}

// PTZPosition represents a PTZ position
type PTZPosition struct {
	PanTilt *Vector2D `xml:"tt:PanTilt,omitempty"`
	Zoom    *Vector1D `xml:"tt:Zoom,omitempty"`
}

// PTZVector represents a PTZ vector (for AbsoluteMove/RelativeMove)
type PTZVector struct {
	PanTilt *Vector2D `xml:"PanTilt,omitempty"`
	Zoom    *Vector1D `xml:"Zoom,omitempty"`
}

// GotoPresetRequest represents GotoPreset request
type GotoPresetRequest struct {
	XMLName      xml.Name `xml:"GotoPreset"`
	ProfileToken string   `xml:"ProfileToken"`
	PresetToken  string   `xml:"PresetToken"`
	Speed        *PTZSpeed `xml:"Speed,omitempty"`
}

// GotoPresetResponse represents GotoPreset response
type GotoPresetResponse struct {
	XMLName xml.Name `xml:"tptz:GotoPresetResponse"`
}

// AbsoluteMoveRequest represents AbsoluteMove request
type AbsoluteMoveRequest struct {
	XMLName      xml.Name   `xml:"AbsoluteMove"`
	ProfileToken string     `xml:"ProfileToken"`
	Position     PTZVector  `xml:"Position"`
	Speed        *PTZSpeed  `xml:"Speed,omitempty"`
}

// AbsoluteMoveResponse represents AbsoluteMove response
type AbsoluteMoveResponse struct {
	XMLName xml.Name `xml:"tptz:AbsoluteMoveResponse"`
}

// RelativeMoveRequest represents RelativeMove request
type RelativeMoveRequest struct {
	XMLName      xml.Name   `xml:"RelativeMove"`
	ProfileToken string     `xml:"ProfileToken"`
	Translation  PTZVector  `xml:"Translation"`
	Speed        *PTZSpeed  `xml:"Speed,omitempty"`
}

// RelativeMoveResponse represents RelativeMove response
type RelativeMoveResponse struct {
	XMLName xml.Name `xml:"tptz:RelativeMoveResponse"`
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
					RelativePanTiltTranslationSpace: []Space2D{
						{
							URI: "http://www.onvif.org/ver10/tptz/PanTiltSpaces/TranslationGenericSpace",
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
				HomeSupported:          true,
				FixedHomePosition:      true,
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

	// Check if this is a zoom-only operation
	if velocity.Zoom != nil && velocity.PanTilt == nil {
		// AtomCam doesn't support zoom, ignore zoom-only operations
		return nil
	}

	// Check if this is a zoom operation with pan/tilt
	if velocity.Zoom != nil && velocity.PanTilt != nil {
		// Ignore zoom component, only process pan/tilt
		// (AtomCam doesn't support zoom)
	}

	// Extract velocity
	var velocityX, velocityY float64
	if velocity.PanTilt != nil {
		velocityX = velocity.PanTilt.X
		velocityY = velocity.PanTilt.Y
	}

	// Calculate velocity magnitude
	velocityMag := math.Sqrt(velocityX*velocityX + velocityY*velocityY)
	if velocityMag < 0.01 {
		// Velocity too small, treat as stop
		return profile.Camera.Client.PTZStop()
	}

	// Get current position
	currentPan, currentTilt := profile.Camera.GetPTZPosition()

	// Calculate movement delta based on velocity
	// velocity range: -1.0 to 1.0
	// Scale to reasonable movement: ±5 degrees per command
	const movementScale = 5.0
	deltaPan := int(velocityX * movementScale)
	deltaTilt := int(velocityY * movementScale) // Y: positive = down, negative = up

	// Calculate new position
	newPan := currentPan + deltaPan
	newTilt := currentTilt + deltaTilt

	// Clamp to valid range
	if newPan < 0 {
		newPan = 0
	}
	if newPan > 355 {
		newPan = 355
	}
	if newTilt < 0 {
		newTilt = 0
	}
	if newTilt > 180 {
		newTilt = 180
	}

	// Convert velocity magnitude to speed (5-9)
	// Use higher speeds to avoid firmware issues
	speed := int(math.Round(velocityMag*4.0)) + 5
	if speed < 5 {
		speed = 5
	}
	if speed > 9 {
		speed = 9
	}

	// Update tracked position
	profile.Camera.SetPTZPosition(newPan, newTilt)

	// Send PTZ move command
	return profile.Camera.Client.PTZMove(newPan, newTilt, speed)
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

// GotoHomePosition handles GotoHomePosition request
func (s *Service) GotoHomePosition(profileToken string, speed *PTZSpeed) error {
	// Get camera from profile token
	profile, err := s.registry.GetProfileByToken(profileToken)
	if err != nil {
		return fmt.Errorf("profile not found: %s", profileToken)
	}

	// Check if camera supports PTZ
	if !profile.Camera.Config.Capabilities.PTZ {
		return fmt.Errorf("camera does not support PTZ: %s", profile.Camera.Config.Name)
	}

	// Default speed for home position
	defaultSpeed := 5
	if speed != nil && speed.PanTilt != nil {
		// Convert ONVIF speed (0.0-1.0) to AtomCam speed (1-9)
		speedMag := speed.PanTilt.X*speed.PanTilt.X + speed.PanTilt.Y*speed.PanTilt.Y
		if speedMag > 0.01 {
			// Use average of X and Y components
			avgSpeed := (speed.PanTilt.X + speed.PanTilt.Y) / 2.0
			defaultSpeed = int(avgSpeed*8) + 1 // Map 0.0-1.0 to 1-9
			if defaultSpeed < 1 {
				defaultSpeed = 1
			}
			if defaultSpeed > 9 {
				defaultSpeed = 9
			}
		}
	}

	// Get home position from config, or use default
	pan := 160
	tilt := 130
	if profile.Camera.Config.PTZ.Home != nil {
		pan = profile.Camera.Config.PTZ.Home.Pan
		tilt = profile.Camera.Config.PTZ.Home.Tilt
	}

	// Update tracked position
	profile.Camera.SetPTZPosition(pan, tilt)

	return profile.Camera.Client.PTZMove(pan, tilt, defaultSpeed)
}

// GetPresets handles GetPresets request
func (s *Service) GetPresets(profileToken string) (*GetPresetsResponse, error) {
	// Get camera from profile token
	profile, err := s.registry.GetProfileByToken(profileToken)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %s", profileToken)
	}

	// Check if camera supports PTZ
	if !profile.Camera.Config.Capabilities.PTZ {
		return nil, fmt.Errorf("camera does not support PTZ: %s", profile.Camera.Config.Name)
	}

	presets := []Preset{}

	// Add presets from config
	for i, preset := range profile.Camera.Config.PTZ.Presets {
		token := preset.Token
		if token == "" {
			token = fmt.Sprintf("%d", i+1) // Default to 1-based index
		}

		presets = append(presets, Preset{
			Token: token,
			Name:  preset.Name,
		})
	}

	return &GetPresetsResponse{
		Preset: presets,
	}, nil
}

// GotoPreset handles GotoPreset request
func (s *Service) GotoPreset(profileToken, presetToken string, speed *PTZSpeed) error {
	// Get camera from profile token
	profile, err := s.registry.GetProfileByToken(profileToken)
	if err != nil {
		return fmt.Errorf("profile not found: %s", profileToken)
	}

	// Check if camera supports PTZ
	if !profile.Camera.Config.Capabilities.PTZ {
		return fmt.Errorf("camera does not support PTZ: %s", profile.Camera.Config.Name)
	}

	// Find preset in config
	var preset *config.PTZPreset
	for i := range profile.Camera.Config.PTZ.Presets {
		p := &profile.Camera.Config.PTZ.Presets[i]
		token := p.Token
		if token == "" {
			token = fmt.Sprintf("%d", i+1)
		}
		if token == presetToken {
			preset = p
			break
		}
	}

	if preset == nil {
		return fmt.Errorf("preset not found: %s", presetToken)
	}

	// Check if this is an MQTT preset
	if preset.MQTTBroker != "" && preset.MQTTTopic != "" {
		// MQTT preset: publish message instead of moving camera
		log.Printf("PTZ GotoPreset: MQTT action - broker=%s, topic=%s, message=%s",
			preset.MQTTBroker, preset.MQTTTopic, preset.MQTTMessage)

		if err := mqtt.PublishMessage(preset.MQTTBroker, preset.MQTTTopic, preset.MQTTMessage); err != nil {
			return fmt.Errorf("failed to publish MQTT message for preset %s: %w", presetToken, err)
		}

		log.Printf("PTZ GotoPreset: MQTT message published successfully")
		return nil
	}

	// Default speed
	defaultSpeed := 5
	if speed != nil && speed.PanTilt != nil {
		speedMag := speed.PanTilt.X*speed.PanTilt.X + speed.PanTilt.Y*speed.PanTilt.Y
		if speedMag > 0.01 {
			avgSpeed := (speed.PanTilt.X + speed.PanTilt.Y) / 2.0
			defaultSpeed = int(avgSpeed*8) + 1
			if defaultSpeed < 1 {
				defaultSpeed = 1
			}
			if defaultSpeed > 9 {
				defaultSpeed = 9
			}
		}
	}

	// Update tracked position
	profile.Camera.SetPTZPosition(preset.Pan, preset.Tilt)

	return profile.Camera.Client.PTZMove(preset.Pan, preset.Tilt, defaultSpeed)
}

// AbsoluteMove handles AbsoluteMove request
func (s *Service) AbsoluteMove(profileToken string, position PTZVector, speed *PTZSpeed) error {
	// Get camera from profile token
	profile, err := s.registry.GetProfileByToken(profileToken)
	if err != nil {
		return fmt.Errorf("profile not found: %s", profileToken)
	}

	// Check if camera supports PTZ
	if !profile.Camera.Config.Capabilities.PTZ {
		return fmt.Errorf("camera does not support PTZ: %s", profile.Camera.Config.Name)
	}

	// Check if this is a zoom-only operation
	if position.Zoom != nil && position.PanTilt == nil {
		// AtomCam doesn't support zoom, ignore zoom-only operations
		return nil
	}

	// Extract position coordinates
	if position.PanTilt == nil {
		return fmt.Errorf("pan/tilt position required for AbsoluteMove")
	}

	x := position.PanTilt.X
	y := position.PanTilt.Y

	// Check if FOV is configured
	var pan, tilt int
	if profile.Camera.Config.PTZ.HorizontalFOV > 0 && profile.Camera.Config.PTZ.VerticalFOV > 0 {
		// Use FOV-aware conversion: treat ONVIF coordinates as positions within current view
		// Get current position as the center of the view
		currentPan, currentTilt := profile.Camera.GetPTZPosition()

		// Check if position is uninitialized
		if currentPan == math.MinInt || currentTilt == math.MinInt || currentPan < 0 || currentTilt < 0 {
			// Use home position or center as default
			if profile.Camera.Config.PTZ.Home != nil {
				currentPan = profile.Camera.Config.PTZ.Home.Pan
				currentTilt = profile.Camera.Config.PTZ.Home.Tilt
			} else {
				currentPan = 177  // Center
				currentTilt = 90  // Center
			}
			profile.Camera.SetPTZPosition(currentPan, currentTilt)
			log.Printf("PTZ AbsoluteMove: Initialized position to (%d, %d)", currentPan, currentTilt)
		}

		// Calculate offset from center based on FOV
		// ONVIF x,y ∈ [-1.0, 1.0] represents position within the current view
		halfHorizontalFOV := profile.Camera.Config.PTZ.HorizontalFOV / 2.0
		halfVerticalFOV := profile.Camera.Config.PTZ.VerticalFOV / 2.0

		deltaPan := int(math.Round(x * halfHorizontalFOV))
		deltaTilt := int(math.Round((1.0 - y) * halfVerticalFOV)) - int(halfVerticalFOV)

		pan = currentPan + deltaPan
		tilt = currentTilt + deltaTilt

		// Clamp to valid range
		if pan < 0 {
			pan = 0
		}
		if pan > 355 {
			pan = 355
		}
		if tilt < 0 {
			tilt = 0
		}
		if tilt > 180 {
			tilt = 180
		}

		log.Printf("PTZ AbsoluteMove (FOV-aware): ONVIF=(%.2f, %.2f), current=(%d, %d), delta=(%d, %d), new AtomCam=(%d, %d)",
			x, y, currentPan, currentTilt, deltaPan, deltaTilt, pan, tilt)
	} else {
		// Fallback to legacy conversion (absolute position mapping)
		var defaultSpeed int
		pan, tilt, defaultSpeed = ONVIFToAtomCam(x, y, 0.5)
		_ = defaultSpeed // unused in this path
		log.Printf("PTZ AbsoluteMove (legacy): ONVIF=(%.2f, %.2f) -> AtomCam=(%d, %d)", x, y, pan, tilt)
	}

	defaultSpeed := 5

	// Override speed if provided
	if speed != nil && speed.PanTilt != nil {
		speedMag := math.Sqrt(speed.PanTilt.X*speed.PanTilt.X + speed.PanTilt.Y*speed.PanTilt.Y)
		if speedMag > 0.01 {
			// Convert ONVIF speed (0.0-1.0) to AtomCam speed (5-9)
			defaultSpeed = int(math.Round(speedMag*4.0)) + 5
			if defaultSpeed < 5 {
				defaultSpeed = 5
			}
			if defaultSpeed > 9 {
				defaultSpeed = 9
			}
		}
	}

	// Update tracked position
	profile.Camera.SetPTZPosition(pan, tilt)

	// Send PTZ move command
	return profile.Camera.Client.PTZMove(pan, tilt, defaultSpeed)
}

// RelativeMove handles RelativeMove request
func (s *Service) RelativeMove(profileToken string, translation PTZVector, speed *PTZSpeed) error {
	// Get camera from profile token
	profile, err := s.registry.GetProfileByToken(profileToken)
	if err != nil {
		return fmt.Errorf("profile not found: %s", profileToken)
	}

	// Check if camera supports PTZ
	if !profile.Camera.Config.Capabilities.PTZ {
		return fmt.Errorf("camera does not support PTZ: %s", profile.Camera.Config.Name)
	}

	// Check if this is a zoom-only operation
	if translation.Zoom != nil && translation.PanTilt == nil {
		// AtomCam doesn't support zoom, ignore zoom-only operations
		return nil
	}

	// Extract translation vector
	if translation.PanTilt == nil {
		return fmt.Errorf("pan/tilt translation required for RelativeMove")
	}

	translationX := translation.PanTilt.X
	translationY := translation.PanTilt.Y

	// Validate translation values (reject NaN)
	if math.IsNaN(translationX) || math.IsNaN(translationY) {
		return fmt.Errorf("invalid translation values: x=%f, y=%f (NaN not allowed)", translationX, translationY)
	}

	// Get current position
	currentPan, currentTilt := profile.Camera.GetPTZPosition()

	// Check if position is uninitialized (int min value indicates uninitialized state)
	if currentPan == math.MinInt || currentTilt == math.MinInt || currentPan < 0 || currentTilt < 0 {
		// Use home position or center as default
		if profile.Camera.Config.PTZ.Home != nil {
			currentPan = profile.Camera.Config.PTZ.Home.Pan
			currentTilt = profile.Camera.Config.PTZ.Home.Tilt
		} else {
			currentPan = 177  // Center
			currentTilt = 90  // Center
		}
		// Update tracked position
		profile.Camera.SetPTZPosition(currentPan, currentTilt)
		log.Printf("PTZ RelativeMove: Initialized position to (%d, %d)", currentPan, currentTilt)
	}

	// Check if FOV is configured
	var pan, tilt int
	if profile.Camera.Config.PTZ.HorizontalFOV > 0 && profile.Camera.Config.PTZ.VerticalFOV > 0 {
		// Use FOV-aware conversion: treat translation as position within current view
		// Translation ∈ [-1.0, 1.0] represents offset within the field of view
		halfHorizontalFOV := profile.Camera.Config.PTZ.HorizontalFOV / 2.0
		halfVerticalFOV := profile.Camera.Config.PTZ.VerticalFOV / 2.0

		// Calculate movement delta based on FOV
		// translation -1.0 = far left/top of view, 0.0 = center, 1.0 = far right/bottom of view
		deltaPan := int(math.Round(translationX * halfHorizontalFOV))
		deltaTilt := int(math.Round(-translationY * halfVerticalFOV)) // Invert Y to match ONVIF (positive = up)

		pan = currentPan + deltaPan
		tilt = currentTilt + deltaTilt

		// Clamp to valid range
		if pan < 0 {
			pan = 0
		}
		if pan > 355 {
			pan = 355
		}
		if tilt < 0 {
			tilt = 0
		}
		if tilt > 180 {
			tilt = 180
		}

		log.Printf("PTZ RelativeMove (FOV-aware): current=(%d, %d), translation=(%.2f, %.2f), FOV=(%.1f°, %.1f°), delta=(%d, %d), new AtomCam=(%d, %d)",
			currentPan, currentTilt, translationX, translationY, profile.Camera.Config.PTZ.HorizontalFOV, profile.Camera.Config.PTZ.VerticalFOV, deltaPan, deltaTilt, pan, tilt)
	} else {
		// Fallback to legacy conversion (ONVIF coordinate space mapping)
		currentX, currentY := AtomCamToONVIF(currentPan, currentTilt)

		// Add translation to current position
		newX := currentX + translationX
		newY := currentY + translationY

		// Clamp to valid ONVIF range [-1.0, 1.0]
		if newX < -1.0 {
			newX = -1.0
		}
		if newX > 1.0 {
			newX = 1.0
		}
		if newY < -1.0 {
			newY = -1.0
		}
		if newY > 1.0 {
			newY = 1.0
		}

		// Convert new ONVIF coordinates to AtomCam coordinates
		var defaultSpeed int
		pan, tilt, defaultSpeed = ONVIFToAtomCam(newX, newY, 0.5)
		_ = defaultSpeed // unused in this path

		log.Printf("PTZ RelativeMove (legacy): current=(%d, %d), ONVIF current=(%.2f, %.2f), translation=(%.2f, %.2f), new ONVIF=(%.2f, %.2f), new AtomCam=(%d, %d)",
			currentPan, currentTilt, currentX, currentY, translationX, translationY, newX, newY, pan, tilt)
	}

	defaultSpeed := 5

	// Override speed if provided
	if speed != nil && speed.PanTilt != nil {
		speedMag := math.Sqrt(speed.PanTilt.X*speed.PanTilt.X + speed.PanTilt.Y*speed.PanTilt.Y)
		if speedMag > 0.01 {
			// Convert ONVIF speed (0.0-1.0) to AtomCam speed (5-9)
			defaultSpeed = int(math.Round(speedMag*4.0)) + 5
			if defaultSpeed < 5 {
				defaultSpeed = 5
			}
			if defaultSpeed > 9 {
				defaultSpeed = 9
			}
		}
	}

	// Update tracked position
	profile.Camera.SetPTZPosition(pan, tilt)

	// Send PTZ move command
	return profile.Camera.Client.PTZMove(pan, tilt, defaultSpeed)
}

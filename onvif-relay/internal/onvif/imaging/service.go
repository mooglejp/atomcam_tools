package imaging

import (
	"encoding/xml"
	"fmt"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/camera"
)

// GetImagingSettingsRequest represents GetImagingSettings request
type GetImagingSettingsRequest struct {
	XMLName         xml.Name `xml:"GetImagingSettings"`
	VideoSourceToken string  `xml:"VideoSourceToken"`
}

// GetImagingSettingsResponse represents GetImagingSettings response
type GetImagingSettingsResponse struct {
	XMLName        xml.Name       `xml:"timg:GetImagingSettingsResponse"`
	ImagingSettings ImagingSettings `xml:"ImagingSettings"`
}

// ImagingSettings represents imaging settings
type ImagingSettings struct {
	Brightness    *float64    `xml:"Brightness,omitempty"`
	ColorSaturation *float64  `xml:"ColorSaturation,omitempty"`
	Contrast      *float64    `xml:"Contrast,omitempty"`
	Sharpness     *float64    `xml:"Sharpness,omitempty"`
	IrCutFilter   *string     `xml:"IrCutFilter,omitempty"`
	Exposure      *Exposure   `xml:"Exposure,omitempty"`
	WideDynamicRange *WideDynamicRange `xml:"WideDynamicRange,omitempty"`
	WhiteBalance  *WhiteBalance `xml:"WhiteBalance,omitempty"`
}

// Exposure represents exposure settings
type Exposure struct {
	Mode        string   `xml:"Mode"`
	Priority    string   `xml:"Priority,omitempty"`
	MinExposureTime *float64 `xml:"MinExposureTime,omitempty"`
	MaxExposureTime *float64 `xml:"MaxExposureTime,omitempty"`
	MinGain     *float64 `xml:"MinGain,omitempty"`
	MaxGain     *float64 `xml:"MaxGain,omitempty"`
	MinIris     *float64 `xml:"MinIris,omitempty"`
	MaxIris     *float64 `xml:"MaxIris,omitempty"`
	ExposureTime *float64 `xml:"ExposureTime,omitempty"`
	Gain        *float64 `xml:"Gain,omitempty"`
	Iris        *float64 `xml:"Iris,omitempty"`
}

// WideDynamicRange represents WDR settings
type WideDynamicRange struct {
	Mode  string   `xml:"Mode"`
	Level *float64 `xml:"Level,omitempty"`
}

// WhiteBalance represents white balance settings
type WhiteBalance struct {
	Mode string   `xml:"Mode"`
	CrGain *float64 `xml:"CrGain,omitempty"`
	CbGain *float64 `xml:"CbGain,omitempty"`
}

// SetImagingSettingsRequest represents SetImagingSettings request
type SetImagingSettingsRequest struct {
	XMLName         xml.Name        `xml:"SetImagingSettings"`
	VideoSourceToken string         `xml:"VideoSourceToken"`
	ImagingSettings ImagingSettings `xml:"ImagingSettings"`
	ForcePersistence bool           `xml:"ForcePersistence,omitempty"`
}

// SetImagingSettingsResponse represents SetImagingSettings response
type SetImagingSettingsResponse struct {
	XMLName xml.Name `xml:"timg:SetImagingSettingsResponse"`
}

// GetOptionsRequest represents GetOptions request
type GetOptionsRequest struct {
	XMLName         xml.Name `xml:"GetOptions"`
	VideoSourceToken string  `xml:"VideoSourceToken"`
}

// GetOptionsResponse represents GetOptions response
type GetOptionsResponse struct {
	XMLName        xml.Name       `xml:"timg:GetOptionsResponse"`
	ImagingOptions ImagingOptions `xml:"ImagingOptions"`
}

// ImagingOptions represents imaging options
type ImagingOptions struct {
	Brightness       *FloatRange      `xml:"Brightness,omitempty"`
	ColorSaturation  *FloatRange      `xml:"ColorSaturation,omitempty"`
	Contrast         *FloatRange      `xml:"Contrast,omitempty"`
	Sharpness        *FloatRange      `xml:"Sharpness,omitempty"`
	IrCutFilterModes []string         `xml:"IrCutFilterModes,omitempty"`
	Exposure         *ExposureOptions `xml:"Exposure,omitempty"`
}

// FloatRange represents a float value range
type FloatRange struct {
	Min float64 `xml:"Min"`
	Max float64 `xml:"Max"`
}

// ExposureOptions represents exposure options
type ExposureOptions struct {
	Mode            []string    `xml:"Mode"`
	Priority        []string    `xml:"Priority,omitempty"`
	MinExposureTime *FloatRange `xml:"MinExposureTime,omitempty"`
	MaxExposureTime *FloatRange `xml:"MaxExposureTime,omitempty"`
	MinGain         *FloatRange `xml:"MinGain,omitempty"`
	MaxGain         *FloatRange `xml:"MaxGain,omitempty"`
	MinIris         *FloatRange `xml:"MinIris,omitempty"`
	MaxIris         *FloatRange `xml:"MaxIris,omitempty"`
	ExposureTime    *FloatRange `xml:"ExposureTime,omitempty"`
	Gain            *FloatRange `xml:"Gain,omitempty"`
	Iris            *FloatRange `xml:"Iris,omitempty"`
}

// Service represents the Imaging service
type Service struct {
	registry *camera.Registry
}

// NewService creates a new Imaging service
func NewService(registry *camera.Registry) *Service {
	return &Service{
		registry: registry,
	}
}

// GetImagingSettings handles GetImagingSettings request
func (s *Service) GetImagingSettings(videoSourceToken string) *GetImagingSettingsResponse {
	// Return default settings
	// Note: AtomCam doesn't provide a way to query current settings,
	// so we return sensible defaults (center values)
	brightness := 0.5
	contrast := 0.5
	saturation := 0.5
	sharpness := 0.5
	irCutFilter := "AUTO"

	return &GetImagingSettingsResponse{
		ImagingSettings: ImagingSettings{
			Brightness:      &brightness,
			ColorSaturation: &saturation,
			Contrast:        &contrast,
			Sharpness:       &sharpness,
			IrCutFilter:     &irCutFilter,
			Exposure: &Exposure{
				Mode: "AUTO",
			},
		},
	}
}

// SetImagingSettings handles SetImagingSettings request
func (s *Service) SetImagingSettings(videoSourceToken string, settings ImagingSettings) error {
	// For now, apply settings to all cameras
	// TODO: Map videoSourceToken to specific camera
	cameras := s.registry.List()
	if len(cameras) == 0 {
		return fmt.Errorf("no cameras configured")
	}

	// Apply to first camera (or all cameras - depends on requirements)
	cam := cameras[0]

	// Apply brightness
	if settings.Brightness != nil {
		if err := cam.Client.SetBrightness(*settings.Brightness); err != nil {
			return fmt.Errorf("failed to set brightness: %w", err)
		}
	}

	// Apply contrast
	if settings.Contrast != nil {
		if err := cam.Client.SetContrast(*settings.Contrast); err != nil {
			return fmt.Errorf("failed to set contrast: %w", err)
		}
	}

	// Apply saturation
	if settings.ColorSaturation != nil {
		if err := cam.Client.SetSaturation(*settings.ColorSaturation); err != nil {
			return fmt.Errorf("failed to set saturation: %w", err)
		}
	}

	// Apply sharpness
	if settings.Sharpness != nil {
		if err := cam.Client.SetSharpness(*settings.Sharpness); err != nil {
			return fmt.Errorf("failed to set sharpness: %w", err)
		}
	}

	// Apply IR cut filter
	if settings.IrCutFilter != nil && cam.Config.Capabilities.IR {
		if err := cam.Client.SetIRCutFilter(*settings.IrCutFilter); err != nil {
			return fmt.Errorf("failed to set IR cut filter: %w", err)
		}
	}

	// Apply exposure settings
	if settings.Exposure != nil {
		// Default exposure time values (from config migration)
		minTime := 1
		maxTime := 1683

		if settings.Exposure.MinExposureTime != nil {
			minTime = int(*settings.Exposure.MinExposureTime)
		}
		if settings.Exposure.MaxExposureTime != nil {
			maxTime = int(*settings.Exposure.MaxExposureTime)
		}

		if err := cam.Client.SetExposureMode(settings.Exposure.Mode, minTime, maxTime); err != nil {
			return fmt.Errorf("failed to set exposure: %w", err)
		}
	}

	return nil
}

// GetOptions handles GetOptions request
func (s *Service) GetOptions(videoSourceToken string) *GetOptionsResponse {
	irCutModes := []string{"ON", "OFF", "AUTO"}

	return &GetOptionsResponse{
		ImagingOptions: ImagingOptions{
			Brightness: &FloatRange{
				Min: 0.0,
				Max: 1.0,
			},
			ColorSaturation: &FloatRange{
				Min: 0.0,
				Max: 1.0,
			},
			Contrast: &FloatRange{
				Min: 0.0,
				Max: 1.0,
			},
			Sharpness: &FloatRange{
				Min: 0.0,
				Max: 1.0,
			},
			IrCutFilterModes: irCutModes,
			Exposure: &ExposureOptions{
				Mode:     []string{"AUTO", "MANUAL"},
				Priority: []string{"LowNoise", "FrameRate"},
				MinExposureTime: &FloatRange{
					Min: 1.0,
					Max: 10000.0,
				},
				MaxExposureTime: &FloatRange{
					Min: 1.0,
					Max: 10000.0,
				},
			},
		},
	}
}

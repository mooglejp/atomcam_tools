package device

import (
	"encoding/xml"
)

// GetCapabilitiesRequest represents GetCapabilities request
type GetCapabilitiesRequest struct {
	XMLName xml.Name `xml:"GetCapabilities"`
	Category []string `xml:"Category,omitempty"`
}

// GetCapabilitiesResponse represents GetCapabilities response
type GetCapabilitiesResponse struct {
	XMLName      xml.Name     `xml:"tds:GetCapabilitiesResponse"`
	Capabilities Capabilities `xml:"Capabilities"`
}

// Capabilities represents device capabilities
type Capabilities struct {
	Device  *DeviceCapabilities  `xml:"Device,omitempty"`
	Media   *MediaCapabilities   `xml:"Media,omitempty"`
	PTZ     *PTZCapabilities     `xml:"PTZ,omitempty"`
	Imaging *ImagingCapabilities `xml:"Imaging,omitempty"`
}

// DeviceCapabilities represents device service capabilities
type DeviceCapabilities struct {
	XAddr   string                    `xml:"XAddr"`
	Network *NetworkCapabilities      `xml:"Network,omitempty"`
	System  *SystemCapabilities       `xml:"System,omitempty"`
	IO      *IOCapabilities           `xml:"IO,omitempty"`
	Security *SecurityCapabilities    `xml:"Security,omitempty"`
}

// NetworkCapabilities represents network capabilities
type NetworkCapabilities struct {
	IPFilter            bool `xml:"IPFilter,attr,omitempty"`
	ZeroConfiguration   bool `xml:"ZeroConfiguration,attr,omitempty"`
	IPVersion6          bool `xml:"IPVersion6,attr,omitempty"`
	DynDNS              bool `xml:"DynDNS,attr,omitempty"`
}

// SystemCapabilities represents system capabilities
type SystemCapabilities struct {
	DiscoveryResolve    bool `xml:"DiscoveryResolve,attr,omitempty"`
	DiscoveryBye        bool `xml:"DiscoveryBye,attr,omitempty"`
	RemoteDiscovery     bool `xml:"RemoteDiscovery,attr,omitempty"`
	SystemBackup        bool `xml:"SystemBackup,attr,omitempty"`
	SystemLogging       bool `xml:"SystemLogging,attr,omitempty"`
	FirmwareUpgrade     bool `xml:"FirmwareUpgrade,attr,omitempty"`
}

// IOCapabilities represents I/O capabilities
type IOCapabilities struct {
	InputConnectors  int `xml:"InputConnectors,attr,omitempty"`
	RelayOutputs     int `xml:"RelayOutputs,attr,omitempty"`
}

// SecurityCapabilities represents security capabilities
type SecurityCapabilities struct {
	TLS11           bool `xml:"TLS1.1,attr,omitempty"`
	TLS12           bool `xml:"TLS1.2,attr,omitempty"`
	OnboardKeyGeneration bool `xml:"OnboardKeyGeneration,attr,omitempty"`
	AccessPolicyConfig bool `xml:"AccessPolicyConfig,attr,omitempty"`
}

// MediaCapabilities represents media service capabilities
type MediaCapabilities struct {
	XAddr            string                      `xml:"XAddr"`
	StreamingCapabilities *StreamingCapabilities `xml:"StreamingCapabilities,omitempty"`
}

// StreamingCapabilities represents streaming capabilities
type StreamingCapabilities struct {
	RTPMulticast bool `xml:"RTPMulticast,attr,omitempty"`
	RTP_TCP      bool `xml:"RTP_TCP,attr,omitempty"`
	RTP_RTSP_TCP bool `xml:"RTP_RTSP_TCP,attr,omitempty"`
}

// PTZCapabilities represents PTZ service capabilities
type PTZCapabilities struct {
	XAddr string `xml:"XAddr"`
}

// ImagingCapabilities represents Imaging service capabilities
type ImagingCapabilities struct {
	XAddr string `xml:"XAddr"`
}

// GetCapabilities handles GetCapabilities request
func (s *Service) GetCapabilities(categories []string) *GetCapabilitiesResponse {
	resp := &GetCapabilitiesResponse{
		Capabilities: Capabilities{},
	}

	// If no categories specified, return all
	if len(categories) == 0 {
		categories = []string{"All"}
	}

	for _, cat := range categories {
		switch cat {
		case "All", "Device":
			resp.Capabilities.Device = &DeviceCapabilities{
				XAddr: s.baseURL + "/onvif/device_service",
				Network: &NetworkCapabilities{
					IPFilter:          false,
					ZeroConfiguration: false,
					IPVersion6:        false,
					DynDNS:            false,
				},
				System: &SystemCapabilities{
					DiscoveryResolve: true,
					DiscoveryBye:     true,
					RemoteDiscovery:  false,
					SystemBackup:     false,
					SystemLogging:    false,
					FirmwareUpgrade:  false,
				},
				IO: &IOCapabilities{
					InputConnectors: 0,
					RelayOutputs:    0,
				},
				Security: &SecurityCapabilities{
					TLS11:                false,
					TLS12:                false,
					OnboardKeyGeneration: false,
					AccessPolicyConfig:   false,
				},
			}
		}

		switch cat {
		case "All", "Media":
			resp.Capabilities.Media = &MediaCapabilities{
				XAddr: s.baseURL + "/onvif/media_service",
				StreamingCapabilities: &StreamingCapabilities{
					RTPMulticast: false,
					RTP_TCP:      true,
					RTP_RTSP_TCP: true,
				},
			}
		}

		switch cat {
		case "All", "PTZ":
			resp.Capabilities.PTZ = &PTZCapabilities{
				XAddr: s.baseURL + "/onvif/ptz_service",
			}
		}

		switch cat {
		case "All", "Imaging":
			resp.Capabilities.Imaging = &ImagingCapabilities{
				XAddr: s.baseURL + "/onvif/imaging_service",
			}
		}
	}

	return resp
}

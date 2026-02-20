package onvif

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/camera"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/onvif/device"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/onvif/imaging"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/onvif/media"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/onvif/ptz"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/onvif/soap"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/snapshot"
)

// Server represents an ONVIF server
type Server struct {
	config         *config.Config
	registry       *camera.Registry
	deviceService  *device.Service
	mediaService   *media.Service
	ptzService     *ptz.Service
	imagingService *imaging.Service
	httpServer     *http.Server
}

// NewServer creates a new ONVIF server
func NewServer(cfg *config.Config, registry *camera.Registry) *Server {
	// Determine base URL for capabilities
	baseURL := fmt.Sprintf("http://localhost:%d", cfg.Server.OnvifPort)

	// Determine mediamtx RTSP host (use rtsp_host if specified, otherwise auto-detect)
	mediamtxHost := cfg.Server.Mediamtx.RTSPHost
	if mediamtxHost == "" {
		// Default to mediamtx service name in Docker network
		mediamtxHost = "mediamtx"
	}

	// Snapshot service uses localhost (or Docker service name)
	snapshotHost := "localhost"
	snapshotPort := cfg.Server.OnvifPort

	s := &Server{
		config:         cfg,
		registry:       registry,
		deviceService:  device.NewService(cfg.Server.DeviceName, baseURL),
		mediaService:   media.NewService(registry, mediamtxHost, cfg.Server.Mediamtx.RTSPPort, snapshotHost, snapshotPort),
		ptzService:     ptz.NewService(registry),
		imagingService: imaging.NewService(registry),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/onvif/device_service", s.handleDeviceService)
	mux.HandleFunc("/onvif/media_service", s.handleMediaService)
	mux.HandleFunc("/onvif/ptz_service", s.handlePTZService)
	mux.HandleFunc("/onvif/imaging_service", s.handleImagingService)

	// Root path handler for non-compliant ONVIF clients
	// Some clients ignore GetCapabilities XAddr and send requests to "/"
	mux.HandleFunc("/", s.handleRootService)

	// Snapshot endpoint with authentication
	snapshotProxy := snapshot.NewProxy(registry, cfg.Server.Auth.Username, cfg.Server.Auth.Password)
	mux.HandleFunc("/snapshot/", snapshotProxy.Handler())

	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.OnvifPort),
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	return s
}

// Start starts the ONVIF server
func (s *Server) Start() error {
	log.Printf("Starting ONVIF server on port %d", s.config.Server.OnvifPort)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// handleDeviceService handles Device service requests
func (s *Server) handleDeviceService(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests for SOAP
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to read request body"))
		return
	}
	defer r.Body.Close()

	action, err := soap.GetAction(body)
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to parse SOAP action"))
		return
	}

	log.Printf("Device service action: %s", action)

	// GetSystemDateAndTime is exempt from authentication per ONVIF spec
	if action != "GetSystemDateAndTime" {
		if err := s.validateAuth(body); err != nil {
			log.Printf("Authentication failed for %s: %v", action, err)
			s.sendFault(w, soap.NewNotAuthorizedFault())
			return
		}
	}

	var response interface{}
	switch action {
	case "GetDeviceInformation":
		response = s.deviceService.GetDeviceInformation()
	case "GetSystemDateAndTime":
		response = s.deviceService.GetSystemDateAndTime()
	case "GetCapabilities":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req device.GetCapabilitiesRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		response = s.deviceService.GetCapabilities(req.Category)
	default:
		s.sendFault(w, soap.NewActionFailedFault(fmt.Sprintf("Unknown action: %s", action)))
		return
	}

	s.sendResponse(w, response)
}

// handleMediaService handles Media service requests
func (s *Server) handleMediaService(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests for SOAP
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to read request body"))
		return
	}
	defer r.Body.Close()

	action, err := soap.GetAction(body)
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to parse SOAP action"))
		return
	}

	log.Printf("Media service action: %s", action)

	// Authentication required
	if err := s.validateAuth(body); err != nil {
		log.Printf("Authentication failed for %s: %v", action, err)
		s.sendFault(w, soap.NewNotAuthorizedFault())
		return
	}

	var response interface{}
	switch action {
	case "GetProfiles":
		response = s.mediaService.GetProfiles()
	case "GetStreamUri":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req media.GetStreamUriRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		resp, err := s.mediaService.GetStreamUri(req.ProfileToken)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault(err.Error()))
			return
		}
		response = resp
	case "GetSnapshotUri":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req media.GetSnapshotUriRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		resp, err := s.mediaService.GetSnapshotUri(req.ProfileToken)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault(err.Error()))
			return
		}
		response = resp
	default:
		s.sendFault(w, soap.NewActionFailedFault(fmt.Sprintf("Unknown action: %s", action)))
		return
	}

	s.sendResponse(w, response)
}

// handlePTZService handles PTZ service requests
func (s *Server) handlePTZService(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests for SOAP
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to read request body"))
		return
	}
	defer r.Body.Close()

	action, err := soap.GetAction(body)
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to parse SOAP action"))
		return
	}

	log.Printf("PTZ service action: %s", action)

	// Authentication required
	if err := s.validateAuth(body); err != nil {
		log.Printf("Authentication failed for %s: %v", action, err)
		s.sendFault(w, soap.NewNotAuthorizedFault())
		return
	}

	var response interface{}
	switch action {
	case "GetNodes":
		response = s.ptzService.GetNodes()
	case "GetConfigurations":
		response = s.ptzService.GetConfigurations()
	case "ContinuousMove":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.ContinuousMoveRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.ContinuousMove(req.ProfileToken, req.Velocity); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.ContinuousMoveResponse{}
	case "Stop":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.StopRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.Stop(req.ProfileToken); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.StopResponse{}
	case "GotoHomePosition":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			log.Printf("Failed to extract body content: %v", err)
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.GotoHomePositionRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			log.Printf("Failed to unmarshal GotoHomePosition request: %v", err)
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.GotoHomePosition(req.ProfileToken, req.Speed); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.GotoHomePositionResponse{}
	case "GetPresets":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.GetPresetsRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		resp, err := s.ptzService.GetPresets(req.ProfileToken)
		if err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = resp
	case "GotoPreset":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.GotoPresetRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.GotoPreset(req.ProfileToken, req.PresetToken, req.Speed); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.GotoPresetResponse{}
	case "AbsoluteMove":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.AbsoluteMoveRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.AbsoluteMove(req.ProfileToken, req.Position, req.Speed); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.AbsoluteMoveResponse{}
	case "RelativeMove":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.RelativeMoveRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.RelativeMove(req.ProfileToken, req.Translation, req.Speed); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.RelativeMoveResponse{}
	default:
		s.sendFault(w, soap.NewActionFailedFault(fmt.Sprintf("Unknown action: %s", action)))
		return
	}

	s.sendResponse(w, response)
}

// handleImagingService handles Imaging service requests
func (s *Server) handleImagingService(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests for SOAP
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to read request body"))
		return
	}
	defer r.Body.Close()

	action, err := soap.GetAction(body)
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to parse SOAP action"))
		return
	}

	log.Printf("Imaging service action: %s", action)

	// Authentication required
	if err := s.validateAuth(body); err != nil {
		log.Printf("Authentication failed for %s: %v", action, err)
		s.sendFault(w, soap.NewNotAuthorizedFault())
		return
	}

	var response interface{}
	switch action {
	case "GetImagingSettings":
		var req imaging.GetImagingSettingsRequest
		if err := xml.Unmarshal(body, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		response = s.imagingService.GetImagingSettings(req.VideoSourceToken)
	case "SetImagingSettings":
		var req imaging.SetImagingSettingsRequest
		if err := xml.Unmarshal(body, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.imagingService.SetImagingSettings(req.VideoSourceToken, req.ImagingSettings); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &imaging.SetImagingSettingsResponse{}
	case "GetOptions":
		var req imaging.GetOptionsRequest
		if err := xml.Unmarshal(body, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		response = s.imagingService.GetOptions(req.VideoSourceToken)
	default:
		s.sendFault(w, soap.NewActionFailedFault(fmt.Sprintf("Unknown action: %s", action)))
		return
	}

	s.sendResponse(w, response)
}

// sendResponse sends a SOAP response
func (s *Server) sendResponse(w http.ResponseWriter, response interface{}) {
	data, err := soap.MarshalEnvelope(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// sendFault sends a SOAP fault
func (s *Server) sendFault(w http.ResponseWriter, fault *soap.Fault) {
	data, err := soap.MarshalFault(fault)
	if err != nil {
		log.Printf("Failed to marshal fault: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(data)
}

// handleRootService handles requests to root path from non-compliant ONVIF clients
// Routes requests based on SOAP action to appropriate service handler
func (s *Server) handleRootService(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests for SOAP
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only handle root path exactly, not sub-paths
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to read request body"))
		return
	}
	defer r.Body.Close()

	action, err := soap.GetAction(body)
	if err != nil {
		s.sendFault(w, soap.NewActionFailedFault("Failed to parse SOAP action"))
		return
	}

	log.Printf("Root service request with action: %s (non-compliant client)", action)

	// Route based on SOAP action to appropriate service
	switch action {
	// Device service actions
	case "GetDeviceInformation", "GetSystemDateAndTime", "GetCapabilities":
		s.routeToDeviceService(w, body, action)
	// Media service actions
	case "GetProfiles", "GetStreamUri", "GetSnapshotUri":
		s.routeToMediaService(w, body, action)
	// PTZ service actions
	case "GetNodes", "GetConfigurations", "ContinuousMove", "Stop", "GotoHomePosition", "GetPresets", "GotoPreset", "AbsoluteMove", "RelativeMove":
		s.routeToPTZService(w, body, action)
	// Imaging service actions
	case "GetImagingSettings", "SetImagingSettings", "GetOptions":
		s.routeToImagingService(w, body, action)
	default:
		log.Printf("Unknown action in root path: %s", action)
		s.sendFault(w, soap.NewActionFailedFault(fmt.Sprintf("Unknown action: %s", action)))
	}
}

// routeToDeviceService routes request to device service handler
func (s *Server) routeToDeviceService(w http.ResponseWriter, body []byte, action string) {
	log.Printf("Routing to Device service: %s", action)

	// GetSystemDateAndTime is exempt from authentication per ONVIF spec
	if action != "GetSystemDateAndTime" {
		if err := s.validateAuth(body); err != nil {
			log.Printf("Authentication failed for %s: %v", action, err)
			s.sendFault(w, soap.NewNotAuthorizedFault())
			return
		}
	}

	var response interface{}
	switch action {
	case "GetDeviceInformation":
		response = s.deviceService.GetDeviceInformation()
	case "GetSystemDateAndTime":
		response = s.deviceService.GetSystemDateAndTime()
	case "GetCapabilities":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req device.GetCapabilitiesRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		response = s.deviceService.GetCapabilities(req.Category)
	}

	s.sendResponse(w, response)
}

// routeToMediaService routes request to media service handler
func (s *Server) routeToMediaService(w http.ResponseWriter, body []byte, action string) {
	log.Printf("Routing to Media service: %s", action)

	// Debug: Log GetProfiles response XML
	debugGetProfiles := (action == "GetProfiles")

	// Authentication required
	if err := s.validateAuth(body); err != nil {
		log.Printf("Authentication failed for %s: %v", action, err)
		s.sendFault(w, soap.NewNotAuthorizedFault())
		return
	}

	var response interface{}
	switch action {
	case "GetProfiles":
		response = s.mediaService.GetProfiles()
		// Debug: Log response XML
		if debugGetProfiles {
			if debugResp, err := soap.MarshalEnvelope(response); err == nil {
				log.Printf("GetProfiles Response XML:\n%s", string(debugResp))
			}
		}
	case "GetStreamUri":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req media.GetStreamUriRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		resp, err := s.mediaService.GetStreamUri(req.ProfileToken)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault(err.Error()))
			return
		}
		response = resp
	case "GetSnapshotUri":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req media.GetSnapshotUriRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		resp, err := s.mediaService.GetSnapshotUri(req.ProfileToken)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault(err.Error()))
			return
		}
		response = resp
	}

	s.sendResponse(w, response)
}

// routeToPTZService routes request to PTZ service handler
func (s *Server) routeToPTZService(w http.ResponseWriter, body []byte, action string) {
	log.Printf("Routing to PTZ service: %s", action)

	// Authentication required
	if err := s.validateAuth(body); err != nil {
		log.Printf("Authentication failed for %s: %v", action, err)
		s.sendFault(w, soap.NewNotAuthorizedFault())
		return
	}

	var response interface{}
	switch action {
	case "GetNodes":
		response = s.ptzService.GetNodes()
	case "GetConfigurations":
		response = s.ptzService.GetConfigurations()
	case "ContinuousMove":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.ContinuousMoveRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.ContinuousMove(req.ProfileToken, req.Velocity); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.ContinuousMoveResponse{}
	case "Stop":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.StopRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.Stop(req.ProfileToken); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.StopResponse{}
	case "GotoHomePosition":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			log.Printf("Failed to extract body content: %v", err)
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.GotoHomePositionRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			log.Printf("Failed to unmarshal GotoHomePosition request: %v", err)
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.GotoHomePosition(req.ProfileToken, req.Speed); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.GotoHomePositionResponse{}
	case "GetPresets":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.GetPresetsRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		resp, err := s.ptzService.GetPresets(req.ProfileToken)
		if err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = resp
	case "GotoPreset":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.GotoPresetRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.GotoPreset(req.ProfileToken, req.PresetToken, req.Speed); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.GotoPresetResponse{}
	case "AbsoluteMove":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.AbsoluteMoveRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.AbsoluteMove(req.ProfileToken, req.Position, req.Speed); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.AbsoluteMoveResponse{}
	case "RelativeMove":
		bodyContent, err := soap.GetBodyContent(body)
		if err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		var req ptz.RelativeMoveRequest
		if err := xml.Unmarshal(bodyContent, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.ptzService.RelativeMove(req.ProfileToken, req.Translation, req.Speed); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &ptz.RelativeMoveResponse{}
	}

	s.sendResponse(w, response)
}

// routeToImagingService routes request to imaging service handler
func (s *Server) routeToImagingService(w http.ResponseWriter, body []byte, action string) {
	log.Printf("Routing to Imaging service: %s", action)

	// Authentication required
	if err := s.validateAuth(body); err != nil {
		log.Printf("Authentication failed for %s: %v", action, err)
		s.sendFault(w, soap.NewNotAuthorizedFault())
		return
	}

	var response interface{}
	switch action {
	case "GetImagingSettings":
		var req imaging.GetImagingSettingsRequest
		if err := xml.Unmarshal(body, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		response = s.imagingService.GetImagingSettings(req.VideoSourceToken)
	case "SetImagingSettings":
		var req imaging.SetImagingSettingsRequest
		if err := xml.Unmarshal(body, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		if err := s.imagingService.SetImagingSettings(req.VideoSourceToken, req.ImagingSettings); err != nil {
			s.sendFault(w, soap.NewActionFailedFault(err.Error()))
			return
		}
		response = &imaging.SetImagingSettingsResponse{}
	case "GetOptions":
		var req imaging.GetOptionsRequest
		if err := xml.Unmarshal(body, &req); err != nil {
			s.sendFault(w, soap.NewInvalidArgsFault("Invalid request"))
			return
		}
		response = s.imagingService.GetOptions(req.VideoSourceToken)
	}

	s.sendResponse(w, response)
}

// validateAuth validates WS-UsernameToken authentication
func (s *Server) validateAuth(body []byte) error {
	var envelope soap.Envelope
	if err := xml.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("failed to parse SOAP envelope: %w", err)
	}

	if envelope.Header == nil || len(envelope.Header.Content) == 0 {
		return fmt.Errorf("missing security header")
	}

	return soap.ValidateUsernameToken(
		envelope.Header.Content,
		s.config.Server.Auth.Username,
		s.config.Server.Auth.Password,
	)
}

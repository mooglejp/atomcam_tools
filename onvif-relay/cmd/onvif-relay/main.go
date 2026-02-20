package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/camera"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/config"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/discovery"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/mediamtx"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/onvif/soap"
	"github.com/mooglejp/atomcam_tools/onvif-relay/internal/onvif"
)

func main() {
	configPath := flag.String("config", "/config/config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	log.Printf("Loading configuration from %s", *configPath)
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	log.Printf("Configuration loaded: %d cameras, device: %s", len(cfg.Cameras), cfg.Server.DeviceName)

	// Warn about default/weak credentials
	if cfg.Server.Auth.Username == "admin" && cfg.Server.Auth.Password == "admin" {
		log.Printf("WARNING: Using default credentials (admin/admin) - CHANGE THESE IN PRODUCTION!")
	}

	// Create camera registry
	registry, err := camera.NewRegistry(cfg)
	if err != nil {
		log.Fatalf("Failed to create camera registry: %v", err)
	}

	// Configure mediamtx if enabled (API endpoint is set)
	if cfg.Server.Mediamtx.API != "" {
		mtxClient := mediamtx.NewClient(cfg.Server.Mediamtx.API)

		log.Printf("Waiting for mediamtx API at %s", cfg.Server.Mediamtx.API)
		if err := mtxClient.WaitReady(30 * time.Second); err != nil {
			log.Fatalf("mediamtx not ready: %v", err)
		}
		log.Printf("mediamtx API ready")

		log.Printf("Configuring mediamtx paths...")
		if err := configureMediamtxPaths(cfg, mtxClient); err != nil {
			log.Fatalf("Failed to configure mediamtx paths: %v", err)
		}
		log.Printf("All mediamtx paths configured")
	} else {
		log.Printf("mediamtx disabled (api not set); streams must specify rtsp_url directly")
	}

	// Start health checker
	healthChecker := camera.NewHealthChecker(registry, 30*time.Second)
	healthChecker.Start()
	log.Printf("Health checker started")

	// Start WS-Discovery responder if enabled
	var discoveryResponder *discovery.Responder
	if cfg.Server.Discovery {
		baseURL := fmt.Sprintf("http://localhost:%d", cfg.Server.OnvifPort)
		discoveryResponder, err = discovery.NewResponder(cfg.Server.DeviceName, baseURL)
		if err != nil {
			log.Fatalf("Failed to create WS-Discovery responder: %v", err)
		}
		if err := discoveryResponder.Start(); err != nil {
			log.Fatalf("Failed to start WS-Discovery responder: %v", err)
		}
		log.Printf("WS-Discovery responder started")
	}

	// Create and start ONVIF server
	onvifServer := onvif.NewServer(cfg, registry)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("Received shutdown signal")

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Gracefully shutdown HTTP server
		if err := onvifServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		} else {
			log.Printf("HTTP server shut down gracefully")
		}

		// Stop SOAP nonce cleanup goroutine
		soap.StopCleanup()

		// Close camera registry (stops digest transport cleanup goroutines)
		registry.Close()

		// Stop other services
		healthChecker.Stop()
		if discoveryResponder != nil {
			discoveryResponder.Stop()
		}

		log.Printf("Shutdown complete")
		os.Exit(0)
	}()

	log.Printf("Starting ONVIF server on port %d...", cfg.Server.OnvifPort)
	if err := onvifServer.Start(); err != nil {
		log.Fatalf("ONVIF server failed: %v", err)
	}
}

// configureMediamtxPaths configures all camera stream paths in mediamtx
func configureMediamtxPaths(cfg *config.Config, client *mediamtx.Client) error {
	for _, cam := range cfg.Cameras {
		for _, stream := range cam.Streams {
			pathName := fmt.Sprintf("%s/%s", cam.Name, stream.Path)
			log.Printf("Configuring path: %s (codec: %s)", pathName, stream.Codec)

			// Build ffmpeg command
			ffmpegCmd := mediamtx.BuildFFmpegCommand(&cam, &stream, &cfg.Server.Mediamtx)

			// Create path configuration
			pathConfig := mediamtx.PathConfig{
				RunOnDemand:           ffmpegCmd,
				RunOnDemandRestart:    true,
				RunOnDemandCloseAfter: "60s",
			}

			// Configure path with idempotent logic
			if err := client.ConfigurePath(pathName, pathConfig); err != nil {
				return fmt.Errorf("failed to configure path %s: %w", pathName, err)
			}

			log.Printf("Path configured: %s", pathName)
		}
	}

	return nil
}

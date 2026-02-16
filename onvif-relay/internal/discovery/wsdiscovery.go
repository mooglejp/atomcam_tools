package discovery

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"log"
	"net"
	"strings"
	"time"
)

const (
	// WS-Discovery multicast address and port
	multicastAddr = "239.255.255.250:3702"

	// ONVIF namespaces
	wsaNamespace = "http://schemas.xmlsoap.org/ws/2004/08/addressing"
	wsdNamespace = "http://schemas.xmlsoap.org/ws/2005/04/discovery"
	onvifNamespace = "http://www.onvif.org/ver10/network/wsdl"
)

// Responder represents a WS-Discovery responder
type Responder struct {
	deviceUUID   string
	deviceName   string
	xaddrs       string
	scopes       []string
	metadataVersion int
	conn         *net.UDPConn
	ctx          context.Context
	cancel       context.CancelFunc
}

// ProbeMessage represents a WS-Discovery Probe message
type ProbeMessage struct {
	XMLName xml.Name `xml:"Envelope"`
	Header  ProbeHeader
	Body    ProbeBody
}

type ProbeHeader struct {
	MessageID string `xml:"MessageID"`
	Action    string `xml:"Action"`
	To        string `xml:"To"`
}

type ProbeBody struct {
	Probe Probe
}

type Probe struct {
	XMLName xml.Name `xml:"Probe"`
	Types   string   `xml:"Types,omitempty"`
	Scopes  string   `xml:"Scopes,omitempty"`
}

// NewResponder creates a new WS-Discovery responder
func NewResponder(deviceName, baseURL string) (*Responder, error) {
	ctx, cancel := context.WithCancel(context.Background())

	uuid := generateUUID(deviceName)
	xaddrs := baseURL + "/onvif/device_service"

	return &Responder{
		deviceUUID:      uuid,
		deviceName:      deviceName,
		xaddrs:          xaddrs,
		scopes:          []string{
			"onvif://www.onvif.org/type/video_encoder",
			"onvif://www.onvif.org/type/ptz",
			"onvif://www.onvif.org/hardware/" + deviceName,
			"onvif://www.onvif.org/name/" + deviceName,
		},
		metadataVersion: 1,
		ctx:             ctx,
		cancel:          cancel,
	}, nil
}

// Start starts the WS-Discovery responder
func (r *Responder) Start() error {
	// Resolve multicast address
	addr, err := net.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve multicast address: %w", err)
	}

	// Listen on multicast address
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to listen on multicast: %w", err)
	}

	r.conn = conn
	log.Printf("WS-Discovery responder started on %s", multicastAddr)

	go r.run()
	return nil
}

// Stop stops the WS-Discovery responder
func (r *Responder) Stop() {
	r.cancel()
	if r.conn != nil {
		r.conn.Close()
	}
}

// run handles incoming WS-Discovery messages
func (r *Responder) run() {
	buffer := make([]byte, 8192)

	for {
		select {
		case <-r.ctx.Done():
			log.Printf("WS-Discovery responder stopped")
			return
		default:
			// Set read deadline to allow context checking
			r.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			n, remoteAddr, err := r.conn.ReadFromUDP(buffer)
			if err != nil {
				// Check if context was cancelled
				select {
				case <-r.ctx.Done():
					return
				default:
					// Ignore timeout errors, continue listening
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						continue
					}
					log.Printf("WS-Discovery read error: %v", err)
					continue
				}
			}

			// Copy buffer to avoid race condition
			dataCopy := make([]byte, n)
			copy(dataCopy, buffer[:n])

			// Handle probe message
			go r.handleProbe(dataCopy, remoteAddr)
		}
	}
}

// handleProbe handles a Probe message
func (r *Responder) handleProbe(data []byte, remoteAddr *net.UDPAddr) {
	var probe ProbeMessage
	if err := xml.Unmarshal(data, &probe); err != nil {
		// Not a valid probe message, ignore
		return
	}

	// Check if this is a Probe action
	if !strings.Contains(probe.Header.Action, "Probe") {
		return
	}

	log.Printf("Received WS-Discovery Probe from %s", remoteAddr.String())

	// Send ProbeMatch response
	response := r.buildProbeMatch(probe.Header.MessageID)
	r.sendResponse(response, remoteAddr)
}

// buildProbeMatch builds a ProbeMatch response
func (r *Responder) buildProbeMatch(relatesTo string) string {
	// Escape all user/config-provided values to prevent XML injection
	relatesTo = html.EscapeString(relatesTo)
	scopesStr := html.EscapeString(strings.Join(r.scopes, " "))
	xaddrs := html.EscapeString(r.xaddrs)

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope
    xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope"
    xmlns:wsa="%s"
    xmlns:wsd="%s"
    xmlns:dn="%s">
  <SOAP-ENV:Header>
    <wsa:MessageID>uuid:%s</wsa:MessageID>
    <wsa:RelatesTo>%s</wsa:RelatesTo>
    <wsa:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</wsa:To>
    <wsa:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/ProbeMatches</wsa:Action>
  </SOAP-ENV:Header>
  <SOAP-ENV:Body>
    <wsd:ProbeMatches>
      <wsd:ProbeMatch>
        <wsa:EndpointReference>
          <wsa:Address>uuid:%s</wsa:Address>
        </wsa:EndpointReference>
        <wsd:Types>dn:NetworkVideoTransmitter</wsd:Types>
        <wsd:Scopes>%s</wsd:Scopes>
        <wsd:XAddrs>%s</wsd:XAddrs>
        <wsd:MetadataVersion>%d</wsd:MetadataVersion>
      </wsd:ProbeMatch>
    </wsd:ProbeMatches>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`,
		wsaNamespace, wsdNamespace, onvifNamespace,
		generateMessageID(), relatesTo, r.deviceUUID,
		scopesStr, xaddrs, r.metadataVersion)
}

// sendResponse sends a response to the client
func (r *Responder) sendResponse(response string, addr *net.UDPAddr) {
	_, err := r.conn.WriteToUDP([]byte(response), addr)
	if err != nil {
		log.Printf("Failed to send WS-Discovery response: %v", err)
		return
	}

	log.Printf("Sent ProbeMatch to %s", addr.String())
}

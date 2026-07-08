package soap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// Envelope represents a SOAP envelope
type Envelope struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
	Header  *Header  `xml:"Header,omitempty"`
	Body    Body     `xml:"Body"`
}

// Header represents a SOAP header
type Header struct {
	Security *Security `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd Security"`
}

// Body represents a SOAP body
type Body struct {
	Content []byte `xml:",innerxml"`
}

// ParseEnvelope parses a SOAP envelope from XML
func ParseEnvelope(r io.Reader) (*Envelope, error) {
	var env Envelope
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&env); err != nil {
		return nil, fmt.Errorf("failed to decode SOAP envelope: %w", err)
	}
	return &env, nil
}

// MarshalEnvelope marshals a SOAP envelope to XML
func MarshalEnvelope(body interface{}) ([]byte, error) {
	envelope := struct {
		XMLName   xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
		XmlnsTds  string   `xml:"xmlns:tds,attr"`
		XmlnsTrt  string   `xml:"xmlns:trt,attr"`
		XmlnsTptz string   `xml:"xmlns:tptz,attr"`
		XmlnsTimg string   `xml:"xmlns:timg,attr"`
		XmlnsTt   string   `xml:"xmlns:tt,attr"`
		Body      struct {
			Content interface{} `xml:",any"`
		} `xml:"Body"`
	}{
		XmlnsTds:  "http://www.onvif.org/ver10/device/wsdl",
		XmlnsTrt:  "http://www.onvif.org/ver10/media/wsdl",
		XmlnsTptz: "http://www.onvif.org/ver10/ptz/wsdl",
		XmlnsTimg: "http://www.onvif.org/ver10/imaging/wsdl",
		XmlnsTt:   "http://www.onvif.org/ver10/schema",
	}
	envelope.Body.Content = body

	output, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP envelope: %w", err)
	}

	return append([]byte(xml.Header), output...), nil
}

// GetAction extracts the SOAP action from the request body
func GetAction(body []byte) (string, error) {
	// Parse envelope to find the action inside Body element
	decoder := xml.NewDecoder(bytes.NewReader(body))
	inBody := false

	for {
		token, err := decoder.Token()
		if err != nil {
			return "", fmt.Errorf("failed to parse SOAP body: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			// Check if we entered the Body element
			if se.Name.Local == "Body" {
				inBody = true
				continue
			}
			// Return the first element inside Body
			if inBody {
				return se.Name.Local, nil
			}
		}
	}
}

// GetBodyContent extracts the content inside the SOAP Body element
func GetBodyContent(body []byte) ([]byte, error) {
	var env Envelope
	if err := xml.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("failed to parse SOAP envelope: %w", err)
	}
	return env.Body.Content, nil
}

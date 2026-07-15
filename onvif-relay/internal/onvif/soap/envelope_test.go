package soap

import (
	"strings"
	"testing"
)

func TestMarshalEnvelopeUsesCurrentPTZNamespace(t *testing.T) {
	type response struct {
		Value string `xml:"tptz:Value"`
	}

	data, err := MarshalEnvelope(&response{Value: "test"})
	if err != nil {
		t.Fatalf("MarshalEnvelope returned an error: %v", err)
	}
	if !strings.Contains(string(data), `xmlns:tptz="http://www.onvif.org/ver20/ptz/wsdl"`) {
		t.Fatalf("response does not contain the ONVIF ver20 PTZ namespace: %s", data)
	}
}

package soap

import (
	"encoding/xml"
)

// Fault represents a SOAP fault
type Fault struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Fault"`
	Code    FaultCode
	Reason  FaultReason
	Detail  string `xml:"Detail,omitempty"`
}

// FaultCode represents a SOAP fault code
type FaultCode struct {
	Value   string       `xml:"Value"`
	Subcode *FaultSubcode `xml:"Subcode,omitempty"`
}

// FaultSubcode represents a SOAP fault subcode
type FaultSubcode struct {
	Value string `xml:"Value"`
}

// FaultReason represents a SOAP fault reason
type FaultReason struct {
	Text string `xml:"Text"`
}

// Common SOAP fault codes
const (
	FaultCodeSender   = "s:Sender"
	FaultCodeReceiver = "s:Receiver"
)

// Common ONVIF fault subcodes
const (
	SubcodeNotAuthorized  = "ter:NotAuthorized"
	SubcodeInvalidArgVal  = "ter:InvalidArgVal"
	SubcodeActionFailed   = "ter:Action/Failure"
	SubcodeNoSuchService  = "ter:NoSuchService"
	SubcodeOperationProhibited = "ter:OperationProhibited"
)

// NewFault creates a new SOAP fault
func NewFault(code, subcode, reason string) *Fault {
	f := &Fault{
		Code: FaultCode{
			Value: code,
		},
		Reason: FaultReason{
			Text: reason,
		},
	}

	if subcode != "" {
		f.Code.Subcode = &FaultSubcode{
			Value: subcode,
		}
	}

	return f
}

// NewNotAuthorizedFault creates a "Not Authorized" fault
func NewNotAuthorizedFault() *Fault {
	return NewFault(FaultCodeSender, SubcodeNotAuthorized, "The action requested requires authorization and the sender is not authorized")
}

// NewInvalidArgsFault creates an "Invalid Arguments" fault
func NewInvalidArgsFault(message string) *Fault {
	if message == "" {
		message = "Invalid arguments"
	}
	return NewFault(FaultCodeSender, SubcodeInvalidArgVal, message)
}

// NewActionFailedFault creates an "Action Failed" fault
func NewActionFailedFault(message string) *Fault {
	if message == "" {
		message = "The requested action failed"
	}
	return NewFault(FaultCodeReceiver, SubcodeActionFailed, message)
}

// MarshalFault marshals a SOAP fault to XML
func MarshalFault(fault *Fault) ([]byte, error) {
	envelope := struct {
		XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
		Body    struct {
			Fault *Fault
		} `xml:"Body"`
	}{}

	envelope.Body.Fault = fault

	output, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}

package device

import (
	"encoding/xml"
	"time"
)

// GetDeviceInformationRequest represents GetDeviceInformation request
type GetDeviceInformationRequest struct {
	XMLName xml.Name `xml:"GetDeviceInformation"`
}

// GetDeviceInformationResponse represents GetDeviceInformation response
type GetDeviceInformationResponse struct {
	XMLName      xml.Name `xml:"tds:GetDeviceInformationResponse"`
	Manufacturer string   `xml:"tds:Manufacturer"`
	Model        string   `xml:"tds:Model"`
	FirmwareVersion string `xml:"tds:FirmwareVersion"`
	SerialNumber string   `xml:"tds:SerialNumber"`
	HardwareId   string   `xml:"tds:HardwareId"`
}

// GetSystemDateAndTimeRequest represents GetSystemDateAndTime request
type GetSystemDateAndTimeRequest struct {
	XMLName xml.Name `xml:"GetSystemDateAndTime"`
}

// GetSystemDateAndTimeResponse represents GetSystemDateAndTime response
type GetSystemDateAndTimeResponse struct {
	XMLName        xml.Name       `xml:"tds:GetSystemDateAndTimeResponse"`
	SystemDateAndTime SystemDateAndTime `xml:"tds:SystemDateAndTime"`
}

// SystemDateAndTime represents system date and time
type SystemDateAndTime struct {
	DateTimeType string    `xml:"tt:DateTimeType"`
	DaylightSavings bool   `xml:"tt:DaylightSavings"`
	TimeZone     TimeZone  `xml:"tt:TimeZone"`
	UTCDateTime  DateTime  `xml:"tt:UTCDateTime"`
	LocalDateTime DateTime `xml:"tt:LocalDateTime"`
}

// TimeZone represents timezone information
type TimeZone struct {
	TZ string `xml:"tt:TZ"`
}

// DateTime represents date and time
type DateTime struct {
	Time Time `xml:"tt:Time"`
	Date Date `xml:"tt:Date"`
}

// Time represents time
type Time struct {
	Hour   int `xml:"tt:Hour"`
	Minute int `xml:"tt:Minute"`
	Second int `xml:"tt:Second"`
}

// Date represents date
type Date struct {
	Year  int `xml:"tt:Year"`
	Month int `xml:"tt:Month"`
	Day   int `xml:"tt:Day"`
}

// Service represents the Device service
type Service struct {
	deviceName string
	baseURL    string
}

// NewService creates a new Device service
func NewService(deviceName, baseURL string) *Service {
	return &Service{
		deviceName: deviceName,
		baseURL:    baseURL,
	}
}

// GetDeviceInformation handles GetDeviceInformation request
func (s *Service) GetDeviceInformation() *GetDeviceInformationResponse {
	return &GetDeviceInformationResponse{
		Manufacturer:    "AtomCam",
		Model:           s.deviceName,
		FirmwareVersion: "2.5.19",
		SerialNumber:    "ONVIF-RELAY-001",
		HardwareId:      "ONVIF-RELAY",
	}
}

// GetSystemDateAndTime handles GetSystemDateAndTime request
func (s *Service) GetSystemDateAndTime() *GetSystemDateAndTimeResponse {
	now := time.Now()
	utc := now.UTC()

	return &GetSystemDateAndTimeResponse{
		SystemDateAndTime: SystemDateAndTime{
			DateTimeType:    "NTP",
			DaylightSavings: false,
			TimeZone: TimeZone{
				TZ: "UTC",
			},
			UTCDateTime: DateTime{
				Time: Time{
					Hour:   utc.Hour(),
					Minute: utc.Minute(),
					Second: utc.Second(),
				},
				Date: Date{
					Year:  utc.Year(),
					Month: int(utc.Month()),
					Day:   utc.Day(),
				},
			},
			LocalDateTime: DateTime{
				Time: Time{
					Hour:   now.Hour(),
					Minute: now.Minute(),
					Second: now.Second(),
				},
				Date: Date{
					Year:  now.Year(),
					Month: int(now.Month()),
					Day:   now.Day(),
				},
			},
		},
	}
}

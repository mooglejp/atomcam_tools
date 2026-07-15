package config

import "testing"

func TestPTZPresetValidateTracking(t *testing.T) {
	preset := PTZPreset{Name: "Tracking On", Tracking: "ON"}
	if err := preset.Validate(); err != nil {
		t.Fatalf("Validate returned an error: %v", err)
	}
	if preset.Tracking != "on" {
		t.Fatalf("Tracking = %q, want on", preset.Tracking)
	}
}

func TestPTZPresetValidateRejectsInvalidTracking(t *testing.T) {
	preset := PTZPreset{Name: "Tracking", Tracking: "toggle"}
	if err := preset.Validate(); err == nil {
		t.Fatal("Validate returned nil for invalid tracking action")
	}
}

func TestPTZPresetValidateRejectsCombinedActions(t *testing.T) {
	preset := PTZPreset{
		Name:       "Combined",
		Tracking:   "on",
		MQTTBroker: "tcp://localhost:1883",
		MQTTTopic:  "test/topic",
	}
	if err := preset.Validate(); err == nil {
		t.Fatal("Validate returned nil for combined tracking and MQTT actions")
	}
}

func TestPTZPresetValidateRequiresCompleteMQTTConfiguration(t *testing.T) {
	preset := PTZPreset{Name: "MQTT", MQTTTopic: "test/topic"}
	if err := preset.Validate(); err == nil {
		t.Fatal("Validate returned nil without mqtt_broker")
	}
}

package mqtt

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// PublishMessage publishes a message to an MQTT broker
func PublishMessage(broker, topic, message string) error {
	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(fmt.Sprintf("onvif-relay-%d", time.Now().Unix()))
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(false) // One-shot connection

	// Create and connect client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker %s: %w", broker, token.Error())
	}
	defer client.Disconnect(100)

	// Publish message
	token := client.Publish(topic, 0, false, message)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish MQTT message: %w", token.Error())
	}

	return nil
}

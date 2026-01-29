package interfaces

import "context"

// IRDatabase defines database operations for IR code lookup
// This interface allows for testing without a real database connection
type IRDatabase interface {
	// LookupCode retrieves the IR code for a specific AC state
	LookupCode(ctx context.Context, modelID, mode string, temperature int, fanSpeed string) (string, error)

	// LookupOffCode retrieves the IR code to turn off the AC
	LookupOffCode(ctx context.Context, modelID string) (string, error)
}

// MQTTPublisher defines MQTT publishing operations
// This interface allows for testing without a real MQTT broker
type MQTTPublisher interface {
	// Publish sends a message to an MQTT topic
	Publish(topic string, qos byte, retained bool, payload interface{}) error

	// IsConnected returns true if the client is connected to the broker
	IsConnected() bool
}

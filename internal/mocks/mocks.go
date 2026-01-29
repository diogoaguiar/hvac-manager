package mocks

import (
	"context"
	"fmt"
)

// MockDatabase is a mock implementation of interfaces.IRDatabase for testing
type MockDatabase struct {
	// Codes maps state keys to IR codes
	// Format: "modelID:mode:temp:fan" -> IR code
	Codes map[string]string

	// OffCodes maps model IDs to off codes
	// Format: "modelID" -> IR code
	OffCodes map[string]string

	// Err forces an error response for testing error handling
	Err error

	// Calls tracks all lookup calls for verification
	Calls []string
}

// LookupCode implements interfaces.IRDatabase
func (m *MockDatabase) LookupCode(ctx context.Context, modelID, mode string, temperature int, fanSpeed string) (string, error) {
	key := fmt.Sprintf("%s:%s:%d:%s", modelID, mode, temperature, fanSpeed)
	m.Calls = append(m.Calls, key)

	if m.Err != nil {
		return "", m.Err
	}

	if code, ok := m.Codes[key]; ok {
		return code, nil
	}

	return "", fmt.Errorf("code not found for %s", key)
}

// LookupOffCode implements interfaces.IRDatabase
func (m *MockDatabase) LookupOffCode(ctx context.Context, modelID string) (string, error) {
	m.Calls = append(m.Calls, fmt.Sprintf("%s:off", modelID))

	if m.Err != nil {
		return "", m.Err
	}

	if code, ok := m.OffCodes[modelID]; ok {
		return code, nil
	}

	return "", fmt.Errorf("off code not found for model %s", modelID)
}

// MockMQTT is a mock implementation of interfaces.MQTTPublisher for testing
type MockMQTT struct {
	// Published tracks all publish calls
	Published []PublishCall

	// Err forces an error response for testing error handling
	Err error

	// Connected simulates connection state
	Connected bool
}

// PublishCall records the details of a Publish call
type PublishCall struct {
	Topic    string
	QoS      byte
	Retained bool
	Payload  interface{}
}

// Publish implements interfaces.MQTTPublisher
func (m *MockMQTT) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	m.Published = append(m.Published, PublishCall{
		Topic:    topic,
		QoS:      qos,
		Retained: retained,
		Payload:  payload,
	})

	return m.Err
}

// IsConnected implements interfaces.MQTTPublisher
func (m *MockMQTT) IsConnected() bool {
	return m.Connected
}

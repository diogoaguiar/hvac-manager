//go:build integration
// +build integration

package mqtt

import (
	"sync"
	"testing"
	"time"
)

const testBroker = "tcp://localhost:1884" // Port from docker-compose.test.yml

func TestMQTTClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create client
	client, err := NewClient(Config{
		Broker:   testBroker,
		ClientID: "test-client-integration",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Connect
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to broker: %v\nMake sure test broker is running: docker-compose -f docker-compose.test.yml up -d", err)
	}
	defer client.Disconnect()

	// Verify connection
	if !client.IsConnected() {
		t.Error("Client should be connected")
	}

	// Test publish and subscribe
	testTopic := "test/integration/topic"
	testPayload := "hello from integration test"

	var receivedPayload string
	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe first
	err = client.Subscribe(testTopic, 0, func(topic string, payload []byte) {
		receivedPayload = string(payload)
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Give subscription time to register
	time.Sleep(100 * time.Millisecond)

	// Publish message
	err = client.Publish(testTopic, 0, false, testPayload)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait for message with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if receivedPayload != testPayload {
			t.Errorf("Received payload = %q, want %q", receivedPayload, testPayload)
		}
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestMQTTClient_PublishQoS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewClient(Config{
		Broker:   testBroker,
		ClientID: "test-qos",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Test different QoS levels
	qosLevels := []byte{0, 1, 2}
	for _, qos := range qosLevels {
		t.Run("QoS"+string(rune(qos+'0')), func(t *testing.T) {
			topic := "test/qos/level"
			payload := "test payload"

			err := client.Publish(topic, qos, false, payload)
			if err != nil {
				t.Errorf("Publish with QoS %d failed: %v", qos, err)
			}
		})
	}
}

func TestMQTTClient_RetainedMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewClient(Config{
		Broker:   testBroker,
		ClientID: "test-retained",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	testTopic := "test/retained/message"
	testPayload := "retained message"

	// Publish retained message
	err = client.Publish(testTopic, 1, true, testPayload)
	if err != nil {
		t.Fatalf("Failed to publish retained message: %v", err)
	}

	// Disconnect and create new client
	client.Disconnect()

	client2, err := NewClient(Config{
		Broker:   testBroker,
		ClientID: "test-retained-subscriber",
	})
	if err != nil {
		t.Fatalf("Failed to create second client: %v", err)
	}

	if err := client2.Connect(); err != nil {
		t.Fatalf("Failed to connect second client: %v", err)
	}
	defer client2.Disconnect()

	var receivedPayload string
	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe - should immediately receive retained message
	err = client2.Subscribe(testTopic, 1, func(topic string, payload []byte) {
		receivedPayload = string(payload)
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Wait for retained message
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if receivedPayload != testPayload {
			t.Errorf("Retained payload = %q, want %q", receivedPayload, testPayload)
		}
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for retained message")
	}

	// Clean up: remove retained message
	client2.Publish(testTopic, 1, true, "")
}

func TestMQTTClient_WildcardSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewClient(Config{
		Broker:   testBroker,
		ClientID: "test-wildcard",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	var receivedTopics []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Subscribe to wildcard
	err = client.Subscribe("test/wildcard/+", 0, func(topic string, payload []byte) {
		mu.Lock()
		receivedTopics = append(receivedTopics, topic)
		mu.Unlock()
		wg.Done()
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Publish to multiple topics
	topics := []string{
		"test/wildcard/one",
		"test/wildcard/two",
		"test/wildcard/three",
	}

	wg.Add(len(topics))
	for _, topic := range topics {
		if err := client.Publish(topic, 0, false, "test"); err != nil {
			t.Errorf("Failed to publish to %s: %v", topic, err)
		}
	}

	// Wait for all messages
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mu.Lock()
		if len(receivedTopics) != len(topics) {
			t.Errorf("Received %d topics, want %d", len(receivedTopics), len(topics))
		}
		mu.Unlock()
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for wildcard messages")
	}
}

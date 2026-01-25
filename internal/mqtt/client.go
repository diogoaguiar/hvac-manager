package mqtt

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client wraps the Paho MQTT client with our application logic
type Client struct {
	client   mqtt.Client
	clientID string
}

// Config holds MQTT connection configuration
type Config struct {
	Broker   string // e.g., "tcp://localhost:1883"
	ClientID string
	Username string
	Password string
}

// MessageHandler is a callback function for incoming MQTT messages
type MessageHandler func(topic string, payload []byte)

// NewClient creates a new MQTT client with the given configuration
func NewClient(config Config) (*Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetClientID(config.ClientID)

	if config.Username != "" {
		opts.SetUsername(config.Username)
		opts.SetPassword(config.Password)
	}

	// Configure connection parameters
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(5 * time.Second)

	// Connection handlers
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("MQTT: Connected to broker")
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("MQTT: Connection lost: %v", err)
	})

	opts.SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
		log.Println("MQTT: Reconnecting...")
	})

	client := mqtt.NewClient(opts)

	return &Client{
		client:   client,
		clientID: config.ClientID,
	}, nil
}

// Connect establishes connection to the MQTT broker
func (c *Client) Connect() error {
	token := c.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("connection timeout")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	return nil
}

// Disconnect closes the connection to the MQTT broker
func (c *Client) Disconnect() {
	c.client.Disconnect(250)
	log.Println("MQTT: Disconnected from broker")
}

// Publish sends a message to a topic
func (c *Client) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	token := c.client.Publish(topic, qos, retained, payload)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}
	return nil
}

// Subscribe subscribes to a topic with a message handler
func (c *Client) Subscribe(topic string, qos byte, handler MessageHandler) error {
	callback := func(client mqtt.Client, msg mqtt.Message) {
		handler(msg.Topic(), msg.Payload())
	}

	token := c.client.Subscribe(topic, qos, callback)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("subscribe timeout")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	log.Printf("MQTT: Subscribed to %s", topic)
	return nil
}

// IsConnected returns true if the client is connected to the broker
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

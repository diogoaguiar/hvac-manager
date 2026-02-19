package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/diogoaguiar/hvac-manager/internal/database"
	"github.com/diogoaguiar/hvac-manager/internal/homeassistant"
	"github.com/diogoaguiar/hvac-manager/internal/integration"
	"github.com/diogoaguiar/hvac-manager/internal/logger"
	"github.com/diogoaguiar/hvac-manager/internal/mqtt"
	"github.com/diogoaguiar/hvac-manager/internal/state"
)

const (
	defaultBroker   = "tcp://localhost:1883"
	defaultDeviceID = "living_room"
)

// loadEnv loads environment variables from .env file if it exists
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		// .env file is optional, so don't error if it doesn't exist
		return
	}
	defer file.Close()

	fmt.Println("📄 Loading .env file...")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first = sign
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := parts[1] // Don't trim the value yet

		// Only remove surrounding quotes if they match
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		} else {
			// If no quotes, trim whitespace
			value = strings.TrimSpace(value)
		}

		// Set environment variable (overwrite if from .env)
		os.Setenv(key, value)
		if key == "MQTT_PASSWORD" {
			fmt.Printf("   ✓ %s=***\n", key)
		} else {
			fmt.Printf("   ✓ %s=%s\n", key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Warn("Error reading .env file: %v", err)
	}
}

func main() {
	// Load .env file if it exists
	loadEnv()

	// Initialize logger from environment (LOG_LEVEL)
	logger.InitFromEnv()

	fmt.Println("🌡️  HVAC Manager - E2E POC")
	fmt.Println("=" + string(make([]byte, 50)) + "=")

	// Configuration from environment or defaults
	broker := getEnv("MQTT_BROKER", defaultBroker)
	deviceID := getEnv("DEVICE_ID", defaultDeviceID)
	username := getEnv("MQTT_USERNAME", "")
	password := getEnv("MQTT_PASSWORD", "")

	logger.Info("Config: Broker=%s, Device=%s", broker, deviceID)

	// Database configuration
	dbPath := getEnv("DATABASE_PATH", "./hvac.db")
	modelID := getEnv("AC_MODEL_ID", "1109")
	irBlasterID := getEnv("IR_BLASTER_ID", "ir-blaster")

	// Initialize database
	logger.Info("📦 Initializing IR code database...")
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run schema migrations
	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Load SmartIR IR codes for configured model
	smartirFile := filepath.Join("docs", "smartir", "reference", fmt.Sprintf("%s_tuya.json", modelID))
	if err := db.LoadFromJSON(ctx, modelID, smartirFile); err != nil {
		log.Fatalf("Failed to load IR codes from %s: %v", smartirFile, err)
	}
	logger.Info("✅ Database ready with model: %s", modelID)

	// Create MQTT client
	mqttConfig := mqtt.Config{
		Broker:   broker,
		ClientID: fmt.Sprintf("hvac-manager-%s", deviceID),
		Username: username,
		Password: password,
	}

	client, err := mqtt.NewClient(mqttConfig)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}

	// Connect to MQTT broker
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Initialize AC state
	acState := state.NewACState()
	logger.Info("Initial state: %s", acState.String())

	// Publish Home Assistant MQTT Discovery
	if err := publishDiscovery(client, deviceID); err != nil {
		log.Fatalf("Failed to publish discovery: %v", err)
	}

	// Publish availability (online)
	availTopic := fmt.Sprintf("homeassistant/climate/%s/availability", deviceID)
	if err := client.Publish(availTopic, 1, true, "online"); err != nil {
		logger.Warn("Failed to publish availability: %v", err)
	}

	// Publish initial state
	if err := publishState(client, deviceID, acState); err != nil {
		logger.Warn("Failed to publish initial state: %v", err)
	}

	// Subscribe to command topic
	cmdTopic := fmt.Sprintf("homeassistant/climate/%s/set", deviceID)
	if err := client.Subscribe(cmdTopic, 1, func(topic string, payload []byte) {
		handleCommand(client, db, modelID, irBlasterID, deviceID, acState, payload)
	}); err != nil {
		log.Fatalf("Failed to subscribe to command topic: %v", err)
	}

	fmt.Println("\n✅ Phase 4 Integration Active!")
	fmt.Printf("   📡 MQTT Broker: %s\n", broker)
	fmt.Printf("   🏠 HA Device ID: %s\n", deviceID)
	fmt.Printf("   🎛️  AC Model: %s\n", modelID)
	fmt.Printf("   📡 IR Blaster: %s\n", irBlasterID)
	fmt.Printf("   📥 Listening on: %s\n", cmdTopic)
	fmt.Printf("   📤 State topic: homeassistant/climate/%s/state\n", deviceID)
	fmt.Println("📡 IR codes will be transmitted via Zigbee2MQTT")
	fmt.Println("   Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("\n🛑 Shutting down...")
	// Publish offline status
	if err := client.Publish(availTopic, 1, true, "offline"); err != nil {
		logger.Warn("Failed to publish offline status: %v", err)
	}
}

// publishDiscovery publishes the Home Assistant MQTT Discovery payload
func publishDiscovery(client *mqtt.Client, deviceID string) error {
	discovery := homeassistant.NewClimateDiscovery(deviceID, "Living Room AC")
	payload, err := discovery.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal discovery: %w", err)
	}

	topic := discovery.ConfigTopic(deviceID)
	if err := client.Publish(topic, 2, true, payload); err != nil {
		return fmt.Errorf("failed to publish discovery: %w", err)
	}

	logger.Info("✅ Published discovery to: %s", topic)
	return nil
}

// publishState publishes the current AC state to Home Assistant
func publishState(client *mqtt.Client, deviceID string, acState *state.ACState) error {
	haState := &homeassistant.ClimateState{
		Temperature: acState.Temperature,
		Mode:        acState.Mode,
		FanMode:     acState.FanMode,
	}

	payload, err := homeassistant.StateToJSON(haState)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	topic := fmt.Sprintf("homeassistant/climate/%s/state", deviceID)

	// Log what we're about to publish
	logger.Info("📤 Publishing HA state to: %s", topic)
	logger.Info("   JSON: %s", string(payload))

	if err := client.Publish(topic, 0, true, payload); err != nil {
		return fmt.Errorf("failed to publish state: %w", err)
	}

	return nil
}

// handleCommand processes commands received from Home Assistant
func handleCommand(client *mqtt.Client, db *database.DB, modelID, irBlasterID, deviceID string, acState *state.ACState, payload []byte) {
	fmt.Println("\n" + strings.Repeat("─", 60))
	logger.Info("📥 Received command: %s", string(payload))

	// Try to parse as JSON first
	cmd, err := homeassistant.ParseCommand(payload)
	if err != nil {
		// If JSON parsing fails, treat as plain text (temperature or mode value)
		payloadStr := string(payload)
		logger.Debug("📋 Plain text command: %s", payloadStr)

		// Try to parse as temperature (numeric)
		if temp, err := strconv.ParseFloat(payloadStr, 64); err == nil {
			// Save original state before modification
			originalTemp := acState.Temperature

			if err := acState.SetTemperature(temp); err != nil {
				logger.Error("Invalid temperature: %v", err)
				return
			}
			logger.Info("🌡️  Temperature set to: %.1f°C", temp)

			// Try to send IR code
			ctx := context.Background()
			if err := integration.SendIRCode(ctx, db, client, modelID, irBlasterID, acState); err != nil {
				logger.Error("❌ Failed to send IR code: %v", err)
				// Revert to original state on failure
				acState.Temperature = originalTemp
				logger.Warn("⏪ Reverted temperature to: %.1f°C", originalTemp)
			} else {
				logger.Info("✅ IR code sent successfully")
			}

			// Always publish actual state (new if success, reverted if failure)
			if err := publishState(client, deviceID, acState); err != nil {
				logger.Error("Failed to publish state: %v", err)
			}
			fmt.Println(strings.Repeat("─", 60))
			return
		}

		// Otherwise treat as mode or fan mode
		originalMode := acState.Mode
		if err := acState.SetMode(payloadStr); err == nil {
			logger.Info("🔄 Mode set to: %s", payloadStr)

			// Try to send IR code
			ctx := context.Background()
			if err := integration.SendIRCode(ctx, db, client, modelID, irBlasterID, acState); err != nil {
				logger.Error("❌ Failed to send IR code: %v", err)
				// Revert to original state on failure
				acState.Mode = originalMode
				logger.Warn("⏪ Reverted mode to: %s", originalMode)
			} else {
				logger.Info("✅ IR code sent successfully")
			}

			// Always publish actual state (new if success, reverted if failure)
			if err := publishState(client, deviceID, acState); err != nil {
				logger.Error("Failed to publish state: %v", err)
			}
			fmt.Println(strings.Repeat("─", 60))
			return
		}

		originalFan := acState.FanMode
		if err := acState.SetFanMode(payloadStr); err == nil {
			logger.Info("💨 Fan mode set to: %s", payloadStr)

			// Try to send IR code
			ctx := context.Background()
			if err := integration.SendIRCode(ctx, db, client, modelID, irBlasterID, acState); err != nil {
				logger.Error("❌ Failed to send IR code: %v", err)
				// Revert to original state on failure
				acState.FanMode = originalFan
				logger.Warn("⏪ Reverted fan mode to: %s", originalFan)
			} else {
				logger.Info("✅ IR code sent successfully")
			}

			// Always publish actual state (new if success, reverted if failure)
			if err := publishState(client, deviceID, acState); err != nil {
				logger.Error("Failed to publish state: %v", err)
			}
			fmt.Println(strings.Repeat("─", 60))
			return
		}

		logger.Error("Could not parse command as JSON or plain text: %s", payloadStr)
		return
	}

	// Pretty print the command for visibility
	cmdJSON, _ := json.MarshalIndent(cmd, "", "  ")
	logger.Debug("📋 Parsed command:\n%s", string(cmdJSON))

	// Save original state before any modifications
	originalState := state.ACState{
		Temperature: acState.Temperature,
		Mode:        acState.Mode,
		FanMode:     acState.FanMode,
	}

	// Apply changes to state
	stateChanged := false

	if cmd.Temperature != nil {
		if err := acState.SetTemperature(*cmd.Temperature); err != nil {
			logger.Error("Invalid temperature: %v", err)
			return
		}
		stateChanged = true
		logger.Info("🌡️  Temperature set to: %.1f°C", *cmd.Temperature)
	}

	if cmd.Mode != nil {
		if err := acState.SetMode(*cmd.Mode); err != nil {
			logger.Error("Invalid mode: %v", err)
			return
		}
		stateChanged = true
		logger.Info("🔄 Mode set to: %s", *cmd.Mode)
	}

	if cmd.FanMode != nil {
		if err := acState.SetFanMode(*cmd.FanMode); err != nil {
			logger.Error("Invalid fan mode: %v", err)
			return
		}
		stateChanged = true
		logger.Info("💨 Fan mode set to: %s", *cmd.FanMode)
	}

	if !stateChanged {
		logger.Warn("⚠️  No valid state changes in command")
		return
	}

	// Try to send IR code to IR blaster
	ctx := context.Background()
	if err := integration.SendIRCode(ctx, db, client, modelID, irBlasterID, acState); err != nil {
		logger.Error("❌ Failed to send IR code: %v", err)
		// Revert to original state on failure
		acState.Temperature = originalState.Temperature
		acState.Mode = originalState.Mode
		acState.FanMode = originalState.FanMode
		logger.Warn("⏪ Reverted to original state: %s", originalState.String())
	} else {
		logger.Info("✅ IR code sent successfully")
	}

	// Always publish actual state (new if success, reverted if failure)
	if err := publishState(client, deviceID, acState); err != nil {
		logger.Error("Failed to publish state: %v", err)
	}

	fmt.Println(strings.Repeat("─", 60))
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

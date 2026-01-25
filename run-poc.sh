#!/bin/bash
# Quick setup script for HVAC Manager E2E POC

set -e

echo "🌡️  HVAC Manager - E2E POC Setup"
echo "================================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.25+ first."
    exit 1
fi

echo "✅ Go is installed: $(go version)"
echo ""

# Check for MQTT_BROKER environment variable
if [ -z "$MQTT_BROKER" ]; then
    echo "⚠️  MQTT_BROKER environment variable is not set!"
    echo ""
    echo "Please configure your MQTT broker connection:"
    echo ""
    echo "  export MQTT_BROKER=\"tcp://YOUR_HA_BROKER_IP:1883\""
    echo ""
    echo "If your broker requires authentication:"
    echo "  export MQTT_USERNAME=\"your_username\""
    echo "  export MQTT_PASSWORD=\"your_password\""
    echo ""
    echo "Optional: customize device ID"
    echo "  export DEVICE_ID=\"living_room\""
    echo ""
    echo "Examples:"
    echo "  # Local HA with Mosquitto add-on"
    echo "  export MQTT_BROKER=\"tcp://192.168.1.100:1883\""
    echo "  export MQTT_USERNAME=\"mqtt_user\""
    echo "  export MQTT_PASSWORD=\"your_password\""
    echo ""
    echo "  # No authentication"
    echo "  export MQTT_BROKER=\"tcp://homeassistant.local:1883\""
    echo ""
    exit 1
fi

echo "✅ MQTT Configuration:"
echo "   Broker: $MQTT_BROKER"
echo "   Device ID: ${DEVICE_ID:-living_room (default)}"
if [ -n "$MQTT_USERNAME" ]; then
    echo "   Username: $MQTT_USERNAME"
    echo "   Password: ${MQTT_PASSWORD:+***configured***}"
fi
echo ""

# Download dependencies
echo "📦 Downloading Go dependencies..."
go mod download
echo "✅ Dependencies ready"
echo ""

# Build
echo "🔨 Building application..."
go build -o /tmp/hvac-manager-poc cmd/main.go
echo "✅ Build successful"
echo ""

echo "🚀 Starting HVAC Manager POC..."
echo "   Press Ctrl+C to stop"
echo ""

# Run
exec go run cmd/main.go

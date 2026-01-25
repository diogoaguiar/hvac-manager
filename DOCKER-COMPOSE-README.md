# Docker Compose Setup (Optional)

This docker-compose file is **optional** and only needed if you want to test without an existing MQTT broker.

## When to Use This

- Testing without Home Assistant
- Development and debugging
- Isolated testing environment
- Learning/experimentation

## When NOT to Use This

- You already have Home Assistant with MQTT configured ✅ **Use your existing broker instead!**
- You want to test actual HA integration

## Usage

### Start Standalone MQTT Broker

```bash
docker-compose up -d
```

This starts:
- Mosquitto MQTT broker on `localhost:1883`
- MQTT Explorer web UI on `localhost:4000`

### Configure POC to Use It

```bash
export MQTT_BROKER="tcp://localhost:1883"
unset MQTT_USERNAME MQTT_PASSWORD
go run cmd/main.go
```

### Test with MQTT Client

```bash
# Subscribe to all topics
mosquitto_sub -h localhost -t "#" -v

# Publish test command
mosquitto_pub -h localhost -t "homeassistant/climate/living_room/set" \
  -m '{"temperature": 21, "mode": "cool"}'
```

## MQTT Explorer Web UI

Access at `http://localhost:4000` for a visual MQTT client.

## Stop Services

```bash
docker-compose down
```

---

**💡 Tip:** For actual Home Assistant integration, connect to your existing MQTT broker using environment variables. See [poc-setup.md](docs/poc-setup.md) for details.

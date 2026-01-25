# API & MQTT Documentation

This document specifies all MQTT topics, message formats, and Home Assistant integration details for Go-Climate-Sidecar.

## Table of Contents

1. [MQTT Overview](#mqtt-overview)
2. [Topic Structure](#topic-structure)
3. [Message Formats](#message-formats)
4. [Home Assistant Integration](#home-assistant-integration)
5. [Zigbee2MQTT Integration](#zigbee2mqtt-integration)
6. [Error Handling](#error-handling)
7. [Examples](#examples)

---

## MQTT Overview

### Broker Configuration

- **Protocol:** MQTT 3.1.1 or 5.0
- **Default Port:** 1883 (unencrypted), 8883 (TLS)
- **QoS Levels:**
  - QoS 0: State updates (best effort)
  - QoS 1: Commands (at least once)
  - QoS 2: Discovery payloads (exactly once)

### Client Information

```
Client ID: go-climate-sidecar-{device_id}
Clean Session: true
Keep Alive: 60 seconds
Will Topic: homeassistant/climate/{device_id}/availability
Will Payload: offline
Will QoS: 1
Will Retain: true
```

---

## Topic Structure

### Naming Convention

```
{domain}/{component}/{device_id}/{action}
```

**Examples:**
- `homeassistant/climate/living_room/set` - Command from HA
- `homeassistant/climate/living_room/state` - State update to HA
- `homeassistant/climate/living_room/config` - Discovery payload
- `zigbee2mqtt/ir_blaster_01/set` - Command to IR blaster

### Topic Categories

#### 1. Home Assistant Topics

| Topic | Direction | QoS | Retain | Purpose |
|-------|-----------|-----|--------|---------|
| `homeassistant/climate/{device}/config` | Publish | 2 | Yes | Discovery payload |
| `homeassistant/climate/{device}/set` | Subscribe | 1 | No | Commands from HA |
| `homeassistant/climate/{device}/state` | Publish | 0 | Yes | State updates to HA |
| `homeassistant/climate/{device}/availability` | Publish | 1 | Yes | Online/offline status |

#### 2. Zigbee2MQTT Topics

| Topic | Direction | QoS | Retain | Purpose |
|-------|-----------|-----|--------|---------|
| `zigbee2mqtt/{device}/set` | Publish | 1 | No | IR code transmission |
| `zigbee2mqtt/{device}/get` | Publish | 0 | No | Query device state |
| `zigbee2mqtt/bridge/devices` | Subscribe | 0 | No | Device discovery |
| `zigbee2mqtt/{device}` | Subscribe | 0 | No | Device state updates |

---

## Message Formats

### Home Assistant Command Messages

Topic: `homeassistant/climate/{device_id}/set`

#### Set Temperature

```json
{
  "temperature": 21.0
}
```

#### Set Mode

```json
{
  "mode": "cool"
}
```

Valid modes: `off`, `cool`, `heat`, `dry`, `fan_only`, `auto`

#### Set Fan Mode

```json
{
  "fan_mode": "auto"
}
```

Valid fan modes: `auto`, `quiet`, `1`, `2`, `3`, `4`, `5`

#### Set Swing Mode

```json
{
  "swing_mode": "vertical"
}
```

Valid swing modes: `off`, `vertical`, `horizontal`, `both`

#### Combined Command

```json
{
  "mode": "cool",
  "temperature": 21.0,
  "fan_mode": "auto",
  "swing_mode": "off"
}
```

### Home Assistant State Messages

Topic: `homeassistant/climate/{device_id}/state`

```json
{
  "mode": "cool",
  "temperature": 21.0,
  "current_temperature": 24.5,
  "fan_mode": "auto",
  "swing_mode": "off",
  "action": "cooling"
}
```

**Fields:**
- `mode` (string, required): Current mode
- `temperature` (number, required): Target temperature
- `current_temperature` (number, optional): Measured room temperature (if sensor available)
- `fan_mode` (string, required): Current fan speed
- `swing_mode` (string, required): Current swing setting
- `action` (string, optional): Current action (`idle`, `cooling`, `heating`, `drying`, `fan`)

### Zigbee2MQTT Command Messages

Topic: `zigbee2mqtt/{device_id}/set`

#### Send IR Code

```json
{
  "ir_code_to_send": "C/MgAQUBFAUUBRQFFAUUBRQFFAUUBRQFFAUUBRQFFAUUBRQF..."
}
```

**Fields:**
- `ir_code_to_send` (string, required): Tuya-compressed Base64 IR code

#### Query Device State

Topic: `zigbee2mqtt/{device_id}/get`

```json
{
  "state": ""
}
```

### Availability Messages

Topic: `homeassistant/climate/{device_id}/availability`

```json
"online"
```

or

```json
"offline"
```

Simple string payload (not JSON).

---

## Home Assistant Integration

### MQTT Discovery

On startup, HVAC Manager publishes a discovery payload that creates a Climate entity in Home Assistant.

#### Discovery Topic

```
homeassistant/climate/{device_id}/config
```

#### Discovery Payload

```json
{
  "name": "Living Room AC",
  "unique_id": "daikin_ac_living_room",
  "object_id": "living_room_ac",
  "device": {
    "identifiers": ["daikin_ac_living_room"],
    "name": "Living Room Daikin AC",
    "model": "Daikin FTXM35",
    "manufacturer": "Daikin",
    "sw_version": "1.0.0"
  },
  "mode_command_topic": "homeassistant/climate/living_room/set",
  "mode_state_topic": "homeassistant/climate/living_room/state",
  "mode_state_template": "{{ value_json.mode }}",
  "temperature_command_topic": "homeassistant/climate/living_room/set",
  "temperature_state_topic": "homeassistant/climate/living_room/state",
  "temperature_state_template": "{{ value_json.temperature }}",
  "current_temperature_topic": "homeassistant/climate/living_room/state",
  "current_temperature_template": "{{ value_json.current_temperature }}",
  "fan_mode_command_topic": "homeassistant/climate/living_room/set",
  "fan_mode_state_topic": "homeassistant/climate/living_room/state",
  "fan_mode_state_template": "{{ value_json.fan_mode }}",
  "swing_mode_command_topic": "homeassistant/climate/living_room/set",
  "swing_mode_state_topic": "homeassistant/climate/living_room/state",
  "swing_mode_state_template": "{{ value_json.swing_mode }}",
  "action_topic": "homeassistant/climate/living_room/state",
  "action_template": "{{ value_json.action }}",
  "availability_topic": "homeassistant/climate/living_room/availability",
  "modes": ["off", "cool", "heat", "dry", "fan_only", "auto"],
  "fan_modes": ["auto", "quiet", "1", "2", "3", "4", "5"],
  "swing_modes": ["off", "vertical", "horizontal", "both"],
  "min_temp": 16,
  "max_temp": 32,
  "temp_step": 1,
  "precision": 1.0,
  "temperature_unit": "C",
  "qos": 0,
  "retain": true,
  "optimistic": false
}
```

#### Discovery Payload Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name in HA UI |
| `unique_id` | string | Unique identifier (must be unique across HA) |
| `object_id` | string | Entity ID suffix (e.g., `climate.living_room_ac`) |
| `device` | object | Device information for HA device registry |
| `*_command_topic` | string | Topic for receiving commands |
| `*_state_topic` | string | Topic for publishing state |
| `*_template` | string | Jinja2 template to extract value from JSON |
| `modes` | array | Available HVAC modes |
| `fan_modes` | array | Available fan speeds |
| `swing_modes` | array | Available swing positions |
| `min_temp` | number | Minimum temperature (°C) |
| `max_temp` | number | Maximum temperature (°C) |
| `temp_step` | number | Temperature adjustment step |
| `precision` | number | Temperature precision (decimals) |
| `temperature_unit` | string | `C` or `F` |

### Discovery Best Practices

1. **Publish on startup** with retain flag
2. **Republish on reconnection** to ensure HA has current config
3. **Use unique IDs** to prevent conflicts
4. **Include device info** for better HA organization
5. **Set appropriate QoS** (QoS 2 for discovery)

### Removing Entities

To remove an entity from Home Assistant, publish an empty payload to the config topic:

```bash
mosquitto_pub -t 'homeassistant/climate/living_room/config' \
  -n -r
```

---

## Zigbee2MQTT Integration

### Device Requirements

Compatible IR blasters:
- **ZS06** (Tuya Universal IR Remote)
- Other Tuya-compatible Zigbee IR blasters

### Zigbee2MQTT Configuration

Ensure your `configuration.yaml` includes:

```yaml
devices:
  '0x00124b001234abcd':
    friendly_name: 'ir_blaster_01'
    retain: false
```

### IR Code Format

Zigbee2MQTT expects Tuya-compressed IR codes in the `ir_code_to_send` field:

```json
{
  "ir_code_to_send": "C/MgAQUBFAUUBRQFFAUUBRQFFAU..."
}
```

**Format:** Base64-encoded Tuya-compressed pulse timings, prefixed with `C/` or `M/`

See [protocols.md](protocols.md) for encoding details.

### Device State Updates

Zigbee2MQTT publishes device state to `zigbee2mqtt/{device_id}`:

```json
{
  "linkquality": 120,
  "last_seen": "2026-01-24T15:30:00Z"
}
```

Note: Most IR blasters don't provide feedback on transmission success. Monitor Zigbee2MQTT logs for errors.

---

## Error Handling

### Invalid Commands

If HVAC Manager receives an invalid command, it publishes an error state:

```json
{
  "mode": "off",
  "error": "Invalid temperature: must be between 16 and 32"
}
```

**Error Field:** Temporary, cleared on next successful command

### Connection Failures

#### MQTT Broker Unreachable

- Service logs error and retries with exponential backoff
- Commands queued in memory (max 100)
- Availability set to `offline`

#### IR Blaster Unreachable

- Detect via Zigbee2MQTT availability topic
- Queue commands (max 10)
- Retry on device return

### Rate Limiting

To prevent command spam:
- **Max rate:** 1 command per second per device
- **Burst:** Up to 3 commands in quick succession
- **Behavior:** Excess commands are dropped with warning log

---

## Examples

### Complete Flow Example

#### 1. Service Startup

```bash
# Service connects and publishes discovery
→ MQTT: homeassistant/climate/living_room/config
  Payload: {Discovery JSON}
  Retain: true, QoS: 2

# Service publishes availability
→ MQTT: homeassistant/climate/living_room/availability
  Payload: "online"
  Retain: true, QoS: 1

# Service publishes initial state
→ MQTT: homeassistant/climate/living_room/state
  Payload: {"mode": "off", "temperature": 24, "fan_mode": "auto", "swing_mode": "off"}
  Retain: true, QoS: 0
```

#### 2. User Sets Temperature

```bash
# User changes temp to 21°C in HA UI
← MQTT: homeassistant/climate/living_room/set
  Payload: {"temperature": 21, "mode": "cool"}
  QoS: 1

# Service generates IR code and sends to blaster
→ MQTT: zigbee2mqtt/ir_blaster_01/set
  Payload: {"ir_code_to_send": "C/MgAQUBFAU..."}
  QoS: 1

# Service updates state in HA
→ MQTT: homeassistant/climate/living_room/state
  Payload: {"mode": "cool", "temperature": 21, "fan_mode": "auto", "swing_mode": "off", "action": "cooling"}
  Retain: true, QoS: 0
```

#### 3. Service Shutdown

```bash
# Service publishes offline status (via MQTT will)
→ MQTT: homeassistant/climate/living_room/availability
  Payload: "offline"
  Retain: true, QoS: 1
```

### Testing with Mosquitto Tools

#### Subscribe to All Topics

```bash
mosquitto_sub -h localhost -t '#' -v
```

#### Simulate HA Command

```bash
mosquitto_pub -h localhost \
  -t 'homeassistant/climate/living_room/set' \
  -m '{"temperature": 21, "mode": "cool"}' \
  -q 1
```

#### Check Device State

```bash
mosquitto_sub -h localhost \
  -t 'homeassistant/climate/living_room/state' \
  -C 1
```

#### Remove Discovery

```bash
mosquitto_pub -h localhost \
  -t 'homeassistant/climate/living_room/config' \
  -n -r
```

### MQTT Explorer

For visual debugging, use MQTT Explorer:

1. Connect to broker
2. Navigate to `homeassistant/climate/{device}`
3. Publish test messages
4. Observe state changes
5. Inspect retained messages

---

## Security Considerations

### Authentication

Configure MQTT broker with username/password:

```bash
# Create password file
mosquitto_passwd -c /etc/mosquitto/passwd go-climate

# Configure mosquitto.conf
allow_anonymous false
password_file /etc/mosquitto/passwd
```

Service configuration:

```bash
MQTT_USERNAME=go-climate
MQTT_PASSWORD=secure-password
```

### TLS Encryption

For secure communication:

```bash
# Generate certificates (self-signed for testing)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout mqtt-key.pem -out mqtt-cert.pem

# Configure mosquitto.conf
listener 8883
cafile /etc/mosquitto/ca.crt
certfile /etc/mosquitto/mqtt-cert.pem
keyfile /etc/mosquitto/mqtt-key.pem
```

Service configuration:

```bash
MQTT_BROKER=ssl://homeassistant.local:8883
MQTT_TLS_CERT=/path/to/mqtt-cert.pem
MQTT_TLS_KEY=/path/to/mqtt-key.pem
```

### Access Control

Use MQTT ACLs to restrict topic access:

```
# /etc/mosquitto/acl
user go-climate
topic readwrite homeassistant/climate/#
topic write zigbee2mqtt/+/set
topic read zigbee2mqtt/bridge/devices
```

---

## Future API Enhancements

### Planned Features (Phase 4+)

1. **HTTP REST API** for direct control (bypass MQTT)
2. **WebSocket API** for real-time updates
3. **GraphQL API** for complex queries
4. **Configuration API** for runtime settings
5. **Metrics API** for Prometheus integration

### Webhook Support

Allow external triggers:

```bash
POST /api/webhook/climate/living_room
Content-Type: application/json

{
  "temperature": 21,
  "mode": "cool"
}
```

---

## References

- [MQTT Protocol](https://mqtt.org/mqtt-specification/)
- [Home Assistant MQTT Climate](https://www.home-assistant.io/integrations/climate.mqtt/)
- [Home Assistant MQTT Discovery](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery)
- [Zigbee2MQTT](https://www.zigbee2mqtt.io/)
- [Eclipse Paho MQTT](https://www.eclipse.org/paho/)
- [Mosquitto Documentation](https://mosquitto.org/documentation/)

---

## API Changelog

### v1.0.0 (Current - Phase 1)

- Initial MQTT connectivity
- Basic command publish/subscribe
- Manual IR code transmission

### v1.1.0 (Planned - Phase 2)

- Dynamic IR code generation
- Tuya encoding support

### v1.2.0 (Planned - Phase 3)

- Daikin protocol generator
- State management

### v2.0.0 (Planned - Phase 4)

- MQTT Discovery integration
- Full HA Climate entity support
- Error handling and recovery

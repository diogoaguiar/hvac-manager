# 🎉 Ready to Test with Home Assistant!

Your HVAC Manager E2E POC is now configured to connect to your existing MQTT broker and integrate with Home Assistant.

## Quick Setup

### Option 1: Simple (Recommended)

```bash
# Set your broker details
export MQTT_BROKER="tcp://YOUR_HA_IP:1883"
export MQTT_USERNAME="your_mqtt_user"
export MQTT_PASSWORD="your_password"

# Run the helper script
./run-poc.sh
```

### Option 2: Using .env file

```bash
# Create config file
cp .env.example .env

# Edit with your details
nano .env

# Load and run
set -a; source .env; set +a
go run cmd/main.go
```

## What to Expect

1. **Application starts** and connects to your MQTT broker
2. **Discovery payload published** to `homeassistant/climate/living_room/config`
3. **Climate entity appears** in Home Assistant automatically!
   - Check: Settings → Devices & Services → MQTT
   - Look for "Living Room AC"
4. **Control from HA** - adjust temperature, mode, fan speed
5. **See logs** - commands received, state updated, IR codes logged (not sent)

## Testing the Integration

### In Home Assistant:

1. Go to **Settings → Devices & Services → MQTT**
2. Find "Living Room AC" device
3. Click on it to see the climate entity
4. Try changing:
   - Temperature (16-30°C)
   - Mode (off, cool, heat, dry, fan_only, auto)
   - Fan speed (auto, low, medium, high)

### In the POC Terminal:

You'll see logs like:
```
📥 Received command: {"temperature":21,"mode":"cool"}
🌡️  Temperature set to: 21.0°C
🔄 Mode set to: cool
💡 [POC] Would look up IR code for: Mode: cool, Temp: 21.0°C, Fan: auto
💡 [POC] Would publish to: zigbee2mqtt/ir-blaster/set
📤 Published state: Mode: cool, Temp: 21.0°C, Fan: auto, Power: true
```

## Architecture You're Testing

```
┌─────────────────┐
│  Home Assistant │
│   (Your Setup)  │
└────────┬────────┘
         │ MQTT Commands
         ↓
┌─────────────────┐
│  MQTT Broker    │
│ (Your Existing) │
└────────┬────────┘
         │
         ↓
┌─────────────────┐
│  HVAC Manager   │ ← You are here!
│    (This POC)   │
└─────────────────┘
         │
         ↓ [Logs only]
┌─────────────────┐
│ Zigbee2MQTT +   │ ← Phase 4
│   IR Blaster    │
└─────────────────┘
```

## Common Issues

### Climate entity not appearing?

1. Check POC logs show "Published discovery"
2. Verify MQTT integration is enabled in HA
3. Restart POC to re-publish discovery
4. Check HA MQTT logs: Settings → System → Logs (filter "mqtt")

### Connection refused?

1. Verify MQTT_BROKER is correct: `echo $MQTT_BROKER`
2. Test with mosquitto_pub:
   ```bash
   mosquitto_pub -h YOUR_BROKER_IP -t "test" -m "hello"
   ```
3. Check firewall allows port 1883

### Authentication errors?

1. Verify credentials are correct
2. Check user permissions in MQTT broker
3. Create dedicated user in HA if needed

## What's Next?

This POC validates the complete HA integration flow. Phase 4 will add:

- ✅ IR code database lookup (already implemented in Phase 2)
- 🔜 Connect state changes to database queries
- 🔜 Publish actual IR codes to Zigbee2MQTT
- 🔜 End-to-end IR transmission

## Need Help?

- 📖 [Full POC Setup Guide](docs/poc-setup.md)
- 📖 [Architecture Documentation](docs/architecture.md)
- 📖 [API Reference](docs/api.md)
- 🤖 [AI Agent Context](AGENTS.md)

---

**Enjoy testing! 🎉**

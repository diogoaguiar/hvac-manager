# Debugging Guide

## Log Levels

Control logging with the `LOG_LEVEL` environment variable:

```bash
LOG_LEVEL=DEBUG make run   # Detailed: IR codes, DB queries, MQTT
LOG_LEVEL=INFO make run    # Normal: Commands, state changes (default)
LOG_LEVEL=WARN make run    # Warnings only
LOG_LEVEL=ERROR make run   # Errors only
```

### DEBUG Output Example
```
🔍 [DEBUG] SendIRCode called for state: Mode: heat, Temp: 22.0°C, Fan: low, Power: true
🔍 [DEBUG] Looking up IR code: model=1109 mode=heat temp=22 fan=low
🔍 [DEBUG] Found IR code (length: 352 bytes)
🔍 [DEBUG] IR code: JgBYAAABJ5IVEBUQFRAVNxURFTcVNxU3FRAVEBURFRA...
🔍 [DEBUG] Publishing to topic: zigbee2mqtt/ir-blaster/set
🔍 [DEBUG] MQTT publish successful
ℹ️  [INFO] 📡 IR code sent to ir-blaster
```

## Common Issues

### AC Doesn't Respond

**Check logs with DEBUG:**
```bash
LOG_LEVEL=DEBUG make run
```

Look for:
- ✅ "IR code sent to ir-blaster" - Code transmitted
- ❌ "No IR code found" - Database missing code for temp/mode/fan combo
- ❌ "MQTT publish failed" - Connection issue

**Solutions:**
- Check available codes: `./bin/hvac-manager-db query --model 1109`
- Verify MQTT broker connection
- Check IR blaster battery/power

### Temperature Changes Don't Work

**Fixed!** Temperature-only changes now send IR codes (previously a bug).

If still not working:
- Enable DEBUG logging
- Verify "SendIRCode called" appears in logs
- Check database has code for that temperature

### Database Lookup Failures

DEBUG shows helpful context:
```
⚠️  [WARN] No IR code found in DB for model=1109 mode=heat temp=22 fan=quiet
🔍 [DEBUG] Found 48 codes for model=1109 mode=heat (any temp/fan)
```

This tells you:
- Mode exists (48 codes found)
- Fan mode "quiet" doesn't exist for this model
- Valid fan modes: auto, low, medium, high

## MQTT Delivery

The client confirms delivery with QoS 1:
```
🔍 [DEBUG] MQTT publish successful: topic=zigbee2mqtt/ir-blaster/set
```

**Note:** This confirms MQTT delivery, not physical IR transmission.

## Quick Debugging

```bash
# See all IR transmissions
make run 2>&1 | grep "IR code sent"

# See all errors
make run 2>&1 | grep ERROR

# Save logs for analysis
LOG_LEVEL=DEBUG make run 2>&1 | tee debug.log
```

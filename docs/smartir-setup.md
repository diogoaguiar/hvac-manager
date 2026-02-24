# SmartIR Setup Guide

Complete guide for controlling HVAC units via SmartIR + Zigbee2MQTT with Tuya IR blasters,
replacing the hvac-manager Go microservice.

## Prerequisites

- **Home Assistant** (2024.1+) running in Docker
- **HACS** installed ([instructions](https://hacs.xyz/docs/use/download/download/))
- **Zigbee2MQTT** running with a paired **Tuya IR blaster** (ZS06 or UFO-R11)
- **`smartir-tuya-converter`** tool — [install instructions](https://github.com/diogoaguiar/smartir-tuya-converter)

---

## 1. Install SmartIR

### Via HACS (recommended)

1. Open HA → HACS → Integrations
2. Click the three dots (top right) → **Custom repositories**
3. Add `litinoveweedle/SmartIR` as category **Integration**
4. Search for "SmartIR" in HACS and install it
5. Add to `configuration.yaml`:

```yaml
smartir:
```

6. Restart Home Assistant

### Verify installation

After restart, check **Settings → Devices & Services → Integrations** — SmartIR should
appear. You can also check the logs at **Settings → System → Logs** for any SmartIR errors.

---

## 2. Convert Device Codes

SmartIR's community device code files use Broadlink encoding. Tuya IR blasters (via
Zigbee2MQTT) need the codes in Tuya Raw format. The `smartir-tuya-converter` tool handles
this conversion.

### Find your device code file

Browse the [SmartIR climate codes](https://github.com/litinoveweedle/SmartIR/tree/master/codes/climate)
and download the JSON file for your AC model. For Daikin:

- **1109.json** — Daikin BRC4C158 (3 fan speeds, no swing)
- **1116.json** — Daikin FCQ100KAVEA (3 fan levels + swing modes)

### Convert

```bash
# Install the converter (one-time)
go install github.com/diogoaguiar/smartir-tuya-converter/cmd/smartir-tuya-converter@latest

# Convert Broadlink → Tuya
smartir-tuya-converter 1109.json 1109.json
```

This overwrites the file in-place with Tuya-encoded codes. You can also write to a new file:

```bash
smartir-tuya-converter 1109.json 1109_tuya.json
```

### Verify the output

Open the converted file and confirm:
- `"commandsEncoding": "Raw"` (was `"Base64"`)
- `"supportedController": "MQTT"` (was `"Broadlink"`)
- IR code values look different from the original (Tuya format, still base64 but different data)

### Place the file

Copy the converted JSON to the SmartIR codes directory inside your HA config:

```bash
# From the HA host
cp 1109.json <ha-config>/custom_components/smartir/codes/climate/1109.json
```

If running HA in Docker, the config directory is typically the mounted volume:

```bash
cp 1109.json /path/to/ha-config/custom_components/smartir/codes/climate/1109.json
```

---

## 3. Configure Climate Entity

Add a climate platform entry to your `configuration.yaml`:

```yaml
climate:
  - platform: smartir
    name: Living Room AC
    unique_id: living_room_ac
    device_code: 1109
    controller_data: zigbee2mqtt/<ir-blaster-name>/set/ir_code_to_send
```

### Finding the correct MQTT topic

The `controller_data` value is the MQTT topic that Zigbee2MQTT uses for your IR blaster.
To find it:

1. Open **Zigbee2MQTT UI** → Devices
2. Find your IR blaster (e.g., "Living Room IR Blaster")
3. Note the **friendly name** — this is the `<ir-blaster-name>` in the topic
4. The full topic is: `zigbee2mqtt/<friendly-name>/set/ir_code_to_send`

For example, if your blaster's friendly name is `ir_blaster_living_room`:

```yaml
controller_data: zigbee2mqtt/ir_blaster_living_room/set/ir_code_to_send
```

### Full example with optional sensors

```yaml
climate:
  - platform: smartir
    name: Living Room AC
    unique_id: living_room_ac
    device_code: 1109
    controller_data: zigbee2mqtt/ir_blaster_living_room/set/ir_code_to_send
    temperature_sensor: sensor.living_room_temperature
    humidity_sensor: sensor.living_room_humidity
    power_sensor: binary_sensor.ac_power
```

After editing, restart Home Assistant.

---

## 4. Test

### Verify the entity exists

1. Go to **Settings → Devices & Services → Entities**
2. Search for your climate entity (e.g., "Living Room AC")
3. It should show as a `climate` entity with the configured name

### Test from Developer Tools

1. Go to **Developer Tools → Services**
2. Select `climate.set_hvac_mode`
3. Target your climate entity
4. Set mode to `cool` and call the service
5. Watch the Zigbee2MQTT logs to confirm the IR code was published

### Test from the UI

1. Go to your dashboard
2. Add a **Climate** card pointing to your entity
3. Try changing:
   - **Mode**: off, cool, heat, dry, fan_only
   - **Temperature**: 16–32°C
   - **Fan speed**: low, medium, high
4. Verify the IR blaster LED blinks and the AC responds

### Verify via Zigbee2MQTT logs

Check the Zigbee2MQTT logs for IR transmission messages:

```
Zigbee2MQTT:info  Publishing 'set' 'ir_code_to_send' to 'ir_blaster_living_room'
```

You can also verify from the Zigbee2MQTT UI → your IR blaster device → the "ir_code_to_send"
expose should show the last sent code.

---

## 5. Adding More IR Blasters / HVAC Units

### Pair a new Tuya IR blaster

1. Put the IR blaster in pairing mode (usually hold button 5+ seconds until LED blinks fast)
2. In Zigbee2MQTT UI → click **Permit join**
3. Wait for the device to appear
4. Rename it with a descriptive friendly name (e.g., `ir_blaster_bedroom`)

### Find or create device codes

1. Check the [SmartIR codes repo](https://github.com/litinoveweedle/SmartIR/tree/master/codes/climate)
   for your AC model
2. If your exact model isn't listed, try a similar model from the same manufacturer —
   many AC units in a product line share the same IR protocol
3. If no match exists, you can create a custom code file by learning IR codes
   (see [SmartIR docs](https://github.com/litinoveweedle/SmartIR#creating-your-own-device-files))

### Convert and add the new codes

```bash
smartir-tuya-converter <new-model>.json <new-model>.json
cp <new-model>.json <ha-config>/custom_components/smartir/codes/climate/
```

### Multi-device configuration

```yaml
climate:
  - platform: smartir
    name: Living Room AC
    unique_id: living_room_ac
    device_code: 1109
    controller_data: zigbee2mqtt/ir_blaster_living_room/set/ir_code_to_send
    temperature_sensor: sensor.living_room_temperature

  - platform: smartir
    name: Bedroom AC
    unique_id: bedroom_ac
    device_code: 1116
    controller_data: zigbee2mqtt/ir_blaster_bedroom/set/ir_code_to_send
    temperature_sensor: sensor.bedroom_temperature
```

---

## 6. Optional Enhancements

### Power sensor for state feedback

IR is one-way — SmartIR can't know if the AC actually turned on. A smart plug with power
monitoring solves this:

1. Plug the AC into a Zigbee smart plug with power metering (e.g., Sonoff S31)
2. Create a template binary sensor that detects when the AC is drawing power:

```yaml
template:
  - binary_sensor:
      - name: AC Power
        unique_id: ac_power
        state: "{{ states('sensor.ac_plug_power') | float > 10 }}"
        device_class: power
```

3. Add it to the SmartIR config:

```yaml
climate:
  - platform: smartir
    ...
    power_sensor: binary_sensor.ac_power
```

Now SmartIR will sync its state with the physical AC — if someone uses the remote to turn
it on, SmartIR updates accordingly.

### Temperature and humidity sensors

Any Zigbee temperature/humidity sensor (e.g., Aqara WSDCGQ11LM) placed in the room can
provide current readings in the climate card:

```yaml
climate:
  - platform: smartir
    ...
    temperature_sensor: sensor.living_room_temperature
    humidity_sensor: sensor.living_room_humidity
```

### Automations

Example: turn off AC when nobody is home:

```yaml
automation:
  - alias: Turn off AC when away
    trigger:
      - platform: state
        entity_id: group.family
        to: not_home
        for: "00:10:00"
    action:
      - service: climate.turn_off
        target:
          entity_id: climate.living_room_ac
```

Example: set AC to cool mode at 24°C when arriving home in summer:

```yaml
automation:
  - alias: Cool house on arrival
    trigger:
      - platform: state
        entity_id: group.family
        to: home
    condition:
      - condition: numeric_state
        entity_id: sensor.living_room_temperature
        above: 26
    action:
      - service: climate.set_temperature
        target:
          entity_id: climate.living_room_ac
        data:
          hvac_mode: cool
          temperature: 24
```

---

## 7. Troubleshooting

### Climate entity doesn't appear

- Check HA logs for SmartIR errors: **Settings → System → Logs**, filter for "smartir"
- Verify the device code file exists at
  `<ha-config>/custom_components/smartir/codes/climate/<code>.json`
- Ensure `smartir:` is in `configuration.yaml` (the integration line, not just the climate)
- Restart HA after any configuration changes

### AC doesn't respond to commands

- **Check MQTT topic**: Verify the `controller_data` topic matches your IR blaster's
  Zigbee2MQTT friendly name exactly (case-sensitive)
- **Check IR blaster range**: The IR blaster needs line-of-sight to the AC's IR receiver.
  Try placing it closer or repositioning
- **Verify encoding**: Open the device code JSON and confirm `"commandsEncoding": "Raw"`.
  If it says `"Base64"`, you need to run the converter
- **Test IR blaster directly**: In the Zigbee2MQTT UI, go to your IR blaster device and
  manually send a test IR code through the "ir_code_to_send" expose

### Wrong MQTT topic format

The topic must be exactly:
```
zigbee2mqtt/<friendly-name>/set/ir_code_to_send
```

Common mistakes:
- Missing `/set/` — must include it
- Wrong friendly name — check Zigbee2MQTT UI for the exact name
- Spaces in friendly name — Zigbee2MQTT replaces spaces with underscores by default

### IR codes don't work (AC doesn't react)

- The device code file may not match your specific AC model. Try other device codes for
  the same manufacturer
- Verify the Tuya conversion was successful — the converter should output codes that look
  different from the Broadlink input
- Try the "off" command first — it's the simplest and most likely to work

### State gets out of sync

This happens when someone uses the physical remote. Solutions:
- Add a `power_sensor` (see Section 6) to detect actual AC state
- Use SmartIR as the only way to control the AC

### Checking HA logs

Filter logs for SmartIR and MQTT:

```yaml
# In configuration.yaml, add for debugging:
logger:
  default: warning
  logs:
    custom_components.smartir: debug
    homeassistant.components.mqtt: debug
```

Remove the debug logging once everything is working to avoid log spam.

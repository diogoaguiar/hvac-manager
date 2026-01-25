# IR Code Import Guide

This document explains how to import SmartIR IR code files into the HVAC Manager database. The system automatically handles format conversion, so you can directly import files from the SmartIR project.

## Quick Start

### Import all reference files
```bash
make db-load
```

### Import from a custom directory
```bash
make db-import DIR=/path/to/smartir/files
```

### Import a single model file
```bash
make db-import-model FILE=docs/smartir/reference/1109.json
```

## About SmartIR Files

### Source
SmartIR is a Home Assistant project that maintains a catalog of IR codes for various climate devices:
- Repository: https://github.com/smartHomeHub/SmartIR
- IR codes are located in the `codes/climate/` directory
- Each file represents one AC model with all its control combinations

### Supported Formats

The loader automatically detects and handles both formats:

1. **Broadlink format** (original SmartIR files)
   - `commandsEncoding: "Base64"`
   - `supportedController: "Broadlink"`
   - IR codes use Broadlink's proprietary encoding
   - Files typically named: `1109.json`

2. **Tuya format** (pre-converted files)
   - `commandsEncoding: "Raw"`
   - `supportedController: "MQTT"`
   - IR codes use Tuya LZ-compressed format
   - Files typically named: `1109_tuya.json`

**When importing Broadlink files, the loader automatically converts them to Tuya format during database insertion.**

## Adding New Models

1. Find the model in SmartIR repository:
   ```bash
   # Browse available models
   https://github.com/smartHomeHub/SmartIR/tree/master/codes/climate
   ```

2. Download the JSON file for your AC model

3. Place it in your reference directory (or use directly)

4. Import it:
   ```bash
   # Using db-import-model
   make db-import-model FILE=/path/to/new-model.json
   
   # Or place in docs/smartir/reference/ and run
   make db-load
   ```

The database will automatically:
- Detect the file format
- Convert Broadlink codes to Tuya if needed
- Store the IR codes with proper indexing
- Handle duplicate imports gracefully (UPSERT)

## Hardware Compatibility

This project uses Tuya-compatible Zigbee IR blasters:
- **Model:** ZS06 Universal IR Remote
- **Reference:** https://www.aliexpress.com/item/1005003878194474.html
- **Integration:** Via Zigbee2MQTT

The Tuya format is required for these devices to correctly transmit IR signals.

## Conversion Details

For technical details about the Broadlink-to-Tuya conversion algorithm, see:
- [internal/database/README.md](../internal/database/README.md) - Implementation overview
- [internal/database/tuya_codec.go](../internal/database/tuya_codec.go) - Compression algorithm
- [internal/database/converter.go](../internal/database/converter.go) - Conversion logic

### Technical Background

**IR Signal Format:**
- Raw IR: sequence of mark (ON) and space (OFF) durations in microseconds
- Broadlink: uses ~32.84 µs ticks (269/8192 ms), with 0x00 escape for 16-bit values
- Tuya: 16-bit little-endian durations, LZ-compressed for efficiency

**Compression Algorithm:**
- Sliding window: 8KB (2^13 bytes)
- Max match length: 265 bytes
- Two token types: literal blocks (≤32 bytes) and distance blocks (length-distance pairs)
- Level 2 strategy: use best match found (balance compression vs speed)

### Technical References
- Tuya IR codec specification: https://gist.github.com/mildsunrise/1d576669b63a260d2cff35fda63ec0b5
- Broadlink → Tuya converter examples: https://gist.github.com/svyatogor/7839d00303998a9fa37eb48494dd680f

## Validation

Test that conversion works correctly:
```bash
make db-test-conversion
```

This runs the test suite that validates conversion against known-good reference files.

## Legacy Python Converter

The repository includes a Python reference implementation at [docs/smartir/reference/broadlink_to_tuya.py](smartir/reference/broadlink_to_tuya.py). This is kept for validation purposes but is no longer required for normal operation, as the Go implementation handles conversion automatically.

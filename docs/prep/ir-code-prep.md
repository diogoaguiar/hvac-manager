# IR Code Preparation (SmartIR → Tuya MQTT)

This document explains how we prepare IR code files for use with this project. It covers where the codes come from, why we convert them, the devices involved, the key technical concepts, and the exact workflow to regenerate or extend the Tuya-formatted code files.

## Context
- Original code files contain IR codes for selected Daikin HVAC models that were verified compatible with our use case. Only the relevant models were kept locally.
- These model code files were retrieved from the SmartIR project for Home Assistant.
  - SmartIR repository: https://github.com/smartHomeHub/SmartIR
  - The repo maintains a large catalog of climate IR code maps (by manufacturer / model) under `codes/`.
- Our project does not depend on the Home Assistant integration itself; we only re-use the IR code maps as a source dataset.

## Why convert to Tuya format?
- We use generic Tuya-compatible IR blasters (AliExpress model "ZS06" / universal IR remote) to transmit the HVAC commands.
  - Example reference listing (stable item id): https://www.aliexpress.com/item/1005003878194474.html
- SmartIR stores many codes in Broadlink’s raw format (base64-encoded). Tuya IR blasters, however, expect a Tuya "raw" timing stream, typically compressed with Tuya’s LZ-style codec.
- Therefore, we convert the SmartIR Broadlink-encoded payloads into Tuya-compressed payloads that the IR blaster will publish/transmit via MQTT.

## Converter script and provenance
- The Python converter used in this repo is at:
  - `docs/smartir/reference/broadlink_to_tuya.py`
- It is based on community work that documents Tuya’s IR compression format and provides a converter from Broadlink timing payloads to Tuya’s stream:
  - "Tuya IR codec" reference: https://gist.github.com/mildsunrise/1d576669b63a260d2cff35fda63ec0b5
  - Converter example (Broadlink → Tuya): https://gist.github.com/svyatogor/7839d00303998a9fa37eb48494dd680f
  - Alternate implementation with attribution: https://gist.github.com/vills/590c154b377ac50acab079328e4ddaf9

## Technical background (concise)
- IR basics: A raw IR command is a sequence of alternating mark (carrier ON) and space (carrier OFF) durations, measured in microseconds. Protocols like NEC or Daikin define how those durations encode bits, headers, and repeats. Here we handle raw durations only, not protocol semantics.
- Broadlink representation: Durations are stored in vendor-specific "ticks". The commonly used conversion is ~32.84 µs per tick (constant `BRDLNK_UNIT = 269/8192 ms ≈ 0.03284 ms`). Broadlink payloads store durations as 8-bit values, with an escape `0x00` prefix for 16-bit big‑endian extended durations.
- Tuya "raw" + compression: Tuya expects a byte stream of 16-bit little‑endian durations. To reduce size, Tuya applies an LZ‑style codec with:
  - Sliding window `W = 2^13 = 8192` bytes
  - Maximum match length `L = 256 + 9 = 265`
  - Two token types: literal blocks (up to 32 bytes), and length‑distance matches (distance in 13 bits; length encoded as `(len-2)`, with an extra byte for `len ≥ 10`).
- Practical constraints:
  - Durations must fit in 16‑bit unsigned (`< 65535`) when packed for Tuya.
  - Carrier frequency (e.g., 38 kHz) is implicit; only durations are transmitted.

## Conversion pipeline (how it works)
1. Take a SmartIR code file (Broadlink base64 IR payloads) and parse the Broadlink payload into a list of durations (in ticks), expanding any `0x00` extended tokens.
2. Convert Broadlink ticks to timing quanta using `ceil(value / BRDLNK_UNIT)`, producing a list of unsigned 16‑bit durations.
3. Pack each duration as little‑endian 16‑bit and concatenate into a raw byte stream.
4. Compress the stream with the Tuya LZ‑style codec (literal blocks + length‑distance tokens).
5. Base64‑encode the compressed Tuya payload for transport/publishing (e.g., via MQTT).

## Running the converter
- The script rewrites the `commands` in a SmartIR JSON to Tuya format and sets metadata (`supportedController=MQTT`, `commandsEncoding=Raw`).
- Example usage from the repo root:

```bash
python3 docs/smartir/reference/broadlink_to_tuya.py docs/smartir/reference/1116.json > docs/smartir/reference/1116_tuya.json
```

- You can run the same for other code files you added under `docs/smartir/reference/`.

## Files in this repo related to codes
- SmartIR-derived inputs and Tuya outputs live under:
  - `docs/smartir/reference/1109.json` and `docs/smartir/reference/1109_tuya.json`
  - `docs/smartir/reference/1116.json` and `docs/smartir/reference/1116_tuya.json`
  - Converter: `docs/smartir/reference/broadlink_to_tuya.py`

## Extending this preparation step
- Add more SmartIR code files: Place relevant model files (from https://github.com/smartHomeHub/SmartIR/tree/master/codes) into `docs/smartir/reference/`.
- Run the converter to generate the corresponding `_tuya.json` files.
- Keep this step scoped as a preparatory/asset‑generation process.

## References
- SmartIR (source of code maps): https://github.com/smartHomeHub/SmartIR
- Tuya IR codec notes and Python reference: https://gist.github.com/mildsunrise/1d576669b63a260d2cff35fda63ec0b5
- Broadlink → Tuya converter examples:
  - https://gist.github.com/svyatogor/7839d00303998a9fa37eb48494dd680f
  - https://gist.github.com/vills/590c154b377ac50acab079328e4ddaf9
- ZS06 IR blaster (example listing): https://www.aliexpress.com/item/1005003878194474.html

# Protocol Documentation

> **Note:** This document serves as **reference material** for understanding IR protocols and Tuya encoding. The current implementation uses **pre-translated IR codes from SmartIR** rather than generating protocols dynamically. This documentation is useful for debugging and understanding the format of codes we're using.

This document provides technical specifications for the Daikin IR protocol and Tuya compression format.

## Table of Contents

1. [Infrared Basics](#infrared-basics)
2. [Daikin Protocol](#daikin-protocol)
3. [Tuya Compression Format](#tuya-compression-format)
4. [Broadlink Format](#broadlink-format)
5. [Conversion Pipeline](#conversion-pipeline)

---

## Infrared Basics

### How IR Remote Controls Work

Infrared (IR) remote controls encode data by modulating an IR LED at a specific carrier frequency (typically 38kHz for consumer electronics). The data is transmitted as a series of **marks** (LED on) and **spaces** (LED off).

**Key Concepts:**
- **Mark:** IR LED is pulsing at carrier frequency (e.g., 38kHz)
- **Space:** IR LED is off (no signal)
- **Carrier Frequency:** Typically 38kHz for HVAC units
- **Protocol:** Defines how marks/spaces encode bits and commands

### Timing Representation

A complete IR command consists of alternating mark/space durations measured in microseconds (µs):

```
[mark1, space1, mark2, space2, mark3, space3, ...]
```

Example (simplified):
```
[9000, 4500, 560, 560, 560, 1690, ...]
 ^^^^   ^^^^   ^^^  ^^^   ^^^  ^^^^
 |      |      |    |     |    |
 |      |      |    |     |    Space for bit "1"
 |      |      |    |     Mark for bit "1"
 |      |      |    Space for bit "0"
 |      |      Mark for bit "0"
 |      Header space
 Header mark
```

### Bit Encoding

Different protocols use different timing patterns to encode bits:

**NEC Protocol:**
- Bit "0": 560µs mark + 560µs space
- Bit "1": 560µs mark + 1690µs space

**Daikin Protocol:**
- Bit "0": 400µs mark + 400µs space
- Bit "1": 400µs mark + 1300µs space

---

## Daikin Protocol

Daikin air conditioners use a proprietary multi-frame protocol. Each command consists of **3 frames** separated by gaps.

### Frame Structure Overview

```
[Frame 1 - Header] --gap-- [Frame 2 - State] --gap-- [Frame 3 - Extended]
     8 bytes                   19 bytes                  19 bytes
```

**Inter-frame gap:** ~29ms (29,000µs) of silence

### Frame 1: Header Frame

**Purpose:** Protocol identification  
**Length:** 8 bytes (fixed)  
**Content:** Always constant

```
Bytes: [0x11, 0xDA, 0x27, 0x00, 0xC5, 0x00, 0x00, 0xD7]
```

**Breakdown:**
- `0x11 0xDA 0x27`: Manufacturer ID (Daikin)
- `0x00 0xC5 0x00 0x00`: Protocol version/type
- `0xD7`: Checksum (sum of previous 7 bytes & 0xFF)

### Frame 2: State Frame

**Purpose:** Main AC state (power, mode, temperature, fan)  
**Length:** 19 bytes  
**Structure:**

```
Byte   Bits      Description
----   ----      -----------
0      xxxx xxxx Header (0x11)
1      xxxx xxxx Manufacturer ID part 1 (0xDA)
2      xxxx xxxx Manufacturer ID part 2 (0x27)
3      xxxx xxxx Command type (0x00)
4      xxxx xxxx Zeros (0x00)
5      ---- ---P Power: 0=Off, 1=On
       ---- -MMM Mode: 0=Fan, 2=Dry, 3=Cool, 4=Heat, 6=Auto
6      xxxx xxxx Temperature: (value * 2) - 10 (e.g., 21°C = 32 = 0x20)
7      xxxx xxxx Zeros (0x00)
8      ---- -FFF Fan Speed: 0x30=Auto, 0x40=Silent, 0x50-0x90=Levels 1-5
       ---- S--- Swing Vertical: 0=Fixed, 1=Swing
9      xxxx xxxx Zeros (0x00)
10     ---- ---S Swing Horizontal: 0=Fixed, 1=Swing
11-12  xxxx xxxx Time: Hour and Minute (often set to 0x00 0x00)
13     xxxx xxxx Zeros (0x00)
14     xxxx xxxx Zeros (0x00)
15     xxxx xxxx Special modes (Powerful, Econo, etc.)
16     xxxx xxxx Zeros (0x00)
17     xxxx xxxx Zeros (0x00)
18     xxxx xxxx Checksum: Sum of bytes 0-17, AND 0xFF
```

**Temperature Encoding:**
```
IR_value = (Celsius * 2) - 10

Examples:
  16°C → (16 * 2) - 10 = 22 = 0x16
  21°C → (21 * 2) - 10 = 32 = 0x20
  25°C → (25 * 2) - 10 = 40 = 0x28
  30°C → (30 * 2) - 10 = 50 = 0x32
```

**Mode Values:**
```
0x0 = Fan only
0x2 = Dehumidify (Dry)
0x3 = Cool
0x4 = Heat
0x6 = Auto
```

**Fan Speed Values:**
```
0x30 = Auto
0x40 = Silent/Quiet
0x50 = Level 1
0x60 = Level 2
0x70 = Level 3
0x80 = Level 4
0x90 = Level 5
```

### Frame 3: Extended Frame

**Purpose:** Additional features and final checksum  
**Length:** 19 bytes  
**Structure:** Model-specific; includes timers, advanced modes, and final checksum

### Bit Timing (Physical Layer)

**Leader (Start of Frame):**
- Mark: 3360µs
- Space: 1760µs

**Data Bits:**
- Bit "0": 400µs mark + 400µs space (total: 800µs)
- Bit "1": 400µs mark + 1300µs space (total: 1700µs)

**Bit Order:** LSB first (least significant bit transmitted first)

**Frame Gap:** 29,000µs of silence between frames

### Checksum Calculation

```go
func calculateChecksum(frame []byte) byte {
    sum := 0
    for i := 0; i < len(frame)-1; i++ {
        sum += int(frame[i])
    }
    return byte(sum & 0xFF)
}
```

**Example:**
```
Frame: [0x11, 0xDA, 0x27, 0x00, 0x01, 0x03, 0x20, ...]
Sum = 0x11 + 0xDA + 0x27 + 0x00 + 0x01 + 0x03 + 0x20 + ...
Checksum = sum & 0xFF
```

### Time Field (Bytes 11-12 of Frame 2)

The Daikin remote normally sends the current time (HH:MM) with each command. This causes "rolling codes" where the hex dump changes every minute.

**Our Solution:** Hardcode time to `00:00` (0x00 0x00) for consistent, repeatable signals.

### Complete Transmission Example

**Command:** Turn on AC, Cool mode, 21°C, Auto fan

**Frame 1 (Header):**
```
Hex: 11 DA 27 00 C5 00 00 D7
Bits: [binary representation with mark/space timings]
```

**Frame 2 (State):**
```
Hex: 11 DA 27 00 00 01 03 20 00 30 00 00 00 00 00 00 00 00 [checksum]
     ^^ ^^ ^^          ^^ ^^ ^^    ^^
     |  |  |           |  |  |     |
     |  |  |           |  |  |     Fan: Auto (0x30)
     |  |  |           |  |  Temp: 21°C (0x20)
     |  |  |           |  Mode: Cool (0x03)
     |  |  |           Power: On (0x01)
     |  |  Protocol
     |  Manufacturer ID
     Header
```

**Total Timing Stream:**
```
[3360, 1760, 400, 400, 400, 1300, ...] (Frame 1)
[29000] (gap)
[3360, 1760, 400, 1300, 400, 400, ...] (Frame 2)
[29000] (gap)
[3360, 1760, 400, 1300, 400, 1300, ...] (Frame 3)
```

---

## Tuya Compression Format

Tuya IR blasters expect compressed pulse timing data in a proprietary format. This section documents the compression algorithm and encoding.

### Input Format

Raw pulse timings as 16-bit little-endian unsigned integers (microseconds):

```
[0x90, 0x01, 0x14, 0x05, 0x90, 0x01, 0x90, 0x01, ...]
 ^^^^^ ^^^^^  ^^^^^ ^^^^^
 400µs  276µs  1300µs 400µs
```

### Compression Algorithm

Tuya uses a variant of LZ77 compression optimized for timing data.

**Parameters:**
- **Sliding Window (W):** 8192 bytes (2^13)
- **Maximum Match Length (L):** 265 bytes
- **Minimum Match Length:** 2 bytes

**Token Types:**

#### 1. Literal Block
Encodes raw uncompressed bytes.

```
Format: 0LLLLLLL [L+1 literal bytes]
        ^^^^^^^^
        Bit 7 = 0 indicates literal
        Bits 0-6 = Length - 1 (0-127 → 1-128 bytes)
```

**Example:**
```
0x05 0x11 0xDA 0x27 0x00 0xC5 0x00
^^^^  ^^^^^^^^^^^^^^^^^^^^^^^^
|     6 literal bytes
Length = 5 (means 6 bytes follow)
```

#### 2. Match Token
Encodes a reference to previous data.

```
Format: 1LLLLLLL DDDDDDDD DDDDDDDD [optional extra length byte]
        ^^^^^^^^  ^^^^^^^^^^^^^^^^
        Bit 7 = 1 indicates match
        Bits 0-6 = Length - 2 (0-127 → 2-129 bytes)
        Next 13 bits = Distance (1-8192)
```

**Extended Length:** If length bits = 127 (0x7F), read next byte for additional length:
```
Total Length = 127 + 2 + next_byte = 129 to 384
```

**Example:**
```
0x85 0x10 0x00
^^^^  ^^^^^^^^^
|     Distance = 0x0010 = 16 bytes back
Length = 5 (means match 7 bytes)

Interpretation: Copy 7 bytes from 16 bytes before current position
```

### Compression Algorithm Steps

1. **Initialize:** Position = 0, Output = empty
2. **Search Window:** Look back up to 8192 bytes for longest match
3. **If match >= 2 bytes:**
   - Emit match token
   - Advance position by match length
4. **If match < 2 bytes:**
   - Accumulate literal bytes
   - When literal block full (128 bytes) or match found, emit literal token
5. **Repeat** until all data processed
6. **Finalize:** Emit any remaining literal bytes

### Base64 Encoding

After compression, encode to Base64 and prepend format identifier:

```
Compressed bytes → Base64 → Prefix
```

**Prefixes:**
- `C/` - Standard Tuya compressed format (most common)
- `M/` - Alternative format (model-specific)

**Example Output:**
```
C/MgAQUBFAUUBRQFFAUUBRQFFAUUBRQF...
^^
Format identifier
```

### Decompression (for verification)

```go
func decompressTuya(compressed []byte) []byte {
    output := []byte{}
    pos := 0
    
    for pos < len(compressed) {
        token := compressed[pos]
        pos++
        
        if (token & 0x80) == 0 {
            // Literal block
            length := int(token&0x7F) + 1
            output = append(output, compressed[pos:pos+length]...)
            pos += length
        } else {
            // Match token
            length := int(token&0x7F) + 2
            distHigh := compressed[pos]
            distLow := compressed[pos+1]
            pos += 2
            distance := (int(distHigh) << 8) | int(distLow)
            
            if length == 129 {
                length += int(compressed[pos])
                pos++
            }
            
            // Copy from output buffer
            start := len(output) - distance
            for i := 0; i < length; i++ {
                output = append(output, output[start+i])
            }
        }
    }
    
    return output
}
```

---

## Broadlink Format

**Used by:** SmartIR and many online IR code databases  
**Format:** Base64-encoded proprietary format  
**Prefix:** `JgB` (when Base64-decoded starts with specific header)

### Broadlink Timing Unit

Broadlink stores durations in vendor-specific "ticks":

```
1 tick = 269/8192 ms ≈ 0.03284 ms ≈ 32.84 µs

To convert to microseconds:
microseconds = ticks * 32.84
```

### Broadlink Encoding

1. **Header:** Fixed bytes indicating Broadlink format
2. **Duration Encoding:**
   - **8-bit values:** Standard durations (0x01-0xFF)
   - **16-bit values:** For durations > 0xFF:
     - Emit `0x00` (escape byte)
     - Emit high byte
     - Emit low byte

**Example:**
```
Broadlink bytes: [0x26, 0x00, 0xAE, 0x00, 0x18, 0x16, ...]
                         ^^^^^ ^^^^^
                         0x00AE = 174 ticks (escape sequence)
                                        0x18 = 24 ticks
                                              0x16 = 22 ticks
```

### Conversion to Microseconds

```python
def broadlink_to_microseconds(broadlink_bytes):
    TICK_DURATION = 269 / 8192  # milliseconds
    timings = []
    i = 0
    
    while i < len(broadlink_bytes):
        if broadlink_bytes[i] == 0x00:
            # 16-bit extended value
            high = broadlink_bytes[i+1]
            low = broadlink_bytes[i+2]
            ticks = (high << 8) | low
            i += 3
        else:
            # 8-bit value
            ticks = broadlink_bytes[i]
            i += 1
        
        microseconds = int(ticks * TICK_DURATION * 1000)
        timings.append(microseconds)
    
    return timings
```

---

## Conversion Pipeline

### SmartIR (Broadlink) → Tuya

Complete pipeline for converting SmartIR database codes to Tuya format:

```
1. SmartIR JSON (Broadlink Base64)
   ↓
2. Base64 decode
   ↓
3. Parse Broadlink format (extract timing ticks)
   ↓
4. Convert ticks to microseconds
   ↓
5. Pack as 16-bit LE values
   ↓
6. Apply Tuya compression (LZ77)
   ↓
7. Base64 encode
   ↓
8. Add "C/" prefix
   ↓
9. Tuya format ready for ZS06
```

**Automation:** See [ir-code-prep.md](ir-code-prep.md) and `docs/smartir/reference/broadlink_to_tuya.py`

### Go Implementation Strategy

For Phase 2 (Encoder) and Phase 3 (Generator):

```go
// Phase 3: Generate Daikin Protocol
func GenerateDaikinCommand(state ACState) []byte {
    frame1 := buildFrame1()
    frame2 := buildFrame2(state)
    frame3 := buildFrame3(state)
    return concatenate(frame1, frame2, frame3)
}

// Phase 2: Convert to Tuya
func EncodeTuya(frames []byte) string {
    timings := framesToTimings(frames)  // Bits → µs array
    raw := timingsToBytes(timings)      // Pack as 16-bit LE
    compressed := compressTuya(raw)     // LZ77 compression
    encoded := base64.StdEncoding.EncodeToString(compressed)
    return "C/" + encoded
}
```

---

## Testing & Validation

### Verification Methods

1. **Protocol Correctness:**
   - Generate command for known state
   - Compare checksum with reference
   - Validate bit timing ratios

2. **Compression Verification:**
   - Compress → Decompress → Compare with original
   - Check output matches ZS06 expectations
   - Verify Base64 encoding

3. **Hardware Validation:**
   - Transmit via ZS06
   - Observe AC unit response
   - Confirm state change matches command

### Reference Implementations

- **Arduino-IRremote:** [Daikin protocol](https://github.com/Arduino-IRremote/Arduino-IRremote/blob/master/src/ir_Daikin.cpp)
- **Tuya Codec:** [Gist by mildsunrise](https://gist.github.com/mildsunrise/1d576669b63a260d2cff35fda63ec0b5)
- **Broadlink→Tuya:** [Gist by svyatogor](https://gist.github.com/svyatogor/7839d00303998a9fa37eb079328e4ddaf9)

### Common Pitfalls

1. **Byte Order:** Tuya uses little-endian; Broadlink uses big-endian in some places
2. **Checksum Errors:** Ensure all bytes included, use correct modulo operation
3. **Timing Precision:** Microsecond accuracy matters; avoid rounding errors
4. **Frame Gaps:** Must include proper silence periods between frames
5. **Bit Order:** Daikin transmits LSB first (least significant bit)

---

## External References

- [Daikin IR Protocol Analysis](https://github.com/mharizanov/Daikin-AC-remote-control-over-the-Internet/blob/master/README.md)
- [IRremote Library Documentation](http://www.righto.com/2020/12/reverse-engineering-daikin-infrared.html)
- [Tuya IR Format Specification](https://gist.github.com/mildsunrise/1d576669b63a260d2cff35fda63ec0b5)
- [SmartIR Repository](https://github.com/smartHomeHub/SmartIR)
- [Broadlink Protocol](https://github.com/mjg59/python-broadlink)

# MapleStory Flame Assistant

An automated tool to assist with flaming equipment in MapleStory. This tool uses computer vision and OCR to read flame stats and automatically reroll flames based on configurable criteria.

## Versions

- **Python version (v1)**: Original implementation (deprecated)
- **Go version (v2)**: Current implementation with enhanced performance, reliability, and features

## Features (Go Version)

- **Automatic MapleStory window detection**
- **Real-time OCR of flame stats** (before and after reroll)
- **Configurable main and secondary stats** via command-line flags
- **Intelligent flame score calculation** with weighted stat values
- **Enhanced image processing** with 2x upscaling for better OCR accuracy
- **Direct terminal output** with color-coded stat comparisons
- **Comprehensive logging** with attempt numbers and timestamps
- **Combined screenshot generation** using enhanced images
- **FIFO queue management** (maintains max 7 combined images)
- **CP increase detection** with optional ignore flag
- **Emergency stop** with Ctrl+F1
- **Automatic cleanup** of temp files

## Requirements

- **Go 1.16 or later**
- **Windows OS** (uses Win32 API)
- **Tesseract OCR** installed and in your PATH

## Installation

1. Install Tesseract OCR on your system
2. Navigate to the `goversion` directory
3. Build the executable:
   ```bash
   go build -o flame.exe flame.go
   ```

## Usage

Run the flame scoring tool with your class stats:

```bash
# Basic usage
flame.exe -main=STR -secondary=DEX

# With CP ignore flag (continues rolling despite positive CP increases)
flame.exe -main=INT -secondary=LUK --ignoreCP
```

### Valid Stats
- **STR** (Strength)
- **DEX** (Dexterity) 
- **INT** (Intelligence)
- **LUK** (Luck)

## Controls

- Press **Ctrl+F1** at any time to stop the script
- The script automatically stops when:
  - A better or equal flame score is achieved
  - A positive CP increase is detected (unless `--ignoreCP` is used)
  - The same score appears 3 consecutive times

## How It Works

The Go version calculates flame scores using the formula:
```
Score = Main Stat + (Weapon/Magic Attack × 4) + (% All Stat × 10) + (Secondary Stat ÷ 8)
```

**Process for each attempt:**
1. **Capture before stats** - Screenshots and OCRs current flame stats
2. **Capture after stats** - Screenshots and OCRs reroll results  
3. **Enhanced image processing** - Creates 2x upscaled images for better OCR accuracy
4. **Score calculation** - Computes weighted flame scores for comparison
5. **Decision making** - Keeps better scores or continues rolling
6. **Image combination** - Creates side-by-side comparison images
7. **Cleanup** - Manages temp files and maintains FIFO queue

## Project Structure

```
goversion/
├── flame.go              - Main application logic
├── flame.exe             - Compiled executable
├── internal/
│   ├── automation/       - Mouse/keyboard automation
│   ├── flame/           - Stat extraction and scoring
│   ├── ocr/             - OCR with image enhancement
│   ├── screenshot/      - Screen capture and image processing
│   └── window/          - Windows API integration
└── temp/                - Generated logs and combined images
```

## Safety Features

- **Window detection** to ensure MapleStory is active
- **Emergency stop** hotkey (Ctrl+F1)
- **Enhanced screenshots** saved for verification
- **Configurable delays** between actions (80ms enter key delays)
- **Automatic stuck detection** when stats don't change
- **CP increase detection** to prevent accidental worse flames

## Logging

The tool creates comprehensive logs in `temp/flame_logs.txt` including:
- Timestamped attempts with attempt numbers
- Raw OCR text output
- Extracted stat values
- Calculated flame scores
- Success notifications

## Disclaimer

Use this tool responsibly and at your own risk. Automated tools may be against MapleStory's Terms of Service.

# MapleStory Flame Assistant

An automated tool to assist with flaming equipment in MapleStory. This script uses computer vision and OCR to read flame stats and automatically reroll flames based on configurable criteria.

## Features

- Automatic detection of MapleStory window
- Real-time OCR of flame stats (before and after)
- Supports both weapon and armor flaming
- Configurable main and secondary stats
- Automatic flame score calculation
- Debug screenshots for troubleshooting
- Emergency stop with Ctrl+Esc

## Requirements

- Python 3.x
- Tesseract OCR
- Required Python packages:
  - opencv-python
  - numpy
  - pytesseract
  - Pillow
  - pywin32
  - psutil
  - keyboard
  - pyautogui

## Setup

1. Install Tesseract OCR on your system
2. Install required Python packages:
   ```
   pip install opencv-python numpy pytesseract Pillow pywin32 psutil keyboard pyautogui
   ```
3. Configure your main and secondary stats in `maple_flame.py`:
   ```python
   MAIN_STAT = STR    # Change to your class's main stat (STR/DEX/INT/LUK)
   SECONDARY_STAT = DEX  # Change to your class's secondary stat
   ```

## Usage

1. Open MapleStory and navigate to the flaming interface
2. Run the script:
   ```
   python maple_flame.py
   ```
3. Press Ctrl+Esc at any time to stop the script

## How It Works

The script calculates flame scores using the formula:
```
Score = Main Stat + (Weapon/Magic Attack × 4) + (% All Stat × 8) + (Secondary Stat ÷ 8)
```

For each flame attempt:
1. Captures and OCRs the current stats
2. Calculates the flame score
3. Decides whether to keep or reroll based on target values
4. Automatically clicks appropriate buttons

## Safety Features

- Window detection to ensure MapleStory is active
- Emergency stop hotkey (Ctrl+Esc)
- Debug screenshots saved for verification
- Configurable delays between actions

## Disclaimer

Use this tool responsibly and at your own risk. Automated tools may be against MapleStory's Terms of Service.

# MapleStory Stats OCR Tool (Go Version)

This is a Go implementation of the MapleStory stats OCR automation tool that detects Item Drop Rate and Mesos Obtained stats from the game.

## Features

- **OCR Detection**: Captures and analyzes MapleStory stat windows
- **Prime Line Detection**: Recognizes "Item Drop" and "Mesos Obtained" stats
- **Accumulative Stats**: Sums up multiple instances of the same stat
- **Max Mode**: Optional mode to search for 2+ prime lines (`--max` flag)
- **Colorized Output**: Uses ANSI colors for better terminal readability
- **Comprehensive Logging**: Creates timestamped log files of OCR results
- **Screenshot Management**: Maintains a FIFO queue of screenshots
- **Automatic Cleanup**: Clears temp folder at startup
- **Stuck Detection**: Automatically stops if text doesn't change for 3 consecutive tries

## Requirements

- Go 1.16 or later
- Windows OS (uses Win32 API)
- Tesseract OCR installed and in your PATH

## Project Structure

```
goversion/
├── internal/
│   ├── window/      - Windows API integration for window management
│   ├── screenshot/  - Screen capture functionality
│   ├── ocr/         - Text extraction and pattern matching
│   └── automation/  - Mouse and keyboard automation
├── main.go          - Main application logic
└── README.md        - This file
```

## Usage

Run the application with:

```bash
# Standard mode (stop on first prime line)
go run main.go

# Max mode (search for 2-3 prime lines)
go run main.go --max
```

## Build

To build the executable:

```bash
go build -o maple_stats.exe
```

## Controls

- Press `Ctrl+Esc` or `Shift+Esc` at any time to stop the script
- The script will automatically stop when:
  - It finds a desired stat (standard mode) 
  - It finds 2+ prime lines (max mode)
  - OCR text is unchanged for 3 consecutive tries (stuck detection)

## Implementation Notes

1. The Go version uses the Windows API directly via syscall instead of wrappers
2. The OCR implementation expects Tesseract to be installed on the system
3. The code is organized into reusable modules for better maintainability
4. Error handling is more robust compared to the Python version

## Required External Tools

- Tesseract OCR (for text extraction): https://github.com/tesseract-ocr/tesseract

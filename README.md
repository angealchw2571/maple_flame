# MapleStory Flame Potential Analyzer

A Go application that automates MapleStory flame potential analysis using OCR and computer vision.

## Features

- **Window Detection**: Automatically finds and focuses MapleStory window
- **Screenshot Capture**: Captures specific regions of flame stat interface  
- **OCR Processing**: Extracts text from flame stats using Tesseract
- **Smart Analysis**: Analyzes flame potential and recommends keep/reroll
- **Continuous Mode**: Monitors flame results until good stats are found

## Prerequisites

1. **Go 1.21+** - Download from https://golang.org/
2. **Tesseract OCR** - Download from https://github.com/UB-Mannheim/tesseract/wiki
   - Make sure `tesseract.exe` is in your system PATH
3. **MapleStory** running in windowed mode

## Installation

1. Clone or download this repository
2. Open terminal in the project directory
3. Run: `go mod tidy` (if needed)
4. Run: `go build`
5. Run: `.\maple_flame.exe`

## Usage

### Single Analysis Mode
1. Start the application
2. Choose option 1 (Single flame analysis)
3. Have MapleStory flame interface open
4. Follow the prompts to capture before/after stats
5. Get recommendation on whether to keep the flame

### Continuous Mode  
1. Choose option 2 (Continuous flame analysis)
2. The app will monitor flame results automatically
3. Stops when good stats are detected

### Configuration

Edit the screen regions in `internal/analyzer/analyzer.go` if needed:

```go
// Adjust these coordinates based on your screen resolution
beforeRegion = ScreenRegion{
    X: 100,     // Distance from left edge of MapleStory window
    Y: 200,     // Distance from top edge  
    Width: 200, // Width of stats area
    Height: 150 // Height of stats area
}
```

## What Makes a Good Flame?

The analyzer looks for these in order of priority:

1. **Perfect Roll**: Both Item Drop Rate + Mesos Obtained
2. **High Item Drop**: ≥20% Item Drop Rate  
3. **High Mesos**: ≥20% Mesos Obtained
4. **Improvement**: Better than previous flame
5. **Higher Percentages**: Same stats but higher values

## Troubleshooting

**"MapleStory window not found"**
- Make sure MapleStory is running
- Run MapleStory in windowed mode (not fullscreen)
- Make sure window title contains "MapleStory"

**"Tesseract failed"**  
- Install Tesseract OCR from the link above
- Add tesseract.exe to your system PATH
- The app will use simulated results if Tesseract isn't available

**Wrong screen regions captured**
- Adjust coordinates in `GetTargetScreenRegions()` function
- Use "Test screenshot capture" option to verify regions
- Screenshots are saved in `temp/` folder for verification

## File Structure

```
maple_flame/
├── main.go                    # Main application entry point
├── internal/
│   ├── window/window.go       # MapleStory window detection
│   ├── screenshot/screenshot.go # Screen capture functionality  
│   ├── ocr/ocr.go            # OCR text extraction
│   └── analyzer/analyzer.go   # Flame analysis logic
└── temp/                      # Debug screenshots saved here
```

## License

MIT License - feel free to modify and distribute.

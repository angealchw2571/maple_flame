// Maple Flame Stats OCR Tool (Go version)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"maple_flame/goversion/internal/automation"
	"maple_flame/goversion/internal/ocr"
	"maple_flame/goversion/internal/screenshot"
	"maple_flame/goversion/internal/window"
)

// ANSI color codes
const (
	GREEN = "\033[32m"
	CYAN  = "\033[36m"
	WHITE = "\033[37m"
	RESET = "\033[0m"
)

// Click coordinates for drop rate detection (adjust as needed)
var (
	DROP_CLICK_X = 647 // X offset from window left for reroll button
	DROP_CLICK_Y = 550 // Y offset from window top for reroll button
)

// ScanResult represents the result of a stat scan
type ScanResult struct {
	ItemDropRate    int
	MesosObtained   int
	HasItemKeyword  bool
	HasMesosKeyword bool
	PrimeLineCount  int
	RawText         string
}

// Global variables
var (
	logFile string
)

// setupLogging creates the temp directory and clears all files
func setupLogging() (string, error) {
	tempDir := filepath.Join(".", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Clear all files in temp directory
	fmt.Printf("%sCleaning temp folder...%s\n", CYAN, RESET)
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return "", fmt.Errorf("error reading temp directory: %v", err)
	}

	for _, file := range files {
		filePath := filepath.Join(tempDir, file.Name())
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("Error removing file %s: %v\n", file.Name(), err)
		} else {
			fmt.Printf("Removed: %s\n", file.Name())
		}
	}

	// Create new log file
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(tempDir, fmt.Sprintf("logs_%s.txt", timestamp))

	return logFile, nil
}

// logOcrText writes OCR text and stats to the log file
func logOcrText(logFilePath string, text string, stats *ScanResult) error {
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write timestamp and OCR text
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("\n===== OCR Scan: %s =====\n", timestamp))
	f.WriteString(text)
	f.WriteString("\n")

	// Write extracted stats if available
	if stats != nil {
		f.WriteString("\nExtracted Stats:\n")
		f.WriteString(fmt.Sprintf("item_drop_rate: %d\n", stats.ItemDropRate))
		f.WriteString(fmt.Sprintf("mesos_obtained: %d\n", stats.MesosObtained))
		f.WriteString(fmt.Sprintf("prime_line_count: %d\n", stats.PrimeLineCount))
	}

	f.WriteString("\n" + strings.Repeat("-", 60) + "\n")
	return nil
}

// scanForStats captures a screenshot and extracts stats
func scanForStats(logFilePath string, tryNumber int) (*ScanResult, error) {
	// Get MapleStory window coordinates
	windowRect, err := window.GetMaplestoryWindow()
	if err != nil {
		return nil, fmt.Errorf("error getting MapleStory window: %v", err)
	}

	// Define the region to capture
	regionX := 607 // X coordinate offset from window left
	regionY := 449 // Y coordinate offset from window top
	width := 168   // Width of region to capture
	height := 75   // Height of region to capture

	// Capture the stat box region
	img, err := screenshot.CaptureScreenRegion(windowRect, regionX, regionY, width, height)
	if err != nil {
		return nil, fmt.Errorf("error capturing screen region: %v", err)
	}

	// Save debug image with try number
	imagePath, err := screenshot.SaveDebugImage(img, tryNumber)
	if err != nil {
		return nil, fmt.Errorf("error saving debug image: %v", err)
	}

	// Extract text using OCR with the saved image file
	text, err := ocr.ExtractText(imagePath)
	if err != nil {
		return nil, fmt.Errorf("OCR extraction error: %v", err)
	}

	// Print raw OCR text
	fmt.Printf("\nRaw OCR text:\n")
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("-", 40), RESET)
	fmt.Println(text)
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("-", 40), RESET)

	// Check for keywords and count prime lines
	hasItemKeyword, hasMesosKeyword, primeLineCount := ocr.DetectKeywords(text)

	// Extract specific stats
	itemDropRate := ocr.ExtractItemDropRate(text)
	mesosObtained := ocr.ExtractMesosObtained(text)

	// Determine color for output based on values
	var itemColor, mesosColor string
	if itemDropRate > 0 {
		itemColor = GREEN
	} else {
		itemColor = WHITE
	}

	if mesosObtained > 0 {
		mesosColor = GREEN
	} else {
		mesosColor = WHITE
	}

	// Print stats with colors
	fmt.Printf("%sItem Drop Rate: +%d%%%s\n", itemColor, itemDropRate, RESET)
	fmt.Printf("%sMesos Obtained: +%d%%%s\n", mesosColor, mesosObtained, RESET)
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("=", 40), RESET)

	// Create result
	result := &ScanResult{
		ItemDropRate:    itemDropRate,
		MesosObtained:   mesosObtained,
		HasItemKeyword:  hasItemKeyword,
		HasMesosKeyword: hasMesosKeyword,
		PrimeLineCount:  primeLineCount,
		RawText:         text,
	}

	// Log the results
	logOcrText(logFilePath, text, result)

	return result, nil
}

// logSuccess writes a success message to the log file
func logSuccess(logFilePath string, result *ScanResult, maxMode bool, primeLines int) error {
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	successHeader := "SUCCESS: DESIRED STATS FOUND"
	if maxMode {
		successHeader = fmt.Sprintf("SUCCESS: %d PRIME LINES FOUND", primeLines)
	}

	f.WriteString(fmt.Sprintf("\n===== %s =====\n", successHeader))
	f.WriteString(fmt.Sprintf("Item Drop Rate: +%d%%\n", result.ItemDropRate))
	f.WriteString(fmt.Sprintf("Mesos Obtained: +%d%%\n", result.MesosObtained))
	f.WriteString(fmt.Sprintf("Total Prime Lines: %d\n", primeLines))

	return nil
}

func main() {
	// Parse command line arguments
	maxMode := flag.Bool("max", false, "Search for 2-3 prime lines instead of stopping at first one")
	flag.Parse()

	// Initialize text history for stuck detection
	textHistory := make([]string, 0, 3)

	// Print welcome message
	fmt.Printf("%sMapleStory Stats OCR Tool (Go version)%s\n", CYAN, RESET)
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("=", 40), RESET)
	fmt.Println("This tool will capture a region of your MapleStory window")
	fmt.Println("and extract Item Drop Rate and Mesos Obtained stats.")
	
	if *maxMode {
		fmt.Printf("%sMAX MODE: Searching for 2-3 prime lines%s\n", GREEN, RESET)
	} else {
		fmt.Println("Standard mode: Stopping on first prime line")
	}
	
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("=", 40), RESET)
	fmt.Println("\nPress Ctrl+F1 at any time to exit")
	fmt.Println("Script will automatically stop if text remains unchanged for 3 consecutive tries")

	// Setup logging
	logFilePath, err := setupLogging()
	if err != nil {
		log.Fatalf("Error setting up logging: %v", err)
	}

	// Main loop
	rerollDelay := 1.0 // seconds between rerolls
	splitDelay := 4    // number of parts to split the delay for key checking
	splitTime := time.Duration(float64(rerollDelay) * float64(time.Second) / float64(splitDelay))
	tryCounter := 0

	try:
	for {
		tryCounter++
		// Check for stop key combination
		if automation.CheckStopKey() {
			fmt.Println("\nCtrl+F1 detected. Exiting...")
			break
		}

		// Scan for stats
		fmt.Println("\nScanning for stats...")
		// Add a short delay before scanning to allow text to fully render
		fmt.Println("Waiting for text to render...")
		time.Sleep(500 * time.Millisecond)
		result, err := scanForStats(logFilePath, tryCounter)
		if err != nil {
			fmt.Printf("Error scanning for stats: %v\n", err)
			break
		}

		// Add current text to history and keep only last 3
		if result != nil {
			currentText := result.RawText
			textHistory = append(textHistory, currentText)
			if len(textHistory) > 3 {
				textHistory = textHistory[1:]
			}

			// Check if text hasn't changed for 3 consecutive tries
			if len(textHistory) == 3 && 
			   textHistory[0] == textHistory[1] && 
			   textHistory[1] == textHistory[2] {
				fmt.Printf("\n%s⚠️ OCR text unchanged for 3 consecutive tries. Script might be stuck.%s\n", CYAN, RESET)
				fmt.Println("\nExiting script...")
				
				// Log the issue
				f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err == nil {
					f.WriteString("\n===== SCRIPT STOPPED: TEXT UNCHANGED FOR 3 TRIES =====\n")
					f.WriteString(fmt.Sprintf("Last detected text:\n%s\n", currentText))
					f.Close()
				}
				break
			}
		}

		if result != nil {
			// Count prime lines found
			primeLines := 0
			if result.HasItemKeyword {
				primeLines++
			}
			if result.HasMesosKeyword {
				primeLines++
			}

			// Standard mode: stop on first prime line
			// Max mode: stop only if we have 2+ prime lines
			if (!*maxMode && primeLines > 0) || (*maxMode && primeLines >= 2) {
				successMessage := "Found desired stats!"
				if *maxMode {
					successMessage = fmt.Sprintf("Found %d prime lines!", primeLines)
				}
				
				fmt.Printf("\n%s✅ %s Scanning complete.%s\n", GREEN, successMessage, RESET)
				fmt.Printf("\n%sDetected Text:%s\n%s", GREEN, RESET, result.RawText)
				
				// Log the final successful result
				logSuccess(logFilePath, result, *maxMode, primeLines)
				break
			}
		}

		// Get the window rectangle again for clicking
		windowRect, err := window.GetMaplestoryWindow()
		if err != nil {
			fmt.Printf("Error getting MapleStory window: %v\n", err)
			break
		}

		// No desired stats found, click to reroll
		err = automation.ClickRerollButton(windowRect, DROP_CLICK_X, DROP_CLICK_Y)
		if err != nil {
			fmt.Printf("Error clicking reroll button: %v\n", err)
			break
		}

		// Split the delay into parts for responsive key checking
		for i := 0; i < splitDelay; i++ {
			time.Sleep(splitTime)
			if automation.CheckStopKey() {
				fmt.Printf("\n%sCtrl+F1 detected. Exiting...%s\n", GREEN, RESET)
				break try
			}
		}
	}
}

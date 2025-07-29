// Maple Flame Scoring Tool
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"maple_flame/goversion/internal/automation"
	"maple_flame/goversion/internal/flame"
	"maple_flame/goversion/internal/ocr"
	"maple_flame/goversion/internal/screenshot"
	"maple_flame/goversion/internal/window"
)

// ANSI color codes
const (
	GREEN = "\033[32m"
	RED   = "\033[31m"
	CYAN  = "\033[36m"
	WHITE = "\033[37m"
	RESET = "\033[0m"
)

// Click coordinates for flame scoring (adjust as needed)
var (
	FLAME_CLICK_X = 700 // X offset from window left for reroll button
	FLAME_CLICK_Y = 630 // Y offset from window top for reroll button
)

// FlameResult represents the result of a flame scan
type FlameResult struct {
	Stats   *flame.FlameStats
	Score   float64
	RawText string
	Image   *image.RGBA // Store the actual image for combining
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
	logFile := filepath.Join(tempDir, "flame_logs.txt")

	return logFile, nil
}

// logFlameResult writes flame result to the log file
func logFlameResult(logFilePath string, result *FlameResult, config *flame.FlameConfig, label string, attemptNumber int) error {
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write timestamp and label with attempt number
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("\n===== %s Flame Scan (Attempt #%d): %s =====\n", label, attemptNumber, timestamp))
	f.WriteString(result.RawText)
	f.WriteString("\n")

	// Write extracted stats
	f.WriteString("\nExtracted Stats:\n")
	f.WriteString(fmt.Sprintf("Main Stat (%s): %d\n", config.MainStat, result.Stats.MainStat))
	f.WriteString(fmt.Sprintf("Secondary Stat (%s): %d\n", config.SecondaryStat, result.Stats.SecondaryStat))
	f.WriteString(fmt.Sprintf("Weapon Attack: %d\n", result.Stats.WeaponAttack))
	f.WriteString(fmt.Sprintf("Magic Attack: %d\n", result.Stats.MagicAttack))
	f.WriteString(fmt.Sprintf("All Stat %%: %d\n", result.Stats.AllStatPercent))
	f.WriteString(fmt.Sprintf("CP Increase: %d\n", result.Stats.CPIncrease))
	f.WriteString(fmt.Sprintf("Flame Score: %.3f\n", result.Score))

	f.WriteString("\n" + strings.Repeat("-", 60) + "\n")
	return nil
}

// captureFlameStats captures a screenshot and extracts flame stats
func captureFlameStats(logFilePath string, config *flame.FlameConfig, isBefore bool, tryNumber int) (*FlameResult, error) {
	// Get MapleStory window coordinates
	windowRect, err := window.GetMaplestoryWindow()
	if err != nil {
		return nil, fmt.Errorf("error getting MapleStory window: %v", err)
	}

	// Define the region to capture based on whether it's before or after
	var regionX, regionY int
	width := 167   // Width of region to capture
	height := 118  // Height of region to capture

	if isBefore {
		// Position for BEFORE box (left side)
		regionX = 607 // X coordinate offset from window left
		regionY = 350 // Y coordinate offset from window top
	} else {
		// Position for AFTER box (right side)
		regionX = 607 // X coordinate offset from window left
		regionY = 495 // Y coordinate offset from window top
	}

	// Capture the stat box region
	img, err := screenshot.CaptureScreenRegion(windowRect, regionX, regionY, width, height)
	if err != nil {
		return nil, fmt.Errorf("error capturing screen region: %v", err)
	}

	// Wait for UI to render before OCR
	time.Sleep(300 * time.Millisecond)

	// Create a temporary image file for OCR (we'll delete it after)
	tempDir := filepath.Join(".", "temp")
	os.MkdirAll(tempDir, 0755)
	
	prefix := "before"
	if !isBefore {
		prefix = "after"
	}
	tempImagePath := filepath.Join(tempDir, fmt.Sprintf("temp_%s_%d.png", prefix, tryNumber))
	
	// Save original image for OCR
	f, err := os.Create(tempImagePath)
	if err != nil {
		return nil, fmt.Errorf("error creating temp image file: %v", err)
	}
	defer f.Close()
	
	err = png.Encode(f, img)
	if err != nil {
		return nil, fmt.Errorf("error encoding temp image: %v", err)
	}
	f.Close() // Close before OCR

	// Extract text using OCR optimized for flame stats (using original image)
	text, err := ocr.ExtractFlameText(tempImagePath)
	if err != nil {
		return nil, fmt.Errorf("OCR extraction error: %v", err)
	}
	
	// Don't delete temp files here - they'll be cleaned up after combining

	// OCR text is logged to file, no need to print to terminal in live mode

	// Extract flame stats
	stats, err := flame.ExtractFlameStats(text, config)
	if err != nil {
		return nil, fmt.Errorf("error extracting flame stats: %v", err)
	}

	// Calculate flame score
	score := flame.CalculateFlameScore(stats, config)

	// Create result with a copy of the original image to avoid modification issues
	imageCopy := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			imageCopy.Set(x, y, img.At(x, y))
		}
	}
	
	result := &FlameResult{
		Stats:   stats,
		Score:   score,
		RawText: text,
		Image:   imageCopy, // Store a copy of the captured image
	}

	// Log the results to file
	label := "BEFORE"
	if !isBefore {
		label = "AFTER"
	}
	logFlameResult(logFilePath, result, config, label, tryNumber)

	return result, nil
}

const BLACK = "\033[30m"

// printFlameComparisonBuffer prints before and after flame scores side by side (for buffer system)
func printFlameComparisonBuffer(beforeResult, afterResult *FlameResult, config *flame.FlameConfig) {
	leftWidth := 35  // Fixed width for left column
	rightWidth := 35 // Fixed width for right column

	// Print header
	fmt.Printf("\n\n%s%s%s\n", CYAN, strings.Repeat("=", leftWidth+rightWidth+3), RESET)
	fmt.Printf("%s%-*s%s|%s%-*s%s\n", GREEN, leftWidth, "BEFORE", RESET, GREEN, rightWidth, "AFTER", RESET)
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("=", leftWidth+rightWidth+3), RESET)

	// Clear any residual colors by printing a reset sequence
	fmt.Printf("%s", RESET)
	
	// Print normally with colors
	printComparisonRows(beforeResult, afterResult, config, leftWidth, "")
}

// printComparisonRows prints the data rows of the comparison (with black AFTER values if forceBlack is BLACK)
func printComparisonRows(beforeResult, afterResult *FlameResult, config *flame.FlameConfig, leftWidth int, forceBlack string) {

	// Print main stat with color
	mainStatDiff := afterResult.Stats.MainStat - beforeResult.Stats.MainStat
	mainStatColor := GREEN
	if mainStatDiff < 0 {
		mainStatColor = RED
	} else if mainStatDiff == 0 {
		mainStatColor = WHITE
	}
	
	// Use black color if forceBlack is set, otherwise use calculated color
	if forceBlack == BLACK {
		mainStatColor = BLACK
	}

	beforeMainStat := fmt.Sprintf("Main Stat (%s): %d", config.MainStat, beforeResult.Stats.MainStat)
	afterMainStat := fmt.Sprintf("Main Stat (%s): %s%d%s", config.MainStat, mainStatColor, afterResult.Stats.MainStat, RESET)
	fmt.Printf("\n%-*s|  %s\n", leftWidth, beforeMainStat, afterMainStat)

	// Print secondary stat with color
	ssDiff := afterResult.Stats.SecondaryStat - beforeResult.Stats.SecondaryStat
	ssColor := GREEN
	if ssDiff < 0 {
		ssColor = RED
	} else if ssDiff == 0 {
		ssColor = WHITE
	}
	
	// Use black color if forceBlack is set, otherwise use calculated color
	if forceBlack == BLACK {
		ssColor = BLACK
	}

	beforeSS := fmt.Sprintf("Secondary (%s): %d → %.3f", config.SecondaryStat, beforeResult.Stats.SecondaryStat, float64(beforeResult.Stats.SecondaryStat)/8)
	afterSS := fmt.Sprintf("Secondary (%s): %s%d%s → %s%.3f%s", config.SecondaryStat, ssColor, afterResult.Stats.SecondaryStat, RESET, ssColor, float64(afterResult.Stats.SecondaryStat)/8, RESET)
	fmt.Printf("%-*s|  %s\n", leftWidth, beforeSS, afterSS)

	// Print attack stats with color (weapon or magic based on main stat)
	if config.MainStat == flame.INT {
		// Print magic attack with color
		maDiff := afterResult.Stats.MagicAttack - beforeResult.Stats.MagicAttack
		maColor := GREEN
		if maDiff < 0 {
			maColor = RED
		} else if maDiff == 0 {
			maColor = WHITE
		}
		
		// Use black color if forceBlack is set, otherwise use calculated color
		if forceBlack == BLACK {
			maColor = BLACK
		}

		beforeMA := fmt.Sprintf("Magic Attack: %d → %d", beforeResult.Stats.MagicAttack, beforeResult.Stats.MagicAttack*4)
		afterMA := fmt.Sprintf("Magic Attack: %s%d%s → %s%d%s", maColor, afterResult.Stats.MagicAttack, RESET, maColor, afterResult.Stats.MagicAttack*4, RESET)
		fmt.Printf("%-*s|  %s\n", leftWidth, beforeMA, afterMA)
	} else {
		// Print weapon attack with color
		waDiff := afterResult.Stats.WeaponAttack - beforeResult.Stats.WeaponAttack
		waColor := GREEN
		if waDiff < 0 {
			waColor = RED
		} else if waDiff == 0 {
			waColor = WHITE
		}
		
		// Use black color if forceBlack is set, otherwise use calculated color
		if forceBlack == BLACK {
			waColor = BLACK
		}

		beforeWA := fmt.Sprintf("Weapon Attack: %d → %d", beforeResult.Stats.WeaponAttack, beforeResult.Stats.WeaponAttack*4)
		afterWA := fmt.Sprintf("Weapon Attack: %s%d%s → %s%d%s", waColor, afterResult.Stats.WeaponAttack, RESET, waColor, afterResult.Stats.WeaponAttack*4, RESET)
		fmt.Printf("%-*s|  %s\n", leftWidth, beforeWA, afterWA)
	}

	// Print all stat with color
	asDiff := afterResult.Stats.AllStatPercent - beforeResult.Stats.AllStatPercent
	asColor := GREEN
	if asDiff < 0 {
		asColor = RED
	} else if asDiff == 0 {
		asColor = WHITE
	}
	
	// Use black color if forceBlack is set, otherwise use calculated color
	if forceBlack == BLACK {
		asColor = BLACK
	}

	beforeAS := fmt.Sprintf("All Stat %%: %d%% → %d", beforeResult.Stats.AllStatPercent, beforeResult.Stats.AllStatPercent*10)
	afterAS := fmt.Sprintf("All Stat %%: %s%d%%%s → %s%d%s", asColor, afterResult.Stats.AllStatPercent, RESET, asColor, afterResult.Stats.AllStatPercent*10, RESET)
	fmt.Printf("%-*s|  %s\n", leftWidth, beforeAS, afterAS)

	// Print CP increase with color (positive=green, negative=red, zero=white)
	var cpColor string
	if afterResult.Stats.CPIncrease > 0 {
		cpColor = GREEN
	} else if afterResult.Stats.CPIncrease < 0 {
		cpColor = RED
	} else {
		cpColor = WHITE
	}
	
	// Use black color if forceBlack is set, otherwise use calculated color
	if forceBlack == BLACK {
		cpColor = BLACK
	}

	beforeCP := fmt.Sprintf("CP Increase: %d", beforeResult.Stats.CPIncrease)
	afterCP := fmt.Sprintf("CP Increase: %s%d%s", cpColor, afterResult.Stats.CPIncrease, RESET)
	fmt.Printf("%-*s|  %s\n", leftWidth, beforeCP, afterCP)

	// Print divider
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("-", leftWidth+35+3), RESET)

	// Print total scores with color
	scoreDiff := afterResult.Score - beforeResult.Score
	scoreColor := GREEN
	if scoreDiff < 0 {
		scoreColor = RED
	} else if scoreDiff == 0 {
		scoreColor = WHITE
	}
	
	// Use black color if forceBlack is set, otherwise use calculated color
	if forceBlack == BLACK {
		scoreColor = BLACK
	}

	beforeScore := fmt.Sprintf("Total Score: %.3f", beforeResult.Score)
	afterScore := fmt.Sprintf("Total Score: %s%.3f%s", scoreColor, afterResult.Score, RESET)
	fmt.Printf("%-*s|  %s\n", leftWidth, beforeScore, afterScore)
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("=", leftWidth+35+3), RESET)

	// Print score difference with color (skip if forceBlack is set)
	if forceBlack != BLACK {
		diff := afterResult.Score - beforeResult.Score
		color := GREEN
		if diff < 0 {
			color = RED
		} else if diff == 0 {
			color = WHITE
		}

		if diff > 0 {
			fmt.Printf("\n\nScore Difference: %s+%.3f%s\n\n\n", color, diff, RESET)
		} else {
			fmt.Printf("\n\nScore Difference: %s%.3f%s\n\n\n", color, diff, RESET)
		}
	}
}


// logSuccess writes a success message to the log file
func logSuccess(logFilePath string, beforeResult, afterResult *FlameResult, config *flame.FlameConfig, attemptNumber int) error {
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString(fmt.Sprintf("\n===== SUCCESS: BETTER FLAME SCORE ACHIEVED (Attempt #%d) =====\n", attemptNumber))
	f.WriteString(fmt.Sprintf("Before Score: %.3f\n", beforeResult.Score))
	f.WriteString(fmt.Sprintf("After Score: %.3f\n", afterResult.Score))
	f.WriteString(fmt.Sprintf("Improvement: +%.3f\n", afterResult.Score-beforeResult.Score))
	f.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05")))

	return nil
}

func main() {
	// Parse command line arguments
	mainStatStr := flag.String("main", "", "Main stat (STR/DEX/INT/LUK)")
	secondaryStatStr := flag.String("secondary", "", "Secondary stat (STR/DEX/INT/LUK)")
	ignoreCP := flag.Bool("ignoreCP", false, "Ignore positive CP increase and continue rolling until flame score is higher")
	flag.Parse()

	// Check if required arguments are provided
	if *mainStatStr == "" || *secondaryStatStr == "" {
		fmt.Printf("%sError: Both -main and -secondary arguments are required%s\n", RED, RESET)
		fmt.Println("\nUsage:")
		fmt.Println("  flame.exe -main=STR -secondary=DEX")
		fmt.Println("  flame.exe -main=INT -secondary=LUK")
		fmt.Println("  flame.exe -main=DEX -secondary=STR")
		fmt.Println("  flame.exe -main=LUK -secondary=DEX")
		fmt.Println("  flame.exe -main=STR -secondary=DEX --ignoreCP")
		fmt.Println("\nValid stats: STR, DEX, INT, LUK")
		fmt.Println("Optional flags:")
		fmt.Println("  --ignoreCP: Ignore positive CP increase and continue until flame score is higher")
		os.Exit(1)
	}

	// Convert string arguments to StatType
	var mainStat, secondaryStat flame.StatType
	switch strings.ToUpper(*mainStatStr) {
	case "STR":
		mainStat = flame.STR
	case "DEX":
		mainStat = flame.DEX
	case "INT":
		mainStat = flame.INT
	case "LUK":
		mainStat = flame.LUK
	default:
		log.Fatalf("Invalid main stat: %s. Must be STR, DEX, INT, or LUK", *mainStatStr)
	}

	switch strings.ToUpper(*secondaryStatStr) {
	case "STR":
		secondaryStat = flame.STR
	case "DEX":
		secondaryStat = flame.DEX
	case "INT":
		secondaryStat = flame.INT
	case "LUK":
		secondaryStat = flame.LUK
	default:
		log.Fatalf("Invalid secondary stat: %s. Must be STR, DEX, INT, or LUK", *secondaryStatStr)
	}

	// Create flame configuration
	config := &flame.FlameConfig{
		MainStat:      mainStat,
		SecondaryStat: secondaryStat,
	}

	// Print welcome message
	fmt.Printf("%sMapleStory Flame Scoring Tool%s\n", CYAN, RESET)
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("=", 40), RESET)
	fmt.Printf("Main Stat: %s%s%s\n", GREEN, config.MainStat, RESET)
	fmt.Printf("Secondary Stat: %s%s%s\n", GREEN, config.SecondaryStat, RESET)
	fmt.Printf("%s%s%s\n", CYAN, strings.Repeat("=", 40), RESET)
	fmt.Println("\nPress Ctrl+F1 at any time to exit")
	fmt.Println("Script will automatically stop when a better flame score is achieved")
	fmt.Println("or if the same score appears 5 consecutive times")

	// Setup logging
	logFilePath, err := setupLogging()
	if err != nil {
		log.Fatalf("Error setting up logging: %v", err)
	}

	// Initialize tracking variables
	previousAfterScore := -1.0
	unchangedCount := 0
	tryCounter := 0

	// Main loop
	rerollDelay := 0.5 // seconds between rerolls
	splitDelay := 4    // number of parts to split the delay for key checking
	splitTime := time.Duration(float64(rerollDelay) * float64(time.Second) / float64(splitDelay))

	for {
		tryCounter++
		
		fmt.Printf("\n%s=== Attempt %d ===%s\n", CYAN, tryCounter, RESET)

		// Check for stop key combination
		if automation.CheckStopKey() {
			fmt.Printf("\n%sCtrl+F1 detected. Exiting...%s\n", GREEN, RESET)
			break
		}

		// Capture before stats
		fmt.Printf("Capturing before stats...\n")
		beforeResult, err := captureFlameStats(logFilePath, config, true, tryCounter)
		if err != nil {
			fmt.Printf("Error capturing before stats: %v\n", err)
			break
		}

		// Capture after stats
		fmt.Printf("Capturing after stats...\n")
		afterResult, err := captureFlameStats(logFilePath, config, false, tryCounter)
		if err != nil {
			fmt.Printf("Error capturing after stats: %v\n", err)
			break
		}

		// Print flame comparison
		printFlameComparisonBuffer(beforeResult, afterResult, config)

		// Create combined image using enhanced versions
		_, err = screenshot.CombineEnhancedImages(tryCounter)
		if err != nil {
			// Just log warning, don't break execution
			fmt.Printf("Warning: Failed to combine enhanced images: %v\n", err)
			// Fallback to combining original images
			_, err = screenshot.CombineImagesHorizontal(beforeResult.Image, afterResult.Image, tryCounter)
			if err != nil {
				fmt.Printf("Warning: Failed to combine original images: %v\n", err)
			}
		}

		// Check if score hasn't changed
		if previousAfterScore != -1 && previousAfterScore == afterResult.Score {
			unchangedCount++
			if unchangedCount >= 3 {
				fmt.Printf("\n%sScore hasn't changed for 3 consecutive attempts. Stopping script...%s\n", CYAN, RESET)
				break
			}
		} else {
			// Reset counter if score changed
			unchangedCount = 0
		}

		// Update previous score
		previousAfterScore = afterResult.Score

		// Check for stop key again
		if automation.CheckStopKey() {
			fmt.Printf("\n%sCtrl+F1 detected. Exiting...%s\n", GREEN, RESET)
			break
		}

		// Check for POSITIVE CP increase first (trumps everything unless ignoreCP is set)
		if afterResult.Stats.HasCPIncrease && afterResult.Stats.CPIncrease > 0 {
			if !*ignoreCP {
				fmt.Printf("\n%s🎉 POSITIVE CP INCREASE DETECTED: +%d! This trumps everything - stopping here.%s\n", GREEN, afterResult.Stats.CPIncrease, RESET)
				logSuccess(logFilePath, beforeResult, afterResult, config, tryCounter)
				break
			} else {
				fmt.Printf("\n%s🎉 POSITIVE CP INCREASE DETECTED: +%d! (Ignoring due to --ignoreCP flag)%s\n", CYAN, afterResult.Stats.CPIncrease, RESET)
			}
		}

		// If after score is better or equal, we're done
		if afterResult.Score >= beforeResult.Score {
			fmt.Printf("\n%s✅ Got a better or equal score! Stopping here.%s\n", GREEN, RESET)
			logSuccess(logFilePath, beforeResult, afterResult, config, tryCounter)
			break
		}

		// Show reroll message
		statusMsg := fmt.Sprintf("After score is lower. Rerolling in %.1f seconds...", rerollDelay)
		if unchangedCount > 0 {
			statusMsg = fmt.Sprintf("Score unchanged for %d attempts. Rerolling in %.1f seconds...", unchangedCount, rerollDelay)
		}
		fmt.Printf("\n%s%s%s\n", CYAN, statusMsg, RESET)

		// Get the window rectangle again for clicking
		windowRect, err := window.GetMaplestoryWindow()
		if err != nil {
			fmt.Printf("Error getting MapleStory window: %v\n", err)
			break
		}

		// Click to reroll
		err = automation.ClickRerollButton(windowRect, FLAME_CLICK_X, FLAME_CLICK_Y)
		if err != nil {
			fmt.Printf("Error clicking reroll button: %v\n", err)
			break
		}

		// Split the delay into parts for responsive key checking
		for i := 0; i < splitDelay; i++ {
			time.Sleep(splitTime)
			if automation.CheckStopKey() {
				fmt.Printf("\n%sCtrl+F1 detected. Exiting...%s\n", GREEN, RESET)
				return
			}
		}
	}
}
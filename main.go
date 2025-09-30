package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"maple_flame/internal/ocr"
	"maple_flame/internal/screenshot"
	"maple_flame/internal/window"
)

// Windows API for sending keypress and mouse clicks
var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procKeyboardEvent    = user32.NewProc("keybd_event")
	procFindWindow       = user32.NewProc("FindWindowW")
	procPostMessage      = user32.NewProc("PostMessageW")
	procSetCursorPos     = user32.NewProc("SetCursorPos")
	procMouseEvent       = user32.NewProc("mouse_event")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
)

const (
	VK_SPACE       = 0x20
	VK_RETURN      = 0x0D
	VK_CONTROL     = 0x11
	VK_F1          = 0x70
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	INPUT_KEYBOARD = 1
	
	// Mouse event constants
	MOUSEEVENTF_LEFTDOWN = 0x0002
	MOUSEEVENTF_LEFTUP   = 0x0004
	
	// Global capture area settings
	CAPTURE_X      = 530  // X position relative to MapleStory window
	CAPTURE_Y      = 345  // Y position relative to MapleStory window  
	CAPTURE_WIDTH  = 325  // Width of capture area
	CAPTURE_HEIGHT = 120  // Height of capture area
	
	// Reroll click settings
	CLICK_OFFSET_X = 650  // Click X offset from window
	CLICK_OFFSET_Y = 720  // Click Y offset from window
)

type INPUT struct {
	Type uint32
	Ki   KEYBDINPUT
}

type KEYBDINPUT struct {
	VirtualKeyCode uint16
	ScanCode       uint16
	Flags          uint32
	Time           uint32
	ExtraInfo      uintptr
}

// MainStat enum for the four main stats
type MainStat int

const (
	STR MainStat = iota
	DEX
	INT
	LUK
)

// String returns the string representation of the main stat
func (m MainStat) String() string {
	switch m {
	case STR:
		return "STR"
	case DEX:
		return "DEX"
	case INT:
		return "INT"
	case LUK:
		return "LUK"
	default:
		return "UNKNOWN"
	}
}

// parseMainStat converts a string to MainStat enum
func parseMainStat(s string) (MainStat, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "STR":
		return STR, nil
	case "DEX":
		return DEX, nil
	case "INT":
		return INT, nil
	case "LUK":
		return LUK, nil
	default:
		return STR, fmt.Errorf("invalid main stat: %s (valid options: STR, DEX, INT, LUK)", s)
	}
}

// setupLogging configures logging to write to both console and temp/flame.log
func setupLogging() {
	// Create temp directory if it doesn't exist
	tempDir := "temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		return
	}

	// Create log file (same file each time, clear on each run)
	logPath := filepath.Join(tempDir, "flame.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf("Failed to create log file: %v\n", err)
		return
	}

	// Create multi-writer to write to both original stdout and file
	originalStdout := os.Stdout
	multiWriter := io.MultiWriter(originalStdout, logFile)
	
	// Create a pipe to redirect stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	// Start goroutine to copy from pipe to multi-writer
	go func() {
		defer logFile.Close()
		io.Copy(multiWriter, r)
	}()
	
	fmt.Printf("📝 Logging enabled: %s\n", logPath)
}

func main() {
	// Setup logging to both console and file
	setupLogging()

	fmt.Println("MapleStory Auto Flame Reroller")
	fmt.Println("=============================")

	// Parse command-line flags
	modeFlag := flag.String("mode", "", "Mode: armor or weapon")
	mainStatFlag := flag.String("MAIN_STAT", "", "Main stat to target for armor mode (STR, DEX, INT, LUK)")
	weaponTypeFlag := flag.String("type", "", "Weapon type for weapon mode (ATT, MATT)")
	flag.Parse()

	// Check if no parameters provided
	if len(flag.Args()) == 0 && *modeFlag == "" {
		fmt.Println("❌ Error: No parameters provided!")
		fmt.Println()
		fmt.Println("MapleStory Auto Flame Reroller - Usage Guide")
		fmt.Println("===========================================")
		fmt.Println()
		fmt.Println("🛡️  ARMOR MODE:")
		fmt.Println("   Target main stats (STR/DEX/INT/LUK) + All Stats")
		fmt.Println("   Stops when 2+ lines contain the main stat")
		fmt.Println()
		fmt.Println("   Examples:")
		fmt.Println("     ./maple_flame --mode=armor --MAIN_STAT=STR")
		fmt.Println("     ./maple_flame --mode=armor --MAIN_STAT=DEX")
		fmt.Println("     ./maple_flame --mode=armor --MAIN_STAT=INT")
		fmt.Println("     ./maple_flame --mode=armor --MAIN_STAT=LUK")
		fmt.Println()
		fmt.Println("⚔️  WEAPON MODE:")
		fmt.Println("   Target ATT/MATT + Boss Damage + Ignore Defense")
		fmt.Println("   Stops when 2+ weapon stat lines found")
		fmt.Println()
		fmt.Println("   Examples:")
		fmt.Println("     ./maple_flame --mode=weapon --type=ATT   (Physical weapons)")
		fmt.Println("     ./maple_flame --mode=weapon --type=MATT  (Magic weapons)")
		fmt.Println()
		fmt.Println("🎮 CONTROLS:")
		fmt.Println("   Ctrl+F1  - Stop gracefully")
		fmt.Println("   Ctrl+C   - Force quit")
		fmt.Println()
		fmt.Println("📁 OUTPUT:")
		fmt.Println("   temp/debug_ss_1.png - Latest screenshot")
		fmt.Println("   temp/flame.log      - Complete session log")
		fmt.Println()
		return
	}

	mode := strings.ToLower(strings.TrimSpace(*modeFlag))

	switch mode {
	case "armor", "armour":
		runArmorMode(*mainStatFlag)
	case "weapon":
		runWeaponMode(*weaponTypeFlag)
	default:
		fmt.Printf("❌ Error: Invalid mode '%s'\n", mode)
		fmt.Println("Usage:")
		fmt.Println("  Armor mode:  ./maple_flame --mode=armor --MAIN_STAT=STR")
		fmt.Println("  Weapon mode: ./maple_flame --mode=weapon --type=ATT")
		fmt.Println("               ./maple_flame --mode=weapon --type=MATT")
		return
	}
}

// runArmorMode runs the armor flame analysis (original functionality)
func runArmorMode(mainStatStr string) {
	fmt.Println("🛡️  ARMOR MODE")

	if mainStatStr == "" {
		fmt.Println("❌ Error: MAIN_STAT parameter required for armor mode!")
		fmt.Println("Usage: ./maple_flame --mode=armor --MAIN_STAT=STR/DEX/INT/LUK")
		return
	}

	// Convert string flag to MainStat enum
	MAIN_STAT, err := parseMainStat(mainStatStr)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		fmt.Println("Usage: ./maple_flame --mode=armor --MAIN_STAT=STR/DEX/INT/LUK")
		return
	}

	fmt.Printf("Target main stat: %s\n", MAIN_STAT)
	fmt.Println("Will stop when 2+ lines contain the main stat (including All Stats)")
	fmt.Println()

	// Step 1: Find MapleStory window
	fmt.Print("Finding MapleStory window... ")
	windowRect, err := window.GetMaplestoryWindow()
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		fmt.Println("Make sure MapleStory is running and visible.")
		return
	}
	fmt.Println("✅ Found!")

	// Screen region for flame stats (using global constants)
	fmt.Printf("Monitoring region %dx%d at (%d,%d)\n", CAPTURE_WIDTH, CAPTURE_HEIGHT, CAPTURE_X, CAPTURE_Y)
	fmt.Printf("Reroll click will be at offset (%d,%d) from window\n", CLICK_OFFSET_X, CLICK_OFFSET_Y)
	fmt.Printf("Absolute click position will be around (%d,%d)\n", 
		int(windowRect.Left)+CLICK_OFFSET_X, int(windowRect.Top)+CLICK_OFFSET_Y)
	fmt.Println("Starting auto-reroll... Press Ctrl+F1 to stop gracefully, or Ctrl+C to force quit")
	fmt.Println()

	attemptCount := 0
	var lastThreeTexts [3]string  // Store last 3 OCR results to detect stuck rerolls
	textIndex := 0

	for {
		attemptCount++
		fmt.Printf("=== Attempt #%d ===\n", attemptCount)

		// Check for Ctrl+F1 to stop gracefully
		if CheckStopKey() {
			fmt.Println("\n🛑 Ctrl+F1 pressed - stopping gracefully...")
			break
		}

		// Capture screenshot
		fmt.Print("Capturing... ")
		img, err := screenshot.CaptureScreenRegion(windowRect, CAPTURE_X, CAPTURE_Y, CAPTURE_WIDTH, CAPTURE_HEIGHT)
		if err != nil {
			fmt.Printf("❌ Screenshot failed: %v\n", err)
			continue
		}

		// Save for debugging (max 1 screenshot, always overwrites)
		filename, err := screenshot.SaveDebugImage(img, 1)
		if err != nil {
			fmt.Printf("❌ Save failed: %v\n", err)
			continue
		}
		fmt.Printf("✅ Saved: %s (latest)\n", filename)

		// Apply OCR
		fmt.Print("OCR... ")
		text, err := ocr.ExtractText(filename)
		if err != nil {
			fmt.Printf("❌ OCR failed: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
		fmt.Println("✅ Done")

		// Store this text result in our history for stuck detection
		lastThreeTexts[textIndex] = strings.TrimSpace(text)
		textIndex = (textIndex + 1) % 3

		// Check if stats are stuck (same for 3 consecutive attempts)
		if attemptCount >= 3 {
			if lastThreeTexts[0] == lastThreeTexts[1] && lastThreeTexts[1] == lastThreeTexts[2] && lastThreeTexts[0] != "" {
				fmt.Printf("\n⚠️ STUCK DETECTED: Stats haven't changed for 3 consecutive attempts!\n")
				fmt.Printf("Last OCR result: %s\n", lastThreeTexts[0])
				fmt.Println("🛑 Reroll mechanism may not be working - stopping script...")
				break
			}
		}

		// Check for main stat occurrences
		mainStatCount := countMainStatLines(text, MAIN_STAT)
		fmt.Printf("Text extracted:\n%s\n", text)
		fmt.Printf("%s + All Stats lines found: %d\n", MAIN_STAT, mainStatCount)

		// Check if we should stop (2+ main stat lines)
		if mainStatCount >= 2 {
			fmt.Printf("\n🎉 SUCCESS! Found %d lines with %s!\n", mainStatCount, MAIN_STAT)
			fmt.Println("Stopping reroll - good stats achieved!")
			break
		}

		// Not good enough, click to reroll
		fmt.Println("❌ Not enough main stat lines, rerolling...")
		triggerReroll(windowRect)

		// Wait a moment before next attempt
		time.Sleep(2 * time.Second)
	}
}

// runWeaponMode runs the weapon flame analysis 
func runWeaponMode(weaponTypeStr string) {
	fmt.Println("⚔️  WEAPON MODE")

	if weaponTypeStr == "" {
		fmt.Println("❌ Error: type parameter required for weapon mode!")
		fmt.Println("Usage: ./maple_flame --mode=weapon --type=ATT/MATT")
		return
	}

	weaponType := strings.ToUpper(strings.TrimSpace(weaponTypeStr))
	if weaponType != "ATT" && weaponType != "MATT" {
		fmt.Printf("❌ Error: Invalid weapon type '%s'\n", weaponType)
		fmt.Println("Usage: ./maple_flame --mode=weapon --type=ATT/MATT")
		return
	}

	fmt.Printf("Target weapon type: %s\n", weaponType)
	fmt.Println("Will stop when 2+ lines contain target type + BOSS DMG + IGN DEF")
	fmt.Println("(BOSS MONSTER DAMAGE and IGNORE DEFENSE are always desirable)")
	fmt.Println()

	// Find MapleStory window
	fmt.Print("Finding MapleStory window... ")
	windowRect, err := window.GetMaplestoryWindow()
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		fmt.Println("Make sure MapleStory is running and visible.")
		return
	}
	fmt.Println("✅ Found!")

	// Screen region for flame stats (using global constants)
	fmt.Printf("Monitoring region %dx%d at (%d,%d)\n", CAPTURE_WIDTH, CAPTURE_HEIGHT, CAPTURE_X, CAPTURE_Y)
	fmt.Printf("Reroll click will be at offset (%d,%d) from window\n", CLICK_OFFSET_X, CLICK_OFFSET_Y)
	fmt.Println("Starting auto-reroll... Press Ctrl+F1 to stop gracefully")
	fmt.Println()

	attemptCount := 0
	var lastThreeTexts [3]string
	textIndex := 0

	for {
		attemptCount++
		fmt.Printf("=== Attempt #%d ===\n", attemptCount)

		// Check for Ctrl+F1 to stop gracefully
		if CheckStopKey() {
			fmt.Println("\n🛑 Ctrl+F1 pressed - stopping gracefully...")
			break
		}

		// Capture screenshot
		fmt.Print("Capturing... ")
		img, err := screenshot.CaptureScreenRegion(windowRect, CAPTURE_X, CAPTURE_Y, CAPTURE_WIDTH, CAPTURE_HEIGHT)
		if err != nil {
			fmt.Printf("❌ Screenshot failed: %v\n", err)
			continue
		}

		// Save for debugging (max 1 screenshot, always overwrites)
		filename, err := screenshot.SaveDebugImage(img, 1)
		if err != nil {
			fmt.Printf("❌ Save failed: %v\n", err)
			continue
		}
		fmt.Printf("✅ Saved: %s (latest)\n", filename)

		// Apply OCR
		fmt.Print("OCR... ")
		text, err := ocr.ExtractText(filename)
		if err != nil {
			fmt.Printf("❌ OCR failed: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
		fmt.Println("✅ Done")

		// Store for stuck detection
		lastThreeTexts[textIndex] = strings.TrimSpace(text)
		textIndex = (textIndex + 1) % 3

		// Check if stuck
		if attemptCount >= 3 {
			if lastThreeTexts[0] == lastThreeTexts[1] && lastThreeTexts[1] == lastThreeTexts[2] && lastThreeTexts[0] != "" {
				fmt.Printf("\n⚠️ STUCK DETECTED: Stats haven't changed for 3 consecutive attempts!\n")
				fmt.Printf("Last OCR result: %s\n", lastThreeTexts[0])
				fmt.Println("🛑 Reroll mechanism may not be working - stopping script...")
				break
			}
		}

		// Check for weapon stat occurrences
		weaponStatCount := countWeaponStatLines(text, weaponType)
		fmt.Printf("Text extracted:\n%s\n", text)
		fmt.Printf("Weapon stats (%s + BOSS DMG + IGN DEF) found: %d\n", weaponType, weaponStatCount)

		// Check if we should stop (2+ weapon stat lines)
		if weaponStatCount >= 2 {
			fmt.Printf("\n🎉 SUCCESS! Found %d weapon stat lines!\n", weaponStatCount)
			fmt.Println("Stopping reroll - good stats achieved!")
			break
		}

		// Not good enough, click to reroll
		fmt.Println("❌ Not enough weapon stat lines, rerolling...")
		triggerReroll(windowRect)

		// Wait a moment before next attempt
		time.Sleep(2 * time.Second)
	}
}

// countMainStatLines counts how many lines contain the main stat or All Stats
func countMainStatLines(text string, mainStat MainStat) int {
	if text == "" {
		return 0
	}

	lines := strings.Split(text, "\n")
	count := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		upperLine := strings.ToUpper(line)
		
		// Check if line contains the main stat (case insensitive)
		if strings.Contains(upperLine, strings.ToUpper(mainStat.String())) {
			count++
		} else if strings.Contains(upperLine, "ALL STATS") || 
				  strings.Contains(upperLine, "ALL STAT") ||
				  strings.Contains(upperLine, "ALLSTATS") ||
				  strings.Contains(upperLine, "ALLSTAT") {
			// All Stats also counts as main stat since it boosts all stats
			count++
		}
	}

	return count
}

// countWeaponStatLines counts weapon-relevant stats (ATT/MATT + BOSS DMG + IGN DEF)
func countWeaponStatLines(text, weaponType string) int {
	if text == "" {
		return 0
	}

	lines := strings.Split(text, "\n")
	count := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		upperLine := strings.ToUpper(line)
		
		// Check for target weapon type (ATT or MATT) - more precise matching
		if weaponType == "ATT" {
			// Look for "ATT:" or "ATT " or "ATT%" to avoid matching words like "ATTACK"
			if (strings.Contains(upperLine, "ATT:") || 
				strings.Contains(upperLine, "ATT ") || 
				strings.Contains(upperLine, "ATT%")) && 
				!strings.Contains(upperLine, "MATT") {
				count++
			}
		} else if weaponType == "MATT" {
			// Look for "MATT:" or "MATT " or "MATT%"
			if strings.Contains(upperLine, "MATT:") || 
			   strings.Contains(upperLine, "MATT ") || 
			   strings.Contains(upperLine, "MATT%") {
				count++
			}
		}
		
		// Check for boss damage (always desirable)
		if strings.Contains(upperLine, "BOSS") && strings.Contains(upperLine, "DAMAGE") {
			// Boss Monster Damage is always desirable
			count++
		}
		
		// Check for ignore defense (always desirable)
		if strings.Contains(upperLine, "IGNORE") && strings.Contains(upperLine, "DEFENSE") {
			// Ignore Defense is always desirable (like All Stats for weapons)
			count++
		} else if strings.Contains(upperLine, "IGN") && strings.Contains(upperLine, "DEF") {
			// Alternative format for Ignore Defense
			count++
		}
	}

	return count
}

// triggerReroll clicks on a specific area and presses Enter twice to reroll
func triggerReroll(windowRect *window.WindowRect) {
	fmt.Print("Triggering reroll... ")

	// Calculate absolute screen coordinates using global constants
	clickX := int(windowRect.Left) + CLICK_OFFSET_X
	clickY := int(windowRect.Top) + CLICK_OFFSET_Y

	fmt.Printf("(Click at %d,%d) ", clickX, clickY)

	// Activate MapleStory window first
	_, err := window.FindAndActivateMaplestory()
	if err != nil {
		fmt.Printf("❌ Could not activate MapleStory: %v\n", err)
		return
	}

	time.Sleep(100 * time.Millisecond)

	// // Debug: Capture 20x20 pixel area around click position for debugging
	// fmt.Print("📷 Debug screenshot... ")
	// debugImg, err := screenshot.CaptureScreenRegion(windowRect, 
	// 	clickOffsetX-10, clickOffsetY-10, 50, 50)
	// if err != nil {
	// 	fmt.Printf("⚠️ Debug screenshot failed: %v ", err)
	// } else {
	// 	// debugFilename, err := screenshot.SaveDebugImageWithPrefix(debugImg, "click_debug", 1)
	// 	if err != nil {
	// 		fmt.Printf("⚠️ Debug save failed: %v ", err)
	// 	} else {
	// 		fmt.Printf("✅ Saved click debug: %s ", debugFilename)
	// 	}
	// }

	// Move cursor to click position
	ret, _, _ := procSetCursorPos.Call(uintptr(clickX), uintptr(clickY))
	if ret == 0 {
		fmt.Printf("❌ Failed to set cursor position\n")
		return
	}

	time.Sleep(100 * time.Millisecond)

	// Perform mouse click (left button down and up)
	procMouseEvent.Call(
		MOUSEEVENTF_LEFTDOWN,
		0, 0, 0, 0,
	)
	time.Sleep(50 * time.Millisecond)

	procMouseEvent.Call(
		MOUSEEVENTF_LEFTUP,
		0, 0, 0, 0,
	)

	fmt.Print("✅ Clicked! ")

	// Press Enter twice
	time.Sleep(200 * time.Millisecond) // Wait for click to register
	
	fmt.Print("Enter1... ")
	PressKey(VK_RETURN)
	
	time.Sleep(100 * time.Millisecond)
	
	fmt.Print("Enter2... ")
	PressKey(VK_RETURN)

	fmt.Println("✅ Complete!")
}

// pressSpacebar uses the working keybd_event method from git history
func pressSpacebar() {
	fmt.Print("Pressing Spacebar... ")

	// First, ensure MapleStory window is active
	_, err := window.FindAndActivateMaplestory()
	if err != nil {
		fmt.Printf("❌ Could not activate MapleStory: %v\n", err)
		return
	}

	// Wait for window to be focused
	time.Sleep(100 * time.Millisecond)

	// Use the working PressKey method from git history
	PressKey(VK_SPACE)

	fmt.Println("✅")
}

// pressEnter uses the working keybd_event method from git history
func pressEnter() {
	fmt.Print("Pressing Enter... ")

	// First, ensure MapleStory window is active
	_, err := window.FindAndActivateMaplestory()
	if err != nil {
		fmt.Printf("❌ Could not activate MapleStory: %v\n", err)
		return
	}

	// Wait for window to be focused
	time.Sleep(100 * time.Millisecond)

	// Use the working PressKey method from git history
	PressKey(VK_RETURN)

	fmt.Println("✅")
}

// PressKey simulates a key press using the working method from git history
func PressKey(keyCode int) {
	// Key down
	procKeyboardEvent.Call(
		uintptr(keyCode),
		0,
		0,
		0,
	)
	time.Sleep(50 * time.Millisecond)

	// Key up
	procKeyboardEvent.Call(
		uintptr(keyCode),
		0,
		2, // KEYEVENTF_KEYUP
		0,
	)
}

// CheckStopKey checks if the stop key combination (Ctrl+F1) is pressed
func CheckStopKey() bool {
	ctrlState, _, _ := procGetAsyncKeyState.Call(uintptr(VK_CONTROL))
	f1State, _, _ := procGetAsyncKeyState.Call(uintptr(VK_F1))
	
	// Check if Ctrl+F1 is pressed
	return ctrlState&0x8000 != 0 && f1State&0x8000 != 0
}

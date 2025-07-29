// Package automation provides functions for automation (mouse clicks, keyboard inputs)
package automation

import (
	"fmt"
	"syscall"
	"time"

	"maple_flame/goversion/internal/window"
)

var (
	user32                = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState  = user32.NewProc("GetAsyncKeyState")
	procSetCursorPos      = user32.NewProc("SetCursorPos")
	procMouseEvent        = user32.NewProc("mouse_event")
	procKeyboardEvent     = user32.NewProc("keybd_event")
)

const (
	// Mouse event constants
	MOUSEEVENTF_LEFTDOWN  = 0x0002
	MOUSEEVENTF_LEFTUP    = 0x0004
	
	// Key codes
	VK_CONTROL = 0x11
	VK_ESCAPE  = 0x1B
	VK_SHIFT   = 0x10
	VK_RETURN  = 0x0D
	VK_F1      = 0x70
)

// ClickRerollButton clicks the reroll button and presses enter
func ClickRerollButton(windowRect *window.WindowRect, offsetX, offsetY int) error {
	// Calculate click coordinates using provided offsets
	buttonX := int(windowRect.Left) + offsetX
	buttonY := int(windowRect.Top) + offsetY
	
	// Re-activate the window to ensure it's in focus
	_, err := window.FindAndActivateMaplestory()
	if err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond) // Give window time to come into focus
	
	// Set cursor position
	ret, _, _ := procSetCursorPos.Call(uintptr(buttonX), uintptr(buttonY))
	if ret == 0 {
		return fmt.Errorf("failed to set cursor position")
	}
	time.Sleep(300 * time.Millisecond)
	
	// Perform mouse click
	procMouseEvent.Call(
		MOUSEEVENTF_LEFTDOWN,
		0, 0, 0, 0,
	)
	time.Sleep(50 * time.Millisecond)
	procMouseEvent.Call(
		MOUSEEVENTF_LEFTUP,
		0, 0, 0, 0,
	)
	
	// Press ENTER multiple times
	time.Sleep(50 * time.Millisecond)
	PressKey(VK_RETURN)
	time.Sleep(50 * time.Millisecond)
	PressKey(VK_RETURN)
	time.Sleep(50 * time.Millisecond)
	PressKey(VK_RETURN)
	
	// Success
	time.Sleep(500 * time.Millisecond) // Wait for click to register
	
	return nil
}

// PressKey simulates a key press
func PressKey(keyCode int) {
	procKeyboardEvent.Call(
		uintptr(keyCode),
		0,
		0,
		0,
	)
	time.Sleep(50 * time.Millisecond)
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

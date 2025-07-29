// Package window provides functions for handling window operations for MapleStory
package window

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32                = syscall.NewLazyDLL("user32.dll")
	procFindWindow        = user32.NewProc("FindWindowW")
	procGetWindowRect     = user32.NewProc("GetWindowRect")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

// WindowRect represents a window rectangle
type WindowRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// GetMaplestoryWindow finds the MapleStory window and returns its rectangle
func GetMaplestoryWindow() (*WindowRect, error) {
	// Find the MapleStory window
	hwnd, _, _ := procFindWindow.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("MapleStory"))),
	)

	if hwnd == 0 {
		return nil, fmt.Errorf("MapleStory window not found")
	}

	// Get the window rectangle
	var rect WindowRect
	ret, _, _ := procGetWindowRect.Call(
		hwnd,
		uintptr(unsafe.Pointer(&rect)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("failed to get window rectangle")
	}

	// Activate the window
	procSetForegroundWindow.Call(hwnd)

	return &rect, nil
}

// FindAndActivateMaplestory finds and activates the MapleStory window
func FindAndActivateMaplestory() (uintptr, error) {
	hwnd, _, _ := procFindWindow.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("MapleStory"))),
	)

	if hwnd == 0 {
		return 0, fmt.Errorf("MapleStory window not found")
	}

	// Set as foreground window
	ret, _, _ := procSetForegroundWindow.Call(hwnd)
	if ret == 0 {
		return 0, fmt.Errorf("failed to activate MapleStory window")
	}

	return hwnd, nil
}

// Package screenshot provides functions for capturing screenshots
package screenshot

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"maple_flame/goversion/internal/window"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	gdi32                = syscall.NewLazyDLL("gdi32.dll")
	procGetDC            = user32.NewProc("GetDC")
	procReleaseDC        = user32.NewProc("ReleaseDC")
	procDeleteDC         = gdi32.NewProc("DeleteDC")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procBitBlt           = gdi32.NewProc("BitBlt")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procGetDIBits        = gdi32.NewProc("GetDIBits")
)

const (
	SRCCOPY = 0x00CC0020
)

// CaptureScreenRegion captures a specific region of the screen
func CaptureScreenRegion(windowRect *window.WindowRect, regionX, regionY, width, height int) (*image.RGBA, error) {
	// Calculate absolute coordinates
	x := int(windowRect.Left) + regionX
	y := int(windowRect.Top) + regionY

	// Get device context for entire screen
	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return nil, fmt.Errorf("failed to get DC for screen")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	// Create compatible DC
	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, fmt.Errorf("failed to create compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	// Create compatible bitmap
	hBitmap, _, _ := procCreateCompatibleBitmap.Call(hdcScreen, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		return nil, fmt.Errorf("failed to create compatible bitmap")
	}
	defer procDeleteObject.Call(hBitmap)

	// Select bitmap into DC
	procSelectObject.Call(hdcMem, hBitmap)

	// Copy screen to bitmap
	procBitBlt.Call(
		hdcMem,
		0, 0,
		uintptr(width), uintptr(height),
		hdcScreen,
		uintptr(x), uintptr(y),
		SRCCOPY,
	)

	// Create image to hold bitmap data
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Set up bitmap info header
	type BITMAPINFOHEADER struct {
		BiSize          uint32
		BiWidth         int32
		BiHeight        int32
		BiPlanes        uint16
		BiBitCount      uint16
		BiCompression   uint32
		BiSizeImage     uint32
		BiXPelsPerMeter int32
		BiYPelsPerMeter int32
		BiClrUsed       uint32
		BiClrImportant  uint32
	}

	bmi := BITMAPINFOHEADER{
		BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
		BiWidth:       int32(width),
		BiHeight:      -int32(height), // Negative height for top-down DIB
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: 0, // BI_RGB
	}

	// Get bitmap bits into our image
	procGetDIBits.Call(
		hdcMem,
		hBitmap,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&img.Pix[0])),
		uintptr(unsafe.Pointer(&bmi)),
		0, // DIB_RGB_COLORS
	)

	return img, nil
}

const maxScreenshots = 7

// SaveDebugImage saves a screenshot with a try number for debugging
// and maintains a FIFO queue of screenshots (max 7)
func SaveDebugImage(img *image.RGBA, tryNumber int) (string, error) {
	// Create temp directory if it doesn't exist
	tempDir := filepath.Join(".", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Create filename with try number
	filename := filepath.Join(tempDir, fmt.Sprintf("debug_ss_%d.png", tryNumber))

	// Create file
	f, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %v", err)
	}
	defer f.Close()

	// Encode and save
	if err := png.Encode(f, img); err != nil {
		return "", fmt.Errorf("failed to encode image: %v", err)
	}

	// Clean up old screenshots if we're beyond the max
	if tryNumber > maxScreenshots {
		// Remove the oldest screenshot (tryNumber - maxScreenshots)
		oldFile := filepath.Join(tempDir, fmt.Sprintf("debug_ss_%d.png", tryNumber-maxScreenshots))
		if err := os.Remove(oldFile); err != nil && !os.IsNotExist(err) {
			// Just log the error but don't fail the operation
			fmt.Printf("Warning: Failed to remove old screenshot: %v\n", err)
		}
	}

	return filename, nil
}

// SaveDebugImageWithPrefix saves a screenshot with a prefix and try number for debugging
// Used for flame scoring to distinguish between "before" and "after" images
func SaveDebugImageWithPrefix(img *image.RGBA, prefix string, tryNumber int) (string, error) {
	// Create temp directory if it doesn't exist
	tempDir := filepath.Join(".", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Create filename with prefix and try number
	filename := filepath.Join(tempDir, fmt.Sprintf("%s_flame_%d.png", prefix, tryNumber))

	// Create file
	f, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %v", err)
	}
	defer f.Close()

	// Encode and save
	if err := png.Encode(f, img); err != nil {
		return "", fmt.Errorf("failed to encode image: %v", err)
	}

	// Clean up old screenshots if we're beyond the max
	if tryNumber > maxScreenshots {
		// Remove the oldest screenshot (tryNumber - maxScreenshots)
		oldFile := filepath.Join(tempDir, fmt.Sprintf("%s_flame_%d.png", prefix, tryNumber-maxScreenshots))
		if err := os.Remove(oldFile); err != nil && !os.IsNotExist(err) {
			// Just log the error but don't fail the operation
			fmt.Printf("Warning: Failed to remove old screenshot: %v\n", err)
		}
	}

	return filename, nil
}

// CombineImagesHorizontal combines two images side by side (left + right)
// Used specifically for flame scoring to show before/after comparison
func CombineImagesHorizontal(leftImg, rightImg *image.RGBA, tryNumber int) (string, error) {
	// Get dimensions
	leftBounds := leftImg.Bounds()
	rightBounds := rightImg.Bounds()
	
	// Calculate combined dimensions
	combinedWidth := leftBounds.Dx() + rightBounds.Dx()
	combinedHeight := leftBounds.Dy()
	if rightBounds.Dy() > combinedHeight {
		combinedHeight = rightBounds.Dy()
	}
	
	// Create combined image
	combined := image.NewRGBA(image.Rect(0, 0, combinedWidth, combinedHeight))
	
	// Copy left image to left side
	for y := 0; y < leftBounds.Dy(); y++ {
		for x := 0; x < leftBounds.Dx(); x++ {
			combined.Set(x, y, leftImg.At(x, y))
		}
	}
	
	// Copy right image to right side
	for y := 0; y < rightBounds.Dy(); y++ {
		for x := 0; x < rightBounds.Dx(); x++ {
			combined.Set(x+leftBounds.Dx(), y, rightImg.At(x, y))
		}
	}
	
	// Create temp directory if it doesn't exist
	tempDir := filepath.Join(".", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Create filename with try number
	filename := filepath.Join(tempDir, fmt.Sprintf("combined_flame_%d.png", tryNumber))

	// Create file
	f, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create combined image file: %v", err)
	}
	defer f.Close()

	// Encode and save
	if err := png.Encode(f, combined); err != nil {
		return "", fmt.Errorf("failed to encode combined image: %v", err)
	}

	return filename, nil
}

// EnhanceImageForOCR enhances an image for better OCR accuracy by upscaling and sharpening
func EnhanceImageForOCR(img *image.RGBA, scaleFactor int) *image.RGBA {
	if scaleFactor <= 1 {
		scaleFactor = 3 // Default 3x upscaling
	}
	
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()
	
	newWidth := originalWidth * scaleFactor
	newHeight := originalHeight * scaleFactor
	
	// Create enlarged image using nearest neighbor for crisp edges
	enlarged := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Map back to original coordinates
			origX := x / scaleFactor
			origY := y / scaleFactor
			
			// Ensure we don't go out of bounds
			if origX >= originalWidth {
				origX = originalWidth - 1
			}
			if origY >= originalHeight {
				origY = originalHeight - 1
			}
			
			enlarged.Set(x, y, img.At(origX, origY))
		}
	}
	
	// Apply sharpening filter
	sharpened := applySharpeningFilter(enlarged)
	
	// Convert to high contrast (helpful for small text)
	enhanced := enhanceContrast(sharpened)
	
	return enhanced
}

// applySharpeningFilter applies a 3x3 sharpening kernel to enhance edges
func applySharpeningFilter(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	result := image.NewRGBA(bounds)
	
	// Sharpening kernel
	kernel := [3][3]float64{
		{0, -1, 0},
		{-1, 5, -1},
		{0, -1, 0},
	}
	
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			var r, g, b float64
			
			// Apply convolution
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					pixel := img.RGBAAt(x+kx, y+ky)
					weight := kernel[ky+1][kx+1]
					
					r += float64(pixel.R) * weight
					g += float64(pixel.G) * weight
					b += float64(pixel.B) * weight
				}
			}
			
			// Clamp values to valid range
			if r < 0 { r = 0 }
			if r > 255 { r = 255 }
			if g < 0 { g = 0 }
			if g > 255 { g = 255 }
			if b < 0 { b = 0 }
			if b > 255 { b = 255 }
			
			result.Set(x, y, color.RGBA{
				R: uint8(r),
				G: uint8(g), 
				B: uint8(b),
				A: 255,
			})
		}
	}
	
	// Copy border pixels
	for y := 0; y < height; y++ {
		result.Set(0, y, img.At(0, y))
		result.Set(width-1, y, img.At(width-1, y))
	}
	for x := 0; x < width; x++ {
		result.Set(x, 0, img.At(x, 0))
		result.Set(x, height-1, img.At(x, height-1))
	}
	
	return result
}

// enhanceContrast enhances contrast to make text more readable
func enhanceContrast(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)
	
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := img.RGBAAt(x, y)
			
			// Convert to grayscale for better text recognition
			gray := uint8((uint16(pixel.R)*299 + uint16(pixel.G)*587 + uint16(pixel.B)*114) / 1000)
			
			// Apply contrast enhancement - make bright pixels brighter, dark pixels darker
			var enhanced uint8
			if gray > 128 {
				// Bright pixels - make brighter
				enhanced = uint8(float64(gray)*1.2)
				if enhanced > 255 {
					enhanced = 255
				}
			} else {
				// Dark pixels - make darker
				enhanced = uint8(float64(gray)*0.8)
			}
			
			result.Set(x, y, color.RGBA{
				R: enhanced,
				G: enhanced,
				B: enhanced,
				A: 255,
			})
		}
	}
	
	return result
}

// LightEnhanceForOCR applies light enhancement (2x upscale + gentle sharpening) for OCR
func LightEnhanceForOCR(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()
	
	// 2x upscale using nearest neighbor
	newWidth := originalWidth * 2
	newHeight := originalHeight * 2
	
	enlarged := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			origX := x / 2
			origY := y / 2
			
			if origX >= originalWidth {
				origX = originalWidth - 1
			}
			if origY >= originalHeight {
				origY = originalHeight - 1
			}
			
			enlarged.Set(x, y, img.At(origX, origY))
		}
	}
	
	// Apply very light sharpening
	return lightSharpen(enlarged)
}

// lightSharpen applies a gentle sharpening filter
func lightSharpen(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	result := image.NewRGBA(bounds)
	
	// Light sharpening kernel (less aggressive)
	kernel := [3][3]float64{
		{0, -0.5, 0},
		{-0.5, 3, -0.5},
		{0, -0.5, 0},
	}
	
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			var r, g, b float64
			
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					pixel := img.RGBAAt(x+kx, y+ky)
					weight := kernel[ky+1][kx+1]
					
					r += float64(pixel.R) * weight
					g += float64(pixel.G) * weight
					b += float64(pixel.B) * weight
				}
			}
			
			// Clamp values
			if r < 0 { r = 0 }
			if r > 255 { r = 255 }
			if g < 0 { g = 0 }
			if g > 255 { g = 255 }
			if b < 0 { b = 0 }
			if b > 255 { b = 255 }
			
			result.Set(x, y, color.RGBA{
				R: uint8(r),
				G: uint8(g), 
				B: uint8(b),
				A: 255,
			})
		}
	}
	
	// Copy border pixels
	for y := 0; y < height; y++ {
		result.Set(0, y, img.At(0, y))
		result.Set(width-1, y, img.At(width-1, y))
	}
	for x := 0; x < width; x++ {
		result.Set(x, 0, img.At(x, 0))
		result.Set(x, height-1, img.At(x, height-1))
	}
	
	return result
}
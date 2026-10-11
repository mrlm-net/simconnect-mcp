//go:build windows

// Package capture takes a picture of the simulator's window: PrintWindow
// with PW_RENDERFULLCONTENT, so a DirectX window behind others is drawn
// too (not a minimized one).
package capture

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	enumWindows              = user32.NewProc("EnumWindows")
	getWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	isWindowVisible          = user32.NewProc("IsWindowVisible")
	isIconic                 = user32.NewProc("IsIconic")
	getClientRect            = user32.NewProc("GetClientRect")
	getWindowRect            = user32.NewProc("GetWindowRect")
	printWindow              = user32.NewProc("PrintWindow")
	getDC                    = user32.NewProc("GetDC")
	releaseDC                = user32.NewProc("ReleaseDC")
	createCompatibleDC       = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBitmap   = gdi32.NewProc("CreateCompatibleBitmap")
	selectObject             = gdi32.NewProc("SelectObject")
	getDIBits                = gdi32.NewProc("GetDIBits")
	deleteObject             = gdi32.NewProc("DeleteObject")
	deleteDC                 = gdi32.NewProc("DeleteDC")
	openProcess              = kernel32.NewProc("OpenProcess")
	queryFullProcessImage    = kernel32.NewProc("QueryFullProcessImageNameW")
	closeHandle              = kernel32.NewProc("CloseHandle")
)

// SimProcesses are the simulators' executables, in the order looked for.
var SimProcesses = []string{"FlightSimulator2024.exe", "FlightSimulator.exe"}

// ErrNoWindow: no visible simulator window (not running, or minimized).
var ErrNoWindow = errors.New("no visible simulator window (not running, or minimized)")

type rect struct{ Left, Top, Right, Bottom int32 }

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// processName is the executable of process pid ("" unknown).
func processName(pid uint32) string {
	const queryLimited = 0x1000
	h, _, _ := openProcess.Call(queryLimited, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer closeHandle.Call(h)
	buf := make([]uint16, 520)
	n := uint32(len(buf))
	if r, _, _ := queryFullProcessImage.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))); r == 0 {
		return ""
	}
	return filepath.Base(syscall.UTF16ToString(buf[:n]))
}

// simWindow is the largest visible top-level window of a simulator
// process; 0 none.
func simWindow() (uintptr, error) {
	var best uintptr
	var bestArea int64
	var minimized bool
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if v, _, _ := isWindowVisible.Call(hwnd); v == 0 {
			return 1
		}
		var pid uint32
		getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		name := processName(pid)
		match := false
		for _, s := range SimProcesses {
			if strings.EqualFold(name, s) {
				match = true
			}
		}
		if !match {
			return 1
		}
		if ic, _, _ := isIconic.Call(hwnd); ic != 0 {
			minimized = true
			return 1
		}
		var r rect
		getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		if a := int64(r.Right-r.Left) * int64(r.Bottom-r.Top); a > bestArea {
			best, bestArea = hwnd, a
		}
		return 1
	})
	enumWindows.Call(cb, 0)
	if best == 0 {
		if minimized {
			return 0, fmt.Errorf("the simulator window is minimized")
		}
		return 0, ErrNoWindow
	}
	return best, nil
}

// Window is the simulator window's client area as an image.
func Window() (image.Image, error) {
	hwnd, err := simWindow()
	if err != nil {
		return nil, err
	}
	var r rect
	getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := int(r.Right-r.Left), int(r.Bottom-r.Top)
	if w <= 0 || h <= 0 {
		return nil, ErrNoWindow
	}
	screen, _, _ := getDC.Call(0)
	defer releaseDC.Call(0, screen)
	mem, _, _ := createCompatibleDC.Call(screen)
	defer deleteDC.Call(mem)
	bmp, _, _ := createCompatibleBitmap.Call(screen, uintptr(w), uintptr(h))
	if bmp == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap failed")
	}
	defer deleteObject.Call(bmp)
	old, _, _ := selectObject.Call(mem, bmp)
	const pwClientOnly, pwRenderFullContent = 0x1, 0x2
	ok, _, _ := printWindow.Call(hwnd, mem, pwClientOnly|pwRenderFullContent)
	selectObject.Call(mem, old)
	if ok == 0 {
		return nil, fmt.Errorf("PrintWindow failed")
	}
	hdr := bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32}
	px := make([]byte, w*h*4)
	if n, _, _ := getDIBits.Call(mem, bmp, 0, uintptr(h), uintptr(unsafe.Pointer(&px[0])), uintptr(unsafe.Pointer(&hdr)), 0); n == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(px); i += 4 { // BGRA → RGBA, opaque
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = px[i+2], px[i+1], px[i], 255
	}
	return img, nil
}

// Scale is img at most maxW pixels wide (nearest neighbour); itself when
// already within, or maxW <= 0.
func Scale(img image.Image, maxW int) image.Image {
	b := img.Bounds()
	if maxW <= 0 || b.Dx() <= maxW {
		return img
	}
	h := b.Dy() * maxW / b.Dx()
	out := image.NewRGBA(image.Rect(0, 0, maxW, h))
	for y := 0; y < h; y++ {
		sy := b.Min.Y + y*b.Dy()/h
		for x := 0; x < maxW; x++ {
			out.Set(x, y, img.At(b.Min.X+x*b.Dx()/maxW, sy))
		}
	}
	return out
}

// Encode is img as "png" or "jpeg" (quality 1–100).
func Encode(img image.Image, format string, quality int) ([]byte, string, error) {
	var buf bytes.Buffer
	switch format {
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/png", nil
	default:
		if quality <= 0 || quality > 100 {
			quality = 85
		}
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/jpeg", nil
	}
}

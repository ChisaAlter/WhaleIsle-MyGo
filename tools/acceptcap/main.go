// Acceptcap captures a HWND (or finds by PID / title substring) via PrintWindow.
// Usage:
//
//	acceptcap -pid 1234 -title Launcher -out L-HOME.png -w 1000 -h 680
//	acceptcap -hwnd 0x1234 -out shot.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	gdi32                        = syscall.NewLazyDLL("gdi32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procGetClientRect            = user32.NewProc("GetClientRect")
	procClientToScreen           = user32.NewProc("ClientToScreen")
	procMoveWindow               = user32.NewProc("MoveWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procPrintWindow              = user32.NewProc("PrintWindow")
	procGetDpiForWindow          = user32.NewProc("GetDpiForWindow")
	procGetDC                    = user32.NewProc("GetDC")
	procReleaseDC                = user32.NewProc("ReleaseDC")
	procCreateCompatibleDC       = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap   = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject             = gdi32.NewProc("SelectObject")
	procDeleteObject             = gdi32.NewProc("DeleteObject")
	procDeleteDC                 = gdi32.NewProc("DeleteDC")
	procBitBlt                   = gdi32.NewProc("BitBlt")
	procGetDIBits                = gdi32.NewProc("GetDIBits")
)

type rect struct{ Left, Top, Right, Bottom int32 }

func main() {
	// Match the target app's coordinate space so GetClientRect/MoveWindow and
	// BitBlt/PrintWindow all use physical pixels at 144+ DPI.
	if shcore := syscall.NewLazyDLL("shcore.dll"); shcore != nil {
		if p := shcore.NewProc("SetProcessDpiAwareness"); p != nil {
			// PROCESS_PER_MONITOR_DPI_AWARE = 2
			_, _, _ = p.Call(2)
		}
	}

	pid := flag.Uint("pid", 0, "process id")
	title := flag.String("title", "", "title substring")
	hwndFlag := flag.String("hwnd", "", "hwnd hex like 0x1234")
	out := flag.String("out", "shot.png", "output png")
	w := flag.Int("w", 0, "resize width (0=keep)")
	h := flag.Int("h", 0, "resize height (0=keep)")
	flag.Parse()

	var hwnd uintptr
	if *hwndFlag != "" {
		raw := strings.TrimSpace(*hwndFlag)
		base := 10
		if strings.HasPrefix(strings.ToLower(raw), "0x") {
			raw = raw[2:]
			base = 16
		}
		v, err := strconv.ParseUint(raw, base, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hwnd: %v\n", err)
			os.Exit(2)
		}
		hwnd = uintptr(v)
	} else if *pid != 0 {
		hwnd = findWindow(uint32(*pid), *title)
	} else {
		fmt.Fprintln(os.Stderr, "need -pid or -hwnd")
		os.Exit(2)
	}
	if hwnd == 0 {
		fmt.Fprintln(os.Stderr, "window not found")
		os.Exit(1)
	}
	procShowWindow.Call(hwnd, 9)
	// HWND_TOPMOST then HWND_NOTOPMOST keeps us above overlapping terminals
	// without leaving the acceptance window permanently topmost.
	const hwndTopmost = ^uintptr(0) // -1
	const hwndNotopmost = ^uintptr(1) // -2
	const swpNoMoveNoSize = 0x0001 | 0x0002
	procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMoveNoSize)
	procSetForegroundWindow.Call(hwnd)
	if *w > 0 && *h > 0 {
		// Callers pass logical acceptance sizes. With per-monitor DPI awareness
		// above, MoveWindow expects physical pixels.
		dpi := uint32(96)
		if r1, _, _ := procGetDpiForWindow.Call(hwnd); r1 != 0 {
			dpi = uint32(r1)
		}
		scale := float64(dpi) / 96.0
		pw := int(float64(*w) * scale)
		ph := int(float64(*h) * scale)
		apply := func(ww, hh int) {
			procMoveWindow.Call(hwnd, 40, 40, uintptr(ww), uintptr(hh), 1)
			time.Sleep(350 * time.Millisecond)
			procMoveWindow.Call(hwnd, 40, 40, uintptr(ww), uintptr(hh), 1)
			time.Sleep(150 * time.Millisecond)
		}
		apply(pw, ph)
		// Nudge outer size until client area matches the physical target
		// (frameless vs bordered shells differ by ~10–20px chrome).
		for i := 0; i < 6; i++ {
			var cr rect
			procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
			cw := int(cr.Right - cr.Left)
			ch := int(cr.Bottom - cr.Top)
			if abs(cw-pw) <= 1 && abs(ch-ph) <= 1 {
				break
			}
			var wr rect
			procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
			ow := int(wr.Right - wr.Left)
			oh := int(wr.Bottom - wr.Top)
			nw := ow + (pw - cw)
			nh := oh + (ph - ch)
			if nw < 2 || nh < 2 {
				break
			}
			procMoveWindow.Call(hwnd, 40, 40, uintptr(nw), uintptr(nh), 1)
			time.Sleep(120 * time.Millisecond)
		}
	}
	img, err := capture(hwnd)
	procSetWindowPos.Call(hwnd, hwndNotopmost, 0, 0, 0, 0, swpNoMoveNoSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b := img.Bounds()
	mid := img.At(b.Dx()/2, b.Dy()/2)
	r, g, bl, _ := mid.RGBA()
	fi, _ := f.Stat()
	fmt.Printf("ok hwnd=0x%X %dx%d mid=%d,%d,%d bytes=%d\n", hwnd, b.Dx(), b.Dy(), r>>8, g>>8, bl>>8, fi.Size())
}

func findWindow(pid uint32, titlePart string) uintptr {
	var found uintptr
	var fallback uintptr
	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var p uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&p)))
		vis, _, _ := procIsWindowVisible.Call(hwnd)
		if p != pid || vis == 0 {
			return 1
		}
		buf := make([]uint16, 256)
		procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 256)
		name := syscall.UTF16ToString(buf)
		if titlePart != "" {
			if !containsFold(name, titlePart) {
				return 1
			}
			// Prefer exact title so "Whale Isle" does not match "Whale Isle Launcher".
			if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(titlePart)) {
				found = hwnd
				return 0
			}
			if fallback == 0 {
				fallback = hwnd
			}
			return 1
		} else if name == "" {
			return 1
		}
		found = hwnd
		return 0
	})
	procEnumWindows.Call(cb, 0)
	if found != 0 {
		return found
	}
	return fallback
}

func containsFold(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (indexFold(s, sub) >= 0))
}

func indexFold(s, sub string) int {
	sl := []rune(s)
	su := []rune(sub)
	for i := 0; i+len(su) <= len(sl); i++ {
		ok := true
		for j := 0; j < len(su); j++ {
			a, b := sl[i+j], su[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

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

type point struct{ X, Y int32 }

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func capture(hwnd uintptr) (image.Image, error) {
	// Prefer client area: MyGo frameless windows BitBlt with GetWindowRect
	// often include a strip of desktop above the painted chrome.
	var cr rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
	pt := point{X: cr.Left, Y: cr.Top}
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
	w := int(cr.Right - cr.Left)
	h := int(cr.Bottom - cr.Top)
	srcX, srcY := int(pt.X), int(pt.Y)
	if w < 2 || h < 2 {
		var r rect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		w = int(r.Right - r.Left)
		h = int(r.Bottom - r.Top)
		srcX, srcY = int(r.Left), int(r.Top)
	}
	if w < 2 || h < 2 {
		return nil, fmt.Errorf("bad size %dx%d", w, h)
	}
	hdcScreen, _, _ := procGetDC.Call(0)
	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	hbmp, _, _ := procCreateCompatibleBitmap.Call(hdcScreen, uintptr(w), uintptr(h))
	procSelectObject.Call(hdcMem, hbmp)
	// Prefer PrintWindow with PW_RENDERFULLCONTENT so layered/transparent
	// MyGo chrome (title actions) is composited into the bitmap. Electron
	// often returns a successful but all-black PrintWindow — fall back to
	// BitBlt of the screen client area in that case.
	const pwRenderFullContent = 2
	const srcCopy = 0x00CC0020
	okPW, _, _ := procPrintWindow.Call(hwnd, hdcMem, pwRenderFullContent)
	bi := bitmapInfoHeader{
		Size:     40,
		Width:    int32(w),
		Height:   -int32(h),
		Planes:   1,
		BitCount: 32,
	}
	buf := make([]byte, w*h*4)
	procGetDIBits.Call(hdcMem, hbmp, 0, uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bi)), 0)
	if okPW == 0 || mostlyBlackBGRA(buf) {
		procBitBlt.Call(hdcMem, 0, 0, uintptr(w), uintptr(h), hdcScreen, uintptr(srcX), uintptr(srcY), srcCopy)
		procGetDIBits.Call(hdcMem, hbmp, 0, uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bi)), 0)
	}
	procDeleteObject.Call(hbmp)
	procDeleteDC.Call(hdcMem)
	procReleaseDC.Call(0, hdcScreen)

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			// BGRA
			img.Pix[i+0] = buf[i+2]
			img.Pix[i+1] = buf[i+1]
			img.Pix[i+2] = buf[i+0]
			img.Pix[i+3] = 255
		}
	}
	return img, nil
}

func mostlyBlackBGRA(buf []byte) bool {
	if len(buf) < 16 {
		return true
	}
	step := 4 * 64
	if step >= len(buf) {
		step = 4
	}
	nonzero := 0
	samples := 0
	for i := 0; i+2 < len(buf); i += step {
		samples++
		if buf[i] != 0 || buf[i+1] != 0 || buf[i+2] != 0 {
			nonzero++
			if nonzero > samples/20 {
				return false
			}
		}
	}
	return samples == 0 || nonzero*20 <= samples
}

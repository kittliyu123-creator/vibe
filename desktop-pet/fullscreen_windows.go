//go:build windows

package main

import (
	"image"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	fullDWM      = syscall.NewLazyDLL("dwmapi.dll")
	fullFrame    = fullDWM.NewProc("DwmGetWindowAttribute")
	fullClass    = winUser32.NewProc("GetClassNameW")
	fullVisible  = winUser32.NewProc("IsWindowVisible")
	fullIconic   = winUser32.NewProc("IsIconic")
	fullZoomed   = winUser32.NewProc("IsZoomed")
	fullStyle    = winUser32.NewProc("GetWindowLongPtrW")
	fullEnum     = winUser32.NewProc("EnumWindows")
	fullAtPoint  = winUser32.NewProc("WindowFromPoint")
	fullAncestor = winUser32.NewProc("GetAncestor")
	fullAppBar   = winShell32.NewProc("SHAppBarMessage")
)

func fullWindowClass(window uintptr) string {
	var name [256]uint16
	fullClass.Call(window, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)))
	return syscall.UTF16ToString(name[:])
}

func fullRect(r winRect) image.Rectangle {
	return image.Rectangle{Min: image.Pt(int(r.Left), int(r.Top)), Max: image.Pt(int(r.Right), int(r.Bottom))}
}

func fullVisualBounds(window uintptr) (image.Rectangle, bool) {
	var r winRect
	if fullFrame.Find() == nil {
		// Extended bounds exclude invisible resize borders on maximized apps.
		hr, _, _ := fullFrame.Call(window, 9, uintptr(unsafe.Pointer(&r)), unsafe.Sizeof(r))
		if int32(hr) >= 0 && !fullRect(r).Empty() {
			return fullRect(r), true
		}
	}
	ok, _, _ := winGetWindowRect.Call(window, uintptr(unsafe.Pointer(&r)))
	return fullRect(r), ok != 0 && !fullRect(r).Empty()
}

func fullCloaked(window uintptr) bool {
	if fullFrame.Find() != nil {
		return false
	}
	var value uint32
	hr, _, _ := fullFrame.Call(window, 14, uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value))
	return int32(hr) >= 0 && value != 0
}

// Taskbar windows can retain WS_VISIBLE while completely covered by a
// fullscreen app. Hit-test several real screen points instead of trusting it.
func fullTaskbarVisible(window uintptr, monitor image.Rectangle) bool {
	visible, _, _ := fullVisible.Call(window)
	if visible == 0 || fullCloaked(window) {
		return false
	}
	var wr winRect
	if ok, _, _ := winGetWindowRect.Call(window, uintptr(unsafe.Pointer(&wr))); ok == 0 {
		return false
	}
	r := fullRect(wr).Intersect(monitor)
	if r.Empty() || min(r.Dx(), r.Dy()) < 8 || max(r.Dx(), r.Dy()) < 64 {
		return false
	}
	for _, part := range []float64{.06, .22, .4, .58, .76, .94} {
		x, y := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
		if r.Dx() >= r.Dy() {
			x = r.Min.X + int(float64(r.Dx()-1)*part)
		} else {
			y = r.Min.Y + int(float64(r.Dy()-1)*part)
		}
		// POINT is passed by value in the Windows x64 calling convention.
		pointValue := uintptr(uint64(uint32(int32(x))) | uint64(uint32(int32(y)))<<32)
		at, _, _ := fullAtPoint.Call(pointValue)
		root, _, _ := fullAncestor.Call(at, 2) // GA_ROOT, includes tray child controls.
		if at == window || root == window {
			return true
		}
	}
	return false
}

type fullAppBarData struct {
	Size    uint32
	Window  uintptr
	Message uint32
	Edge    uint32
	Rect    winRect
	Param   uintptr
}

// One stable callback avoids leaking a syscall callback allocation every poll.
// The locked UI thread supplies the synchronous enumeration's result pointer.
type fullTaskbarScan struct {
	monitor image.Rectangle
	visible bool
}

var fullScanTarget *fullTaskbarScan
var fullEnumCallback = syscall.NewCallback(func(window, param uintptr) uintptr {
	scan := fullScanTarget
	if scan == nil {
		return 0
	}
	class := fullWindowClass(window)
	if class != "Shell_TrayWnd" && class != "Shell_SecondaryTrayWnd" {
		return 1
	}
	if fullTaskbarVisible(window, scan.monitor) {
		scan.visible = true
		return 0
	}
	return 1
})

func winFullscreenSnapshot(petWindow uintptr) fullscreenSnapshot {
	s := fullscreenSnapshot{}
	monitor, _, _ := winMonitorFromWindow.Call(petWindow, 2)
	info := winMonitorInfo{Size: uint32(unsafe.Sizeof(winMonitorInfo{}))}
	if ok, _, _ := winGetMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return s
	}
	s.Monitor = fullRect(info.Monitor)
	foreground, _, _ := winGetForegroundWindow.Call()
	if foreground == 0 || foreground == petWindow {
		return s
	}
	root, _, _ := fullAncestor.Call(foreground, 2)
	if root != 0 {
		foreground = root
	}
	class := fullWindowClass(foreground)
	if class == "" || class == winClassName || class == "#32768" {
		return s
	}
	s.Valid = true
	switch strings.ToLower(class) {
	case "progman", "workerw", "#32769", "shell_traywnd", "shell_secondarytraywnd":
		return s
	}
	visible, _, _ := fullVisible.Call(foreground)
	minimized, _, _ := fullIconic.Call(foreground)
	if visible == 0 || minimized != 0 || fullCloaked(foreground) {
		return s
	}
	var ok bool
	s.Foreground, ok = fullVisualBounds(foreground)
	if !ok {
		s.Valid = false
		return s
	}
	s.ForegroundIsApp = true
	s.SameMonitor = !s.Foreground.Intersect(s.Monitor).Empty()
	if !s.SameMonitor {
		return s
	}
	bar := fullAppBarData{Size: uint32(unsafe.Sizeof(fullAppBarData{}))}
	state, _, _ := fullAppBar.Call(4, uintptr(unsafe.Pointer(&bar))) // ABM_GETSTATE
	s.TaskbarAutoHide = state&1 != 0
	maximized, _, _ := fullZoomed.Call(foreground)
	style, _, _ := fullStyle.Call(foreground, ^uintptr(15)) // GWL_STYLE = -16
	s.DecoratedMaximized = maximized != 0 && style&(0x00C00000|0x00040000) != 0
	scan := fullTaskbarScan{monitor: s.Monitor}
	fullScanTarget = &scan
	enumerated, _, _ := fullEnum.Call(fullEnumCallback, 0)
	fullScanTarget = nil
	if enumerated == 0 && !scan.visible {
		s.Valid = false
		return s
	}
	s.TaskbarVisible = scan.visible
	return s
}

func (h *winHost) effectivelyHidden() bool { return h.hidden || h.fullscreen.Hidden }

func (h *winHost) applyVisibility() {
	if h.effectivelyHidden() {
		if h.pressed {
			h.pressed = false
			winReleaseCapture.Call()
			h.app.SetDragging(false)
		}
		winShowWindow.Call(h.window, 0)
		return
	}
	// Restore in place, preserve the user's z-order preference and keyboard focus.
	order := ^uintptr(1)
	if h.topmost {
		order = ^uintptr(0)
	}
	winSetWindowPos.Call(h.window, order, 0, 0, 0, 0, 0x0001|0x0002|0x0010|0x0040)
	h.lastTick = time.Now()
	h.lastPaint = time.Time{}
}

func (h *winHost) checkFullscreen(now time.Time, force bool) {
	if h.menuOpen || (!force && !h.lastFullscreenCheck.IsZero() && now.Sub(h.lastFullscreenCheck) < 250*time.Millisecond) {
		return
	}
	h.lastFullscreenCheck = now
	wasHidden := h.effectivelyHidden()
	h.fullscreen.Update(winFullscreenSnapshot(h.window), now)
	if wasHidden != h.effectivelyHidden() {
		h.applyVisibility()
	}
}

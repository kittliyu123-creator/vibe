//go:build windows

package main

// The Windows host deliberately uses only the standard library. The pet is a
// real, per-pixel transparent desktop window; no browser/runtime is installed.
import (
	"fmt"
	"image"
	"math"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	winUser32   = syscall.NewLazyDLL("user32.dll")
	winGDI32    = syscall.NewLazyDLL("gdi32.dll")
	winShell32  = syscall.NewLazyDLL("shell32.dll")
	winKernel32 = syscall.NewLazyDLL("kernel32.dll")

	winRegisterClass       = winUser32.NewProc("RegisterClassExW")
	winUnregisterClass     = winUser32.NewProc("UnregisterClassW")
	winCreateWindow        = winUser32.NewProc("CreateWindowExW")
	winDefWindowProc       = winUser32.NewProc("DefWindowProcW")
	winDestroyWindow       = winUser32.NewProc("DestroyWindow")
	winShowWindow          = winUser32.NewProc("ShowWindow")
	winSetWindowPos        = winUser32.NewProc("SetWindowPos")
	winGetWindowRect       = winUser32.NewProc("GetWindowRect")
	winGetMessage          = winUser32.NewProc("GetMessageW")
	winTranslateMessage    = winUser32.NewProc("TranslateMessage")
	winDispatchMessage     = winUser32.NewProc("DispatchMessageW")
	winPostQuitMessage     = winUser32.NewProc("PostQuitMessage")
	winPostMessage         = winUser32.NewProc("PostMessageW")
	winSetTimer            = winUser32.NewProc("SetTimer")
	winKillTimer           = winUser32.NewProc("KillTimer")
	winGetDC               = winUser32.NewProc("GetDC")
	winReleaseDC           = winUser32.NewProc("ReleaseDC")
	winUpdateLayered       = winUser32.NewProc("UpdateLayeredWindow")
	winSetCapture          = winUser32.NewProc("SetCapture")
	winReleaseCapture      = winUser32.NewProc("ReleaseCapture")
	winGetCursorPos        = winUser32.NewProc("GetCursorPos")
	winLoadCursor          = winUser32.NewProc("LoadCursorW")
	winLoadIcon            = winUser32.NewProc("LoadIconW")
	winCreatePopupMenu     = winUser32.NewProc("CreatePopupMenu")
	winAppendMenu          = winUser32.NewProc("AppendMenuW")
	winTrackPopupMenu      = winUser32.NewProc("TrackPopupMenu")
	winDestroyMenu         = winUser32.NewProc("DestroyMenu")
	winSetForegroundWindow = winUser32.NewProc("SetForegroundWindow")
	winGetForegroundWindow = winUser32.NewProc("GetForegroundWindow")
	winFindWindow          = winUser32.NewProc("FindWindowW")
	winSystemParameters    = winUser32.NewProc("SystemParametersInfoW")
	winGetSystemMetrics    = winUser32.NewProc("GetSystemMetrics")
	winMonitorFromWindow   = winUser32.NewProc("MonitorFromWindow")
	winMonitorFromRect     = winUser32.NewProc("MonitorFromRect")
	winGetMonitorInfo      = winUser32.NewProc("GetMonitorInfoW")
	winSetDPIContext       = winUser32.NewProc("SetProcessDpiAwarenessContext")
	winSetDPIAware         = winUser32.NewProc("SetProcessDPIAware")
	winGetDPIForWindow     = winUser32.NewProc("GetDpiForWindow")
	winGetDPIForSystem     = winUser32.NewProc("GetDpiForSystem")
	winRegisterMessage     = winUser32.NewProc("RegisterWindowMessageW")
	winMessageBox          = winUser32.NewProc("MessageBoxW")
	winCreateCompatibleDC  = winGDI32.NewProc("CreateCompatibleDC")
	winCreateDIBSection    = winGDI32.NewProc("CreateDIBSection")
	winSelectObject        = winGDI32.NewProc("SelectObject")
	winDeleteObject        = winGDI32.NewProc("DeleteObject")
	winDeleteDC            = winGDI32.NewProc("DeleteDC")
	winNotifyIcon          = winShell32.NewProc("Shell_NotifyIconW")
	winGetModuleHandle     = winKernel32.NewProc("GetModuleHandleW")
	winCreateMutex         = winKernel32.NewProc("CreateMutexW")
	winCloseHandle         = winKernel32.NewProc("CloseHandle")
)

const (
	winClassName  = "VibeDesktopPetWindow2026"
	winTitle      = "桌边小伙伴"
	winTrayMsg    = 0x8001
	winRestoreMsg = 0x8002
	winTimerID    = 1

	winWMDestroy        = 0x0002
	winWMClose          = 0x0010
	winWMSettingChange  = 0x001A
	winWMDisplayChange  = 0x007E
	winWMMouseActivate  = 0x0021
	winWMMouseMove      = 0x0200
	winWMLButtonDown    = 0x0201
	winWMLButtonUp      = 0x0202
	winWMLButtonDbl     = 0x0203
	winWMRButtonUp      = 0x0205
	winWMCaptureChanged = 0x0215
	winWMTimer          = 0x0113
	winWMDPIChanged     = 0x02E0

	winMenuPet     = 100
	winMenuFeed    = 101
	winMenuPlay    = 102
	winMenuSleep   = 103
	winMenuWave    = 104
	winMenuStretch = 105
	winMenuFollow  = 106
	winMenuWander  = 107
	winMenuTopmost = 110
	winMenuSmall   = 120
	winMenuNormal  = 121
	winMenuLarge   = 122
	winMenuReset   = 130
	winMenuHide    = 131
	winMenuRestore = 132
	winMenuExit    = 199
)

type winPoint struct{ X, Y int32 }
type winSize struct{ Width, Height int32 }
type winRect struct{ Left, Top, Right, Bottom int32 }
type winMessage struct {
	Window  uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   winPoint
	Private uint32
}
type winClass struct {
	Size        uint32
	Style       uint32
	WndProc     uintptr
	ClassExtra  int32
	WindowExtra int32
	Instance    uintptr
	Icon        uintptr
	Cursor      uintptr
	Background  uintptr
	MenuName    *uint16
	ClassName   *uint16
	SmallIcon   uintptr
}
type winBitmapInfo struct {
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
	Colors        [1]uint32
}
type winBlend struct {
	Operation     byte
	Flags         byte
	ConstantAlpha byte
	AlphaFormat   byte
}
type winMonitorInfo struct {
	Size    uint32
	Monitor winRect
	Work    winRect
	Flags   uint32
}
type winNotifyData struct {
	Size             uint32
	Window           uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GUID             [16]byte
	BalloonIcon      uintptr
}

type winHost struct {
	app                 *PetApp
	window              uintptr
	instance            uintptr
	icon                uintptr
	dc                  uintptr
	bitmap              uintptr
	previousBitmap      uintptr
	pixels              []byte
	width               int
	height              int
	x                   int
	y                   int
	scale               float64
	dpi                 float64
	topmost             bool
	hidden              bool
	fullscreen          fullscreenGate
	lastFullscreenCheck time.Time
	menuOpen            bool
	pressed             bool
	dragged             bool
	pressedControl      bool
	controlCanceled     bool
	moveCarryX          float64
	moveCarryY          float64
	settingsPath        string
	downCursor          winPoint
	downWindow          winPoint
	lastTick            time.Time
	lastPaint           time.Time
	taskbarCreated      uint32
	lastError           error
}

// A single UI instance owns all callbacks. Win32 dispatches them on the locked
// thread, so the model and backing bitmap never need concurrent access.
var winActiveHost *winHost

func winUTF16(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }

// GUI-subsystem executables have no visible stderr. Surface actionable startup
// errors (for example, a damaged custom pet.png) instead of silently exiting.
func showError(err error) {
	if err == nil {
		return
	}
	winMessageBox.Call(0,
		uintptr(unsafe.Pointer(winUTF16("小伙伴暂时没能启动：\n\n"+err.Error()+"\n\n如果替换过 pet.png，请移走该文件后重试。"))),
		uintptr(unsafe.Pointer(winUTF16(winTitle))), 0x10)
}

func runNative(app *PetApp) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	mutex, _, mutexErr := winCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(winUTF16("Local\\VibeDesktopPet2026"))))
	if mutex == 0 {
		return fmt.Errorf("创建实例锁: %w", mutexErr)
	}
	defer winCloseHandle.Call(mutex)
	if mutexErr == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		other, _, _ := winFindWindow.Call(uintptr(unsafe.Pointer(winUTF16(winClassName))), 0)
		if other != 0 {
			winPostMessage.Call(other, winRestoreMsg, 0, 0)
		}
		return nil
	}

	if winSetDPIContext.Find() == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is the pseudo-handle -4.
		winSetDPIContext.Call(^uintptr(3))
	} else {
		winSetDPIAware.Call()
	}

	settingsPath, _ := petSettingsPath()
	settings, _ := loadPetSettings(settingsPath)
	host := &winHost{app: app, scale: settings.Scale, dpi: 96, topmost: settings.Topmost,
		lastTick: time.Now(), settingsPath: settingsPath}
	app.Follow, app.Wander = settings.Follow, settings.Wander
	winActiveHost = host
	defer func() { winActiveHost = nil }()
	host.instance, _, _ = winGetModuleHandle.Call(0)
	// A resource with ID 1 can be embedded by the build. The standard icon is a
	// safe fallback when running a locally compiled development executable.
	host.icon, _, _ = winLoadIcon.Call(host.instance, 1)
	if host.icon == 0 {
		host.icon, _, _ = winLoadIcon.Call(0, 32512)
	}
	cursor, _, _ := winLoadCursor.Call(0, 32649) // IDC_HAND
	className := winUTF16(winClassName)
	class := winClass{
		Size: uint32(unsafe.Sizeof(winClass{})), Style: 0x0008, // CS_DBLCLKS
		WndProc: syscall.NewCallback(winWindowProcedure), Instance: host.instance,
		Icon: host.icon, SmallIcon: host.icon, Cursor: cursor, ClassName: className,
	}
	if atom, _, err := winRegisterClass.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return fmt.Errorf("注册桌宠窗口: %w", err)
	}
	defer winUnregisterClass.Call(uintptr(unsafe.Pointer(className)), host.instance)
	if winGetDPIForSystem.Find() == nil {
		if dpi, _, _ := winGetDPIForSystem.Call(); dpi > 0 {
			host.dpi = float64(dpi)
		}
	}
	host.width, host.height = host.dimensions()
	if settings.HasPosition {
		host.x, host.y = settings.X, settings.Y
		host.clampPosition(host.workArea())
	} else {
		host.resetPosition(false)
	}
	extendedStyle := uintptr(0x00080000 | 0x00000080 | 0x08000000) // LAYERED | TOOLWINDOW | NOACTIVATE
	if host.topmost {
		extendedStyle |= 0x00000008
	}
	window, _, err := winCreateWindow.Call(
		extendedStyle,
		uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(winUTF16(winTitle))),
		0x80000000, // WS_POPUP
		uintptr(host.x), uintptr(host.y), uintptr(host.width), uintptr(host.height),
		0, 0, host.instance, 0,
	)
	if window == 0 {
		return fmt.Errorf("创建透明窗口: %w", err)
	}
	host.window = window
	defer winDestroyWindow.Call(window)
	if winGetDPIForWindow.Find() == nil {
		if dpi, _, _ := winGetDPIForWindow.Call(window); dpi > 0 {
			host.dpi = float64(dpi)
		}
	}
	host.width, host.height = host.dimensions()
	if settings.HasPosition {
		host.clampPosition(host.workArea())
	} else {
		host.resetPosition(false)
	}
	host.dc, _, err = winCreateCompatibleDC.Call(0)
	if host.dc == 0 {
		return fmt.Errorf("创建绘图上下文: %w", err)
	}
	defer host.releaseGraphics()
	if err = host.createBitmap(); err != nil {
		return err
	}
	if err = host.paint(); err != nil {
		return err
	}
	initialSnapshot := winFullscreenSnapshot(window)
	host.fullscreen.Update(initialSnapshot, time.Now())
	host.fullscreen.Hidden = shouldHideForFullscreen(initialSnapshot)
	host.applyVisibility()
	host.addTrayIcon()
	defer host.removeTrayIcon()
	if message, _, _ := winRegisterMessage.Call(uintptr(unsafe.Pointer(winUTF16("TaskbarCreated")))); message != 0 {
		host.taskbarCreated = uint32(message)
	}
	if timer, _, err := winSetTimer.Call(window, winTimerID, 33, 0); timer == 0 {
		return fmt.Errorf("启动动画时钟: %w", err)
	}
	defer winKillTimer.Call(window, winTimerID)
	var msg winMessage
	for {
		result, _, err := winGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) == -1 {
			return fmt.Errorf("处理窗口消息: %w", err)
		}
		if result == 0 {
			break
		}
		winTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		winDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
	return host.lastError
}

func (h *winHost) dimensions() (int, int) {
	ratio := h.scale * h.dpi / 96
	return int(float64(canvasWidth)*ratio + .5), int(float64(canvasHeight)*ratio + .5)
}

func (h *winHost) workArea() winRect {
	var monitor uintptr
	if h.window != 0 {
		monitor, _, _ = winMonitorFromWindow.Call(h.window, 2) // nearest monitor
	} else {
		// Resolve a saved position before creating the window. This preserves a
		// secondary monitor placement and recovers safely after unplugging it.
		rect := winRect{int32(h.x), int32(h.y), int32(h.x + h.width), int32(h.y + h.height)}
		monitor, _, _ = winMonitorFromRect.Call(uintptr(unsafe.Pointer(&rect)), 2)
	}
	info := winMonitorInfo{Size: uint32(unsafe.Sizeof(winMonitorInfo{}))}
	if ok, _, _ := winGetMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); ok != 0 {
		return info.Work
	}
	var area winRect
	if ok, _, _ := winSystemParameters.Call(0x0030, 0, uintptr(unsafe.Pointer(&area)), 0); ok != 0 {
		return area
	}
	w, _, _ := winGetSystemMetrics.Call(0)
	v, _, _ := winGetSystemMetrics.Call(1)
	return winRect{Right: int32(w), Bottom: int32(v)}
}

func (h *winHost) savePreferences() {
	// Preference failures must not interrupt an interaction or prevent exit.
	// Missing/corrupt settings always recover to safe defaults on next launch.
	_ = savePetSettings(h.settingsPath, petSettings{
		Version: 2, HasPosition: true, X: h.x, Y: h.y, Scale: h.scale,
		Topmost: h.topmost, Follow: h.app.Follow, Wander: h.app.Wander,
	})
}

func (h *winHost) pointerOnCanvas() (float64, float64) {
	var cursor winPoint
	winGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	ratio := h.scale * h.dpi / 96
	return float64(int(cursor.X)-h.x) / ratio, float64(int(cursor.Y)-h.y) / ratio
}

func (h *winHost) resetPosition(move bool) {
	area := h.workArea()
	h.x = int(area.Right) - h.width - 24
	h.y = int(area.Bottom) - h.height - 12
	h.clampPosition(area)
	if move {
		h.moveWindow()
	}
}

func (h *winHost) clampPosition(area winRect) {
	if h.x+h.width > int(area.Right) {
		h.x = int(area.Right) - h.width
	}
	if h.y+h.height > int(area.Bottom) {
		h.y = int(area.Bottom) - h.height
	}
	if h.x < int(area.Left) {
		h.x = int(area.Left)
	}
	if h.y < int(area.Top) {
		h.y = int(area.Top)
	}
}

func (h *winHost) moveWindow() {
	winSetWindowPos.Call(h.window, 0, uintptr(h.x), uintptr(h.y), 0, 0, 0x0001|0x0004|0x0010) // no size/z-order/activation
}

func (h *winHost) createBitmap() error {
	if h.bitmap != 0 {
		winSelectObject.Call(h.dc, h.previousBitmap)
		winDeleteObject.Call(h.bitmap)
		h.bitmap = 0
		h.pixels = nil
	}
	info := winBitmapInfo{Size: 40, Width: int32(h.width), Height: -int32(h.height), Planes: 1, BitCount: 32}
	var bits unsafe.Pointer
	bitmap, _, err := winCreateDIBSection.Call(h.dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 || bits == nil {
		return fmt.Errorf("创建透明位图: %w", err)
	}
	h.bitmap = bitmap
	h.previousBitmap, _, _ = winSelectObject.Call(h.dc, bitmap)
	h.pixels = unsafe.Slice((*byte)(bits), h.width*h.height*4)
	return nil
}

func (h *winHost) releaseGraphics() {
	if h.bitmap != 0 {
		winSelectObject.Call(h.dc, h.previousBitmap)
		winDeleteObject.Call(h.bitmap)
		h.bitmap = 0
		h.pixels = nil
	}
	if h.dc != 0 {
		winDeleteDC.Call(h.dc)
		h.dc = 0
	}
}

func (h *winHost) paint() error {
	h.lastPaint = time.Now()
	source := h.app.Render()
	if source == nil {
		return nil
	}
	winCopyPremultipliedBGRA(h.pixels, h.width, h.height, source)
	position := winPoint{int32(h.x), int32(h.y)}
	size := winSize{int32(h.width), int32(h.height)}
	origin := winPoint{}
	blend := winBlend{ConstantAlpha: 255, AlphaFormat: 1}
	if ok, _, err := winUpdateLayered.Call(h.window, 0,
		uintptr(unsafe.Pointer(&position)), uintptr(unsafe.Pointer(&size)), h.dc,
		uintptr(unsafe.Pointer(&origin)), 0, uintptr(unsafe.Pointer(&blend)), 2); ok == 0 {
		return fmt.Errorf("更新透明画面: %w", err)
	}
	return nil
}

func (h *winHost) paintDue(now time.Time) bool {
	// Keep input, gaze and movement updates on the regular 33 ms timer while
	// drawing quiet idle at roughly 15 fps and quiet sleep at roughly 10 fps.
	// Small timing margins avoid skipping an extra tick due to timer jitter.
	interval := 30 * time.Millisecond
	quiet := !h.app.Hover && h.app.ControlsAlpha < .01 && h.app.Bubble == ""
	if quiet && h.app.Sleeping() {
		interval = 92 * time.Millisecond
	} else if quiet && h.app.Mode == "idle" {
		interval = 60 * time.Millisecond
	}
	return now.Sub(h.lastPaint) >= interval
}

// Go image.RGBA already stores premultiplied values, as UpdateLayeredWindow
// requires. Interpolate those values together to keep soft illustration edges
// smooth at Windows display scales without introducing dark alpha fringes.
func winCopyPremultipliedBGRA(dst []byte, width, height int, src *image.RGBA) {
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	if sw == 0 || sh == 0 {
		clear(dst)
		return
	}
	if width == sw && height == sh {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				si, di := y*src.Stride+x*4, (y*width+x)*4
				dst[di], dst[di+1], dst[di+2], dst[di+3] = src.Pix[si+2], src.Pix[si+1], src.Pix[si], src.Pix[si+3]
			}
		}
		return
	}
	type sample struct{ low, high, weight int }
	xs := make([]sample, width)
	axisSample := func(at, size, source int) sample {
		v := (float64(at)+.5)*float64(source)/float64(size) - .5
		v = math.Max(0, math.Min(float64(source-1), v))
		low := int(v)
		return sample{low, min(low+1, source-1), int((v-float64(low))*256 + .5)}
	}
	for x := range xs {
		xs[x] = axisSample(x, width, sw)
	}
	for y := 0; y < height; y++ {
		ys := axisSample(y, height, sh)
		for x, sx := range xs {
			i00, i10 := ys.low*src.Stride+sx.low*4, ys.low*src.Stride+sx.high*4
			i01, i11 := ys.high*src.Stride+sx.low*4, ys.high*src.Stride+sx.high*4
			di := (y*width + x) * 4
			for destination, channel := range [...]int{2, 1, 0, 3} {
				top := int(src.Pix[i00+channel])*(256-sx.weight) + int(src.Pix[i10+channel])*sx.weight
				bottom := int(src.Pix[i01+channel])*(256-sx.weight) + int(src.Pix[i11+channel])*sx.weight
				dst[di+destination] = byte((top*(256-ys.weight) + bottom*ys.weight + 32768) >> 16)
			}
		}
	}
}

func (h *winHost) addTrayIcon() {
	data := winNotifyData{Size: uint32(unsafe.Sizeof(winNotifyData{})), Window: h.window, ID: 1,
		Flags: 1 | 2 | 4, CallbackMessage: winTrayMsg, Icon: h.icon}
	copy(data.Tip[:], syscall.StringToUTF16("桌边小伙伴 · 点击召回，右键互动 / 退出"))
	winNotifyIcon.Call(0, uintptr(unsafe.Pointer(&data))) // NIM_ADD
}

func (h *winHost) removeTrayIcon() {
	data := winNotifyData{Size: uint32(unsafe.Sizeof(winNotifyData{})), Window: h.window, ID: 1}
	winNotifyIcon.Call(2, uintptr(unsafe.Pointer(&data))) // NIM_DELETE
}

func (h *winHost) restore() {
	h.hidden = false
	h.checkFullscreen(time.Now(), true)
	h.clampPosition(h.workArea())
	h.moveWindow()
	h.applyVisibility()
	if !h.effectivelyHidden() {
		h.app.Action("hello")
	}
	h.lastTick = time.Now()
}

func (h *winHost) menu() {
	h.menuOpen = true
	defer func() { h.menuOpen = false }()
	menu, _, _ := winCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer winDestroyMenu.Call(menu)
	add := func(id int, label string, checked bool) {
		flags := uintptr(0)
		if checked {
			flags |= 8
		} // MF_CHECKED
		winAppendMenu.Call(menu, flags, uintptr(id), uintptr(unsafe.Pointer(winUTF16(label))))
	}
	separator := func() { winAppendMenu.Call(menu, 0x0800, 0, 0) }
	add(winMenuPet, "摸摸脑袋", false)
	add(winMenuFeed, "喂一口零食", false)
	add(winMenuPlay, "一起玩耍", false)
	add(winMenuWave, "挥挥小手", false)
	add(winMenuStretch, "歪歪头", false)
	if h.app.Sleeping() {
		add(winMenuSleep, "轻轻叫醒", false)
	} else {
		add(winMenuSleep, "让它睡一会儿", false)
	}
	separator()
	add(winMenuFollow, "跟着鼠标走", h.app.Follow)
	add(winMenuWander, "自己散散步", h.app.Wander)
	separator()
	add(winMenuTopmost, "始终显示在最前面", h.topmost)
	add(winMenuSmall, "小号  80%", h.scale == .8)
	add(winMenuNormal, "标准  100%", h.scale == 1)
	add(winMenuLarge, "大号  125%", h.scale == 1.25)
	separator()
	add(winMenuReset, "回到屏幕右下角", false)
	if h.hidden {
		add(winMenuRestore, "显示小伙伴", false)
	} else {
		add(winMenuHide, "暂时收起到托盘", false)
	}
	separator()
	add(winMenuExit, "退出小伙伴", false)
	var cursor winPoint
	winGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	previousForeground, _, _ := winGetForegroundWindow.Call()
	winSetForegroundWindow.Call(h.window)
	choice, _, _ := winTrackPopupMenu.Call(menu, 0x0100|0x0002, uintptr(cursor.X), uintptr(cursor.Y), 0, h.window, 0)
	winPostMessage.Call(h.window, 0, 0, 0) // documented foreground-menu dismissal fix
	if previousForeground != 0 && previousForeground != h.window {
		winSetForegroundWindow.Call(previousForeground)
	}
	switch int(choice) {
	case winMenuPet:
		h.app.Action("pet")
	case winMenuFeed:
		h.app.Action("feed")
	case winMenuPlay:
		h.app.Action("play")
	case winMenuWave:
		h.app.Action("wave")
	case winMenuStretch:
		h.app.Action("stretch")
	case winMenuFollow:
		h.app.Action("follow")
	case winMenuWander:
		h.app.Action("wander")
	case winMenuSleep:
		if h.app.Sleeping() {
			h.app.Action("wake")
		} else {
			h.app.Action("sleep")
		}
	case winMenuTopmost:
		h.topmost = !h.topmost
		insertAfter := ^uintptr(1) // HWND_NOTOPMOST (-2)
		if h.topmost {
			insertAfter = ^uintptr(0)
		}
		winSetWindowPos.Call(h.window, insertAfter, 0, 0, 0, 0, 0x0001|0x0002|0x0010)
	case winMenuSmall:
		h.resize(.8, h.dpi)
	case winMenuNormal:
		h.resize(1, h.dpi)
	case winMenuLarge:
		h.resize(1.25, h.dpi)
	case winMenuReset:
		h.resetPosition(true)
		h.restore()
	case winMenuHide:
		h.hidden = true
		h.applyVisibility()
	case winMenuRestore:
		h.restore()
	case winMenuExit:
		winDestroyWindow.Call(h.window)
	}
	if choice != 0 && choice != winMenuExit {
		h.savePreferences()
	}
}

func (h *winHost) resize(scale, dpi float64) {
	oldWidth, oldHeight := h.width, h.height
	h.scale, h.dpi = scale, dpi
	h.width, h.height = h.dimensions()
	h.x += (oldWidth - h.width) / 2
	h.y += oldHeight - h.height
	h.clampPosition(h.workArea())
	if err := h.createBitmap(); err != nil {
		h.fail(err)
		return
	}
	if err := h.paint(); err != nil {
		h.fail(err)
	}
}

func (h *winHost) fail(err error) {
	h.lastError = err
	winDestroyWindow.Call(h.window)
}

func winWindowProcedure(window uintptr, message uint32, wParam, lParam uintptr) uintptr {
	h := winActiveHost
	if h == nil {
		result, _, _ := winDefWindowProc.Call(window, uintptr(message), wParam, lParam)
		return result
	}
	if h.taskbarCreated != 0 && message == h.taskbarCreated {
		h.addTrayIcon()
		return 0
	}
	switch message {
	case winWMMouseActivate:
		return 3 // MA_NOACTIVATE: deliver clicks while preserving keyboard focus.
	case winWMTimer:
		if wParam == winTimerID {
			now := time.Now()
			dt := now.Sub(h.lastTick).Seconds()
			h.lastTick = now
			if dt > .1 {
				dt = .1
			}
			h.checkFullscreen(now, false)
			if h.effectivelyHidden() {
				return 0
			}
			px, py := h.pointerOnCanvas()
			h.app.SetPointer(px, py)
			dx, dy := h.app.Tick(dt)
			if !h.pressed && (dx != 0 || dy != 0) {
				ratio := h.scale * h.dpi / 96
				h.moveCarryX += float64(dx) * ratio
				h.moveCarryY += float64(dy) * ratio
				mx, my := int(math.Trunc(h.moveCarryX)), int(math.Trunc(h.moveCarryY))
				h.moveCarryX -= float64(mx)
				h.moveCarryY -= float64(my)
				h.x += mx
				h.y += my
				wantedX := h.x
				h.clampPosition(h.workArea())
				if h.app.Wander && h.x != wantedX {
					h.app.Facing = -h.app.Facing
					h.moveCarryX = 0
				}
			}
			if h.paintDue(now) {
				if err := h.paint(); err != nil {
					h.fail(err)
				}
			}
		}
		return 0
	case winWMLButtonDown, winWMLButtonDbl:
		h.pressed, h.dragged = true, false
		px, py := h.pointerOnCanvas()
		h.app.SetPointer(px, py)
		h.pressedControl = h.app.ToolbarHit(px, py)
		h.controlCanceled = false
		winGetCursorPos.Call(uintptr(unsafe.Pointer(&h.downCursor)))
		h.downWindow = winPoint{int32(h.x), int32(h.y)}
		winSetCapture.Call(window)
		return 0
	case winWMMouseMove:
		if h.pressed {
			var cursor winPoint
			winGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
			dx, dy := cursor.X-h.downCursor.X, cursor.Y-h.downCursor.Y
			if h.pressedControl {
				if dx*dx+dy*dy >= 16 {
					h.controlCanceled = true
				}
				return 0
			}
			if !h.dragged && (dx*dx+dy*dy >= 16) {
				h.dragged = true
				h.app.SetDragging(true)
			}
			if h.dragged {
				h.x = int(h.downWindow.X + dx)
				h.y = int(h.downWindow.Y + dy)
				h.moveWindow()
			}
		}
		return 0
	case winWMLButtonUp:
		if h.pressed {
			dragged := h.dragged
			h.pressed = false
			winReleaseCapture.Call()
			h.app.SetDragging(false)
			if !dragged {
				px, py := h.pointerOnCanvas()
				h.app.SetPointer(px, py)
				if !h.controlCanceled && !h.app.ActionAt(px, py) && !h.pressedControl {
					h.app.Action("pet")
				}
			} else {
				h.clampPosition(h.workArea())
				h.moveWindow()
				h.savePreferences()
			}
		}
		return 0
	case winWMCaptureChanged:
		if h.pressed {
			h.pressed = false
			h.app.SetDragging(false)
			h.clampPosition(h.workArea())
			h.moveWindow()
			h.savePreferences()
		}
		return 0
	case winWMRButtonUp:
		h.menu()
		return 0
	case winTrayMsg:
		switch uint32(lParam) {
		case winWMLButtonUp, winWMLButtonDbl:
			h.restore()
		case winWMRButtonUp:
			h.menu()
		}
		return 0
	case winRestoreMsg:
		h.restore()
		return 0
	case winWMDPIChanged:
		if h.dc != 0 {
			h.resize(h.scale, float64(wParam&0xffff))
		}
		return 0
	case winWMDisplayChange, winWMSettingChange:
		if h.window != 0 {
			h.clampPosition(h.workArea())
			h.moveWindow()
		}
	case winWMClose:
		winDestroyWindow.Call(window)
		return 0
	case winWMDestroy:
		h.savePreferences()
		winKillTimer.Call(window, winTimerID)
		winPostQuitMessage.Call(0)
		return 0
	}
	result, _, _ := winDefWindowProc.Call(window, uintptr(message), wParam, lParam)
	return result
}

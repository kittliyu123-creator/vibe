//go:build windows

package main

import (
	"image"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// This test creates a simulated shell taskbar and changes window z-order. It is
// deliberately opt-in and Wine-only, so a normal Windows go test never touches
// the user's desktop or competes with Explorer's real taskbar.
func TestWindowsFullscreenSnapshotIntegration(t *testing.T) {
	if os.Getenv("PET_FULLSCREEN_INTEGRATION") != "1" {
		t.Skip("set PET_FULLSCREEN_INTEGRATION=1 in an isolated Wine desktop")
	}
	if syscall.NewLazyDLL("ntdll.dll").NewProc("wine_get_version").Find() != nil {
		t.Skip("simulated shell fixture is restricted to an isolated Wine desktop")
	}
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)

	if window, _, _ := winFindWindow.Call(uintptr(unsafe.Pointer(winUTF16("Shell_TrayWnd"))), 0); window != 0 {
		t.Skip("Wine already has a shell taskbar; use an isolated desktop without Explorer")
	}
	if winSetDPIContext.Find() == nil {
		winSetDPIContext.Call(^uintptr(3))
	}
	instance, _, _ := winGetModuleHandle.Call(0)
	callback := syscall.NewCallback(func(window uintptr, message uint32, wParam, lParam uintptr) uintptr {
		result, _, _ := winDefWindowProc.Call(window, uintptr(message), wParam, lParam)
		return result
	})
	register := func(name string) *uint16 {
		t.Helper()
		className := winUTF16(name)
		class := winClass{Size: uint32(unsafe.Sizeof(winClass{})), WndProc: callback,
			Instance: instance, ClassName: className, Background: 6}
		if atom, _, err := winRegisterClass.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
			t.Fatalf("register fixture class %s: %v", name, err)
		}
		t.Cleanup(func() { winUnregisterClass.Call(uintptr(unsafe.Pointer(className)), instance) })
		return className
	}
	appClass := register("PetFullscreenIntegrationApp")
	petClass := register("PetFullscreenIntegrationPet")
	taskbarClass := register("Shell_TrayWnd")
	create := func(class *uint16, title string, rect image.Rectangle, extended uintptr) uintptr {
		t.Helper()
		window, _, err := winCreateWindow.Call(extended, uintptr(unsafe.Pointer(class)),
			uintptr(unsafe.Pointer(winUTF16(title))), 0x80000000,
			uintptr(rect.Min.X), uintptr(rect.Min.Y), uintptr(rect.Dx()), uintptr(rect.Dy()),
			0, 0, instance, 0)
		if window == 0 {
			t.Fatalf("create %s fixture: %v", title, err)
		}
		t.Cleanup(func() { winDestroyWindow.Call(window) })
		winShowWindow.Call(window, 4)
		return window
	}
	w, _, _ := winGetSystemMetrics.Call(0)
	h, _, _ := winGetSystemMetrics.Call(1)
	monitor := image.Rect(0, 0, int(w), int(h))
	if monitor.Dx() < 640 || monitor.Dy() < 480 {
		t.Fatalf("test desktop too small: %v", monitor)
	}
	work := image.Rect(0, 0, monitor.Max.X, monitor.Max.Y-48)
	app := create(appClass, "Fullscreen fixture", work, 0)
	pet := create(petClass, "Pet fixture", image.Rect(50, 50, 170, 170), 0x08000000|0x80|0x8)
	taskbar := create(taskbarClass, "Simulated taskbar", image.Rect(0, work.Max.Y, monitor.Max.X, monitor.Max.Y), 0x08000000|0x80|0x8)

	peekMessage := winUser32.NewProc("PeekMessageW")
	pump := func() {
		deadline := time.Now().Add(80 * time.Millisecond)
		for time.Now().Before(deadline) {
			var message winMessage
			for {
				ok, _, _ := peekMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0, 1)
				if ok == 0 {
					break
				}
				winTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
				winDispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
			}
			time.Sleep(time.Millisecond)
		}
	}
	place := func(window, after uintptr, rect image.Rectangle) {
		t.Helper()
		if ok, _, err := winSetWindowPos.Call(window, after, uintptr(rect.Min.X), uintptr(rect.Min.Y),
			uintptr(rect.Dx()), uintptr(rect.Dy()), 0x0010|0x0040); ok == 0 {
			t.Fatalf("position fixture: %v", err)
		}
	}
	activateApp := func() {
		t.Helper()
		winSetForegroundWindow.Call(app)
		pump()
		if foreground, _, _ := winGetForegroundWindow.Call(); foreground != app {
			t.Fatalf("fixture could not take foreground: got %#x, want %#x", foreground, app)
		}
	}
	assertSnapshot := func(label string, wantBar, wantHide bool) fullscreenSnapshot {
		t.Helper()
		pump()
		snapshot := winFullscreenSnapshot(pet)
		if !snapshot.Valid || !snapshot.ForegroundIsApp || !snapshot.SameMonitor {
			t.Fatalf("%s: missing foreground-app observation: %+v", label, snapshot)
		}
		if got := fullTaskbarVisible(taskbar, monitor); got != wantBar {
			t.Errorf("%s: actual taskbar visibility = %t, want %t", label, got, wantBar)
		}
		// Ordinary windows deliberately skip the more expensive taskbar scan.
		if monitor.In(snapshot.Foreground) && snapshot.TaskbarVisible != wantBar {
			t.Errorf("%s: taskbar visibility = %t, want %t; %+v", label, snapshot.TaskbarVisible, wantBar, snapshot)
		}
		if got := shouldHideForFullscreen(snapshot); got != wantHide {
			t.Errorf("%s: hide = %t, want %t; %+v", label, got, wantHide, snapshot)
		}
		return snapshot
	}

	activateApp()
	assertSnapshot("ordinary browser sized to work area", true, false)

	// Covering the entire monitor is insufficient: a taskbar that remains
	// exposed must keep the pet visible, matching the user's browser case.
	place(app, ^uintptr(1), monitor) // HWND_NOTOPMOST
	place(taskbar, ^uintptr(0), image.Rect(0, work.Max.Y, monitor.Max.X, monitor.Max.Y))
	activateApp()
	assertSnapshot("monitor-filling app with exposed taskbar", true, false)

	// WS_VISIBLE remains set on the taskbar, but the fullscreen app now covers
	// its pixels. The detector must use actual visibility, not the style bit.
	place(taskbar, ^uintptr(1), image.Rect(0, work.Max.Y, monitor.Max.X, monitor.Max.Y))
	place(app, ^uintptr(0), monitor)
	activateApp()
	hidden := assertSnapshot("fullscreen app occluding taskbar", false, true)
	gate := fullscreenGate{}
	start := time.Now()
	gate.Update(hidden, start)
	if !gate.Update(hidden, start.Add(fullscreenHideDelay)) {
		t.Error("real fullscreen sample did not hide after the debounce")
	}

	// Moving back to the work area exposes the taskbar and must automatically
	// restore, without requiring a tray click or an application restart.
	place(app, ^uintptr(1), work)
	activateApp()
	restored := assertSnapshot("exit fullscreen", true, false)
	gate.Update(restored, start.Add(time.Second))
	if gate.Update(restored, start.Add(time.Second+fullscreenRestoreDelay)) {
		t.Error("leaving fullscreen did not restore after the debounce")
	}

	// A temporarily hidden taskbar alone must not hide the pet when the active
	// application is still windowed.
	winShowWindow.Call(taskbar, 0)
	assertSnapshot("windowed app with hidden taskbar", false, false)
	place(app, ^uintptr(0), monitor)
	activateApp()
	assertSnapshot("fullscreen app with hidden taskbar", false, true)

	// Exercise the actual host's show/hide transitions using this real HWND.
	// Automatic restoration must never cancel an explicit manual hide or take
	// the keyboard focus away from the user's application.
	host := &winHost{window: pet, app: newPetApp(), topmost: true}
	assertVisible := func(label string, want bool) {
		t.Helper()
		visible, _, _ := fullVisible.Call(pet)
		if (visible != 0) != want {
			t.Errorf("%s: pet window visible = %t, want %t", label, visible != 0, want)
		}
		if foreground, _, _ := winGetForegroundWindow.Call(); foreground != app {
			t.Errorf("%s: visibility change stole foreground focus", label)
		}
	}
	clock := time.Now()
	host.checkFullscreen(clock, true)
	host.checkFullscreen(clock.Add(fullscreenHideDelay), true)
	assertVisible("automatic fullscreen hide", false)
	if !host.fullscreen.Hidden || host.hidden {
		t.Errorf("automatic hide mutated manual state: manual=%t auto=%t", host.hidden, host.fullscreen.Hidden)
	}

	place(app, ^uintptr(1), work)
	activateApp()
	host.checkFullscreen(clock.Add(time.Second), true)
	host.checkFullscreen(clock.Add(time.Second+fullscreenRestoreDelay), true)
	assertVisible("automatic fullscreen restore", true)

	host.hidden = true
	host.applyVisibility()
	assertVisible("manual hide", false)
	place(app, ^uintptr(0), monitor)
	activateApp()
	host.checkFullscreen(clock.Add(2*time.Second), true)
	host.checkFullscreen(clock.Add(2*time.Second+fullscreenHideDelay), true)
	place(app, ^uintptr(1), work)
	activateApp()
	host.checkFullscreen(clock.Add(3*time.Second), true)
	host.checkFullscreen(clock.Add(3*time.Second+fullscreenRestoreDelay), true)
	assertVisible("manual hide survives fullscreen exit", false)
	if host.fullscreen.Hidden || !host.hidden {
		t.Errorf("fullscreen exit changed wrong state: manual=%t auto=%t", host.hidden, host.fullscreen.Hidden)
	}
	host.hidden = false
	host.applyVisibility()
	assertVisible("explicit restore after manual hide", true)
}

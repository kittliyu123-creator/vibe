package main

import (
	"image"
	"time"
)

// fullscreenSnapshot contains observations from one foreground-window sample.
// TaskbarVisible means that a usable taskbar is actually exposed on this monitor,
// rather than merely having the Win32 WS_VISIBLE style set behind another window.
type fullscreenSnapshot struct {
	Valid              bool
	ForegroundIsApp    bool
	SameMonitor        bool
	Monitor            image.Rectangle
	Foreground         image.Rectangle
	TaskbarVisible     bool
	TaskbarAutoHide    bool
	DecoratedMaximized bool
}

func shouldHideForFullscreen(s fullscreenSnapshot) bool {
	if !s.Valid || !s.ForegroundIsApp || !s.SameMonitor || s.TaskbarVisible {
		return false
	}
	if s.Monitor.Empty() || s.Foreground.Empty() {
		return false
	}
	// A normal maximized window may fill the monitor when Windows is configured
	// to auto-hide its taskbar. Keeping its normal window decorations distinguishes
	// that case from an immersive full-screen application.
	if s.TaskbarAutoHide && s.DecoratedMaximized {
		return false
	}
	const tolerance = 2
	// Overscan and windows spanning monitors still cover this entire monitor.
	// Small inward discrepancies allow for rounded frame and DPI measurements.
	return s.Foreground.Min.X <= s.Monitor.Min.X+tolerance &&
		s.Foreground.Min.Y <= s.Monitor.Min.Y+tolerance &&
		s.Foreground.Max.X >= s.Monitor.Max.X-tolerance &&
		s.Foreground.Max.Y >= s.Monitor.Max.Y-tolerance
}

const (
	fullscreenHideDelay    = 250 * time.Millisecond
	fullscreenRestoreDelay = 200 * time.Millisecond
	fullscreenInvalidGrace = time.Second
)

// fullscreenGate debounces transitions while allowing brief unavailable samples,
// such as GetForegroundWindow returning zero during an application switch.
// Hidden is only the automatic full-screen decision; manual hiding belongs to
// the host and must not be overwritten when this gate restores the pet.
type fullscreenGate struct {
	Hidden       bool
	hasValid     bool
	lastValid    time.Time
	pending      bool
	pendingHide  bool
	pendingSince time.Time
}

func (g *fullscreenGate) Update(s fullscreenSnapshot, now time.Time) bool {
	if !s.Valid {
		g.pending = false
		if !g.hasValid || now.Sub(g.lastValid) >= fullscreenInvalidGrace {
			g.Hidden = false
		}
		return g.Hidden
	}
	g.hasValid = true
	g.lastValid = now
	wantHidden := shouldHideForFullscreen(s)
	if wantHidden == g.Hidden {
		g.pending = false
		return g.Hidden
	}
	if !g.pending || g.pendingHide != wantHidden || now.Before(g.pendingSince) {
		g.pending = true
		g.pendingHide = wantHidden
		g.pendingSince = now
		return g.Hidden
	}
	delay := fullscreenRestoreDelay
	if wantHidden {
		delay = fullscreenHideDelay
	}
	if now.Sub(g.pendingSince) >= delay {
		g.Hidden = wantHidden
		g.pending = false
	}
	return g.Hidden
}

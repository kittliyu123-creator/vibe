package main

import (
	"image"
	"testing"
	"time"
)

func fullScreenSample() fullscreenSnapshot {
	return fullscreenSnapshot{
		Valid: true, ForegroundIsApp: true, SameMonitor: true,
		Monitor: image.Rect(0, 0, 1920, 1080), Foreground: image.Rect(0, 0, 1920, 1080),
	}
}

func TestShouldHideForFullscreen(t *testing.T) {
	tests := []struct {
		name   string
		change func(*fullscreenSnapshot)
		want   bool
	}{
		{"immersive full screen", func(s *fullscreenSnapshot) {}, true},
		{"maximized browser in work area", func(s *fullscreenSnapshot) { s.Foreground.Max.Y = 1040 }, false},
		{"full screen with exposed taskbar", func(s *fullscreenSnapshot) { s.TaskbarVisible = true }, false},
		{"desktop and shell", func(s *fullscreenSnapshot) { s.ForegroundIsApp = false }, false},
		{"full screen on another monitor", func(s *fullscreenSnapshot) { s.SameMonitor = false }, false},
		{"negative monitor origin", func(s *fullscreenSnapshot) {
			s.Monitor = image.Rect(-1920, -200, 0, 880)
			s.Foreground = s.Monitor
		}, true},
		{"two pixel overscan", func(s *fullscreenSnapshot) { s.Foreground = image.Rect(-2, -2, 1922, 1082) }, true},
		{"two pixel inset", func(s *fullscreenSnapshot) { s.Foreground = image.Rect(2, 2, 1918, 1078) }, true},
		{"spanning full screen covers pet monitor", func(s *fullscreenSnapshot) { s.Foreground.Min.X = -1920 }, true},
		{"overscan border", func(s *fullscreenSnapshot) { s.Foreground = image.Rect(-8, -8, 1928, 1088) }, true},
		{"left edge short", func(s *fullscreenSnapshot) { s.Foreground.Min.X += 3 }, false},
		{"right edge short", func(s *fullscreenSnapshot) { s.Foreground.Max.X -= 3 }, false},
		{"top edge short", func(s *fullscreenSnapshot) { s.Foreground.Min.Y += 3 }, false},
		{"empty monitor", func(s *fullscreenSnapshot) { s.Monitor = image.Rectangle{} }, false},
		{"empty window", func(s *fullscreenSnapshot) { s.Foreground = image.Rectangle{} }, false},
		{"reversed window edges", func(s *fullscreenSnapshot) {
			s.Foreground = image.Rectangle{Min: image.Pt(20, 20), Max: image.Pt(10, 10)}
		}, false},
		{"invalid sample", func(s *fullscreenSnapshot) { s.Valid = false }, false},
		{"auto hide taskbar and ordinary maximized app", func(s *fullscreenSnapshot) {
			s.TaskbarAutoHide, s.DecoratedMaximized = true, true
		}, false},
		{"auto hide taskbar and borderless full screen", func(s *fullscreenSnapshot) { s.TaskbarAutoHide = true }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fullScreenSample()
			tt.change(&s)
			if got := shouldHideForFullscreen(s); got != tt.want {
				t.Fatalf("shouldHideForFullscreen(%+v) = %v, want %v", s, got, tt.want)
			}
		})
	}
}

func TestFullscreenGateDebouncesBothDirections(t *testing.T) {
	start := time.Now() // Preserve Go's monotonic clock component through Add/Sub.
	var g fullscreenGate
	full := fullScreenSample()
	visible := full
	visible.TaskbarVisible = true
	steps := []struct {
		at   time.Duration
		s    fullscreenSnapshot
		want bool
	}{
		{0, visible, false},
		{time.Second, full, false},
		{time.Second + 249*time.Millisecond, full, false},
		{time.Second + 250*time.Millisecond, full, true},
		{2 * time.Second, visible, true},
		{2*time.Second + 199*time.Millisecond, visible, true},
		{2*time.Second + 200*time.Millisecond, visible, false},
	}
	for _, step := range steps {
		if got := g.Update(step.s, start.Add(step.at)); got != step.want {
			t.Fatalf("at %v: hidden = %v, want %v", step.at, got, step.want)
		}
	}
}

func TestFullscreenGateCancelsFlappingTransitions(t *testing.T) {
	start := time.Now()
	full := fullScreenSample()
	visible := full
	visible.TaskbarVisible = true
	var g fullscreenGate
	for _, at := range []time.Duration{0, 200 * time.Millisecond, 400 * time.Millisecond} {
		if g.Update(full, start.Add(at)) || g.Update(visible, start.Add(at+100*time.Millisecond)) {
			t.Fatal("brief full-screen samples must not hide the pet")
		}
	}
	if g.Update(full, start.Add(time.Second)) || !g.Update(full, start.Add(1250*time.Millisecond)) {
		t.Fatal("stable full screen must hide after the debounce interval")
	}
	if !g.Update(visible, start.Add(1500*time.Millisecond)) || !g.Update(full, start.Add(1600*time.Millisecond)) {
		t.Fatal("a brief visible-taskbar sample must not flash the pet over full screen")
	}
}

func TestFullscreenGateInvalidSampleGraceAndRecovery(t *testing.T) {
	start := time.Now()
	full := fullScreenSample()
	invalid := fullscreenSnapshot{}
	var g fullscreenGate
	if g.Update(invalid, start) {
		t.Fatal("an unavailable initial foreground window must leave the pet visible")
	}
	g.Update(full, start.Add(time.Second))
	if !g.Update(full, start.Add(1250*time.Millisecond)) {
		t.Fatal("expected stable full screen to hide")
	}
	if !g.Update(invalid, start.Add(1500*time.Millisecond)) || !g.Update(invalid, start.Add(2249*time.Millisecond)) {
		t.Fatal("transient missing foreground window must preserve hidden state")
	}
	if g.Update(invalid, start.Add(2250*time.Millisecond)) {
		t.Fatal("persistent unavailable foreground must restore the pet after one second")
	}
	if g.Update(full, start.Add(2500*time.Millisecond)) || !g.Update(full, start.Add(2750*time.Millisecond)) {
		t.Fatal("recovery must use the normal debounce interval")
	}
}

func TestFullscreenGateInvalidSampleResetsPendingTransition(t *testing.T) {
	start := time.Now()
	full := fullScreenSample()
	var g fullscreenGate
	g.Update(full, start)
	g.Update(fullscreenSnapshot{}, start.Add(200*time.Millisecond))
	if g.Update(full, start.Add(300*time.Millisecond)) {
		t.Fatal("time spent with unavailable observations must not complete a hide transition")
	}
	if !g.Update(full, start.Add(550*time.Millisecond)) {
		t.Fatal("two stable samples must complete the restarted transition")
	}
}

func TestFullscreenGateRebasesReversedClock(t *testing.T) {
	start := time.Now()
	full := fullScreenSample()
	var g fullscreenGate
	g.Update(full, start)
	if g.Update(full, start.Add(-time.Second)) || g.Update(full, start.Add(-800*time.Millisecond)) {
		t.Fatal("a reversed test clock must restart, not prematurely complete, a transition")
	}
	if !g.Update(full, start.Add(-750*time.Millisecond)) {
		t.Fatal("debounce must recover after a reversed clock")
	}
}

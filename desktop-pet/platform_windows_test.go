//go:build windows

package main

import (
	"image"
	"image/color"
	"testing"
)

func TestWindowsScaleKeepsPremultipliedTransparentEdges(t *testing.T) {
	// Scaling an opaque red edge into transparency must interpolate alpha
	// and premultiplied red equally, avoiding a dark outline on the desktop.
	source := image.NewRGBA(image.Rect(0, 0, 2, 1))
	source.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	out := make([]byte, 3*4)
	winCopyPremultipliedBGRA(out, 3, 1, source)
	if out[4] != 0 || out[5] != 0 || out[6] != 128 || out[7] != 128 {
		t.Fatalf("middle edge must be half-transparent red in BGRA: %v", out)
	}
	if out[2] != 255 || out[3] != 255 || out[10] != 0 || out[11] != 0 {
		t.Fatalf("scaling changed edge endpoints: %v", out)
	}
}

func TestWindowsBitmapCopyHandlesStrideAndChannelOrder(t *testing.T) {
	parent := image.NewRGBA(image.Rect(0, 0, 5, 4))
	source := parent.SubImage(image.Rect(1, 1, 3, 3)).(*image.RGBA)
	source.SetRGBA(1, 1, color.RGBA{R: 90, G: 70, B: 40, A: 100})
	source.SetRGBA(2, 2, color.RGBA{R: 20, G: 30, B: 40, A: 60})
	out := make([]byte, 2*2*4)
	winCopyPremultipliedBGRA(out, 2, 2, source)
	if got := [4]byte(out[:4]); got != [4]byte{40, 70, 90, 100} {
		t.Fatalf("first pixel changed premultiplied values or channel order: %v", got)
	}
	if got := [4]byte(out[12:]); got != [4]byte{40, 30, 20, 60} {
		t.Fatalf("non-contiguous source stride copied the wrong final pixel: %v", got)
	}
}

func TestWindowsPositionRecoveryAndDPIScale(t *testing.T) {
	host := &winHost{x: -4000, y: 3000, width: 440, height: 480}
	host.clampPosition(winRect{Left: -1920, Top: -200, Right: 0, Bottom: 880})
	if host.x != -1920 || host.y != 400 {
		t.Fatalf("offscreen saved position did not recover on negative-coordinate monitor: %d,%d", host.x, host.y)
	}
	host.scale, host.dpi = 1.25, 144
	w, h := host.dimensions()
	if w != 825 || h != 900 {
		t.Fatalf("DPI and pet size must both apply: %dx%d", w, h)
	}
}

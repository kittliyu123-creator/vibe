package main

import (
	"bytes"
	"testing"
)

func TestArtworkAndWindowsAlpha(t *testing.T) {
	if err := loadArtwork(); err != nil {
		t.Fatal(err)
	}
	if petArt == nil || sleepArt == nil {
		t.Fatal("both pet expressions must be embedded")
	}
	for _, action := range []string{"hello", "pet", "feed", "play", "sleep"} {
		a := newPetApp()
		a.Action(action)
		a.Tick(.6)
		frame := a.Render()
		if frame.RGBAAt(0, 0).A != 0 {
			t.Fatal("window corner must be transparent")
		}
		visible := 0
		for i := 0; i < len(frame.Pix); i += 4 {
			alpha := frame.Pix[i+3]
			if alpha > 0 {
				visible++
			}
			for k := 0; k < 3; k++ {
				if frame.Pix[i+k] > alpha {
					t.Fatalf("%s: invalid premultiplied pixel at %d", action, i/4)
				}
			}
		}
		if visible < 10000 {
			t.Fatalf("%s has insufficient visible art", action)
		}
	}
}

func TestIdleFrameActuallyAnimates(t *testing.T) {
	if err := loadArtwork(); err != nil {
		t.Fatal(err)
	}
	a := newPetApp()
	a.Bubble = ""
	before := a.Render()
	a.Tick(.7)
	after := a.Render()
	if bytes.Equal(before.Pix, after.Pix) {
		t.Fatal("idle animation must change rendered pixels")
	}
}

func BenchmarkRender(b *testing.B) {
	if err := loadArtwork(); err != nil {
		b.Fatal(err)
	}
	a := newPetApp()
	a.Action("pet")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Tick(1.0 / 30)
		a.Render()
	}
}

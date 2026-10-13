package main

import (
	"math"
	"testing"
)

func TestInteractionsFinishAndBubblesExpire(t *testing.T) {
	for _, test := range []struct {
		command, mode string
		duration      float64
	}{
		{"pet", "happy", 2.2},
		{"feed", "eating", 3.2},
		{"play", "playing", 4},
	} {
		t.Run(test.command, func(t *testing.T) {
			p := newPetApp()
			p.Action(test.command)
			if p.Mode != test.mode || p.Bubble == "" || len(p.Particles) == 0 {
				t.Fatalf("interaction did not start: %+v", p)
			}
			p.Tick(test.duration - 0.01)
			if p.Mode != test.mode {
				t.Fatalf("action ended early: %q", p.Mode)
			}
			p.Tick(0.02)
			if p.Mode != "idle" {
				t.Fatalf("action did not finish: %q", p.Mode)
			}
			p.Tick(10)
			if p.Bubble != "" || len(p.Particles) != 0 || len(p.emissions) != 0 {
				t.Fatalf("effects did not expire: %+v", p)
			}
		})
	}
}

func TestSleepPersistsAndCanWake(t *testing.T) {
	p := newPetApp()
	p.Energy = 20
	p.Action("sleep")
	p.Tick(90)
	if !p.Sleeping() || p.Energy != 100 {
		t.Fatalf("sleep did not persist or replenish energy: %+v", p)
	}
	p.Action("sleep")
	if p.Sleeping() || p.Mode != "happy" {
		t.Fatal("selecting sleep again should wake the pet")
	}
	p.Action("sleep")
	p.Action("pet")
	if p.Sleeping() || p.Mode != "happy" {
		t.Fatal("petting should gently wake the pet")
	}
	p.Action("sleep")
	p.Action("wake")
	if p.Sleeping() {
		t.Fatal("explicit wake did not wake the pet")
	}
}

func TestDraggingRestoresSleepOrIdle(t *testing.T) {
	for _, sleeping := range []bool{false, true} {
		p := newPetApp()
		if sleeping {
			p.Action("sleep")
		} else {
			p.Action("play")
		}
		p.SetDragging(true)
		p.SetDragging(true)
		p.Tick(30)
		p.Action("feed")
		if p.Mode != "dragging" {
			t.Fatal("dragging was interrupted by an action or timer")
		}
		p.SetDragging(false)
		if p.Sleeping() != sleeping || (!sleeping && p.Mode != "idle") {
			t.Fatalf("incorrect drag restoration (sleep=%v): %q", sleeping, p.Mode)
		}
		if p.Bubble != "轻轻放下我" {
			t.Fatal("missing drag feedback")
		}
		p.SetDragging(false)
		if p.Sleeping() != sleeping {
			t.Fatal("repeated drag end should have no effect")
		}
	}
}

func TestRapidActionsKeepStatsAndEffectsBounded(t *testing.T) {
	p := newPetApp()
	for i := 0; i < 1000; i++ {
		p.Action("feed")
		p.Action("play")
		p.Action("pet")
	}
	for name, value := range map[string]float64{
		"energy": p.Energy, "fullness": p.Fullness, "bond": p.Bond,
	} {
		if value < 0 || value > 100 || math.IsNaN(value) {
			t.Fatalf("%s out of bounds: %f", name, value)
		}
	}
	if len(p.Particles) > 48 || len(p.emissions) > 10 {
		t.Fatalf("unbounded effects: %d particles, %d queued", len(p.Particles), len(p.emissions))
	}
	p.Tick(10)
	if len(p.Particles) != 0 || len(p.emissions) != 0 {
		t.Fatal("effects from rapid interactions did not finish")
	}
}

func TestParticlesAreEmittedOverTime(t *testing.T) {
	p := newPetApp()
	p.Action("pet")
	if len(p.Particles) != 1 {
		t.Fatalf("expected an immediate first particle, got %d", len(p.Particles))
	}
	y := p.Particles[0].Y
	p.Tick(0.4)
	if len(p.Particles) != 3 || p.Particles[0].Y >= y {
		t.Fatalf("particles should stagger and rise: %+v", p.Particles)
	}
	p.Tick(5)
	if len(p.Particles) != 0 {
		t.Fatal("particles should fade away")
	}
}

func TestElapsedTimeValidationAndLongResume(t *testing.T) {
	p := newPetApp()
	p.Action("play")
	for _, dt := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		p.Tick(dt)
	}
	if p.Time != 0 || p.ActionTime != 0 || p.Mode != "playing" {
		t.Fatalf("invalid durations changed state: %+v", p)
	}
	energy, fullness := p.Energy, p.Fullness
	dx, dy := p.Tick(60 * 60 * 24 * 365)
	if dx != 0 || dy != 0 || p.Mode != "idle" || len(p.Particles) != 0 || len(p.emissions) != 0 {
		t.Fatalf("unsafe long-resume state: %+v", p)
	}
	if p.Energy < energy-2 || p.Fullness < fullness-1 {
		t.Fatal("long machine suspension should not punish the owner")
	}
}

func TestUnknownActionDoesNotInterruptSleep(t *testing.T) {
	p := newPetApp()
	p.Action("sleep")
	p.Action("unknown")
	if !p.Sleeping() {
		t.Fatal("unknown command should leave state unchanged")
	}
}

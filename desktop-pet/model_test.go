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
	p.Tick(0.5)
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

func TestMovementIsOptInAndPreferencesAreExclusive(t *testing.T) {
	p := newPetApp()
	p.SetPointer(1500, -100)
	for i := 0; i < 1800; i++ {
		if dx, dy := p.Tick(1.0 / 60); dx != 0 || dy != 0 {
			t.Fatal("default pet must not move the desktop window")
		}
	}
	p.Action("follow")
	if !p.Follow || p.Wander {
		t.Fatal("follow preference did not enable exclusively")
	}
	p.Action("wander")
	if p.Follow || !p.Wander {
		t.Fatal("wander preference did not replace follow")
	}
	p.Action("wander")
	if p.Follow || p.Wander {
		t.Fatal("toggling wander again should park the pet")
	}
}

func TestFollowingSpeedDeadZoneAndDirection(t *testing.T) {
	for _, direction := range []float64{-1, 1} {
		p := newPetApp()
		p.Action("follow")
		p.SetPointer(220+direction*1000, 50)
		moved := 0
		for i := 0; i < 120; i++ {
			dx, dy := p.Tick(1.0 / 60)
			if dy != 0 || math.Abs(float64(dx)) > 1 {
				t.Fatalf("unexpected frame movement: %d, %d", dx, dy)
			}
			moved += dx
		}
		if math.Abs(float64(moved)-direction*70) > 1 || p.Facing != direction {
			t.Fatalf("expected 35 px/s in direction %g, got %d", direction, moved)
		}
		// A stale cursor observation must still stop at its target.
		p.SetPointer(220+direction*120.25, 50)
		moved = 0
		for i := 0; i < 600; i++ {
			dx, _ := p.Tick(1.0 / 60)
			moved += dx
		}
		if float64(moved) != direction*20 || p.Mode != "idle" || p.moveCarry != 0 {
			t.Fatalf("did not stop cleanly outside dead zone: moved=%d, mode=%s, carry=%g", moved, p.Mode, p.moveCarry)
		}
		p.SetPointer(220-direction*1000, 50)
		for i := 0; i < 10; i++ {
			dx, _ := p.Tick(1.0 / 60)
			if float64(dx)*direction > 0 {
				t.Fatal("old fractional movement continued in the previous direction")
			}
		}
		if p.Facing != -direction {
			t.Fatal("pet did not turn toward cursor")
		}
	}
}

func TestMovementPausesForSleepDragAndResume(t *testing.T) {
	for _, pause := range []string{"sleep", "drag"} {
		p := newPetApp()
		p.Action("follow")
		p.SetPointer(1500, 50)
		p.Tick(0.1)
		if pause == "sleep" {
			p.Action("sleep")
		} else {
			p.SetDragging(true)
		}
		for i := 0; i < 120; i++ {
			if dx, dy := p.Tick(0.1); dx != 0 || dy != 0 {
				t.Fatalf("pet moved during %s", pause)
			}
		}
		if !p.Follow || p.moveCarry != 0 {
			t.Fatal("pause should retain preference and discard old motion")
		}
		if pause == "sleep" {
			p.Action("wake")
			p.Tick(2)
		} else {
			p.SetDragging(false)
		}
		if dx, _ := p.Tick(0.1); dx <= 0 {
			t.Fatalf("pet did not resume following after %s", pause)
		}
		if dx, dy := p.Tick(60 * 60); dx != 0 || dy != 0 || p.Mode != "idle" || p.moveCarry != 0 {
			t.Fatal("resuming after system suspension must not jump")
		}
	}
}

func TestWanderingWaitsForControlsInteraction(t *testing.T) {
	p := newPetApp()
	p.Action("wander")
	p.SetPointer(800, 40)
	if dx, _ := p.Tick(0.1); dx == 0 || p.Mode != "walking" {
		t.Fatal("explicit wandering did not start")
	}
	p.SetPointer(192, 444)
	if dx, _ := p.Tick(0.1); dx != 0 || p.Mode != "idle" {
		t.Fatal("pet should wait while owner approaches its controls")
	}
}

func TestPointerValidationSmoothingAndSleepGaze(t *testing.T) {
	p := newPetApp()
	p.SetPointer(400, 50)
	p.Tick(0.05)
	if p.LookX <= 0 || p.LookX >= 1 || p.LookY >= 0 || p.LookY <= -1 {
		t.Fatalf("gaze should ease toward cursor: %g, %g", p.LookX, p.LookY)
	}
	x, y := p.pointerX, p.pointerY
	for _, invalid := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		p.SetPointer(invalid, 0)
		p.SetPointer(0, invalid)
	}
	if p.pointerX != x || p.pointerY != y {
		t.Fatal("invalid cursor input changed gaze target")
	}
	p.Action("sleep")
	p.Tick(0.5)
	if math.Abs(p.LookX) >= 0.01 || math.Abs(p.LookY) >= 0.01 {
		t.Fatal("sleeping gaze should settle toward center")
	}
}

func TestToolbarTargetsVisibilityGraceAndDrag(t *testing.T) {
	p := newPetApp()
	if p.ActionAt(136, 444) || p.ControlHit(136, 444) {
		t.Fatal("hidden controls must not handle mouse clicks")
	}
	p.SetPointer(220, 230)
	p.Tick(0.1)
	for _, test := range []struct {
		x    float64
		mode string
	}{{136, "happy"}, {192, "eating"}, {248, "playing"}, {304, "sleeping"}} {
		if !p.ControlHit(test.x, 444) || !p.ActionAt(test.x, 444) || p.Mode != test.mode {
			t.Fatalf("wrong action at %g: %s", test.x, p.Mode)
		}
	}
	for _, point := range [][2]float64{{164, 444}, {136, 410}, {math.NaN(), 444}, {136, math.Inf(1)}} {
		if p.ActionAt(point[0], point[1]) {
			t.Fatalf("handled click outside a circular control: %v", point)
		}
	}
	p.SetPointer(700, 700)
	p.Tick(0.5)
	if p.ControlsAlpha < 0.8 || !p.ControlHit(136, 444) {
		t.Fatal("brief pointer exit should preserve toolbar for easier targeting")
	}
	p.Tick(0.6)
	p.Tick(0.4)
	if p.ControlsAlpha != 0 || p.ControlHit(136, 444) {
		t.Fatal("toolbar should fade and stop handling input after grace expires")
	}
	p.SetPointer(136, 444)
	p.Tick(0.1)
	p.SetDragging(true)
	if p.ControlsAlpha != 0 || p.ActionAt(136, 444) || p.ControlHit(136, 444) {
		t.Fatal("dragging should immediately hide and disable toolbar")
	}
}

func TestIdleGesturesFinishAndNewActionsResetTheirClock(t *testing.T) {
	p := newPetApp()
	gestures := 0
	previous := p.Mode
	for i := 0; i < 3600; i++ {
		p.Tick(1.0 / 60)
		if p.Mode != previous && (p.Mode == "waving" || p.Mode == "stretching") {
			gestures++
		}
		previous = p.Mode
	}
	if gestures < 2 || gestures > 5 {
		t.Fatalf("idle gestures should be occasional, got %d in a minute", gestures)
	}
	p.Action("wave")
	p.Tick(2)
	p.Action("stretch")
	if p.Mode != "stretching" || p.ActionTime != 0 {
		t.Fatal("new action must restart the animation clock")
	}
	p.Tick(2.9)
	if p.Mode != "stretching" {
		t.Fatal("previous action clock prematurely ended the new action")
	}
	p.Tick(0.2)
	if p.Mode != "idle" {
		t.Fatal("stretch did not finish")
	}
}

package main

import "math"

// Particle positions and velocities are normalized to the entire pet canvas.
type Particle struct {
	X, Y   float64
	VX, VY float64
	Age    float64
	Life   float64
	Kind   string
}

type particleEmission struct {
	At   float64
	Kind string
}

// PetApp owns interaction state without accessing the desktop or wall clock.
// Pointer coordinates use the renderer's 440 by 480 logical canvas, including
// coordinates outside it. Movement returned by Tick uses the same units.
type PetApp struct {
	Time               float64
	Mode               string
	ActionTime         float64
	Bubble             string
	BubbleUntil        float64
	Energy             float64
	Fullness           float64
	Bond               float64
	Particles          []Particle
	Hover              bool
	LookX, LookY       float64
	Follow             bool
	Wander             bool
	Facing             float64
	ControlsAlpha      float64
	PreviousMode       string
	PreviousActionTime float64
	ModeChangedAt      float64

	modeDuration float64
	dragRestore  string
	emissions    []particleEmission
	randomState  uint32
	petCount     int
	pointerX     float64
	pointerY     float64
	pointerKnown bool
	hoverUntil   float64
	nextIdleAt   float64
	nextTurnAt   float64
	moveCarry    float64
}

func newPetApp() *PetApp {
	p := &PetApp{
		Mode: "idle", Bubble: "摸摸我吧\n靠近就有小菜单", BubbleUntil: 6,
		Energy: 88, Fullness: 75, Bond: 40, Facing: 1,
		pointerX: 220, pointerY: 230, randomState: 0x51a7cafe,
	}
	p.scheduleIdle()
	return p
}

// SetPointer accepts global cursor positions converted to logical canvas
// coordinates. Keeping outside positions lets gaze and opt-in following work
// without requiring the mouse to remain over the pet.
func (p *PetApp) SetPointer(x, y float64) {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		return
	}
	p.pointerX = math.Max(-100000, math.Min(100000, x))
	p.pointerY = math.Max(-100000, math.Min(100000, y))
	p.pointerKnown = true
	p.Hover = x >= 65 && x <= 375 && y >= 125 && y <= 470
	if p.Hover {
		p.hoverUntil = p.Time + 1
	}
}

func toolbarIndex(x, y float64) int {
	if math.IsNaN(x) || math.IsNaN(y) {
		return -1
	}
	for i, center := range [...]float64{136, 192, 248, 304} {
		if math.Hypot(x-center, y-444) <= 23 {
			return i
		}
	}
	return -1
}

// ToolbarHit can be used on mouse-down to keep a button press from dragging
// the window. It uses precisely the same visible targets as ActionAt.
func (p *PetApp) ToolbarHit(x, y float64) bool {
	return p.Mode != "dragging" && (p.Hover || p.ControlsAlpha > 0.15) && toolbarIndex(x, y) >= 0
}

// ControlHit is the native window's mouse-down hit test. Keep the renderer,
// drag handling and action dispatch on the same circular button targets.
func (p *PetApp) ControlHit(x, y float64) bool { return p.ToolbarHit(x, y) }

func (p *PetApp) ActionAt(x, y float64) bool {
	if !p.ToolbarHit(x, y) {
		return false
	}
	p.Action(([...]string{"pet", "feed", "play", "sleep"})[toolbarIndex(x, y)])
	return true
}

// Tick advances state in seconds. Care updates and desktop movement remain
// bounded after a machine resumes from suspension. Movement is opt-in.
func (p *PetApp) Tick(dt float64) (dx, dy int) {
	if dt <= 0 || math.IsNaN(dt) || math.IsInf(dt, 0) {
		return 0, 0
	}
	clockDelta := math.Min(dt, 86400)
	p.Time += clockDelta
	p.ActionTime += clockDelta
	careDelta := math.Min(dt, 60)
	if p.Sleeping() {
		p.Energy += careDelta * 1.8
	} else {
		p.Energy -= careDelta * 0.025
	}
	p.Fullness -= careDelta * 0.015
	p.clampStats()

	lookX, lookY := 0.0, 0.0
	if p.pointerKnown && !p.Sleeping() {
		lookX = math.Max(-1, math.Min(1, (p.pointerX-220)/180))
		lookY = math.Max(-1, math.Min(1, (p.pointerY-230)/180))
	}
	smoothing := 1 - math.Exp(-8*dt)
	p.LookX += (lookX - p.LookX) * smoothing
	p.LookY += (lookY - p.LookY) * smoothing
	controlsTarget := 0.0
	if p.Mode != "dragging" && (p.Hover || p.Time < p.hoverUntil) {
		controlsTarget = 1
	}
	p.ControlsAlpha += (controlsTarget - p.ControlsAlpha) * (1 - math.Exp(-12*dt))
	if p.ControlsAlpha < 0.002 {
		p.ControlsAlpha = 0
	} else if p.ControlsAlpha > 0.998 {
		p.ControlsAlpha = 1
	}

	if p.Bubble != "" && p.Time >= p.BubbleUntil {
		p.Bubble = ""
	}
	wasBusy := p.Mode != "idle" && p.Mode != "walking"
	if p.modeDuration > 0 && p.ActionTime >= p.modeDuration {
		p.setMode("idle", 0)
		p.scheduleIdle()
	}
	p.tickParticles(clockDelta)

	// Never turn elapsed suspension time into a jump across the desktop or a
	// backlog of autonomous animations. Resume from a fresh idle deadline.
	if dt > 1 {
		p.moveCarry = 0
		if p.Mode == "walking" {
			p.setMode("idle", 0)
		}
		p.scheduleIdle()
		return 0, 0
	}
	if !wasBusy && (p.Mode == "idle" || p.Mode == "walking") {
		dx = p.tickMovement(dt)
		if p.Mode == "idle" && p.Time >= p.nextIdleAt {
			if p.random() < 0.5 {
				p.setMode("waving", 2.5)
			} else {
				p.setMode("stretching", 3)
			}
			p.scheduleIdle()
		}
	}
	return dx, 0
}

func (p *PetApp) tickMovement(dt float64) int {
	direction, available := 0.0, math.Inf(1)
	if p.Follow && p.pointerKnown {
		delta := p.pointerX - 220
		// Leave sub-pixel remainders alone: native window motion is integral.
		if math.Abs(delta) >= 101 {
			direction = math.Copysign(1, delta)
			available = math.Abs(delta) - 100
		}
	} else if p.Wander && !p.Hover {
		// Stay still while a user approaches the pet or its controls. Otherwise
		// a wandering window can move out from under the button being clicked.
		if p.Time >= p.nextTurnAt {
			if p.random() < 0.5 {
				p.Facing = -1
			} else {
				p.Facing = 1
			}
			p.nextTurnAt = p.Time + 5 + p.random()*5
		}
		direction = p.Facing
	}
	if direction == 0 {
		p.moveCarry = 0
		if p.Mode == "walking" {
			p.setMode("idle", 0)
			p.scheduleIdle()
		}
		return 0
	}
	if p.moveCarry*direction < 0 {
		p.moveCarry = 0
	}
	p.Facing = direction
	if p.Mode != "walking" {
		p.setMode("walking", 0)
	}
	p.moveCarry += direction * math.Min(35*dt, available)
	dx := int(p.moveCarry)
	if !math.IsInf(available, 0) && math.Abs(float64(dx)) > available {
		dx = int(math.Copysign(math.Floor(available), direction))
	}
	p.moveCarry -= float64(dx)
	// Keep the stored relative pointer consistent even if the caller cannot
	// poll the cursor on every frame. A single stale event cannot cause runaway.
	p.pointerX -= float64(dx)
	return dx
}

func (p *PetApp) tickParticles(dt float64) {
	kept := p.Particles[:0]
	for _, particle := range p.Particles {
		particle.Age += dt
		if particle.Age >= particle.Life {
			continue
		}
		particle.X += particle.VX * dt
		particle.Y += particle.VY * dt
		kept = append(kept, particle)
	}
	p.Particles = kept
	waiting := p.emissions[:0]
	for _, emission := range p.emissions {
		if emission.At > p.Time {
			waiting = append(waiting, emission)
			continue
		}
		particle := p.makeParticle(emission.Kind)
		particle.Age = p.Time - emission.At
		if particle.Age >= particle.Life {
			continue
		}
		particle.X += particle.VX * particle.Age
		particle.Y += particle.VY * particle.Age
		p.addParticle(particle)
	}
	p.emissions = waiting
}

// Action handles direct interaction and menu choices. Following and wandering
// are explicit mutually exclusive preferences; sleep and dragging pause them.
func (p *PetApp) Action(command string) {
	if p.Mode == "dragging" {
		return
	}
	switch command {
	case "pet":
		p.setMode("happy", 2.2)
		responses := [...]string{"摸摸收到啦", "再摸一下嘛", "喜欢和你待着", "今天也要开心呀"}
		p.say(responses[p.petCount%len(responses)], 2.7)
		p.petCount = (p.petCount + 1) % len(responses)
		p.Bond += 3
		p.emit("heart", 5, 0.22)
	case "feed":
		p.setMode("eating", 3.2)
		p.say("啊呜，好好吃", 3.4)
		if p.Fullness >= 90 {
			p.say("这一口留给你", 3.4)
		}
		p.Fullness += 18
		p.Energy += 4
		p.Bond += 2
		p.emit("crumb", 6, 0.3)
	case "play":
		p.setMode("playing", 4)
		p.say("一起玩一会儿", 3.4)
		p.Energy -= 5
		p.Bond += 4
		p.emit("star", 6, 0.4)
	case "wave":
		p.setMode("waving", 2.5)
		p.say("嗨，我在这里", 2.7)
	case "stretch":
		p.setMode("stretching", 3)
		p.say("伸个懒腰～", 2.7)
	case "sleep":
		if p.Sleeping() {
			p.wake()
		} else {
			p.setMode("sleeping", 0)
			p.emissions = nil
			p.say("晚安，陪你安静待着", 3.5)
		}
	case "wake":
		p.wake()
	case "follow":
		p.Follow = !p.Follow
		if p.Follow {
			p.Wander = false
			p.say("你去哪，我就跟到哪", 3)
		} else {
			p.say("好，我在这里陪你", 3)
		}
		p.stopWalking()
	case "wander":
		p.Wander = !p.Wander
		if p.Wander {
			p.Follow = false
			p.nextTurnAt = p.Time + 6
			p.say("我去旁边溜达一下", 3)
		} else {
			p.say("回来陪你啦", 3)
		}
		p.stopWalking()
	case "hello":
		p.say("单击摸摸，按住拖动\n右键还有更多互动", 6)
	default:
		return
	}
	p.scheduleIdle()
	p.clampStats()
}

func (p *PetApp) SetDragging(dragging bool) {
	if dragging {
		if p.Mode == "dragging" {
			return
		}
		p.dragRestore = "idle"
		if p.Sleeping() {
			p.dragRestore = "sleeping"
		}
		p.setMode("dragging", 0)
		p.emissions = nil
		p.Bubble = ""
		p.ControlsAlpha = 0
		return
	}
	if p.Mode != "dragging" {
		return
	}
	p.setMode(p.dragRestore, 0)
	p.dragRestore = ""
	p.scheduleIdle()
	p.say("轻轻放下我", 2.3)
}

func (p *PetApp) Sleeping() bool { return p.Mode == "sleeping" }

func (p *PetApp) wake() {
	p.setMode("happy", 1.6)
	p.say("醒啦，你还在呀", 2.8)
	p.emit("heart", 3, 0.3)
}

func (p *PetApp) stopWalking() {
	p.moveCarry = 0
	if p.Mode == "walking" {
		p.setMode("idle", 0)
	}
}

func (p *PetApp) setMode(mode string, duration float64) {
	p.PreviousMode, p.PreviousActionTime, p.ModeChangedAt = p.Mode, p.ActionTime, p.Time
	p.Mode = mode
	p.ActionTime = 0
	p.modeDuration = duration
	if mode != "walking" {
		p.moveCarry = 0
	}
}

func (p *PetApp) scheduleIdle() { p.nextIdleAt = p.Time + 12 + p.random()*10 }

func (p *PetApp) say(message string, duration float64) {
	p.Bubble = message
	p.BubbleUntil = p.Time + duration
}

func (p *PetApp) clampStats() {
	p.Energy = clampStat(p.Energy)
	p.Fullness = clampStat(p.Fullness)
	p.Bond = clampStat(p.Bond)
}

func clampStat(value float64) float64 {
	if math.IsNaN(value) {
		return 50
	}
	return math.Max(0, math.Min(100, value))
}

func (p *PetApp) emit(kind string, count int, spacing float64) {
	p.emissions = p.emissions[:0]
	p.addParticle(p.makeParticle(kind))
	for i := 1; i < count && i < 8; i++ {
		p.emissions = append(p.emissions, particleEmission{At: p.Time + float64(i)*spacing, Kind: kind})
	}
}

func (p *PetApp) addParticle(particle Particle) {
	const maxParticles = 24
	if len(p.Particles) >= maxParticles {
		copy(p.Particles, p.Particles[len(p.Particles)-maxParticles+1:])
		p.Particles = p.Particles[:maxParticles-1]
	}
	p.Particles = append(p.Particles, particle)
}

func (p *PetApp) makeParticle(kind string) Particle {
	side := -1.0
	if p.random() < 0.5 {
		side = 1
	}
	particle := Particle{
		X: 0.5 + side*(0.22+p.random()*0.04), Y: 0.65 + p.random()*0.035,
		VX: side * (0.025 + p.random()*0.025), VY: -0.07 - p.random()*0.035,
		Life: 1.3 + p.random()*0.6, Kind: kind,
	}
	if kind == "crumb" {
		particle.X = 0.47 + p.random()*0.06
		particle.Y = 0.70
		particle.VY = 0.035 + p.random()*0.035
		particle.Life = 0.5 + p.random()*0.35
	}
	return particle
}

func (p *PetApp) random() float64 {
	p.randomState = p.randomState*1664525 + 1013904223
	return float64(p.randomState>>8) / float64(1<<24)
}

package main

import "math"

// Particle positions and velocities are normalized to the entire pet canvas.
// A renderer can multiply X and VX by its width, and Y and VY by its height.
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

// PetApp contains only interaction state. It does not access the desktop,
// network, filesystem, clock, or graphical assets.
type PetApp struct {
	Time        float64
	Mode        string
	ActionTime  float64
	Bubble      string
	BubbleUntil float64
	Energy      float64
	Fullness    float64
	Bond        float64
	Particles   []Particle

	modeDuration float64
	dragRestore  string
	emissions    []particleEmission
	randomState  uint32
	petCount     int
}

func newPetApp() *PetApp {
	return &PetApp{
		Mode:        "idle",
		Bubble:      "拖动我看看\n右键有小菜单",
		BubbleUntil: 8,
		Energy:      88,
		Fullness:    75,
		Bond:        40,
		randomState: 0x51a7cafe,
	}
}

// Tick advances animation and care state in seconds. Invalid elapsed times are
// ignored. Large finite time steps expire effects in constant, bounded work.
// The pet stays where it was placed; movement is controlled by user dragging.
func (p *PetApp) Tick(dt float64) (dx, dy int) {
	if dt <= 0 || math.IsNaN(dt) || math.IsInf(dt, 0) {
		return 0, 0
	}

	// A machine resuming after a long sleep should not punish its owner. Timers
	// still catch up, while care changes account for at most one minute per tick.
	careDelta := math.Min(dt, 60)
	p.Time += dt
	p.ActionTime += dt
	if p.Sleeping() {
		p.Energy += careDelta * 1.8
	} else {
		p.Energy -= careDelta * 0.025
	}
	p.Fullness -= careDelta * 0.015
	p.clampStats()

	if p.Bubble != "" && p.Time >= p.BubbleUntil {
		p.Bubble = ""
	}
	if p.modeDuration > 0 && p.ActionTime >= p.modeDuration {
		p.setMode("idle", 0)
	}

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
	return 0, 0
}

// Action handles a context-menu choice or a click on the pet. Every action is
// usable repeatedly, with no cooldown or care threshold locking the user out.
func (p *PetApp) Action(command string) {
	if p.Mode == "dragging" {
		return
	}

	switch command {
	case "pet":
		p.setMode("happy", 2.2)
		responses := [...]string{"摸摸收到啦", "再靠近一点点", "有你在，真好", "把今天的好运给你"}
		p.say(responses[p.petCount%len(responses)], 2.7)
		p.petCount++
		p.Bond += 3
		p.emit("heart", 7, 0.18)
	case "feed":
		p.setMode("eating", 3.2)
		if p.Fullness >= 90 {
			p.say("留一点甜，慢慢吃", 3.4)
		} else {
			p.say("一口快乐，啊呜", 3.4)
		}
		p.Fullness += 18
		p.Energy += 4
		p.Bond += 2
		p.emit("crumb", 9, 0.24)
	case "play":
		p.setMode("playing", 4)
		if p.Energy < 25 {
			p.say("慢慢玩，也很开心", 3.4)
		} else {
			p.say("快乐转一圈", 3.4)
		}
		p.Energy -= 5
		p.Bond += 4
		p.emit("star", 10, 0.28)
	case "sleep":
		if p.Sleeping() {
			p.wake()
		} else {
			p.setMode("sleeping", 0)
			p.emissions = nil
			p.say("晚安，我就在这里", 3.5)
		}
	case "wake":
		p.wake()
	case "hello":
		p.say("拖动我看看\n右键有小菜单", 6)
	default:
		return
	}
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
		return
	}
	if p.Mode != "dragging" {
		return
	}
	p.setMode(p.dragRestore, 0)
	p.dragRestore = ""
	p.say("轻轻放下我", 2.3)
}

func (p *PetApp) Sleeping() bool { return p.Mode == "sleeping" }

func (p *PetApp) wake() {
	p.setMode("happy", 1.6)
	p.say("醒啦，陪你待一会", 2.8)
	p.emit("heart", 4, 0.2)
}

func (p *PetApp) setMode(mode string, duration float64) {
	p.Mode = mode
	p.ActionTime = 0
	p.modeDuration = duration
}

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
	// Replace future emissions on a new interaction; rapid clicking must never
	// build an unbounded queue. Previously visible particles finish naturally.
	p.emissions = p.emissions[:0]
	p.addParticle(p.makeParticle(kind))
	for i := 1; i < count; i++ {
		p.emissions = append(p.emissions, particleEmission{
			At: p.Time + float64(i)*spacing, Kind: kind,
		})
	}
}

func (p *PetApp) addParticle(particle Particle) {
	const maxParticles = 48
	if len(p.Particles) >= maxParticles {
		copy(p.Particles, p.Particles[len(p.Particles)-maxParticles+1:])
		p.Particles = p.Particles[:maxParticles-1]
	}
	p.Particles = append(p.Particles, particle)
}

func (p *PetApp) makeParticle(kind string) Particle {
	particle := Particle{
		X: 0.4 + p.random()*0.2, Y: 0.48 + p.random()*0.1,
		VX: (p.random() - 0.5) * 0.1, VY: -0.06 - p.random()*0.045,
		Life: 1.3 + p.random()*0.7, Kind: kind,
	}
	if kind == "crumb" {
		particle.Y = 0.61
		particle.VY = 0.03 + p.random()*0.035
		particle.Life = 0.55 + p.random()*0.45
	}
	return particle
}

func (p *PetApp) random() float64 {
	p.randomState = p.randomState*1664525 + 1013904223
	return float64(p.randomState>>8) / float64(1<<24)
}

package main

import (
	"image"
	"image/color"
	"math"
)

// The reference is kept whole. One uniform scale preserves the original face,
// head/body proportions, seated pose, limbs and bow in every animation state.
type characterPose struct{ x, y, scale, angle float64 }

func (a *PetApp) characterPose() characterPose {
	t, q := a.Time, a.ActionTime
	p := characterPose{x: 220, y: 246, scale: 1}
	p.y += math.Sin(t*1.65) * .65
	switch a.Mode {
	case "happy":
		p.y -= 5 * math.Abs(math.Sin(q*4.4)) * math.Exp(-q*.6)
		p.angle = .018 * math.Sin(q*4.4) * math.Exp(-q*.6)
	case "waving":
		p.angle = .012 * math.Sin(q*5.4)
	case "stretching":
		amount := math.Sin(math.Min(1, q/3) * math.Pi)
		p.y -= 3 * amount
		p.angle = -.018 * amount
	case "walking":
		step := math.Sin(q * 6.5)
		p.y -= 2.5 * math.Abs(step)
		p.angle = .012 * step
	case "eating":
		p.y += .65 * math.Sin(q*7)
		p.angle = .006 * math.Sin(q*7)
	case "playing":
		p.y -= 3.5 * math.Abs(math.Sin(q*4.2))
		p.angle = .015 * math.Sin(q*4.2)
	case "sleeping":
		p.y += 1 + math.Sin(t*1.3)*.9
	case "dragging":
		p.angle = .025 * math.Sin(t*4)
		p.y -= 3
	}
	return p
}

func (a *PetApp) characterArt() *image.RGBA {
	art, _ := petArt.(*image.RGBA)
	blink := math.Mod(a.Time, 6.1)
	closed := a.Sleeping() || (blink > 5.82 && blink < 5.97)
	if a.Mode == "happy" && a.ActionTime > .35 && a.ActionTime < 1.05 {
		closed = true
	}
	if closed && sleepArt != nil {
		if asleep, ok := sleepArt.(*image.RGBA); ok {
			return asleep
		}
	}
	if a.Mode == "waving" && waveArt != nil && math.Sin(a.ActionTime*6) > .1 {
		if wave, ok := waveArt.(*image.RGBA); ok {
			return wave
		}
	}
	return art
}

func (a *PetApp) Render() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
	p := a.characterPose()
	if elapsed := a.Time - a.ModeChangedAt; a.PreviousMode != "" && elapsed < .24 {
		previous := *a
		previous.Mode, previous.ActionTime = a.PreviousMode, a.PreviousActionTime+elapsed
		from := previous.characterPose()
		mix := math.Max(0, elapsed/.24)
		mix = mix * mix * (3 - 2*mix)
		p.x = from.x + (p.x-from.x)*mix
		p.y = from.y + (p.y-from.y)*mix
		p.angle = from.angle + (p.angle-from.angle)*mix
	}
	art := a.characterArt()
	if art != nil {
		width := 360 * p.scale
		height := width * float64(art.Rect.Dy()) / float64(art.Rect.Dx())
		layer(out, art, p.x, p.y, width, height, p.angle, .5, .5, 1)
	}
	if a.Mode == "eating" {
		drawCookie(out, 204, 287+math.Sin(a.ActionTime*7), 10, a.ActionTime)
	}
	if a.Mode == "playing" {
		q := a.ActionTime
		x, y := 343+20*math.Sin(q*2.8), 390-33*math.Abs(math.Sin(q*4.2))
		ellipse(out, x, 413, 13, 3, color.RGBA{100, 66, 93, 18})
		circle(out, x, y, 14, color.RGBA{205, 146, 173, 255})
		circle(out, x-1, y-1, 12.5, color.RGBA{251, 193, 212, 255})
		line(out, x-11, y+2, x+11, y-2, 1.7, color.RGBA{255, 241, 247, 255})
		line(out, x-3, y-12, x+3, y+12, 1.7, color.RGBA{255, 241, 247, 255})
		circle(out, x-5, y-6, 2.4, color.RGBA{255, 255, 255, 215})
	}
	if a.Sleeping() {
		for i := 0; i < 3; i++ {
			q := math.Mod(a.Time*.22+float64(i)/3, 1)
			s := 4 + q*5
			x, y := 378+q*12, 178-q*60
			c := color.RGBA{176, 148, 174, uint8(170 * (1 - q))}
			line(out, x, y, x+s, y, 1.7, c)
			line(out, x+s, y, x, y+s, 1.7, c)
			line(out, x, y+s, x+s, y+s, 1.7, c)
		}
	}
	for _, v := range a.Particles {
		alpha := uint8(math.Max(0, math.Min(1, 1-v.Age/v.Life)) * 205)
		x, y := v.X*canvasWidth, v.Y*canvasHeight
		switch v.Kind {
		case "heart":
			heart(out, x, y, 5+1.5*math.Sin(v.Age*3), color.RGBA{242, 150, 178, alpha})
		case "star":
			star(out, x, y, 5, color.RGBA{228, 183, 113, alpha})
		default:
			circle(out, x, y, 1.7, color.RGBA{218, 168, 111, alpha})
		}
	}
	if a.Bubble != "" && a.Time < a.BubbleUntil {
		bubble := image.NewRGBA(out.Rect)
		drawBubble(bubble, a.Bubble)
		alpha := math.Min(1, math.Max(0, (a.BubbleUntil-a.Time)/.3))
		fadeComposite(out, bubble, alpha)
	}
	a.drawControls(out)
	return out
}

// Draw one premultiplied sprite around a normalized anatomical pivot.
func layer(dst *image.RGBA, src *image.RGBA, x, y, w, h, angle, px, py, alpha float64) {
	if src == nil || w <= 0 || h <= 0 || alpha <= 0 {
		return
	}
	co, si := math.Cos(angle), math.Sin(angle)
	minX, minY, maxX, maxY := 1e9, 1e9, -1e9, -1e9
	for _, q := range []point{{-px * w, -py * h}, {(1 - px) * w, -py * h}, {-px * w, (1 - py) * h}, {(1 - px) * w, (1 - py) * h}} {
		xx, yy := x+q.x*co-q.y*si, y+q.x*si+q.y*co
		minX = math.Min(minX, xx)
		maxX = math.Max(maxX, xx)
		minY = math.Min(minY, yy)
		maxY = math.Max(maxY, yy)
	}
	for yy := max(0, int(math.Floor(minY))-1); yy < min(dst.Rect.Dy(), int(math.Ceil(maxY))+1); yy++ {
		for xx := max(0, int(math.Floor(minX))-1); xx < min(dst.Rect.Dx(), int(math.Ceil(maxX))+1); xx++ {
			dx, dy := float64(xx)+.5-x, float64(yy)+.5-y
			u, v := (dx*co+dy*si)/w+px, (-dx*si+dy*co)/h+py
			if u < 0 || u >= 1 || v < 0 || v >= 1 {
				continue
			}
			c := sampleSprite(src, u*float64(src.Rect.Dx()), v*float64(src.Rect.Dy()))
			if alpha < 1 {
				c.R = uint8(float64(c.R) * alpha)
				c.G = uint8(float64(c.G) * alpha)
				c.B = uint8(float64(c.B) * alpha)
				c.A = uint8(float64(c.A) * alpha)
			}
			if c.A > 0 {
				blendPremult(dst, xx, yy, c.R, c.G, c.B, c.A)
			}
		}
	}
}

func fadeComposite(dst, src *image.RGBA, alpha float64) {
	for y := 0; y < src.Rect.Dy(); y++ {
		for x := 0; x < src.Rect.Dx(); x++ {
			i := src.PixOffset(x, y)
			if src.Pix[i+3] == 0 {
				continue
			}
			blendPremult(dst, x, y, uint8(float64(src.Pix[i])*alpha), uint8(float64(src.Pix[i+1])*alpha), uint8(float64(src.Pix[i+2])*alpha), uint8(float64(src.Pix[i+3])*alpha))
		}
	}
}

func drawCookie(d *image.RGBA, x, y, r, t float64) {
	circle(d, x, y, r, color.RGBA{164, 108, 68, 255})
	circle(d, x, y-1, r-2, color.RGBA{240, 201, 143, 255})
	for i := 0; i < 5; i++ {
		angle := float64(i) * 2.4
		circle(d, x+math.Cos(angle)*r*.54, y+math.Sin(angle)*r*.54, 1.6, color.RGBA{164, 108, 68, 255})
	}
	if t > 0 {
		heart(d, x, y, 4, color.RGBA{233, 147, 158, 255})
	}
}

func (a *PetApp) drawControls(dst *image.RGBA) {
	alpha := a.ControlsAlpha
	if alpha < .003 {
		return
	}
	d := image.NewRGBA(dst.Rect)
	roundRect(d, 106, 420, 228, 51, 23, color.RGBA{92, 65, 83, 24})
	roundRect(d, 106, 417, 228, 51, 23, color.RGBA{255, 247, 249, 250})
	hover := toolbarIndex(a.pointerX, a.pointerY)
	for i, x := range []float64{136, 192, 248, 304} {
		if i == hover {
			circle(d, x, 444, 21, color.RGBA{250, 223, 233, 255})
		}
		switch i {
		case 0:
			heart(d, x, 444, 10, color.RGBA{223, 130, 163, 255})
		case 1:
			drawCookie(d, x, 444, 11, 0)
		case 2:
			circle(d, x, 444, 11, color.RGBA{165, 186, 201, 255})
			line(d, x-9, 442, x+9, 446, 2, cream)
			line(d, x-3, 434, x+3, 454, 2, cream)
		case 3:
			// A crescent constructed without painting opaque pixels outside it.
			for yy := 432; yy < 456; yy++ {
				for xx := int(x) - 12; xx < int(x)+12; xx++ {
					r := math.Hypot(float64(xx)+.5-x, float64(yy)+.5-444)
					cut := math.Hypot(float64(xx)+.5-(x+5), float64(yy)+.5-440)
					if r < 11 && cut > 9 {
						pixel(d, xx, yy, color.RGBA{169, 149, 190, 255}, math.Min(11-r, cut-9)+.5)
					}
				}
			}
		}
	}
	fadeComposite(dst, d, alpha)
}

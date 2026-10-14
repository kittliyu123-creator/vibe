package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

var rigArt map[string]*image.RGBA
var petBrown = color.RGBA{116, 60, 37, 255}

func loadRig() error {
	rigArt = make(map[string]*image.RGBA)
	for _, name := range []string{"head", "body", "paw", "foot"} {
		data, err := assetFS.ReadFile("assets/rig-" + name + ".png")
		if err != nil {
			return err
		}
		art, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		// Layer bounds are determined once; the animation never distorts a full
		// character image to approximate moving its limbs.
		b := art.Bounds()
		crop := image.Rectangle{Min: b.Max, Max: b.Min}
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				_, _, _, a := art.At(x, y).RGBA()
				if a < 8*257 {
					continue
				}
				if x < crop.Min.X {
					crop.Min.X = x
				}
				if y < crop.Min.Y {
					crop.Min.Y = y
				}
				if x >= crop.Max.X {
					crop.Max.X = x + 1
				}
				if y >= crop.Max.Y {
					crop.Max.Y = y + 1
				}
			}
		}
		if crop.Empty() {
			return fmt.Errorf("empty %s layer", name)
		}
		crop = crop.Inset(-3).Intersect(b)
		tight := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
		draw.Draw(tight, tight.Rect, art, crop.Min, draw.Src)
		rigArt[name] = prepareSprite(tight)
	}
	return nil
}

type pose struct {
	headX, headY, headAngle                float64
	bodyY, bodyAngle                       float64
	leftX, leftY, leftAngle                float64
	rightX, rightY, rightAngle             float64
	leftFootX, leftFootY, leftFootAngle    float64
	rightFootX, rightFootY, rightFootAngle float64
	armHeight, eyesOpen                    float64
}

func blendPose(a, b pose, t float64) pose {
	mix := func(x, y float64) float64 { return x + (y-x)*t }
	return pose{
		headX: mix(a.headX, b.headX), headY: mix(a.headY, b.headY), headAngle: mix(a.headAngle, b.headAngle),
		bodyY: mix(a.bodyY, b.bodyY), bodyAngle: mix(a.bodyAngle, b.bodyAngle),
		leftX: mix(a.leftX, b.leftX), leftY: mix(a.leftY, b.leftY), leftAngle: mix(a.leftAngle, b.leftAngle),
		rightX: mix(a.rightX, b.rightX), rightY: mix(a.rightY, b.rightY), rightAngle: mix(a.rightAngle, b.rightAngle),
		leftFootX: mix(a.leftFootX, b.leftFootX), leftFootY: mix(a.leftFootY, b.leftFootY), leftFootAngle: mix(a.leftFootAngle, b.leftFootAngle),
		rightFootX: mix(a.rightFootX, b.rightFootX), rightFootY: mix(a.rightFootY, b.rightFootY), rightFootAngle: mix(a.rightFootAngle, b.rightFootAngle),
		armHeight: mix(a.armHeight, b.armHeight), eyesOpen: mix(a.eyesOpen, b.eyesOpen),
	}
}

func (a *PetApp) currentPose() pose {
	t, p := a.Time, a.ActionTime
	breath := math.Sin(t * 1.85)
	v := pose{
		headX: 220 + a.LookX*3, headY: 238 + breath*1.1 + a.LookY*1.8, headAngle: a.LookX*.045 + math.Sin(t*.9)*.008,
		bodyY: 351 + breath*.8, bodyAngle: math.Sin(t*.9) * .012,
		leftX: 184, leftY: 348, leftAngle: -.15 - breath*.025,
		rightX: 256, rightY: 348, rightAngle: .15 + breath*.025,
		leftFootX: 189, leftFootY: 390, leftFootAngle: -.13,
		rightFootX: 251, rightFootY: 390, rightFootAngle: .13,
		armHeight: 66, eyesOpen: 1,
	}
	// Blink has closing/closed/opening phases, including an occasional double blink.
	blink := math.Mod(t, 5.9)
	if blink > 5.46 && blink < 5.70 {
		v.eyesOpen = math.Min(1, math.Abs(blink-5.58)/.10)
	}
	if math.Mod(t, 17) > 16.7 {
		v.eyesOpen = math.Min(v.eyesOpen, math.Abs(math.Mod(t, 17)-16.85)/.12)
	}
	strength := math.Min(1, p/.20)
	switch a.Mode {
	case "happy":
		wave := math.Sin(p * 6)
		v.headAngle += .095 * wave * math.Exp(-p*.6)
		v.headY -= 4 * math.Sin(p*3) * math.Exp(-p*.5)
		v.leftAngle -= .12 * wave
		v.rightAngle += .12 * wave
		v.leftFootY -= 3 * math.Max(0, wave)
		v.rightFootY -= 3 * math.Max(0, -wave)
		if p > .3 && p < 1.6 {
			v.eyesOpen = .12
		}
	case "waving":
		strength *= math.Min(1, math.Max(0, (2.5-p)/.35))
		v.rightX += 25 * strength
		v.rightY -= 24 * strength
		v.rightAngle += (.55 + .35*math.Sin(p*10)) * strength
		v.headAngle -= .06 * strength
	case "stretching":
		strength = math.Sin(math.Min(1, p/3) * math.Pi)
		v.leftX -= 16 * strength
		v.rightX += 16 * strength
		v.leftY -= 27 * strength
		v.rightY -= 27 * strength
		v.leftAngle -= .55 * strength
		v.rightAngle += .55 * strength
		v.headY -= 7 * strength
		v.bodyY -= 4 * strength
		v.eyesOpen = 1 - .9*strength
	case "walking":
		stride := math.Sin(p * 7)
		v.headY -= 2 * math.Abs(stride)
		v.bodyY -= 2.4 * math.Abs(stride)
		v.headAngle += .035 * stride
		v.bodyAngle += .04 * stride
		v.leftFootX += 8 * stride
		v.rightFootX -= 8 * stride
		v.leftFootY -= 8 * math.Max(0, stride)
		v.rightFootY -= 8 * math.Max(0, -stride)
		v.leftFootAngle += .22 * stride
		v.rightFootAngle -= .22 * stride
		v.leftY += 10
		v.rightY += 10
		v.armHeight = 58
		v.leftAngle = .35*stride - .4
		v.rightAngle = .35*stride + .4
	case "eating":
		v.leftAngle = .65
		v.rightAngle = -.65
		v.armHeight = 53
		v.leftX = 190
		v.rightX = 250
		v.leftY = 348
		v.rightY = 348
		v.headY += 2 * math.Abs(math.Sin(p*8))
		v.headAngle = .025 * math.Sin(p*8)
	case "playing":
		kick := math.Sin(p * 5)
		v.rightX += 23
		v.rightAngle = .6 + .45*kick
		v.rightY += 6 * math.Max(0, kick)
		v.headAngle += .06 * kick
		v.headY -= 5 * math.Abs(kick)
		v.rightFootAngle += .38 * math.Max(0, -kick)
	case "sleeping":
		v.eyesOpen = 0
		v.headY = 250 + breath*1.8
		v.headAngle = -.12
		v.leftX = 194
		v.rightX = 241
		v.leftY = 350
		v.rightY = 350
		v.leftAngle = .45
		v.rightAngle = -.25
		v.armHeight = 55
	case "dragging":
		v.headAngle = .10 * math.Sin(t*6)
		v.leftY = 355
		v.rightY = 355
		v.leftAngle = -2.8 + .2*math.Sin(t*5)
		v.rightAngle = 2.8 + .2*math.Sin(t*5)
		v.armHeight = 53
		v.leftFootY += 8
		v.rightFootY += 8
		v.leftFootAngle -= .18
		v.rightFootAngle += .18
	}
	return v
}

func (a *PetApp) Render() *image.RGBA {
	if len(rigArt) != 4 {
		return a.legacyRender()
	}
	out := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
	p := a.currentPose()
	if elapsed := a.Time - a.ModeChangedAt; a.PreviousMode != "" && elapsed < .24 {
		previous := *a
		previous.Mode, previous.ActionTime = a.PreviousMode, a.PreviousActionTime+elapsed
		t := math.Max(0, elapsed/.24)
		t = t * t * (3 - 2*t)
		p = blendPose(previous.currentPose(), p, t)
	}
	// The grounded shadow stays separate from the independently posed parts.
	for i := 3; i > 0; i-- {
		ellipse(out, 220, 414, float64(73+i*4), float64(5+i*2), color.RGBA{98, 67, 88, uint8(7 + i)})
	}
	layer(out, rigArt["body"], 230, p.bodyY, 147, 105, p.bodyAngle, .5, .5, 1)
	layer(out, rigArt["foot"], p.leftFootX, p.leftFootY, 59, 61, p.leftFootAngle, .5, .5, 1)
	layer(out, rigArt["foot"], p.rightFootX, p.rightFootY, 59, 61, p.rightFootAngle, .5, .5, 1)
	// Compose facial features in head space, then rotate the entire head.
	head := a.faceLayer(p.eyesOpen)
	headH := 262 * float64(head.Rect.Dy()) / float64(head.Rect.Dx())
	layer(out, head, p.headX, p.headY, 262, headH, p.headAngle, .5, .5, 1)
	layer(out, rigArt["paw"], p.leftX, p.leftY, 44, p.armHeight, p.leftAngle, .5, .92, 1)
	layer(out, rigArt["paw"], p.rightX, p.rightY, 44, p.armHeight, p.rightAngle, .5, .92, 1)
	if a.Mode == "eating" {
		drawCookie(out, 220, 315+2*math.Sin(a.ActionTime*8), 15, a.ActionTime)
	}
	if a.Mode == "playing" {
		q := a.ActionTime
		x, y := 316+25*math.Sin(q*2.8), 391-36*math.Abs(math.Sin(q*5))
		ellipse(out, x, 415, 17, 4, color.RGBA{100, 66, 93, 24})
		circle(out, x, y, 17, color.RGBA{204, 147, 174, 255})
		circle(out, x-1, y-2, 15, color.RGBA{251, 188, 209, 255})
		line(out, x-13, y+2, x+12, y-3, 2, color.RGBA{255, 239, 245, 255})
		line(out, x-4, y-14, x+4, y+14, 2, color.RGBA{255, 239, 245, 255})
		circle(out, x-6, y-7, 3, color.RGBA{255, 255, 255, 210})
	}
	if a.Sleeping() {
		for i := 0; i < 3; i++ {
			q := math.Mod(a.Time*.25+float64(i)/3, 1)
			s := 5 + q*6
			x, y := 304+q*29, 219-q*76
			c := color.RGBA{166, 142, 177, uint8(190 * (1 - q))}
			line(out, x, y, x+s, y, 2, c)
			line(out, x+s, y, x, y+s, 2, c)
			line(out, x, y+s, x+s, y+s, 2, c)
		}
	}
	for _, v := range a.Particles {
		alpha := uint8(math.Max(0, math.Min(1, 1-v.Age/v.Life)) * 210)
		x, y := v.X*canvasWidth, v.Y*canvasHeight
		switch v.Kind {
		case "heart":
			heart(out, x, y, 5+2*math.Sin(v.Age*3), color.RGBA{242, 150, 178, alpha})
		case "star":
			star(out, x, y, 6, color.RGBA{228, 183, 113, alpha})
		default:
			circle(out, x, y, 2.1, color.RGBA{218, 168, 111, alpha})
		}
	}
	if a.Bubble != "" && a.Time < a.BubbleUntil {
		bubble := image.NewRGBA(out.Rect)
		drawBubble(bubble, a.Bubble)
		alpha := math.Min(1, math.Max(0, (a.BubbleUntil-a.Time)/.30))
		fadeComposite(out, bubble, alpha)
	}
	a.drawControls(out)
	return out
}

func (a *PetApp) faceLayer(open float64) *image.RGBA {
	head := rigArt["head"]
	out := image.NewRGBA(head.Rect)
	copy(out.Pix, head.Pix)
	w, h := float64(out.Rect.Dx()), float64(out.Rect.Dy())
	eyeY := h*.675 + a.LookY*3
	for _, fx := range []float64{.325, .675} {
		x := w*fx + a.LookX*6
		if open < .22 {
			// A soft curved eyelid, rather than a shrunken oval.
			lastX, lastY := x-w*.022, eyeY
			for i := 1; i <= 10; i++ {
				q := float64(i) / 10
				xx := x - w*.022 + q*w*.044
				yy := eyeY + math.Sin(q*math.Pi)*h*.014
				line(out, lastX, lastY, xx, yy, 3.4, petBrown)
				lastX, lastY = xx, yy
			}
		} else {
			ellipse(out, x, eyeY, w*.019, h*.032*open, petBrown)
		}
	}
	nx, ny := w*.5+a.LookX*2, h*.752+a.LookY*1.8
	ellipse(out, nx, ny, w*.031, h*.026, petBrown)
	ellipse(out, nx, ny, w*.023, h*.0175, color.RGBA{255, 222, 129, 255})
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

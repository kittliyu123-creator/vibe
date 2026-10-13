package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const canvasWidth = 360
const canvasHeight = 380

//go:embed assets/*
var assetFS embed.FS

var petArt image.Image
var sleepArt image.Image
var fontAtlas image.Image
var fontGlyphs map[string]int
var ink = color.RGBA{66, 60, 76, 255}
var cream = color.RGBA{255, 250, 239, 255}
var rose = color.RGBA{246, 136, 159, 255}

func loadArtwork() error {
	b, err := assetFS.ReadFile("assets/font.png")
	if err != nil {
		return err
	}
	fontAtlas, err = png.Decode(bytes.NewReader(b))
	if err != nil {
		return err
	}
	b, err = assetFS.ReadFile("assets/glyphs.json")
	if err != nil {
		return err
	}
	if err = json.Unmarshal(b, &fontGlyphs); err != nil {
		return err
	}
	// An adjacent PNG lets the user change the pet without recompiling.
	if exe, e := os.Executable(); e == nil {
		p := filepath.Join(filepath.Dir(exe), "pet.png")
		if b, e = os.ReadFile(p); e == nil {
			petArt, e = png.Decode(bytes.NewReader(b))
			if e != nil {
				return fmt.Errorf("pet.png: %w", e)
			}
			petArt = prepareSprite(petArt)
			return nil
		}
	}
	if b, err = assetFS.ReadFile("assets/pet.png"); err == nil {
		petArt, err = png.Decode(bytes.NewReader(b))
		if err != nil {
			return err
		}
		petArt = prepareSprite(petArt)
	}
	if b, err = assetFS.ReadFile("assets/pet-sleep.png"); err == nil {
		sleepArt, err = png.Decode(bytes.NewReader(b))
		if err != nil {
			return err
		}
		sleepArt = prepareSprite(sleepArt)
	}
	return nil
}

func (a *PetApp) Render() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
	t := a.Time
	phase := a.ActionTime
	bob := 2.8 * math.Sin(t*2.1)
	sx, sy := 1.0+0.018*math.Sin(t*2.1), 1.0-0.018*math.Sin(t*2.1)
	angle := 0.016 * math.Sin(t*1.4)
	y := 292.0 + bob
	switch a.Mode {
	case "happy":
		y -= 15 * math.Abs(math.Sin(phase*8)) * math.Exp(-phase*0.6)
		angle = 0.06 * math.Sin(phase*12)
	case "eating":
		sy -= 0.05 * math.Abs(math.Sin(phase*10))
		angle = 0.04 * math.Sin(phase*8)
	case "playing":
		y -= 32 * math.Abs(math.Sin(phase*5))
		angle = 0.12 * math.Sin(phase*5)
	case "sleeping":
		sx = 1.07 + 0.025*math.Sin(t*1.8)
		sy = 0.88 - 0.025*math.Sin(t*1.8)
		y = 303
		angle = -0.05
	case "dragging":
		sy, sx = 1.07, 0.96
		angle = 0.10 * math.Sin(t*7)
	}
	ellipse(out, 180, 324, 75-0.25*(292-y), 10, color.RGBA{60, 50, 70, 34})
	if petArt == nil {
		drawPlaceholder(out, 180, y, sx, sy, angle, a.Mode, t)
	} else {
		art := petArt
		if sleepArt != nil && (a.Mode == "sleeping" || (a.Mode == "idle" && math.Mod(t, 5.3) > 5.12)) {
			art = sleepArt
		}
		drawSprite(out, art, 180, y+26, sx, sy, angle)
	}
	if a.Mode == "eating" {
		roundRect(out, 150, 308, 60, 18, 8, color.RGBA{192, 178, 222, 255})
		ellipse(out, 180, 308, 30, 7, color.RGBA{114, 99, 145, 255})
		for i := 0; i < 6; i++ {
			circle(out, 160+float64(i)*7, 307+2*math.Sin(float64(i)*2), 3, color.RGBA{217, 174, 107, 255})
		}
	}
	if a.Mode == "playing" {
		bx, by := 266+28*math.Sin(phase*3), 310-38*math.Abs(math.Sin(phase*5))
		circle(out, bx, by, 16, color.RGBA{151, 200, 204, 255})
		line(out, bx-10, by-9, bx+10, by+9, 3, cream)
		line(out, bx-10, by+9, bx+10, by-9, 3, cream)
	}
	if a.Mode == "sleeping" {
		for i := 0; i < 3; i++ {
			p := math.Mod(t*0.4+float64(i)/3, 1)
			c := color.RGBA{133, 119, 163, uint8(220 * (1 - p))}
			x, y := 240+p*34, 186-p*70
			line(out, x, y, x+12, y, 2.5, c)
			line(out, x+12, y, x, y+13, 2.5, c)
			line(out, x, y+13, x+12, y+13, 2.5, c)
		}
	}
	for _, p := range a.Particles {
		x, y := p.X*canvasWidth, p.Y*canvasHeight
		alpha := uint8(math.Max(0, math.Min(255, 255*(1-p.Age/p.Life))))
		switch p.Kind {
		case "heart":
			heart(out, x, y, 9, color.RGBA{246, 136, 159, alpha})
		case "star":
			star(out, x, y, 8, color.RGBA{238, 194, 106, alpha})
		default:
			circle(out, x, y, 3, color.RGBA{217, 174, 107, alpha})
		}
	}
	if a.Bubble != "" && a.Time < a.BubbleUntil {
		drawBubble(out, a.Bubble)
	}
	return out
}

// Neutral construction placeholder, replaced after the user's reference arrives.
func drawPlaceholder(dst *image.RGBA, x, y, sx, sy, angle float64, mode string, t float64) {
	p := image.NewRGBA(image.Rect(0, 0, 220, 230))
	outline := color.RGBA{107, 98, 112, 255}
	ellipse(p, 70, 45, 28, 36, outline)
	ellipse(p, 150, 45, 28, 36, outline)
	ellipse(p, 70, 44, 24, 32, cream)
	ellipse(p, 150, 44, 24, 32, cream)
	ellipse(p, 70, 46, 12, 21, color.RGBA{244, 198, 201, 255})
	ellipse(p, 150, 46, 12, 21, color.RGBA{244, 198, 201, 255})
	ellipse(p, 110, 133, 89, 83, outline)
	ellipse(p, 110, 132, 85, 80, cream)
	ellipse(p, 62, 203, 29, 14, outline)
	ellipse(p, 158, 203, 29, 14, outline)
	ellipse(p, 62, 201, 26, 12, cream)
	ellipse(p, 158, 201, 26, 12, cream)
	ellipse(p, 58, 140, 14, 8, color.RGBA{250, 204, 204, 255})
	ellipse(p, 162, 140, 14, 8, color.RGBA{250, 204, 204, 255})
	closed := mode == "sleeping" || math.Mod(t, 4.7) < 0.16
	if closed {
		line(p, 73, 119, 88, 123, 4, ink)
		line(p, 132, 123, 147, 119, 4, ink)
	} else {
		ellipse(p, 80, 120, 5.5, 9, ink)
		ellipse(p, 140, 120, 5.5, 9, ink)
		circle(p, 81.5, 117, 1.8, cream)
		circle(p, 141.5, 117, 1.8, cream)
	}
	ellipse(p, 110, 136, 4.5, 3.3, rose)
	if mode == "happy" || mode == "playing" {
		ellipse(p, 110, 149, 7, 6, ink)
		ellipse(p, 110, 151, 4, 2, rose)
	} else {
		line(p, 103, 146, 110, 149, 2, ink)
		line(p, 110, 149, 117, 146, 2, ink)
	}
	// Soft scarf keeps the placeholder visibly provisional but lively.
	roundRect(p, 74, 172, 72, 14, 6, color.RGBA{147, 185, 188, 255})
	roundRect(p, 127, 178, 15, 24, 5, color.RGBA{147, 185, 188, 255})
	drawSprite(dst, p, x, y+26, sx, sy, angle)
}

func drawSprite(dst *image.RGBA, src image.Image, cx, base, sx, sy, angle float64) {
	b := src.Bounds()
	fit := math.Min(218/float64(b.Dx()), 228/float64(b.Dy()))
	sx *= fit
	sy *= fit
	co, si := math.Cos(angle), math.Sin(angle)
	cy := base - float64(b.Dy())*sy/2
	for y := 82; y < 344; y++ {
		for x := 35; x < 325; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			u, v := (dx*co+dy*si)/sx+float64(b.Dx())/2, (-dx*si+dy*co)/sy+float64(b.Dy())/2
			if u < 0 || v < 0 || u >= float64(b.Dx()) || v >= float64(b.Dy()) {
				continue
			}
			c := sampleSprite(src, u, v)
			if c.A > 0 {
				blendPremult(dst, x, y, c.R, c.G, c.B, c.A)
			}
		}
	}
}

// Prefilter large source art once, then bilinear sample animation frames.
// This preserves soft outlines at desktop size and avoids allocating per pixel.
func prepareSprite(src image.Image) *image.RGBA {
	b := src.Bounds()
	scale := math.Min(1, 420.0/math.Max(float64(b.Dx()), float64(b.Dy())))
	w, h := int(float64(b.Dx())*scale), int(float64(b.Dy())*scale)
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			x0, x1 := x*b.Dx()/w, (x+1)*b.Dx()/w
			y0, y1 := y*b.Dy()/h, (y+1)*b.Dy()/h
			var sum [4]uint64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					r, g, bl, a := src.At(b.Min.X+xx, b.Min.Y+yy).RGBA()
					sum[0] += uint64(r)
					sum[1] += uint64(g)
					sum[2] += uint64(bl)
					sum[3] += uint64(a)
				}
			}
			n := uint64((x1-x0)*(y1-y0)) * 257
			i := out.PixOffset(x, y)
			for k := 0; k < 4; k++ {
				out.Pix[i+k] = uint8(sum[k] / n)
			}
		}
	}
	return out
}

func sampleSprite(src image.Image, u, v float64) color.RGBA {
	if rgba, ok := src.(*image.RGBA); ok {
		u -= .5
		v -= .5
		x, y := int(math.Floor(u)), int(math.Floor(v))
		fx, fy := u-float64(x), v-float64(y)
		var ch [4]float64
		for iy := 0; iy < 2; iy++ {
			for ix := 0; ix < 2; ix++ {
				xx, yy := x+ix, y+iy
				if xx < 0 || yy < 0 || xx >= rgba.Rect.Dx() || yy >= rgba.Rect.Dy() {
					continue
				}
				wx, wy := 1-fx, 1-fy
				if ix == 1 {
					wx = fx
				}
				if iy == 1 {
					wy = fy
				}
				i := rgba.PixOffset(xx+rgba.Rect.Min.X, yy+rgba.Rect.Min.Y)
				for k := 0; k < 4; k++ {
					ch[k] += float64(rgba.Pix[i+k]) * wx * wy
				}
			}
		}
		return color.RGBA{uint8(ch[0] + .5), uint8(ch[1] + .5), uint8(ch[2] + .5), uint8(ch[3] + .5)}
	}
	r, g, b, a := src.At(int(u), int(v)).RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func drawBubble(dst *image.RGBA, text string) {
	lines := strings.Split(text, "\n")
	var wrapped []string
	for _, ln := range lines {
		r := []rune(ln)
		for len(r) > 12 {
			wrapped = append(wrapped, string(r[:12]))
			r = r[12:]
		}
		if len(r) > 0 {
			wrapped = append(wrapped, string(r))
		}
	}
	if len(wrapped) > 2 {
		wrapped = wrapped[:2]
	}
	if len(wrapped) == 0 {
		return
	}
	w := 0
	for _, ln := range wrapped {
		if len([]rune(ln)) > w {
			w = len([]rune(ln))
		}
	}
	bw, bh := float64(w*20+34), float64(len(wrapped)*27+24)
	x, y := (canvasWidth-bw)/2, 82.0-bh
	roundRect(dst, x+1, y+4, bw, bh, 17, color.RGBA{69, 57, 76, 26})
	roundRect(dst, x, y, bw, bh, 17, color.RGBA{255, 251, 244, 249})
	polygon(dst, []point{{173, y + bh - 1}, {186, y + bh - 1}, {180, y + bh + 9}}, color.RGBA{255, 251, 244, 249})
	for i, ln := range wrapped {
		drawText(dst, ln, 180-len([]rune(ln))*10, int(y)+12+i*27)
	}
}

func drawText(dst *image.RGBA, s string, x, y int) {
	for _, r := range s {
		i, ok := fontGlyphs[string(r)]
		if !ok {
			i = fontGlyphs["?"]
		}
		src := image.Pt((i%32)*24, (i/32)*28)
		draw.Draw(dst, image.Rect(x, y, x+24, y+28), fontAtlas, src, draw.Over)
		x += 20
	}
}

type point struct{ x, y float64 }

func blendPremult(d *image.RGBA, x, y int, r, g, b, a uint8) {
	if !image.Pt(x, y).In(d.Rect) {
		return
	}
	i := d.PixOffset(x, y)
	inv := uint32(255 - a)
	d.Pix[i] = uint8(uint32(r) + uint32(d.Pix[i])*inv/255)
	d.Pix[i+1] = uint8(uint32(g) + uint32(d.Pix[i+1])*inv/255)
	d.Pix[i+2] = uint8(uint32(b) + uint32(d.Pix[i+2])*inv/255)
	d.Pix[i+3] = uint8(uint32(a) + uint32(d.Pix[i+3])*inv/255)
}

func pixel(d *image.RGBA, x, y int, c color.RGBA, coverage float64) {
	a := uint8(float64(c.A) * math.Max(0, math.Min(1, coverage)))
	blendPremult(d, x, y, uint8(uint32(c.R)*uint32(a)/255), uint8(uint32(c.G)*uint32(a)/255), uint8(uint32(c.B)*uint32(a)/255), a)
}

func ellipse(d *image.RGBA, cx, cy, rx, ry float64, c color.RGBA) {
	if rx <= 0 || ry <= 0 {
		return
	}
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			q := math.Sqrt(math.Pow((float64(x)+.5-cx)/rx, 2) + math.Pow((float64(y)+.5-cy)/ry, 2))
			pixel(d, x, y, c, (1-q)*math.Min(rx, ry)+.5)
		}
	}
}
func circle(d *image.RGBA, x, y, r float64, c color.RGBA) { ellipse(d, x, y, r, r, c) }

func roundRect(d *image.RGBA, x, y, w, h, r float64, c color.RGBA) {
	for py := int(y - 1); py < int(y+h+1); py++ {
		for px := int(x - 1); px < int(x+w+1); px++ {
			dx := math.Max(math.Abs(float64(px)+.5-(x+w/2))-(w/2-r), 0)
			dy := math.Max(math.Abs(float64(py)+.5-(y+h/2))-(h/2-r), 0)
			pixel(d, px, py, c, r-math.Hypot(dx, dy)+.5)
		}
	}
}
func line(d *image.RGBA, x1, y1, x2, y2, w float64, c color.RGBA) {
	len2 := (x2-x1)*(x2-x1) + (y2-y1)*(y2-y1)
	if len2 == 0 {
		circle(d, x1, y1, w/2, c)
		return
	}
	for y := int(math.Min(y1, y2) - w); y <= int(math.Max(y1, y2)+w); y++ {
		for x := int(math.Min(x1, x2) - w); x <= int(math.Max(x1, x2)+w); x++ {
			p := math.Max(0, math.Min(1, ((float64(x)+.5-x1)*(x2-x1)+(float64(y)+.5-y1)*(y2-y1))/len2))
			pixel(d, x, y, c, w/2-math.Hypot(float64(x)+.5-x1-p*(x2-x1), float64(y)+.5-y1-p*(y2-y1))+.5)
		}
	}
}
func polygon(d *image.RGBA, pts []point, c color.RGBA) {
	minX, minY, maxX, maxY := 1e9, 1e9, -1e9, -1e9
	for _, p := range pts {
		minX = math.Min(minX, p.x)
		minY = math.Min(minY, p.y)
		maxX = math.Max(maxX, p.x)
		maxY = math.Max(maxY, p.y)
	}
	for y := int(minY); y <= int(maxY); y++ {
		for x := int(minX); x <= int(maxX); x++ {
			inside := false
			j := len(pts) - 1
			for i, p := range pts {
				q := pts[j]
				if (p.y > float64(y)) != (q.y > float64(y)) && float64(x) < (q.x-p.x)*(float64(y)-p.y)/(q.y-p.y)+p.x {
					inside = !inside
				}
				j = i
			}
			if inside {
				pixel(d, x, y, c, 1)
			}
		}
	}
}
func star(d *image.RGBA, x, y, r float64, c color.RGBA) {
	pts := make([]point, 10)
	for i := range pts {
		rr := r
		if i%2 == 1 {
			rr *= .44
		}
		a := -math.Pi/2 + float64(i)*math.Pi/5
		pts[i] = point{x + rr*math.Cos(a), y + rr*math.Sin(a)}
	}
	polygon(d, pts, c)
}
func heart(d *image.RGBA, x, y, r float64, c color.RGBA) {
	circle(d, x-r*.35, y-r*.25, r*.52, c)
	circle(d, x+r*.35, y-r*.25, r*.52, c)
	polygon(d, []point{{x - r*.83, y - r*.05}, {x + r*.83, y - r*.05}, {x, y + r*.9}}, c)
}

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

const canvasWidth = 440
const canvasHeight = 480

//go:embed assets/*
var assetFS embed.FS

var petArt image.Image
var sleepArt image.Image
var waveArt image.Image
var fontAtlas image.Image
var fontGlyphs map[string]int
var cream = color.RGBA{255, 250, 239, 255}

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
	// An adjacent complete PNG overrides the illustration without changing its proportions.
	if exe, e := os.Executable(); e == nil {
		p := filepath.Join(filepath.Dir(exe), "pet.png")
		if b, e = os.ReadFile(p); e == nil {
			petArt, e = png.Decode(bytes.NewReader(b))
			if e != nil {
				return fmt.Errorf("pet.png: %w", e)
			}
			petArt = prepareSprite(petArt)
			sleepArt = nil // A custom image must not blink into the embedded character.
			waveArt = nil
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
	if b, err = assetFS.ReadFile("assets/pet-wave.png"); err == nil {
		waveArt, err = png.Decode(bytes.NewReader(b))
		if err != nil {
			return err
		}
		waveArt = prepareSprite(waveArt)
	}
	return nil
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
	center := float64(canvasWidth) / 2
	polygon(dst, []point{{center - 7, y + bh - 1}, {center + 6, y + bh - 1}, {center, y + bh + 9}}, color.RGBA{255, 251, 244, 249})
	for i, ln := range wrapped {
		drawText(dst, ln, canvasWidth/2-len([]rune(ln))*10, int(y)+12+i*27)
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

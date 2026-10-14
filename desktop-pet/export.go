package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func exportIcon(path string) error {
	if petArt == nil {
		return fmt.Errorf("missing pet artwork")
	}
	out := image.NewRGBA(image.Rect(0, 0, 256, 256))
	b := petArt.Bounds()
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			out.SetRGBA(x, y, sampleSprite(petArt, (float64(x)+.5)*float64(b.Dx())/256, (float64(y)+.5)*float64(b.Dy())/256))
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, out); err != nil {
		return err
	}
	var ico bytes.Buffer
	for _, v := range []uint16{0, 1, 1} {
		if err := binary.Write(&ico, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	ico.Write([]byte{0, 0, 0, 0})
	for _, v := range []uint16{1, 32} {
		if err := binary.Write(&ico, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	for _, v := range []uint32{uint32(encoded.Len()), 22} {
		if err := binary.Write(&ico, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	ico.Write(encoded.Bytes())
	return os.WriteFile(path, ico.Bytes(), 0644)
}

// Export uses the same renderer and behavior as the live Windows pet.
func exportDemo(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	a := newPetApp()
	for i := 0; i < 320; i++ {
		switch i {
		case 0:
			a.Action("wave")
		case 56:
			a.Action("stretch")
		case 116:
			a.Action("feed")
		case 180:
			a.Action("play")
		case 260:
			a.Action("sleep")
		}
		a.SetPointer(240+85*math.Sin(float64(i)*.035), 260)
		a.Tick(1.0 / 20)
		out := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
		draw.Draw(out, out.Rect, &image.Uniform{C: color.RGBA{245, 238, 241, 255}}, image.Point{}, draw.Src)
		draw.Draw(out, out.Rect, a.Render(), image.Point{}, draw.Over)
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%04d.png", i)))
		if err != nil {
			return err
		}
		err = png.Encode(f, out)
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func exportWalkDemo(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	a := newPetApp()
	a.Action("follow")
	position := 80.0
	for i := 0; i < 200; i++ {
		target := 640.0
		if i >= 100 {
			target = 20
		}
		a.SetPointer(target-position, 230)
		dx, _ := a.Tick(.05)
		position += float64(dx)
		out := image.NewRGBA(image.Rect(0, 0, 720, 480))
		draw.Draw(out, out.Rect, &image.Uniform{C: color.RGBA{245, 238, 241, 255}}, image.Point{}, draw.Src)
		draw.Draw(out, image.Rect(int(position), 0, int(position)+canvasWidth, canvasHeight), a.Render(), image.Point{}, draw.Over)
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%04d.png", i)))
		if err != nil {
			return err
		}
		err = png.Encode(f, out)
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

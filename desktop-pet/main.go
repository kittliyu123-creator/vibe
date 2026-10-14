package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	preview := flag.String("preview", "", "render interaction previews to this directory")
	demo := flag.String("demo", "", "render a short animation to this directory")
	walkDemo := flag.String("walk-demo", "", "render desktop walking demonstration")
	icon := flag.String("icon", "", "export Windows application icon")
	flag.Parse()
	if err := loadArtwork(); err != nil {
		showError(err)
		return
	}
	app := newPetApp()
	if *walkDemo != "" {
		if err := exportWalkDemo(*walkDemo); err != nil {
			showError(err)
			os.Exit(1)
		}
		return
	}
	if *icon != "" {
		if err := exportIcon(*icon); err != nil {
			showError(err)
			os.Exit(1)
		}
		return
	}
	if *demo != "" {
		if err := exportDemo(*demo); err != nil {
			showError(err)
			os.Exit(1)
		}
		return
	}
	if *preview != "" {
		if err := previewFrames(*preview); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := runNative(app); err != nil {
		showError(err)
	}
}

func previewFrames(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, action := range []string{"hello", "pet", "wave", "stretch", "feed", "play", "sleep", "follow"} {
		app := newPetApp()
		app.SetPointer(300, 250)
		if action == "follow" {
			app.SetPointer(650, 250)
		}
		app.Action(action)
		for i := 0; i < 18; i++ {
			app.Tick(1.0 / 30)
		}
		f, err := os.Create(filepath.Join(dir, action+".png"))
		if err != nil {
			return err
		}
		err = png.Encode(f, app.Render())
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

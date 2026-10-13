//go:build !windows

package main

import (
	"fmt"
	"os"
)

func showError(err error) { fmt.Fprintln(os.Stderr, err) }

func runNative(app *PetApp) error {
	return fmt.Errorf("desktop pet requires Windows 10/11; use -preview DIR to render previews")
}

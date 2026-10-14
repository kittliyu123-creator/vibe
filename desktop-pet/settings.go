package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Settings contain only desktop preferences. They never enable autostart,
// network access, or restore a hidden window that the user cannot find.
type petSettings struct {
	Version     int     `json:"version"`
	HasPosition bool    `json:"has_position"`
	X           int     `json:"x"`
	Y           int     `json:"y"`
	Scale       float64 `json:"scale"`
	Topmost     bool    `json:"topmost"`
	Follow      bool    `json:"follow"`
	Wander      bool    `json:"wander"`
}

func defaultPetSettings() petSettings {
	return petSettings{Version: 2, Scale: 1, Topmost: true}
}

func petSettingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "HelloKittyDesktopPet", "settings.json"), nil
}

func normalizePetSettings(s petSettings) petSettings {
	s.Version = 2
	if s.Scale != .8 && s.Scale != 1 && s.Scale != 1.25 {
		s.Scale = 1
	}
	// Negative coordinates are valid for monitors left of/above the primary.
	// Corrupt coordinates fall back to a visible default placement.
	if s.X < -100000 || s.X > 100000 || s.Y < -100000 || s.Y > 100000 {
		s.HasPosition = false
		s.X, s.Y = 0, 0
	}
	if s.Follow {
		s.Wander = false
	}
	return s
}

func loadPetSettings(path string) (petSettings, error) {
	s := defaultPetSettings()
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	defer f.Close()
	decoder := json.NewDecoder(io.LimitReader(f, 64*1024))
	if err = decoder.Decode(&s); err != nil {
		return defaultPetSettings(), fmt.Errorf("读取桌宠设置: %w", err)
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return defaultPetSettings(), fmt.Errorf("桌宠设置包含多余内容")
	}
	return normalizePetSettings(s), nil
}

func savePetSettings(path string, s petSettings) error {
	if path == "" {
		return fmt.Errorf("没有可写的用户配置目录")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// Write and flush a sibling temporary file before replacing the previous
	// settings. An interrupted write never leaves half a JSON document behind.
	f, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(normalizePetSettings(s)); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
}

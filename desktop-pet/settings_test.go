package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsDefaultAndMissingFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s, err := loadPetSettings(path)
	if err != nil || !s.Topmost || s.Scale != 1 || s.HasPosition || s.Follow || s.Wander {
		t.Fatalf("unexpected first-run settings: %+v, %v", s, err)
	}
	if err = os.WriteFile(path, []byte(`{"topmost":false,"x":-1600,"y":120,"has_position":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	s, err = loadPetSettings(path)
	if err != nil || s.Topmost || !s.HasPosition || s.X != -1600 || s.Scale != 1 {
		t.Fatalf("missing fields lost defaults or valid negative monitor coordinate: %+v, %v", s, err)
	}
}

func TestSettingsRejectCorruptAndUnsafeValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	for _, document := range []string{`{broken`, `{"scale":1} {"scale":0.8}`, `{"scale":"large"}`} {
		if err := os.WriteFile(path, []byte(document), 0600); err != nil {
			t.Fatal(err)
		}
		s, err := loadPetSettings(path)
		if err == nil || s != defaultPetSettings() {
			t.Fatalf("corrupt settings must recover to safe defaults: %q => %+v, %v", document, s, err)
		}
	}
	if err := os.WriteFile(path, []byte(`{"scale":99999,"has_position":true,"x":2000000,"follow":true,"wander":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := loadPetSettings(path)
	if err != nil || s.Scale != 1 || s.HasPosition || !s.Follow || s.Wander {
		t.Fatalf("unsafe values were not normalized: %+v, %v", s, err)
	}
}

func TestSettingsReplaceAndRestoreUserPreferences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config", "settings.json")
	if err := savePetSettings(path, defaultPetSettings()); err != nil {
		t.Fatal(err)
	}
	want := petSettings{Version: 2, HasPosition: true, X: -1850, Y: 214, Scale: 1.25, Topmost: false, Wander: true}
	if err := savePetSettings(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadPetSettings(path)
	if err != nil || got != want {
		t.Fatalf("saved preferences did not survive replacement/reload: %+v, %v", got, err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 || entries[0].Name() != "settings.json" {
		t.Fatalf("temporary files were left behind: %v, %v", entries, err)
	}
}

func TestSettingsFailedReplacementKeepsExistingTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := savePetSettings(path, defaultPetSettings()); err == nil {
		t.Fatal("replacement of a directory must fail")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("failed save damaged target or leaked temporary files: %v, %v", entries, err)
	}
}

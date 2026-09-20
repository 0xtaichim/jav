package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Cookies != "" || cfg.Proxy != "" {
		t.Fatalf("%+v", cfg)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := &Config{Cookies: "sid=1", Proxy: "127.0.0.1:1080", Locale: "zh"}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	path, err := GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm %s", info.Mode().Perm())
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Cookies != cfg.Cookies || loaded.Proxy != cfg.Proxy || loaded.Locale != cfg.Locale {
		t.Fatalf("loaded %+v", loaded)
	}
}

func TestSetGetUnknownKey(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("nope", "x"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := cfg.Get("nope"); err == nil {
		t.Fatal("expected error")
	}
	if err := cfg.Set("proxy", "1.2.3.4:1"); err != nil {
		t.Fatal(err)
	}
	got, err := cfg.Get("proxy")
	if err != nil || got != "1.2.3.4:1" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestToMapOmitsEmpty(t *testing.T) {
	cfg := &Config{Cookies: "a"}
	m := cfg.ToMap()
	if m["cookies"] != "a" {
		t.Fatalf("%v", m)
	}
	if _, ok := m["proxy"]; ok {
		t.Fatal("proxy should be omitted")
	}
}

func TestGetConfigDirUsesXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	got, err := GetConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "jav") {
		t.Fatalf("got %s", got)
	}
}

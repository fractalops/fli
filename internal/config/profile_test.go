package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	if cfg.SchemaVersion != CurrentConfigSchemaVersion {
		t.Errorf("expected schema version %d, got %d", CurrentConfigSchemaVersion, cfg.SchemaVersion)
	}
	if cfg.Profiles == nil {
		t.Error("expected non-nil profiles map")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fli", "config.yaml")
	os.MkdirAll(filepath.Dir(path), DirPermissions)

	cfg := NewConfig()
	cfg.ActiveProfile = "staging"
	cfg.SetProfile("staging", ProfileConfig{
		Region:   "us-east-1",
		LogGroup: "/fli/flow-logs/vpc-123",
		Version:  5,
	})
	cfg.SetProfile("default", ProfileConfig{
		Region:   "us-west-2",
		LogGroup: "/fli/flow-logs/vpc-456",
		Version:  2,
	})

	if err := os.WriteFile(path, nil, FilePermissions); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.SchemaVersion != 1 {
		t.Errorf("expected schema version 1, got %d", loaded.SchemaVersion)
	}
	if loaded.ActiveProfile != "staging" {
		t.Errorf("expected active profile 'staging', got %q", loaded.ActiveProfile)
	}

	p, ok := loaded.GetProfile("staging")
	if !ok {
		t.Fatal("expected staging profile to exist")
	}
	if p.Region != "us-east-1" {
		t.Errorf("expected region us-east-1, got %q", p.Region)
	}
	if p.Version != 5 {
		t.Errorf("expected version 5, got %d", p.Version)
	}
}

func TestLoadConfigNotExist(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(cfg.Profiles) != 0 {
		t.Errorf("expected empty profiles, got %d", len(cfg.Profiles))
	}
}

func TestConfigDeleteProfile(t *testing.T) {
	cfg := NewConfig()
	cfg.ActiveProfile = "test"
	cfg.SetProfile("test", ProfileConfig{LogGroup: "/test"})

	cfg.DeleteProfile("test")

	if _, ok := cfg.GetProfile("test"); ok {
		t.Error("expected profile to be deleted")
	}
	if cfg.ActiveProfile != "" {
		t.Error("expected active profile to be cleared")
	}
}

func TestConfigResolveProfile(t *testing.T) {
	cfg := NewConfig()
	cfg.ActiveProfile = "staging"

	if got := cfg.ResolveProfile("explicit"); got != "explicit" {
		t.Errorf("expected 'explicit', got %q", got)
	}
	if got := cfg.ResolveProfile(""); got != "staging" {
		t.Errorf("expected 'staging', got %q", got)
	}

	cfg.ActiveProfile = ""
	if got := cfg.ResolveProfile(""); got != "default" {
		t.Errorf("expected 'default', got %q", got)
	}
}

func TestSchemaVersionTooNew(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("schema_version: 999\n"), FilePermissions)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for newer schema version")
	}
}

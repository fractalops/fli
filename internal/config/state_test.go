package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewState(t *testing.T) {
	s := NewState()
	if s.SchemaVersion != CurrentStateSchemaVersion {
		t.Errorf("expected schema version %d, got %d", CurrentStateSchemaVersion, s.SchemaVersion)
	}
	if s.Profiles == nil {
		t.Error("expected non-nil profiles map")
	}
}

func TestStateAddAndRemoveResource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.yaml")

	s := NewState()

	r := Resource{
		Type: ResourceTypeLogGroup,
		Name: "/fli/flow-logs/vpc-123",
		ARN:  "arn:aws:logs:us-east-1:123:log-group:/fli/flow-logs/vpc-123",
	}

	if err := s.AddResource(path, "default", "us-east-1", r); err != nil {
		t.Fatalf("AddResource failed: %v", err)
	}

	if !s.HasResources("default") {
		t.Error("expected resources for default profile")
	}

	found, ok := s.FindResource("default", ResourceTypeLogGroup)
	if !ok {
		t.Fatal("expected to find log group resource")
	}
	if found.Name != "/fli/flow-logs/vpc-123" {
		t.Errorf("expected name '/fli/flow-logs/vpc-123', got %q", found.Name)
	}

	// Verify it was saved to disk
	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}
	if !loaded.HasResources("default") {
		t.Error("expected loaded state to have resources")
	}

	// Remove the resource
	if err := s.RemoveResource(path, "default", r); err != nil {
		t.Fatalf("RemoveResource failed: %v", err)
	}

	if s.HasResources("default") {
		t.Error("expected no resources after removal")
	}
}

func TestStateFindResourceNotFound(t *testing.T) {
	s := NewState()
	_, ok := s.FindResource("nonexistent", ResourceTypeFlowLog)
	if ok {
		t.Error("expected not found")
	}
}

func TestLoadStateNotExist(t *testing.T) {
	s, err := LoadState("/nonexistent/path")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(s.Profiles) != 0 {
		t.Errorf("expected empty profiles, got %d", len(s.Profiles))
	}
}

func TestStateSchemaVersionTooNew(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.yaml")
	os.WriteFile(path, []byte("schema_version: 999\n"), FilePermissions)

	_, err := LoadState(path)
	if err == nil {
		t.Fatal("expected error for newer schema version")
	}
}

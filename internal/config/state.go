package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// CurrentStateSchemaVersion is the current schema version for state.yaml.
	CurrentStateSchemaVersion = 1
)

// Resource types tracked in state.
const (
	ResourceTypeLogGroup   = "cloudwatch_log_group"
	ResourceTypeIAMRole    = "iam_role"
	ResourceTypeRolePolicy = "iam_role_policy"
	ResourceTypeFlowLog    = "vpc_flow_log"
)

// Resource represents a single AWS resource created by fli.
type Resource struct {
	Type       string `yaml:"type"`
	Name       string `yaml:"name,omitempty"`
	ARN        string `yaml:"arn,omitempty"`
	ID         string `yaml:"id,omitempty"`
	ResourceID string `yaml:"resource_id,omitempty"`
	RoleName   string `yaml:"role_name,omitempty"`
	PolicyName string `yaml:"policy_name,omitempty"`
}

// ProfileState tracks the resources created for a single profile.
type ProfileState struct {
	CreatedAt time.Time  `yaml:"created_at"`
	Region    string     `yaml:"region"`
	Resources []Resource `yaml:"resources"`
}

// State holds the top-level state from state.yaml.
type State struct {
	SchemaVersion int                      `yaml:"schema_version"`
	Profiles      map[string]*ProfileState `yaml:"profiles"`
}

// NewState creates a new State with defaults.
func NewState() *State {
	return &State{
		SchemaVersion: CurrentStateSchemaVersion,
		Profiles:      make(map[string]*ProfileState),
	}
}

// LoadState reads and parses the state file. Returns a new empty state if the file doesn't exist.
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewState(), nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if err := state.checkSchemaVersion(); err != nil {
		return nil, err
	}

	if state.Profiles == nil {
		state.Profiles = make(map[string]*ProfileState)
	}

	return &state, nil
}

// Save writes the state to the given path.
func (s *State) Save(path string) error {
	if err := EnsureFliDir(); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	s.SchemaVersion = CurrentStateSchemaVersion

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(path, data, FilePermissions); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// AddResource appends a resource to a profile's state and saves immediately.
func (s *State) AddResource(path, profile, region string, r Resource) error {
	ps, ok := s.Profiles[profile]
	if !ok {
		ps = &ProfileState{
			CreatedAt: time.Now().UTC(),
			Region:    region,
		}
		s.Profiles[profile] = ps
	}
	ps.Resources = append(ps.Resources, r)
	return s.Save(path)
}

// RemoveResource removes a resource from a profile's state and saves immediately.
func (s *State) RemoveResource(path, profile string, r Resource) error {
	ps, ok := s.Profiles[profile]
	if !ok {
		return nil
	}

	filtered := make([]Resource, 0, len(ps.Resources))
	for _, existing := range ps.Resources {
		if !resourcesMatch(existing, r) {
			filtered = append(filtered, existing)
		}
	}
	ps.Resources = filtered

	// Remove profile entry if no resources remain
	if len(ps.Resources) == 0 {
		delete(s.Profiles, profile)
	}

	return s.Save(path)
}

// GetProfileState returns the state for a profile and whether it exists.
func (s *State) GetProfileState(profile string) (*ProfileState, bool) {
	ps, ok := s.Profiles[profile]
	return ps, ok
}

// HasResources returns true if the profile has any tracked resources.
func (s *State) HasResources(profile string) bool {
	ps, ok := s.Profiles[profile]
	return ok && len(ps.Resources) > 0
}

// FindResource returns the first resource matching the given type for a profile.
func (s *State) FindResource(profile, resourceType string) (Resource, bool) {
	ps, ok := s.Profiles[profile]
	if !ok {
		return Resource{}, false
	}
	for _, r := range ps.Resources {
		if r.Type == resourceType {
			return r, true
		}
	}
	return Resource{}, false
}

func resourcesMatch(a, b Resource) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case ResourceTypeLogGroup:
		return a.Name == b.Name
	case ResourceTypeIAMRole:
		return a.Name == b.Name
	case ResourceTypeRolePolicy:
		return a.RoleName == b.RoleName && a.PolicyName == b.PolicyName
	case ResourceTypeFlowLog:
		return a.ID == b.ID
	default:
		return a.Name == b.Name && a.ID == b.ID
	}
}

func (s *State) checkSchemaVersion() error {
	if s.SchemaVersion == 0 {
		s.SchemaVersion = 1
		return nil
	}
	if s.SchemaVersion > CurrentStateSchemaVersion {
		return fmt.Errorf("state was created by a newer version of fli (schema version %d). Please upgrade", s.SchemaVersion)
	}
	return nil
}

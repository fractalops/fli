package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	// CurrentConfigSchemaVersion is the current schema version for config.yaml.
	CurrentConfigSchemaVersion = 1
)

// ProfileConfig holds the configuration for a single profile.
type ProfileConfig struct {
	Region   string `yaml:"region"`
	LogGroup string `yaml:"log_group"`
	Version  int    `yaml:"version"`
}

// Config holds the top-level configuration from config.yaml.
type Config struct {
	SchemaVersion int                      `yaml:"schema_version"`
	ActiveProfile string                   `yaml:"active_profile"`
	Profiles      map[string]ProfileConfig `yaml:"profiles"`
}

// NewConfig creates a new Config with defaults.
func NewConfig() *Config {
	return &Config{
		SchemaVersion: CurrentConfigSchemaVersion,
		Profiles:      make(map[string]ProfileConfig),
	}
}

// LoadConfig reads and parses the config file. Returns a new empty config if the file doesn't exist.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.checkSchemaVersion(); err != nil {
		return nil, err
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]ProfileConfig)
	}

	return &cfg, nil
}

// Save writes the config to the given path.
func (c *Config) Save(path string) error {
	if err := EnsureFliDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	c.SchemaVersion = CurrentConfigSchemaVersion

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, FilePermissions); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SetProfile adds or updates a profile in the config.
func (c *Config) SetProfile(name string, profile ProfileConfig) {
	c.Profiles[name] = profile
}

// GetProfile returns a profile by name and whether it exists.
func (c *Config) GetProfile(name string) (ProfileConfig, bool) {
	p, ok := c.Profiles[name]
	return p, ok
}

// DeleteProfile removes a profile from the config.
func (c *Config) DeleteProfile(name string) {
	delete(c.Profiles, name)
	if c.ActiveProfile == name {
		c.ActiveProfile = ""
	}
}

// ResolveProfile returns the profile name to use based on the resolution order:
// explicit > active_profile > "default".
func (c *Config) ResolveProfile(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if c.ActiveProfile != "" {
		return c.ActiveProfile
	}
	return "default"
}

func (c *Config) checkSchemaVersion() error {
	if c.SchemaVersion == 0 {
		// Treat missing version as v1
		c.SchemaVersion = 1
		return nil
	}
	if c.SchemaVersion > CurrentConfigSchemaVersion {
		return fmt.Errorf("config was created by a newer version of fli (schema version %d). Please upgrade", c.SchemaVersion)
	}
	return nil
}

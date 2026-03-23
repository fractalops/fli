package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// FliDirName is the name of the fli configuration directory.
	FliDirName = ".fli"

	// ConfigFileName is the name of the configuration file.
	ConfigFileName = "config.yaml"

	// StateFileName is the name of the state file.
	StateFileName = "state.yaml"
)

// FliDir returns the path to the fli configuration directory (~/.fli).
func FliDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, FliDirName), nil
}

// ConfigPath returns the path to the configuration file (~/.fli/config.yaml).
//
//nolint:revive // ConfigPath stutters with package name but renaming would break callers.
func ConfigPath() (string, error) {
	dir, err := FliDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}

// StatePath returns the path to the state file (~/.fli/state.yaml).
func StatePath() (string, error) {
	dir, err := FliDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, StateFileName), nil
}

// EnsureFliDir creates the fli configuration directory if it doesn't exist.
func EnsureFliDir() error {
	dir, err := FliDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, DirPermissions); err != nil {
		return fmt.Errorf("failed to create fli directory %s: %w", dir, err)
	}
	return nil
}

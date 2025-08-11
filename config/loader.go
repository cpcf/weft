// Package config provides generic configuration loading capabilities for weft
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Validator defines an interface that configuration types can implement
// to provide custom validation logic
type Validator interface {
	Validate() error
}

// LoadYAML loads any YAML configuration into the provided target struct.
// The target must be a pointer to the struct you want to unmarshal into.
// If the target implements the Validator interface, validation will be called.
func LoadYAML[T any](path string, target *T) error {
	// Handle relative paths by making them absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", path, err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", absPath)
	}

	// Read the file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file %q: %w", absPath, err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	// Validate if the target implements Validator interface
	if validator, ok := any(target).(Validator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	return nil
}

// LoadYAMLFromString loads YAML configuration from a string instead of a file.
// Useful for testing or when configuration comes from other sources.
func LoadYAMLFromString[T any](yamlContent string, target *T) error {
	// Parse YAML
	if err := yaml.Unmarshal([]byte(yamlContent), target); err != nil {
		return fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	// Validate if the target implements Validator interface
	if validator, ok := any(target).(Validator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	return nil
}

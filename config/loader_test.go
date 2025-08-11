package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestConfig is a simple configuration struct for testing
type TestConfig struct {
	Name    string            `yaml:"name"`
	Version string            `yaml:"version"`
	Options map[string]string `yaml:"options"`
}

// ValidatedTestConfig is a configuration struct that implements validation
type ValidatedTestConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	Required string `yaml:"required"`
}

// Validate implements the Validator interface
func (c *ValidatedTestConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.Required == "" {
		return fmt.Errorf("required field is required")
	}
	return nil
}

func TestLoadYAML_BasicConfig(t *testing.T) {
	yamlContent := `
name: "Test Config"
version: "1.0.0"
options:
  debug: "true"
  timeout: "30s"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	err := os.WriteFile(configPath, []byte(yamlContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	var config TestConfig
	err = LoadYAML(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	// Verify the loaded data
	if config.Name != "Test Config" {
		t.Errorf("Expected name 'Test Config', got '%s'", config.Name)
	}
	if config.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", config.Version)
	}
	if len(config.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(config.Options))
	}
	if config.Options["debug"] != "true" {
		t.Errorf("Expected debug option 'true', got '%s'", config.Options["debug"])
	}
}

func TestLoadYAML_WithValidation_Success(t *testing.T) {
	yamlContent := `
name: "Valid Config"
version: "2.0.0"
required: "present"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "valid-config.yaml")

	err := os.WriteFile(configPath, []byte(yamlContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	var config ValidatedTestConfig
	err = LoadYAML(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to load validated YAML config: %v", err)
	}

	// Verify the loaded data
	if config.Name != "Valid Config" {
		t.Errorf("Expected name 'Valid Config', got '%s'", config.Name)
	}
	if config.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", config.Version)
	}
	if config.Required != "present" {
		t.Errorf("Expected required 'present', got '%s'", config.Required)
	}
}

func TestLoadYAML_WithValidation_Failure(t *testing.T) {
	yamlContent := `
name: "Invalid Config"
version: "2.0.0"
# missing required field
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-config.yaml")

	err := os.WriteFile(configPath, []byte(yamlContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	var config ValidatedTestConfig
	err = LoadYAML(configPath, &config)
	if err == nil {
		t.Fatal("Expected validation error for missing required field, got nil")
	}

	expectedError := "configuration validation failed: required field is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestLoadYAML_FileNotExists(t *testing.T) {
	var config TestConfig
	err := LoadYAML("/nonexistent/file.yaml", &config)
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
}

func TestLoadYAML_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	var config TestConfig
	err = LoadYAML(configPath, &config)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestLoadYAMLFromString_BasicConfig(t *testing.T) {
	yamlContent := `
name: "String Config"
version: "3.0.0"
options:
  mode: "test"
`

	var config TestConfig
	err := LoadYAMLFromString(yamlContent, &config)
	if err != nil {
		t.Fatalf("Failed to load YAML from string: %v", err)
	}

	// Verify the loaded data
	if config.Name != "String Config" {
		t.Errorf("Expected name 'String Config', got '%s'", config.Name)
	}
	if config.Version != "3.0.0" {
		t.Errorf("Expected version '3.0.0', got '%s'", config.Version)
	}
	if config.Options["mode"] != "test" {
		t.Errorf("Expected mode option 'test', got '%s'", config.Options["mode"])
	}
}

func TestLoadYAMLFromString_WithValidation(t *testing.T) {
	yamlContent := `
name: "String Validated Config"
version: "1.5.0"
required: "here"
`

	var config ValidatedTestConfig
	err := LoadYAMLFromString(yamlContent, &config)
	if err != nil {
		t.Fatalf("Failed to load validated YAML from string: %v", err)
	}

	// Verify the loaded data
	if config.Required != "here" {
		t.Errorf("Expected required 'here', got '%s'", config.Required)
	}
}

func TestLoadYAMLFromString_InvalidYAML(t *testing.T) {
	var config TestConfig
	err := LoadYAMLFromString("invalid: yaml: content: [", &config)
	if err == nil {
		t.Fatal("Expected error for invalid YAML string, got nil")
	}
}
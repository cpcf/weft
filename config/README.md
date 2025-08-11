# Config Package

The config package provides generic YAML configuration loading capabilities for gogenkit users. It allows you to easily load YAML configuration files into your custom specification types.

## Features

- **Generic YAML Loading**: Load any YAML file into your custom struct types
- **Validation Support**: Implement the `Validator` interface for custom validation logic
- **Error Handling**: Comprehensive error messages for common issues (missing files, invalid YAML, validation failures)
- **Multiple Sources**: Load from files or strings (useful for testing)
- **Type Safety**: Uses Go generics for compile-time type safety

## Basic Usage

### Simple Configuration Loading

```go
package main

import (
    "fmt"
    "github.com/cpcf/gogenkit/config"
)

// Define your configuration struct with YAML tags
type MyConfig struct {
    Name    string            `yaml:"name"`
    Version string            `yaml:"version"`
    Options map[string]string `yaml:"options"`
}

func main() {
    var cfg MyConfig
    
    // Load configuration from file
    err := config.LoadYAML("config.yaml", &cfg)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Loaded config: %+v\n", cfg)
}
```

### With Custom Validation

```go
// Define a configuration struct that implements validation
type ValidatedConfig struct {
    Name     string `yaml:"name"`
    Version  string `yaml:"version"`
    Required string `yaml:"required"`
}

// Implement the Validator interface
func (c *ValidatedConfig) Validate() error {
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

func main() {
    var cfg ValidatedConfig
    
    // Load and validate configuration
    err := config.LoadYAML("config.yaml", &cfg)
    if err != nil {
        // This will include validation errors if validation fails
        panic(err)
    }
    
    fmt.Printf("Valid config loaded: %+v\n", cfg)
}
```

### Loading from String (Testing)

```go
func TestMyConfig(t *testing.T) {
    yamlContent := `
name: "Test Config"
version: "1.0.0"
options:
  debug: "true"
`
    
    var cfg MyConfig
    err := config.LoadYAMLFromString(yamlContent, &cfg)
    if err != nil {
        t.Fatalf("Failed to load config: %v", err)
    }
    
    // Test your configuration...
}
```

## Validator Interface

The optional `Validator` interface allows your configuration structs to implement custom validation logic:

```go
type Validator interface {
    Validate() error
}
```

When your configuration struct implements this interface, the `Validate()` method will be called automatically after YAML unmarshaling. If validation fails, the error will be wrapped and returned.

## Error Handling

The config package provides detailed error messages for common scenarios:

- **File not found**: Clear message indicating the absolute path that was checked
- **Invalid YAML syntax**: YAML parsing errors with line/column information
- **Validation failures**: Custom validation errors wrapped with context

## Examples

See the `examples/api-client-generator` directory for a complete example of how to use the config package with a complex API specification type.

## Best Practices

1. **Always use YAML tags** on your struct fields for explicit field mapping
2. **Implement validation** for required fields and business logic constraints
3. **Use meaningful error messages** in your validation logic
4. **Test with `LoadYAMLFromString`** for unit testing your configurations
5. **Handle absolute vs relative paths** - the loader automatically resolves relative paths
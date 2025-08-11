// Package spec defines the database schema specification types for the YAML tutorial.
//
// This file demonstrates how to create your own custom specification types
// that work with the gogenkit config system. The key principles are:
//
// 1. Use YAML tags on all struct fields for explicit field mapping
// 2. Implement the config.Validator interface for custom validation
// 3. Use clear, descriptive field names and types
// 4. Provide comprehensive validation with helpful error messages

package spec

import (
	"fmt"
	"strings"
)

// DatabaseSchema represents the root configuration for a database schema generator.
// This is your main specification type that users will define in their YAML files.
//
// TUTORIAL NOTE: When creating your own spec, think about what top-level
// configuration your users need. Common patterns include:
// - Metadata (name, version, description)
// - Output configuration (package name, target directory)
// - Core business logic (the main data your templates need)
type DatabaseSchema struct {
	// Basic metadata about the schema
	Name        string `yaml:"name"`        // Human-readable name for the schema
	Version     string `yaml:"version"`     // Schema version (useful for compatibility)
	Description string `yaml:"description"` // Optional description

	// Output configuration
	Package string `yaml:"package"` // Go package name for generated code

	// Database configuration
	Database DatabaseConfig `yaml:"database"` // Database connection settings

	// Core data - the tables that define our schema
	Tables []Table `yaml:"tables"` // List of database tables to generate
}

// Validate implements the config.Validator interface.
// This method is called automatically after YAML loading to ensure
// the configuration is valid and complete.
//
// TUTORIAL NOTE: Good validation should check:
// - Required fields are present
// - Field values are in valid ranges/formats
// - Business logic constraints are met
// - References between objects are valid
func (s *DatabaseSchema) Validate() error {
	// Check required top-level fields
	if s.Name == "" {
		return fmt.Errorf("schema name is required")
	}
	if s.Version == "" {
		return fmt.Errorf("schema version is required")
	}
	if s.Package == "" {
		return fmt.Errorf("package name is required")
	}

	// Validate package name format (basic Go package name rules)
	if !isValidGoPackageName(s.Package) {
		return fmt.Errorf("package name %q is not a valid Go package name", s.Package)
	}

	// Validate database configuration
	if err := s.Database.Validate(); err != nil {
		return fmt.Errorf("database configuration: %w", err)
	}

	// Must have at least one table
	if len(s.Tables) == 0 {
		return fmt.Errorf("at least one table is required")
	}

	// Validate each table and collect table names for reference checking
	tableNames := make(map[string]bool)
	for i, table := range s.Tables {
		if err := table.Validate(); err != nil {
			return fmt.Errorf("table %d (%s): %w", i, table.Name, err)
		}

		// Check for duplicate table names
		if tableNames[table.Name] {
			return fmt.Errorf("duplicate table name: %s", table.Name)
		}
		tableNames[table.Name] = true
	}

	// Validate foreign key references
	for i, table := range s.Tables {
		for j, field := range table.Fields {
			if field.ForeignKey != "" {
				if err := validateForeignKeyReference(field.ForeignKey, tableNames); err != nil {
					return fmt.Errorf("table %d (%s), field %d (%s): %w", i, table.Name, j, field.Name, err)
				}
			}
		}
	}

	return nil
}

// DatabaseConfig represents database connection settings.
// This shows how to create nested configuration objects.
type DatabaseConfig struct {
	Driver string `yaml:"driver"` // Database driver (postgres, mysql, sqlite, etc.)
	Host   string `yaml:"host"`   // Database host
	Port   int    `yaml:"port"`   // Database port
	Name   string `yaml:"name"`   // Database name
}

// Validate validates the database configuration
func (d *DatabaseConfig) Validate() error {
	if d.Driver == "" {
		return fmt.Errorf("database driver is required")
	}
	if d.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if d.Port <= 0 || d.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535, got %d", d.Port)
	}
	if d.Name == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate supported drivers
	supportedDrivers := []string{"postgres", "mysql", "sqlite", "mssql"}
	for _, supported := range supportedDrivers {
		if d.Driver == supported {
			return nil
		}
	}

	return fmt.Errorf("unsupported database driver %q, supported drivers: %s",
		d.Driver, strings.Join(supportedDrivers, ", "))
}

// Table represents a database table specification.
// This demonstrates how to model the core entities in your domain.
type Table struct {
	Name        string  `yaml:"name"`        // Table name (must be unique within schema)
	Description string  `yaml:"description"` // Human-readable description
	Fields      []Field `yaml:"fields"`      // Table fields/columns
}

// Validate validates a table specification
func (t *Table) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("table name is required")
	}

	// Validate table name format (basic SQL identifier rules)
	if !isValidSQLIdentifier(t.Name) {
		return fmt.Errorf("table name %q is not a valid SQL identifier", t.Name)
	}

	// Must have at least one field
	if len(t.Fields) == 0 {
		return fmt.Errorf("table must have at least one field")
	}

	// Validate fields and check for duplicates
	fieldNames := make(map[string]bool)
	hasPrimaryKey := false

	for i, field := range t.Fields {
		if err := field.Validate(); err != nil {
			return fmt.Errorf("field %d (%s): %w", i, field.Name, err)
		}

		// Check for duplicate field names
		if fieldNames[field.Name] {
			return fmt.Errorf("duplicate field name: %s", field.Name)
		}
		fieldNames[field.Name] = true

		// Track primary keys
		if field.PrimaryKey {
			if hasPrimaryKey {
				return fmt.Errorf("table can only have one primary key field")
			}
			hasPrimaryKey = true
		}
	}

	// Every table should have a primary key
	if !hasPrimaryKey {
		return fmt.Errorf("table must have exactly one primary key field")
	}

	return nil
}

// Field represents a database table field/column.
// This shows how to handle complex field attributes and constraints.
type Field struct {
	Name        string `yaml:"name"`        // Field name
	Type        string `yaml:"type"`        // Data type (string, integer, uuid, etc.)
	Description string `yaml:"description"` // Human-readable description

	// Constraints
	Required   bool `yaml:"required"`   // NOT NULL constraint
	PrimaryKey bool `yaml:"primaryKey"` // PRIMARY KEY constraint
	Unique     bool `yaml:"unique"`     // UNIQUE constraint

	// Relationships
	ForeignKey string `yaml:"foreignKey"` // Foreign key reference (format: "table.field")

	// Additional attributes (you can extend this based on your needs)
	DefaultValue string `yaml:"defaultValue"` // Default value
	MaxLength    int    `yaml:"maxLength"`    // For string types
}

// Validate validates a field specification
func (f *Field) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("field name is required")
	}
	if f.Type == "" {
		return fmt.Errorf("field type is required")
	}

	// Validate field name format
	if !isValidSQLIdentifier(f.Name) {
		return fmt.Errorf("field name %q is not a valid SQL identifier", f.Name)
	}

	// Validate field type
	validTypes := []string{
		"string", "text", "integer", "bigint", "decimal", "float",
		"boolean", "date", "timestamp", "uuid", "json",
	}

	isValidType := false
	for _, validType := range validTypes {
		if f.Type == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		return fmt.Errorf("invalid field type %q, supported types: %s",
			f.Type, strings.Join(validTypes, ", "))
	}

	// Primary keys must be required
	if f.PrimaryKey && !f.Required {
		return fmt.Errorf("primary key fields must be required")
	}

	// Validate MaxLength for string types
	if f.MaxLength < 0 {
		return fmt.Errorf("maxLength cannot be negative")
	}
	if f.MaxLength > 0 && f.Type != "string" {
		return fmt.Errorf("maxLength can only be specified for string fields")
	}

	return nil
}

// Helper functions for validation

// isValidGoPackageName checks if a string is a valid Go package name
func isValidGoPackageName(name string) bool {
	if name == "" {
		return false
	}

	// Basic checks - Go package names should be lowercase letters and digits
	// and start with a letter
	if !((name[0] >= 'a' && name[0] <= 'z') || name[0] == '_') {
		return false
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	return true
}

// isValidSQLIdentifier checks if a string is a valid SQL identifier
func isValidSQLIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// Basic checks - SQL identifiers should start with letter or underscore
	// and contain only letters, digits, and underscores
	if !((name[0] >= 'a' && name[0] <= 'z') ||
		(name[0] >= 'A' && name[0] <= 'Z') ||
		name[0] == '_') {
		return false
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_') {
			return false
		}
	}

	return true
}

// validateForeignKeyReference validates a foreign key reference format
func validateForeignKeyReference(fkRef string, availableTables map[string]bool) error {
	// Foreign key format should be "table.field"
	parts := strings.Split(fkRef, ".")
	if len(parts) != 2 {
		return fmt.Errorf("foreign key reference must be in format 'table.field', got %q", fkRef)
	}

	tableName := parts[0]
	fieldName := parts[1]

	if tableName == "" {
		return fmt.Errorf("foreign key table name cannot be empty")
	}
	if fieldName == "" {
		return fmt.Errorf("foreign key field name cannot be empty")
	}

	// Check if referenced table exists
	if !availableTables[tableName] {
		return fmt.Errorf("foreign key references unknown table: %s", tableName)
	}

	return nil
}

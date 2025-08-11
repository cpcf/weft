package spec

import (
	"strings"
	"testing"

	"github.com/cpcf/weft/config"
)

// TestDatabaseSchema_Validate_Success tests validation with a complete, valid schema
func TestDatabaseSchema_Validate_Success(t *testing.T) {
	schema := &DatabaseSchema{
		Name:        "Test Schema",
		Version:     "1.0.0",
		Description: "Test database schema",
		Package:     "testdb",
		Database: DatabaseConfig{
			Driver: "postgres",
			Host:   "localhost",
			Port:   5432,
			Name:   "testdb",
		},
		Tables: []Table{
			{
				Name:        "users",
				Description: "User accounts",
				Fields: []Field{
					{Name: "id", Type: "uuid", Required: true, PrimaryKey: true, Description: "User ID"},
					{Name: "email", Type: "string", Required: true, Unique: true, Description: "Email address"},
					{Name: "name", Type: "string", Required: true, Description: "Full name"},
					{Name: "created_at", Type: "timestamp", Required: true, Description: "Creation timestamp"},
				},
			},
			{
				Name:        "posts",
				Description: "Blog posts",
				Fields: []Field{
					{Name: "id", Type: "uuid", Required: true, PrimaryKey: true, Description: "Post ID"},
					{Name: "user_id", Type: "uuid", Required: true, ForeignKey: "users.id", Description: "Author ID"},
					{Name: "title", Type: "string", Required: true, Description: "Post title"},
					{Name: "content", Type: "text", Required: true, Description: "Post content"},
				},
			},
		},
	}

	err := schema.Validate()
	if err != nil {
		t.Fatalf("Expected validation to succeed, got error: %v", err)
	}
}

// TestDatabaseSchema_Validate_MissingRequiredFields tests validation failures for missing required fields
func TestDatabaseSchema_Validate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		schema DatabaseSchema
		want   string
	}{
		{
			name:   "missing name",
			schema: DatabaseSchema{Version: "1.0", Package: "test"},
			want:   "schema name is required",
		},
		{
			name:   "missing version",
			schema: DatabaseSchema{Name: "Test", Package: "test"},
			want:   "schema version is required",
		},
		{
			name:   "missing package",
			schema: DatabaseSchema{Name: "Test", Version: "1.0"},
			want:   "package name is required",
		},
		{
			name: "invalid package name",
			schema: DatabaseSchema{
				Name:    "Test",
				Version: "1.0",
				Package: "Invalid-Package",
			},
			want: "package name \"Invalid-Package\" is not a valid Go package name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if err == nil {
				t.Fatalf("Expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestDatabaseSchema_Validate_DatabaseConfig tests database configuration validation
func TestDatabaseSchema_Validate_DatabaseConfig(t *testing.T) {
	baseSchema := DatabaseSchema{
		Name:    "Test Schema",
		Version: "1.0.0",
		Package: "testdb",
		Tables: []Table{
			{
				Name: "test_table",
				Fields: []Field{
					{Name: "id", Type: "uuid", Required: true, PrimaryKey: true},
				},
			},
		},
	}

	tests := []struct {
		name     string
		database DatabaseConfig
		want     string
	}{
		{
			name:     "missing driver",
			database: DatabaseConfig{Host: "localhost", Port: 5432, Name: "test"},
			want:     "database driver is required",
		},
		{
			name:     "missing host",
			database: DatabaseConfig{Driver: "postgres", Port: 5432, Name: "test"},
			want:     "database host is required",
		},
		{
			name:     "invalid port - zero",
			database: DatabaseConfig{Driver: "postgres", Host: "localhost", Port: 0, Name: "test"},
			want:     "database port must be between 1 and 65535",
		},
		{
			name:     "invalid port - too high",
			database: DatabaseConfig{Driver: "postgres", Host: "localhost", Port: 70000, Name: "test"},
			want:     "database port must be between 1 and 65535",
		},
		{
			name:     "missing database name",
			database: DatabaseConfig{Driver: "postgres", Host: "localhost", Port: 5432},
			want:     "database name is required",
		},
		{
			name:     "unsupported driver",
			database: DatabaseConfig{Driver: "oracle", Host: "localhost", Port: 1521, Name: "test"},
			want:     "unsupported database driver \"oracle\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := baseSchema
			schema.Database = tt.database

			err := schema.Validate()
			if err == nil {
				t.Fatalf("Expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestDatabaseSchema_Validate_Tables tests table validation
func TestDatabaseSchema_Validate_Tables(t *testing.T) {
	baseSchema := DatabaseSchema{
		Name:    "Test Schema",
		Version: "1.0.0",
		Package: "testdb",
		Database: DatabaseConfig{
			Driver: "postgres",
			Host:   "localhost",
			Port:   5432,
			Name:   "testdb",
		},
	}

	tests := []struct {
		name   string
		tables []Table
		want   string
	}{
		{
			name:   "no tables",
			tables: []Table{},
			want:   "at least one table is required",
		},
		{
			name: "duplicate table names",
			tables: []Table{
				{
					Name: "users",
					Fields: []Field{
						{Name: "id", Type: "uuid", Required: true, PrimaryKey: true},
					},
				},
				{
					Name: "users",
					Fields: []Field{
						{Name: "id", Type: "uuid", Required: true, PrimaryKey: true},
					},
				},
			},
			want: "duplicate table name: users",
		},
		{
			name: "invalid foreign key reference",
			tables: []Table{
				{
					Name: "posts",
					Fields: []Field{
						{Name: "id", Type: "uuid", Required: true, PrimaryKey: true},
						{Name: "user_id", Type: "uuid", Required: true, ForeignKey: "nonexistent.id"},
					},
				},
			},
			want: "foreign key references unknown table: nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := baseSchema
			schema.Tables = tt.tables

			err := schema.Validate()
			if err == nil {
				t.Fatalf("Expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestTable_Validate tests individual table validation
func TestTable_Validate(t *testing.T) {
	tests := []struct {
		name  string
		table Table
		want  string
	}{
		{
			name:  "missing table name",
			table: Table{Fields: []Field{{Name: "id", Type: "uuid", Required: true, PrimaryKey: true}}},
			want:  "table name is required",
		},
		{
			name:  "invalid table name",
			table: Table{Name: "invalid-name", Fields: []Field{{Name: "id", Type: "uuid", Required: true, PrimaryKey: true}}},
			want:  "table name \"invalid-name\" is not a valid SQL identifier",
		},
		{
			name:  "no fields",
			table: Table{Name: "users", Fields: []Field{}},
			want:  "table must have at least one field",
		},
		{
			name: "no primary key",
			table: Table{
				Name: "users",
				Fields: []Field{
					{Name: "email", Type: "string", Required: true},
				},
			},
			want: "table must have exactly one primary key field",
		},
		{
			name: "multiple primary keys",
			table: Table{
				Name: "users",
				Fields: []Field{
					{Name: "id1", Type: "uuid", Required: true, PrimaryKey: true},
					{Name: "id2", Type: "uuid", Required: true, PrimaryKey: true},
				},
			},
			want: "table can only have one primary key field",
		},
		{
			name: "duplicate field names",
			table: Table{
				Name: "users",
				Fields: []Field{
					{Name: "id", Type: "uuid", Required: true, PrimaryKey: true},
					{Name: "email", Type: "string", Required: true},
					{Name: "email", Type: "string", Required: true},
				},
			},
			want: "duplicate field name: email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.table.Validate()
			if err == nil {
				t.Fatalf("Expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestField_Validate tests field validation
func TestField_Validate(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		want  string
	}{
		{
			name:  "missing field name",
			field: Field{Type: "string", Required: true},
			want:  "field name is required",
		},
		{
			name:  "missing field type",
			field: Field{Name: "test", Required: true},
			want:  "field type is required",
		},
		{
			name:  "invalid field name",
			field: Field{Name: "invalid-name", Type: "string"},
			want:  "field name \"invalid-name\" is not a valid SQL identifier",
		},
		{
			name:  "invalid field type",
			field: Field{Name: "test", Type: "invalid_type"},
			want:  "invalid field type \"invalid_type\"",
		},
		{
			name:  "primary key not required",
			field: Field{Name: "id", Type: "uuid", PrimaryKey: true, Required: false},
			want:  "primary key fields must be required",
		},
		{
			name:  "negative max length",
			field: Field{Name: "test", Type: "string", MaxLength: -1},
			want:  "maxLength cannot be negative",
		},
		{
			name:  "max length on non-string",
			field: Field{Name: "test", Type: "integer", MaxLength: 100},
			want:  "maxLength can only be specified for string fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.Validate()
			if err == nil {
				t.Fatalf("Expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

// TestDatabaseSchema_LoadFromYAML tests loading a complete schema from YAML
func TestDatabaseSchema_LoadFromYAML(t *testing.T) {
	yamlContent := `
name: "Blog Database"
version: "1.0.0"
description: "Simple blog database schema"
package: "blogdb"

database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  name: "blog_db"

tables:
  - name: "authors"
    description: "Blog authors"
    fields:
      - name: "id"
        type: "uuid"
        description: "Author ID"
        required: true
        primaryKey: true
      - name: "name"
        type: "string"
        description: "Author name"
        required: true
        maxLength: 100
      - name: "email"
        type: "string"
        description: "Author email"
        required: true
        unique: true
        maxLength: 255

  - name: "posts"
    description: "Blog posts"
    fields:
      - name: "id"
        type: "uuid"
        description: "Post ID"
        required: true
        primaryKey: true
      - name: "author_id"
        type: "uuid"
        description: "Post author"
        required: true
        foreignKey: "authors.id"
      - name: "title"
        type: "string"
        description: "Post title"
        required: true
        maxLength: 200
      - name: "content"
        type: "text"
        description: "Post content"
        required: true
      - name: "published"
        type: "boolean"
        description: "Is published"
        required: true
        defaultValue: "false"
`

	var schema DatabaseSchema
	err := config.LoadYAMLFromString(yamlContent, &schema)
	if err != nil {
		t.Fatalf("Failed to load schema from YAML: %v", err)
	}

	// Verify basic fields
	if schema.Name != "Blog Database" {
		t.Errorf("Expected name 'Blog Database', got '%s'", schema.Name)
	}
	if schema.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", schema.Version)
	}
	if schema.Package != "blogdb" {
		t.Errorf("Expected package 'blogdb', got '%s'", schema.Package)
	}

	// Verify database config
	if schema.Database.Driver != "postgres" {
		t.Errorf("Expected driver 'postgres', got '%s'", schema.Database.Driver)
	}
	if schema.Database.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", schema.Database.Port)
	}

	// Verify tables
	if len(schema.Tables) != 2 {
		t.Fatalf("Expected 2 tables, got %d", len(schema.Tables))
	}

	// Verify authors table
	authors := schema.Tables[0]
	if authors.Name != "authors" {
		t.Errorf("Expected table name 'authors', got '%s'", authors.Name)
	}
	if len(authors.Fields) != 3 {
		t.Errorf("Expected 3 fields in authors table, got %d", len(authors.Fields))
	}

	// Verify posts table and foreign key
	posts := schema.Tables[1]
	if posts.Name != "posts" {
		t.Errorf("Expected table name 'posts', got '%s'", posts.Name)
	}
	
	authorIdField := posts.Fields[1]
	if authorIdField.ForeignKey != "authors.id" {
		t.Errorf("Expected foreign key 'authors.id', got '%s'", authorIdField.ForeignKey)
	}
}

// TestDatabaseSchema_LoadFromYAML_ValidationFailure tests YAML loading with validation failures
func TestDatabaseSchema_LoadFromYAML_ValidationFailure(t *testing.T) {
	yamlContent := `
name: "Invalid Schema"
version: "1.0.0"
package: "test"
database:
  driver: "postgres"
  host: "localhost"  
  port: 5432
  name: "test_db"
tables:
  - name: "invalid_table"
    fields:
      - name: "field1"
        type: "string"
        # Missing primary key - validation should fail
`

	var schema DatabaseSchema
	err := config.LoadYAMLFromString(yamlContent, &schema)
	if err == nil {
		t.Fatal("Expected validation error for schema without primary key, got nil")
	}

	// Should be a validation error about missing primary key
	if !strings.Contains(err.Error(), "primary key") {
		t.Errorf("Expected error about missing primary key, got: %v", err)
	}
}

// TestHelperFunctions tests the validation helper functions
func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func(string) bool
		input    string
		expected bool
	}{
		// Test isValidGoPackageName
		{"valid go package", isValidGoPackageName, "mypackage", true},
		{"valid go package with underscore", isValidGoPackageName, "my_package", true},
		{"valid go package with numbers", isValidGoPackageName, "package123", true},
		{"invalid go package - starts with number", isValidGoPackageName, "123package", false},
		{"invalid go package - contains dash", isValidGoPackageName, "my-package", false},
		{"invalid go package - contains uppercase", isValidGoPackageName, "MyPackage", false},
		{"invalid go package - empty", isValidGoPackageName, "", false},

		// Test isValidSQLIdentifier
		{"valid sql identifier lowercase", isValidSQLIdentifier, "tablename", true},
		{"valid sql identifier uppercase", isValidSQLIdentifier, "TABLENAME", true},
		{"valid sql identifier mixed case", isValidSQLIdentifier, "TableName", true},
		{"valid sql identifier with underscore", isValidSQLIdentifier, "table_name", true},
		{"valid sql identifier with numbers", isValidSQLIdentifier, "table123", true},
		{"invalid sql identifier - starts with number", isValidSQLIdentifier, "123table", false},
		{"invalid sql identifier - contains dash", isValidSQLIdentifier, "table-name", false},
		{"invalid sql identifier - empty", isValidSQLIdentifier, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v for input %q, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

// TestValidateForeignKeyReference tests foreign key reference validation
func TestValidateForeignKeyReference(t *testing.T) {
	availableTables := map[string]bool{
		"users":  true,
		"posts":  true,
		"orders": true,
	}

	tests := []struct {
		name   string
		fkRef  string
		want   string
		isErr  bool
	}{
		{"valid reference", "users.id", "", false},
		{"valid reference 2", "posts.user_id", "", false},
		{"invalid format - no dot", "users", "foreign key reference must be in format 'table.field'", true},
		{"invalid format - too many dots", "users.posts.id", "foreign key reference must be in format 'table.field'", true},
		{"empty table name", ".field", "foreign key table name cannot be empty", true},
		{"empty field name", "users.", "foreign key field name cannot be empty", true},
		{"unknown table", "unknown.id", "foreign key references unknown table: unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateForeignKeyReference(tt.fkRef, availableTables)

			if tt.isErr {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.want) {
					t.Errorf("Expected error containing %q, got %q", tt.want, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}
		})
	}
}
# YAML Configuration Tutorial for weft

This tutorial teaches you how to create your own YAML-based configuration system using the weft toolkit. By the end, you'll understand how to define custom specification types, validate YAML configurations, and generate code from your specifications.

## 🎯 What You'll Learn

1. **Custom Specifications**: Define your own struct types with YAML tags
2. **Configuration Validation**: Implement the `Validator` interface for robust validation
3. **YAML Loading**: Use the generic `config.LoadYAML` function
4. **Template Integration**: Generate code using your YAML data in templates
5. **Real-world Patterns**: Best practices for extensible configuration systems

## 📁 Tutorial Structure

```
yaml-tutorial-guide/
├── main.go              # Main application with step-by-step tutorial comments
├── spec/                # Custom specification types
│   └── schema.go        # Database schema specification with validation
├── templates/           # Code generation templates
│   ├── models.go.tmpl   # Go struct models
│   ├── schema.sql.tmpl  # SQL DDL statements  
│   └── repository.go.tmpl # Database repository layer
├── configs/             # Sample YAML configurations
│   ├── ecommerce-schema.yaml # Complete e-commerce example
│   └── simple-blog.yaml      # Simple blog example
└── README.md           # This tutorial guide
```

## 🚀 Quick Start

### Step 1: Try a YAML Configuration

Run with one of the provided YAML configurations:

```bash
cd examples/yaml-tutorial-guide

# E-commerce schema (complex example)
go run main.go --config configs/ecommerce-schema.yaml

# Blog schema (simple example)  
go run main.go --config configs/simple-blog.yaml
```

Check the `./generated/` directory to see what was created.

### Step 2: Create Your Own Schema

Copy one of the example YAML files and modify it:

```bash
cp configs/simple-blog.yaml my-schema.yaml
# Edit my-schema.yaml to match your needs
go run main.go --config my-schema.yaml
```

## 📖 Detailed Tutorial

### Part 1: Understanding the Specification Types

The tutorial uses a database schema generator as an example. Let's examine the key types in `spec/schema.go`:

#### Root Configuration Type

```go
type DatabaseSchema struct {
    Name        string          `yaml:"name"`        // Schema name
    Version     string          `yaml:"version"`     // Version for compatibility
    Description string          `yaml:"description"` // Optional description
    Package     string          `yaml:"package"`     // Go package name
    Database    DatabaseConfig  `yaml:"database"`    // DB connection config
    Tables      []Table         `yaml:"tables"`      // List of tables
}
```

**Key points:**
- Every field has a `yaml` tag for explicit mapping
- Mix of required and optional fields
- Nested objects and arrays are supported
- Clear, descriptive field names

#### Implementing Validation

```go
func (s *DatabaseSchema) Validate() error {
    if s.Name == "" {
        return fmt.Errorf("schema name is required")
    }
    // ... more validation logic
    return nil
}
```

**Validation best practices:**
- Check required fields first
- Validate field formats and constraints  
- Cross-reference related objects
- Provide clear, actionable error messages

### Part 2: YAML Configuration Format

Here's the structure your users will write:

```yaml
# Basic metadata
name: "My Application Schema"
version: "1.0.0"
description: "Database schema for my application"
package: "myapp"

# Database connection settings
database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  name: "myapp_db"

# Table definitions
tables:
  - name: "users"
    description: "Application users"
    fields:
      - name: "id"
        type: "uuid"
        required: true
        primaryKey: true
      - name: "email"  
        type: "string"
        required: true
        unique: true
        maxLength: 255
```

### Part 3: Loading Configuration in Code

The main application shows how to use the generic config loader:

```go
// Define your config variable
var dbSchema *spec.DatabaseSchema

// Load from YAML file
dbSchema = &spec.DatabaseSchema{}
err = config.LoadYAML(configPath, dbSchema)
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}
// dbSchema now contains validated data from your YAML file!
```

**Key benefits:**
- Type-safe loading with Go generics
- Automatic validation via the `Validator` interface
- Clear error messages for debugging
- Works with any struct type you define

### Part 4: Using Configuration in Templates

Templates receive your configuration as template data:

```go
// In main.go - pass config to templates
err := eng.RenderDir(ctx, "templates", map[string]any{
    "Schema": dbSchema,  // Your YAML config is now available as .Schema
})
```

```go
// In templates - access your configuration
{{ .Schema.Name }}              // Schema name
{{ .Schema.Package }}           // Package name
{{ range .Schema.Tables }}      // Iterate over tables
    {{ .Name }}                 // Table name  
    {{ range .Fields }}         // Iterate over fields
        {{ .Name }}: {{ .Type }} // Field name and type
    {{ end }}
{{ end }}
```

### Part 5: Template Patterns

The tutorial includes three template examples:

#### 1. Go Structs (`models.go.tmpl`)
- Maps database types to Go types
- Generates struct tags for JSON/DB mapping
- Demonstrates type conversion logic

#### 2. SQL DDL (`schema.sql.tmpl`)  
- Creates database tables from your schema
- Handles database-specific SQL syntax
- Generates indexes and constraints

#### 3. Repository Layer (`repository.go.tmpl`)
- Creates CRUD operations for each table
- Demonstrates complex template logic
- Shows method generation patterns

## 🛠️ Creating Your Own Configuration System

### Step 1: Define Your Domain

Think about what your users need to configure:
- What's the core data your templates need?
- What metadata is helpful (name, version, description)?
- What output settings do you need (package name, target directory)?

### Step 2: Design Your Specification Types

Follow these patterns:

```go
// Root configuration type
type MySpec struct {
    // Metadata
    Name        string `yaml:"name"`
    Version     string `yaml:"version"`
    Description string `yaml:"description"`
    
    // Output configuration  
    Package string `yaml:"package"`
    
    // Your core business logic
    Items []MyItem `yaml:"items"`
}

// Implement validation
func (s *MySpec) Validate() error {
    // Check required fields, validate business logic
    return nil
}
```

### Step 3: Create Sample YAML

Write example configurations that show your users the expected format:

```yaml
name: "My Configuration"
version: "1.0.0"
package: "mypackage"
items:
  - name: "example"
    type: "sample"
```

### Step 4: Build Templates

Create templates that use your configuration:

```go
// Access your config in templates
{{ .MySpec.Name }}
{{ range .MySpec.Items }}
  {{ .Name }}: {{ .Type }}
{{ end }}
```

### Step 5: Test and Iterate

- Test with invalid YAML to ensure validation works
- Try edge cases and complex configurations
- Get feedback from users on the configuration format

## 🧪 Testing Your Configuration System

The tutorial includes comprehensive tests in `spec/schema_test.go`:

```bash
# Run the tests
go test ./spec -v

# Test specific validation scenarios
go test ./spec -run TestValidation
```

Key testing patterns:
- **Valid configurations**: Ensure parsing and validation succeed
- **Invalid configurations**: Test validation catches errors
- **Edge cases**: Empty values, missing fields, invalid references
- **YAML parsing**: Test with malformed YAML syntax

## 🎨 Advanced Patterns

### Configuration Inheritance

```yaml
# base-config.yaml
base: &default
  database:
    driver: "postgres"
    host: "localhost"

# app-config.yaml  
<<: *default
name: "My App"
database:
  name: "myapp_db"
```

### Environment-specific Configuration

```go
// Support environment variables in your spec
type DatabaseConfig struct {
    Host string `yaml:"host" env:"DB_HOST"`
    Port int    `yaml:"port" env:"DB_PORT"`
}
```

### Conditional Generation

```yaml
# Enable/disable features in your schema
features:
  authentication: true
  caching: false
  metrics: true
```

```go
// In templates - conditional generation
{{ if .Schema.Features.Authentication }}
// Generate auth-related code
{{ end }}
```

## 🔧 Troubleshooting

### Common Issues

**YAML parsing errors:**
```
Error: yaml: line 15: found character that cannot start any token
```
- Check YAML syntax (indentation, quotes, special characters)
- Use a YAML validator online to check your file

**Validation failures:**
```
Error: table 0 (users): field 2 (email): type field is required
```
- Read the error message carefully - it shows the exact path to the problem
- Check that all required fields are present and have valid values

**Template errors:**
```
Error: template: models.go.tmpl:25: function "title" not defined
```
- Ensure template helper functions are registered with the engine
- Check template syntax and function names

### Getting Help

1. **Check the example configurations** - `configs/` directory has working examples
2. **Read the validation errors** - they point to the exact problem location
3. **Test with simple configurations first** - start small and add complexity
4. **Use the hardcoded example** - run without `--config` to see a working setup

## 🎉 Next Steps

Now that you understand the basics, you can:

1. **Create your own domain-specific generator** (REST APIs, GraphQL schemas, documentation, etc.)
2. **Extend the database example** with more features (migrations, seeds, relationships)
3. **Add advanced template features** (conditional generation, multiple output formats)
4. **Build a CLI tool** around your generator for distribution
5. **Add configuration validation rules** specific to your domain

The patterns shown in this tutorial work for any type of code generation task. The key is defining clear specification types, robust validation, and templates that leverage your configuration data effectively.

Happy coding! 🚀
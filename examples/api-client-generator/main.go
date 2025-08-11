package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/cpcf/weft/engine"
	"github.com/cpcf/weft/processors"
)

//go:embed templates
var templateFS embed.FS

// APISpec represents a simplified API specification
type APISpec struct {
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	BaseURL   string     `json:"baseUrl"`
	Package   string     `json:"package"`
	Endpoints []Endpoint `json:"endpoints"`
	Types     []TypeDef  `json:"types"`
	Auth      AuthConfig `json:"auth"`
}

// Endpoint represents an API endpoint
type Endpoint struct {
	Name        string            `json:"name"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Description string            `json:"description"`
	Request     *TypeRef          `json:"request,omitempty"`
	Response    TypeRef           `json:"response"`
	Parameters  []Parameter       `json:"parameters,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// TypeDef represents a data type definition
type TypeDef struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // "struct", "enum", "alias"
	Description string            `json:"description"`
	Fields      []Field           `json:"fields,omitempty"`
	Values      []string          `json:"values,omitempty"`   // for enums
	BaseType    string            `json:"baseType,omitempty"` // for aliases
	Tags        map[string]string `json:"tags,omitempty"`
}

// Field represents a struct field
type Field struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Required    bool              `json:"required"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// Parameter represents a request parameter
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	In          string `json:"in"` // "query", "path", "header"
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// TypeRef represents a reference to a type
type TypeRef struct {
	Type      string `json:"type"`
	IsArray   bool   `json:"isArray,omitempty"`
	IsPointer bool   `json:"isPointer,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type   string `json:"type"` // "apikey", "bearer", "basic"
	Header string `json:"header,omitempty"`
}

func main() {
	// Create the API specification data
	apiSpec := createSampleAPISpec()

	// Create and configure the engine
	eng := engine.New(
		engine.WithOutputRoot("./generated"),
		engine.WithFailureMode(engine.FailFast),
	)

	// Add post-processors for generated files
	eng.AddPostProcessor(processors.NewGoImports())                       // Fix Go imports
	eng.AddPostProcessor(processors.NewTrimWhitespace())                  // Clean up whitespace
	eng.AddPostProcessor(processors.NewAddGeneratedHeader("weft", ".go")) // Add generated headers

	// Create context with embedded filesystem
	ctx := engine.NewContext(templateFS, "./generated", apiSpec.Package)

	fmt.Printf("Generating Go API client for %s v%s...\n", apiSpec.Name, apiSpec.Version)

	// Render templates using the API specification data
	if err := eng.RenderDir(ctx, "templates", map[string]any{
		"API": apiSpec,
	}); err != nil {
		log.Fatalf("Failed to generate client: %v", err)
	}

	fmt.Printf("✅ Successfully generated Go API client in ./generated/\n")
	fmt.Printf("📁 Files generated:\n")
	fmt.Printf("   - client.go    (Main client implementation)\n")
	fmt.Printf("   - types.go     (Data type definitions)\n")
	fmt.Printf("   - errors.go    (Error handling)\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("   1. cd generated\n")
	fmt.Printf("   2. go mod init %s\n", apiSpec.Package)
	fmt.Printf("   3. go mod tidy\n")
}

func createSampleAPISpec() APISpec {
	return APISpec{
		Name:    "User Management API",
		Version: "1.0.0",
		BaseURL: "https://api.example.com/v1",
		Package: "userapi",
		Auth: AuthConfig{
			Type:   "bearer",
			Header: "Authorization",
		},
		Types: []TypeDef{
			{
				Name:        "User",
				Type:        "struct",
				Description: "Represents a user in the system",
				Fields: []Field{
					{
						Name: "ID", Type: "int64", Description: "Unique user identifier",
						Required: true, Tags: map[string]string{"json": "id"},
					},
					{
						Name: "Email", Type: "string", Description: "User email address",
						Required: true, Tags: map[string]string{"json": "email"},
					},
					{
						Name: "FirstName", Type: "string", Description: "User's first name",
						Required: true, Tags: map[string]string{"json": "first_name"},
					},
					{
						Name: "LastName", Type: "string", Description: "User's last name",
						Required: true, Tags: map[string]string{"json": "last_name"},
					},
					{
						Name: "Status", Type: "UserStatus", Description: "Current user status",
						Required: true, Tags: map[string]string{"json": "status"},
					},
					{
						Name: "CreatedAt", Type: "time.Time", Description: "Account creation timestamp",
						Required: true, Tags: map[string]string{"json": "created_at"},
					},
				},
			},
			{
				Name:        "UserStatus",
				Type:        "enum",
				Description: "Possible user status values",
				Values:      []string{"active", "inactive", "suspended", "pending"},
				BaseType:    "string",
			},
			{
				Name:        "CreateUserRequest",
				Type:        "struct",
				Description: "Request payload for creating a new user",
				Fields: []Field{
					{
						Name: "Email", Type: "string", Description: "User email address",
						Required: true, Tags: map[string]string{"json": "email"},
					},
					{
						Name: "FirstName", Type: "string", Description: "User's first name",
						Required: true, Tags: map[string]string{"json": "first_name"},
					},
					{
						Name: "LastName", Type: "string", Description: "User's last name",
						Required: true, Tags: map[string]string{"json": "last_name"},
					},
				},
			},
			{
				Name:        "UpdateUserRequest",
				Type:        "struct",
				Description: "Request payload for updating user information",
				Fields: []Field{
					{
						Name: "FirstName", Type: "*string", Description: "User's first name",
						Required: false, Tags: map[string]string{"json": "first_name,omitempty"},
					},
					{
						Name: "LastName", Type: "*string", Description: "User's last name",
						Required: false, Tags: map[string]string{"json": "last_name,omitempty"},
					},
					{
						Name: "Status", Type: "*UserStatus", Description: "User status",
						Required: false, Tags: map[string]string{"json": "status,omitempty"},
					},
				},
			},
		},
		Endpoints: []Endpoint{
			{
				Name:        "GetUser",
				Method:      "GET",
				Path:        "/users/{id}",
				Description: "Retrieve a user by ID",
				Parameters: []Parameter{
					{Name: "id", Type: "int64", In: "path", Required: true, Description: "User ID"},
				},
				Response: TypeRef{Type: "User"},
			},
			{
				Name:        "ListUsers",
				Method:      "GET",
				Path:        "/users",
				Description: "List all users with optional filtering",
				Parameters: []Parameter{
					{Name: "status", Type: "UserStatus", In: "query", Required: false, Description: "Filter by user status"},
					{Name: "limit", Type: "int", In: "query", Required: false, Description: "Maximum number of users to return"},
					{Name: "offset", Type: "int", In: "query", Required: false, Description: "Number of users to skip"},
				},
				Response: TypeRef{Type: "User", IsArray: true},
			},
			{
				Name:        "CreateUser",
				Method:      "POST",
				Path:        "/users",
				Description: "Create a new user",
				Request:     &TypeRef{Type: "CreateUserRequest"},
				Response:    TypeRef{Type: "User"},
			},
			{
				Name:        "UpdateUser",
				Method:      "PATCH",
				Path:        "/users/{id}",
				Description: "Update user information",
				Parameters: []Parameter{
					{Name: "id", Type: "int64", In: "path", Required: true, Description: "User ID"},
				},
				Request:  &TypeRef{Type: "UpdateUserRequest"},
				Response: TypeRef{Type: "User"},
			},
			{
				Name:        "DeleteUser",
				Method:      "DELETE",
				Path:        "/users/{id}",
				Description: "Delete a user by ID",
				Parameters: []Parameter{
					{Name: "id", Type: "int64", In: "path", Required: true, Description: "User ID"},
				},
				Response: TypeRef{Type: "bool"},
			},
		},
	}
}

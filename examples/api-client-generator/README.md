# API Client Generator Example

This example demonstrates how to use weft to generate a complete Go API client from a structured configuration. It showcases a realistic use case where you might need to create client libraries for REST APIs.

## What This Example Does

The example generates a complete Go package for interacting with a "User Management API" including:

- **Type definitions** with proper Go naming conventions and JSON tags
- **HTTP client** with methods for each API endpoint  
- **Error handling** with custom error types for different scenarios
- **Authentication support** (Bearer token)
- **Parameter handling** (path, query, header parameters)
- **Request/response serialization** with JSON

## Generated Files

When you run the example, it generates three main files:

```
generated/
├── types.go     # Data type definitions (structs, enums, aliases)
├── errors.go    # Custom error types and error handling
└── client.go    # Main HTTP client with endpoint methods
```

## Key weft Features Demonstrated

### 1. Template Functions
- **Case conversion**: `pascal`, `camel`, `kebab` for proper Go naming
- **Filtering**: `filter` to separate parameters by type (path, query, header)
- **String manipulation**: Template-based URL building and formatting

### 2. Complex Data Structures
- Nested configuration with API specification
- Arrays of objects (endpoints, parameters, fields)
- Conditional logic based on data values

### 3. Template Organization
- Multiple template files working together
- Consistent package generation across files
- Proper imports and dependencies

### 4. Advanced Logic
- **Conditional generation**: Different code paths for different parameter types
- **Loop control**: Generating methods for each endpoint dynamically
- **Type-aware generation**: Handling arrays, pointers, and various Go types

### 5. Professional Code Generation
- Proper error handling patterns
- Thread-safe HTTP client configuration
- Standard Go conventions (interfaces, options pattern)

## Running the Example

1. **Generate the client code:**
   ```bash
   go run main.go
   ```

2. **Explore the generated code:**
   ```bash
   cd generated
   ls -la
   ```

3. **Use the generated client:**
   ```bash
   cd generated
   go mod init userapi
   go mod tidy
   ```

## Example Usage of Generated Client

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "userapi"
)

func main() {
    // Create client with authentication
    client := userapi.NewClient(
        userapi.WithBearerToken("your-api-token"),
        userapi.WithBaseURL("https://api.example.com/v1"),
    )
    
    // Get a user
    user, err := client.GetUser(context.Background(), 123)
    if err != nil {
        var apiErr *userapi.APIError
        if errors.As(err, &apiErr) {
            if apiErr.IsNotFound() {
                fmt.Println("User not found")
                return
            }
        }
        log.Fatal(err)
    }
    
    fmt.Printf("User: %s %s (%s)\n", user.FirstName, user.LastName, user.Email)
    
    // Create a new user
    newUser := userapi.CreateUserRequest{
        Email:     "john@example.com",
        FirstName: "John",
        LastName:  "Doe",
    }
    
    created, err := client.CreateUser(context.Background(), newUser)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Created user with ID: %d\n", created.ID)
}
```

## Template Structure

### types.go.tmpl
- Generates Go structs from API type definitions
- Creates enums with validation methods
- Handles different field types and JSON tags
- Demonstrates conditional template logic

### errors.go.tmpl  
- Creates custom error types for API interactions
- Provides error classification methods
- Shows consistent error handling patterns

### client.go.tmpl
- Most complex template with HTTP client generation
- Handles different HTTP methods and parameter types
- Demonstrates advanced template functions usage
- Shows request/response handling patterns

## Configuration Structure

The example uses a realistic API specification structure:

```go
type APISpec struct {
    Name      string     // API name
    Version   string     // API version  
    BaseURL   string     // Base URL
    Package   string     // Go package name
    Endpoints []Endpoint // API endpoints
    Types     []TypeDef  // Data type definitions
    Auth      AuthConfig // Authentication config
}
```

This structure mimics real API specifications like OpenAPI/Swagger, making it practical for actual use cases.

## Why This Example Is Realistic

1. **Real-world problem**: Generating API clients is a common need
2. **Production-ready patterns**: Follows Go best practices and conventions
3. **Complex template logic**: Shows weft handling sophisticated generation scenarios
4. **Maintainable structure**: Organized templates that can be extended
5. **Professional output**: Generated code looks hand-written and idiomatic

## Extending the Example

You can easily extend this example to:

- Add more authentication methods (API key, OAuth)
- Generate additional files (tests, documentation, examples)
- Support more HTTP methods or parameter types
- Add request/response interceptors
- Include rate limiting or retry logic
- Generate CLI tools that use the client

The modular template structure makes it easy to add new features without breaking existing functionality.
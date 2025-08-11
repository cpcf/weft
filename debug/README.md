# Debug Package

Package debug provides debugging, error handling, and validation capabilities for weft. It offers configurable debug modes, enhanced error reporting with stack traces, template validation, and a rich set of debugging helper functions.

## Architecture

The debug package is organized into four main components:

- **Error Handling** (`errors.go`): Enhanced error types with stack traces and analysis
- **Debug Mode Management** (`mode.go`): Configurable debug levels and logging
- **Template Helpers** (`helpers.go`): Debug functions available in templates
- **Template Validation** (`validation.go`): Syntax and semantic validation

## Basic Usage

### Enable Debug Mode and Configure Logging

```go
debugMode := debug.NewDebugMode()
debugMode.SetLevel(debug.LevelInfo)
```

### Create Enhanced Errors with Stack Traces

```go
err := debug.NewEnhancedError(originalErr, "template_execution")
fmt.Println(err.FormatDetailed())
```

## Template Integration

The package provides debug functions that can be used within templates:

```html
<!-- In your template -->
{{ debug .Data }}
{{ debugType .Value }}
{{ debugJSON .Config }}
{{ debugLog "Processing item" .Item }}
```

### Register Debug Functions

```go
funcMap := debug.CreateDebugFuncMap(debugMode)
tmpl := template.New("example").Funcs(funcMap)
```

## Configuration

The package supports various configuration options:

```go
config := debug.Config{
    MaxStackFrames:       15,  // Stack trace depth
    ErrorBufferSize:      200, // Error history buffer
    ExecutionBufferSize:  150, // Execution tracking buffer
    MaxStackTraceDisplay: 8,   // Stack frames shown in output
}
debug.SetConfig(config)
```

## Template Validation

Validate templates for syntax and semantic issues:

```go
validator := debug.NewTemplateValidator(templateFS, funcMap, debugMode)
result := validator.ValidateTemplate("path/to/template.tmpl")
if !result.Valid {
    for _, err := range result.Errors {
        fmt.Printf("Error: %s\n", err.Message)
    }
}
```

### Validation Features

- **Syntax Validation**: Checks for proper template syntax
- **Security Validation**: Prevents path traversal attacks
- **Function Validation**: Verifies function availability
- **Performance Warnings**: Identifies potential performance issues

## Debug Levels

The package supports multiple debug levels:

| Level | Description |
|-------|-------------|
| `LevelOff` | Disable all debug output |
| `LevelError` | Show only errors |
| `LevelWarn` | Show warnings and errors |
| `LevelInfo` | Show informational messages, warnings, and errors |
| `LevelDebug` | Show debug messages and above |
| `LevelTrace` | Show all messages including detailed traces |

### Setting Debug Levels

```go
debugMode := debug.NewDebugMode()
debugMode.SetLevel(debug.LevelDebug)

// Check if level is enabled
if debugMode.IsEnabled(debug.LevelDebug) {
    // Perform debug operations
}
```

## Template Helper Functions

### Available Functions

| Function | Description | Example |
|----------|-------------|---------|
| `debug` | Display detailed value information | `{{ debug .User }}` |
| `debugType` | Show type information | `{{ debugType .Value }}` |
| `debugKeys` | List map keys or struct fields | `{{ debugKeys .Config }}` |
| `debugSize` | Get size of collections | `{{ debugSize .Items }}` |
| `debugJSON` | Output JSON representation | `{{ debugJSON .Data }}` |
| `debugPretty` | Pretty-printed JSON | `{{ debugPretty .Config }}` |
| `debugLog` | Log message with context | `{{ debugLog "Processing" .Item }}` |
| `debugTime` | Current timestamp | `{{ debugTime }}` |
| `debugStack` | Stack trace (trace level) | `{{ debugStack }}` |
| `debugContext` | Debug context information | `{{ debugContext }}` |

### Usage Examples

```html
<!-- Display user information -->
{{ debug .User }}

<!-- Show data types -->
Type: {{ debugType .Value }}

<!-- List configuration keys -->
Config keys: {{ debugKeys .Config }}

<!-- Pretty JSON output -->
<pre>{{ debugPretty .Settings }}</pre>

<!-- Conditional debugging -->
{{ if debugContext.debug_level }}
    Debug enabled at level: {{ debugContext.debug_level }}
{{ end }}
```

## Error Handling

### Enhanced Error Creation

```go
// Basic enhanced error
err := debug.NewEnhancedError(originalErr, "template_parse")

// With additional context
err = debug.NewEnhancedError(originalErr, "template_parse").
    WithTemplate("user.tmpl").
    WithLine(42).
    WithContext("variable", "user.name").
    WithSuggestion("Check if user.name exists in the data")
```

### Error Analysis

```go
analyzer := debug.NewErrorAnalyzer()
analyzer.AddError(enhancedErr)

// Get error statistics
stats := analyzer.GetStatistics()
fmt.Printf("Total errors: %d\n", stats.TotalErrors)

// Get errors by operation
templateErrors := analyzer.GetErrorsByTemplate("user.tmpl")
```

### Error Context

```go
ctx := debugMode.NewContext("template_execution")
ctx.SetAttribute("template", "user.tmpl")
ctx.SetAttribute("user_id", userID)

// Log with context
ctx.Info("Starting template execution")
ctx.Debug("Processing user data", "count", len(users))

// Complete operation
ctx.Complete()

// Or complete with error
if err != nil {
    ctx.CompleteWithError(err)
}
```

## Performance Considerations

### Optimization Tips

1. **Disable in Production**: Use `LevelOff` or `LevelError` in production
2. **Lazy Evaluation**: Debug operations are only performed when the level is enabled
3. **Efficient Helpers**: Template helpers check debug level before processing
4. **Caching**: Type information and function maps are cached for performance

### Performance Settings

```go
// Production-optimized configuration
config := debug.Config{
    MaxStackFrames:       5,  // Reduced for performance
    ErrorBufferSize:      50, // Smaller buffer
    ExecutionBufferSize:  25, // Minimal tracking
    MaxStackTraceDisplay: 3,  // Limited display
}
```

### Memory Management

```go
// Clear error history periodically
analyzer.Clear()

// Clear execution history
debugger := debug.NewTemplateDebugger(debugMode)
debugger.ClearExecutions()
```

## Security Notes

### Sensitive Data Protection

- Debug output may contain sensitive information
- File paths in stack traces are filtered for security
- Sensitive field names are automatically redacted
- Template validation prevents path traversal attacks

### Security Best Practices

1. **Never expose debug output** in production environments
2. **Filter sensitive data** before logging
3. **Use secure logging destinations** when debug is enabled
4. **Regular security reviews** of debug output

### Sensitive Field Detection

```go
// Automatically filtered fields:
// - password, passwd, pwd
// - secret, api_key, apikey, private_key
// - access_token, refresh_token, bearer_token
// - certificate, cert, ssl
// - session, cookie, csrf
// - credential, cred, token, auth
```

## Thread Safety

All public functions and types in this package are thread-safe and can be used concurrently from multiple goroutines.

### Concurrent Usage Examples

```go
// Safe concurrent debug mode usage
debugMode := debug.NewDebugMode()

go func() {
    debugMode.SetLevel(debug.LevelDebug)
    debugMode.Info("Goroutine 1", "id", 1)
}()

go func() {
    debugMode.SetLevel(debug.LevelTrace)
    debugMode.Debug("Goroutine 2", "id", 2)
}()
```

## Advanced Features

### Template Debugging

```go
debugger := debug.NewTemplateDebugger(debugMode)
debugger.RegisterTemplate("user", userTemplate)

// Execute with debugging
output, err := debugger.ExecuteWithDebug("user", userTemplate, userData)

// Get execution statistics
stats := debugger.GetExecutionStats()
fmt.Printf("Success rate: %.2f%%\n", stats["success_rate"].(float64)*100)
```

### Custom Validation Rules

```go
validator := debug.NewTemplateValidator(fs, funcMap, debugMode)
validator.SetStrict(true) // Enable strict validation

// Validate entire directory
results := validator.ValidateDirectory("templates/")
for path, result := range results {
    if result.HasErrors() {
        fmt.Printf("Template %s has errors: %s\n", path, result.Summary())
    }
}
```

### Error Recovery

```go
// Implement error recovery patterns
func processTemplate() {
    defer func() {
        if r := recover(); r != nil {
            if err, ok := r.(error); ok {
                enhanced := debug.NewEnhancedError(err, "template_panic")
                log.Printf("Template panic recovered: %s", enhanced.FormatDetailed())
            }
        }
    }()
    
    // Template processing code
}
```

## Integration Examples

### With HTTP Handlers

```go
func templateHandler(w http.ResponseWriter, r *http.Request) {
    ctx := debugMode.NewContext("http_template")
    ctx.SetAttribute("method", r.Method)
    ctx.SetAttribute("path", r.URL.Path)
    
    defer ctx.Complete()
    
    // Template execution with debugging
    output, err := debugger.ExecuteWithDebug("page", tmpl, data)
    if err != nil {
        ctx.CompleteWithError(err)
        http.Error(w, "Template error", http.StatusInternalServerError)
        return
    }
    
    w.Write([]byte(output))
}
```

### With CLI Applications

```go
func main() {
    debugMode := debug.NewDebugMode()
    
    if verbose {
        debugMode.SetLevel(debug.LevelDebug)
    }
    
    // Process templates with debugging
    for _, templateFile := range templateFiles {
        ctx := debugMode.NewContext("cli_process")
        ctx.SetAttribute("file", templateFile)
        
        if err := processTemplate(templateFile); err != nil {
            enhanced := debug.NewEnhancedError(err, "cli_template").
                WithTemplate(templateFile).
                WithSuggestion("Check template syntax and data structure")
            
            fmt.Fprintf(os.Stderr, "Error: %s\n", enhanced.FormatDetailed())
            ctx.CompleteWithError(err)
        } else {
            ctx.Complete()
        }
    }
}
```
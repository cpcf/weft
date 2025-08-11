# Render Package

Package render provides a template rendering system for weft. It offers advanced template discovery, block management, partial and include support, collection operations, and an extensive function registry with string manipulation utilities.

## Architecture

The render package is organized into eight main components:

- **Template Discovery** (`discovery.go`): Configurable template file discovery with pattern matching
- **Block Management** (`blocks.go`): Template block definition, override, and inheritance system
- **Include System** (`includes.go`): Template inclusion with data passing and circular dependency detection
- **Partial Management** (`partials.go`): Partial template loading and registration
- **Function Registry** (`registry.go`): Extensible function management with metadata and documentation
- **Collection Operations** (`collections.go`): Advanced slice and array manipulation functions
- **Function Library** (`funcs.go`): Default and extended template function sets
- **String Utilities** (`strings.go`): Comprehensive string manipulation and formatting functions

## Basic Usage

### Initialize Template Discovery

```go
discovery := render.NewTemplateDiscovery(templateFS)
discovery.AddDefaultRules()
templates, err := discovery.DiscoverTemplates("templates/")
```

### Create Block Manager

```go
blockManager := render.NewBlockManager(templateFS, funcMap)
blockManager.DefineBlock("header", "<h1>{{.Title}}</h1>")
```

### Set Up Function Registry

```go
registry := render.NewFunctionRegistry()
registry.RegisterDefaults()
funcMap := registry.GetFuncMap()
```

## Template Integration

The package provides comprehensive template functions for various operations:

```html
<!-- String manipulation -->
{{ "hello_world" | camel }}     <!-- helloWorld -->
{{ "HelloWorld" | snake }}      <!-- hello_world -->
{{ "hello world" | pascal }}    <!-- HelloWorld -->

<!-- Collection operations -->
{{ .Items | filter (lambda .Active) }}
{{ .Numbers | sort | unique }}
{{ .Users | map (lambda .Name) | join ", " }}

<!-- Block usage -->
{{ block "content" "Default content" }}
{{ override "footer" "Custom footer content" }}

<!-- Include templates -->
{{ include "partials/header" }}
{{ includeWith "partials/user-card" .User }}
```

### Register Function Registry with Templates

```go
registry := render.NewFunctionRegistry()
registry.RegisterDefaults()
registry.RegisterExtended()

tmpl := template.New("main").Funcs(registry.GetFuncMap())
```

## Configuration

### Template Discovery Rules

```go
discovery := render.NewTemplateDiscovery(templateFS)

// Add custom discovery rule
rule := render.DiscoveryRule{
    Name:        "components",
    Patterns:    []string{"*.component.tmpl"},
    Extensions:  []string{".tmpl"},
    Directories: []string{"components"},
    Recursive:   true,
    Priority:    25,
}
discovery.AddRule(rule)
```

### Include Manager Configuration

```go
includeManager := render.NewIncludeManager(templateFS, funcMap)
includeManager.SetMaxDepth(20) // Prevent deep nesting

// Preload commonly used includes
templatePaths := []string{"main.tmpl", "layout.tmpl"}
err := includeManager.PreloadIncludes(templatePaths)
```

### Function Registry Setup

```go
registry := render.NewFunctionRegistry()

// Register custom function with metadata
registry.Register("customFunc", myFunction,
    render.WithDescription("Custom processing function"),
    render.WithCategory("custom"),
    render.WithParameters(
        render.ParamInfo{Name: "input", Type: "string", Required: true},
        render.ParamInfo{Name: "options", Type: "map", Required: false},
    ),
    render.WithReturnType("string"),
    render.WithExamples(`{{ "text" | customFunc }}`),
    render.WithSince("2.0.0"))
```

## Template Discovery

Discover templates using configurable rules and patterns:

```go
discovery := render.NewTemplateDiscovery(templateFS)
discovery.AddDefaultRules()

// Discover all templates
templates, err := discovery.DiscoverTemplates("templates/")
for _, tmpl := range templates {
    fmt.Printf("Found: %s (%s)\n", tmpl.Name, tmpl.Path)
}

// Get templates by type
partials, err := discovery.GetTemplatesByType("templates/", "partial")
includes, err := discovery.GetTemplatesByType("templates/", "include")
```

### Discovery Rule Types

| Rule Type | Description | Example Pattern |
|-----------|-------------|-----------------|
| `standard-templates` | Main template files | `*.tmpl`, `*.tpl` |
| `partials` | Underscore-prefixed partials | `_*.tmpl`, `_*.tpl` |
| `includes` | Include directory templates | Files in `includes/` |

### Advanced Discovery

```go
// Find templates by pattern
matched, err := discovery.GetTemplatesByPattern("templates/", `user.*\.tmpl$`)

// Get discovery statistics
stats, err := discovery.GetDiscoveryStats("templates/")
fmt.Printf("Total templates: %d\n", stats["total"])
fmt.Printf("Partials: %d\n", stats["partials"])
```

## Block Management

### Block Definition and Usage

```go
blockManager := render.NewBlockManager(templateFS, funcMap)

// Define blocks
blockManager.DefineBlock("navigation", `
<nav>
  <ul>
    {{range .NavItems}}
    <li><a href="{{.URL}}">{{.Title}}</a></li>
    {{end}}
  </ul>
</nav>`)

// Override blocks
blockManager.OverrideBlock("navigation", "<nav>Custom navigation</nav>")
```

### Template Block Syntax

```html
<!-- Define a block -->
{{ define "sidebar" }}
<aside>Default sidebar content</aside>
{{ end }}

<!-- Use a block with fallback -->
{{ block "sidebar" "Default content" }}

<!-- Override a block -->
{{ override "sidebar" }}
<aside>Custom sidebar for this template</aside>
{{ end }}
```

### Block Validation

```go
// Validate block references
err := blockManager.ValidateBlocks("templates/main.tmpl")
if err != nil {
    log.Printf("Block validation error: %v", err)
}

// Get block information
blocks := blockManager.GetBlockInfo()
for _, block := range blocks {
    fmt.Printf("Block: %s, Override: %t\n", block.Name, block.Override)
}
```

## Include System

### Basic Includes

```go
includeManager := render.NewIncludeManager(templateFS, funcMap)

// Template usage
// {{ include "header" }}
// {{ includeWith "user-profile" .User }}
```

### Include Path Resolution

```go
// Automatic path resolution tries:
// - exact path
// - path + .tmpl
// - path + .tpl  
// - includes/path
// - includes/path.tmpl
// - includes/path.tpl
```

### Include Validation and Dependencies

```go
// Validate includes
err := includeManager.ValidateIncludes("templates/main.tmpl")

// Get include dependency graph
graph, err := includeManager.GetIncludeGraph("templates/main.tmpl", nil)
for template, includes := range graph {
    fmt.Printf("%s includes: %v\n", template, includes)
}
```

## Partial Templates

### Partial Discovery and Loading

```go
partialManager := render.NewPartialManager(templateFS, funcMap)

// Find partials in directory
partials, err := partialManager.FindPartials("templates/")

// Create template with partials
tmpl, err := partialManager.ParseTemplateWithPartials("main.tmpl")
```

### Partial Naming Convention

```go
// File: _header.tmpl -> Partial name: "header"
// File: _user-card.tmpl -> Partial name: "user-card"
// File: _navigation.tpl -> Partial name: "navigation"
```

### Partial Template Usage

```html
<!-- In your main template -->
{{ template "header" .HeaderData }}
{{ template "user-card" .User }}
{{ template "navigation" .NavItems }}
```

## Function Library

### String Functions

| Function | Description | Example |
|----------|-------------|---------|
| `snake` | Convert to snake_case | `{{ "HelloWorld" \| snake }}` |
| `camel` | Convert to camelCase | `{{ "hello_world" \| camel }}` |
| `pascal` | Convert to PascalCase | `{{ "hello_world" \| pascal }}` |
| `kebab` | Convert to kebab-case | `{{ "HelloWorld" \| kebab }}` |
| `plural` | Pluralize word | `{{ "person" \| plural }}` |
| `singular` | Singularize word | `{{ "people" \| singular }}` |
| `humanize` | Human readable format | `{{ "user_name" \| humanize }}` |
| `indent` | Indent text lines | `{{ .Code \| indent 4 }}` |
| `quote` | Add double quotes | `{{ .String \| quote }}` |
| `comment` | Add comment prefix | `{{ .Text \| comment "//" }}` |

### Collection Functions

| Function | Description | Example |
|----------|-------------|---------|
| `formatSlice` | Format slice with separator | `{{ formatSlice .Items ", " "%s" }}` |
| `filter` | Filter by predicate | `{{ .Users \| filter .IsActive }}` |
| `map` | Transform elements | `{{ .Items \| map .GetName }}` |
| `first` | Get first element | `{{ .Items \| first }}` |
| `last` | Get last element | `{{ .Items \| last }}` |
| `rest` | Get all but first | `{{ .Items \| rest }}` |
| `reverse` | Reverse order | `{{ .Items \| reverse }}` |
| `sort` | Sort elements | `{{ .Numbers \| sort }}` |
| `unique` | Remove duplicates | `{{ .Items \| unique }}` |
| `shuffle` | Random order | `{{ .Cards \| shuffle }}` |
| `chunk` | Split into chunks | `{{ sliceChunk .Items 3 }}` |
| `zip` | Combine slices | `{{ sliceZip .Names .Values }}` |

### Math and Utility Functions

| Function | Description | Example |
|----------|-------------|---------|
| `add` | Add numbers | `{{ add 5 3 }}` |
| `subtract` | Subtract numbers | `{{ subtract 10 3 }}` |
| `multiply` | Multiply numbers | `{{ multiply 4 5 }}` |
| `divide` | Divide numbers | `{{ divide 15 3 }}` |
| `max` | Maximum value | `{{ max .Values }}` |
| `min` | Minimum value | `{{ min .Values }}` |
| `default` | Default value | `{{ default "none" .Value }}` |
| `coalesce` | First non-empty | `{{ coalesce .A .B .C }}` |
| `ternary` | Conditional value | `{{ ternary .Condition "yes" "no" }}` |

### Extended Functions

```go
// Register extended functions for additional capabilities
registry.RegisterExtended()

// Available extended functions:
// - UUID generation: {{ uuid }}
// - Hashing: {{ "text" | md5 }}, {{ "text" | sha256 }}
// - Encoding: {{ "text" | base64 }}, {{ .Base64 | base64dec }}
// - Environment: {{ env "HOME" }}, {{ hasEnv "DEBUG" }}
// - Regex: {{ regexMatch "^test" .String }}
// - Path operations: {{ pathJoin .Dir .File }}
// - Semver: {{ semver "v1.2.3" }}
// - Random: {{ randInt 1 100 }}, {{ genPassword 12 }}
```

## Performance Considerations

### Optimization Tips

1. **Template Caching**: Use discovery cache for repeated template lookups
2. **Preload Includes**: Load commonly used includes at startup
3. **Function Registry**: Register functions once and reuse FuncMap
4. **Block Management**: Clear unused blocks and overrides periodically
5. **Partial Caching**: Cache parsed templates with partials

### Performance Settings

```go
// Optimize include depth for your use case
includeManager.SetMaxDepth(5) // Lower depth for simpler templates

// Clear caches periodically in long-running applications
partialManager.ClearCache()
includeManager.ClearCache()
blockManager.ClearAll()
```

### Memory Management

```go
// Monitor registry size
registry := render.NewFunctionRegistry()
fmt.Printf("Registered functions: %d\n", registry.Count())

// Clear discovery cache when templates change
discovery.ClearCache()

// Use selective function registration
registry.RegisterDefaults() // Essential functions only
// registry.RegisterExtended() // Add only if needed
```

## Security Notes

### Template Security

- Include path resolution prevents directory traversal attacks
- Function registry validates function types before registration
- Template validation catches syntax errors before execution
- Block validation ensures referenced blocks exist

### Security Best Practices

1. **Validate all template inputs** before rendering
2. **Sanitize user-provided data** in templates
3. **Limit include depth** to prevent resource exhaustion
4. **Use allow-lists** for template discovery paths
5. **Regular security audits** of custom functions

### Input Sanitization

```go
// Use built-in escaping functions
// {{ .UserInput | htmlEscape }}
// {{ .JSValue | jsEscape }}
// {{ .URLParam | urlQuery }}
// {{ .YAMLValue | yamlEscape }}
```

## Thread Safety

All public functions and types in this package are thread-safe and can be used concurrently from multiple goroutines. Internal synchronization is handled automatically.

### Concurrent Usage Examples

```go
// Safe concurrent discovery
discovery := render.NewTemplateDiscovery(templateFS)
go func() {
    templates, _ := discovery.DiscoverTemplates("path1")
    // Process templates
}()
go func() {
    templates, _ := discovery.DiscoverTemplates("path2")  
    // Process templates
}()

// Safe concurrent function registration
registry := render.NewFunctionRegistry()
go registry.Register("func1", fn1)
go registry.Register("func2", fn2)
```

## Advanced Features

### Custom Discovery Rules

```go
// Create domain-specific discovery rules
emailRule := render.DiscoveryRule{
    Name:        "email-templates",
    Patterns:    []string{"*.email.tmpl"},
    Extensions:  []string{".tmpl"},
    Directories: []string{"emails", "notifications"},
    Recursive:   true,
    Priority:    30,
}
discovery.AddRule(emailRule)
```

### Function Documentation Generation

```go
registry := render.NewFunctionRegistry()
registry.RegisterDefaults()

// Generate function documentation
docs := registry.GetDocumentation()
fmt.Println(docs) // Markdown documentation

// Export function metadata
metadata := registry.ExportJSON()
// Save to file or API
```

### Complex Template Composition

```go
// Combine all managers for full functionality
blockManager := render.NewBlockManager(templateFS, nil)
includeManager := render.NewIncludeManager(templateFS, nil)  
partialManager := render.NewPartialManager(templateFS, nil)
registry := render.NewFunctionRegistry()

// Merge all function maps
funcMap := registry.MergeFuncMap(blockManager.GetFuncMap())
funcMap = registry.MergeFuncMap(includeManager.GetFuncMap())

// Use combined functionality in templates
tmpl := template.New("complex").Funcs(funcMap)
```

## Integration Examples

### With Web Applications

```go
func setupTemplateEngine() *template.Template {
    templateFS := os.DirFS("templates")
    
    // Set up discovery
    discovery := render.NewTemplateDiscovery(templateFS)
    discovery.AddDefaultRules()
    
    // Set up managers
    blockManager := render.NewBlockManager(templateFS, nil)
    includeManager := render.NewIncludeManager(templateFS, nil)
    partialManager := render.NewPartialManager(templateFS, nil)
    registry := render.NewFunctionRegistry()
    registry.RegisterDefaults()
    
    // Create combined function map
    funcMap := registry.MergeFuncMap(blockManager.GetFuncMap())
    funcMap = registry.MergeFuncMap(includeManager.GetFuncMap())
    
    // Discover and parse templates
    templates, _ := discovery.DiscoverTemplates(".")
    
    rootTemplate := template.New("app").Funcs(funcMap)
    for _, tmpl := range templates {
        content, _ := fs.ReadFile(templateFS, tmpl.Path)
        rootTemplate.New(tmpl.Name).Parse(string(content))
    }
    
    return rootTemplate
}
```

### With CLI Code Generation

```go
func generateCode(templateDir, outputDir string, data interface{}) error {
    templateFS := os.DirFS(templateDir)
    
    // Set up rendering system
    registry := render.NewFunctionRegistry()
    registry.RegisterDefaults()
    registry.RegisterExtended() // For file operations
    
    discovery := render.NewTemplateDiscovery(templateFS)
    discovery.AddDefaultRules()
    
    // Add code generation specific rules
    codeRule := render.DiscoveryRule{
        Name:        "code-templates", 
        Patterns:    []string{"*.go.tmpl", "*.js.tmpl"},
        Extensions:  []string{".tmpl"},
        Recursive:   true,
        Priority:    20,
    }
    discovery.AddRule(codeRule)
    
    // Generate code files
    templates, err := discovery.DiscoverTemplates(".")
    if err != nil {
        return err
    }
    
    for _, tmpl := range templates {
        output, err := renderTemplate(templateFS, tmpl.Path, data, registry.GetFuncMap())
        if err != nil {
            return fmt.Errorf("rendering %s: %w", tmpl.Path, err)
        }
        
        outputFile := filepath.Join(outputDir, tmpl.Name)
        err = os.WriteFile(outputFile, []byte(output), 0644)
        if err != nil {
            return fmt.Errorf("writing %s: %w", outputFile, err)
        }
    }
    
    return nil
}
```

### With Template Validation Pipeline

```go
func validateTemplateSystem(templateFS fs.FS) error {
    // Set up all components
    discovery := render.NewTemplateDiscovery(templateFS)
    discovery.AddDefaultRules()
    
    blockManager := render.NewBlockManager(templateFS, nil)
    includeManager := render.NewIncludeManager(templateFS, nil)
    partialManager := render.NewPartialManager(templateFS, nil)
    registry := render.NewFunctionRegistry()
    registry.RegisterDefaults()
    
    // Validate discovery
    if err := discovery.ValidateDiscovery("."); err != nil {
        return fmt.Errorf("discovery validation failed: %w", err)
    }
    
    // Get all templates
    templates, err := discovery.DiscoverTemplates(".")
    if err != nil {
        return err
    }
    
    // Validate each template
    for _, tmpl := range templates {
        // Validate blocks
        if err := blockManager.ValidateBlocks(tmpl.Path); err != nil {
            return fmt.Errorf("block validation failed for %s: %w", tmpl.Path, err)
        }
        
        // Validate includes  
        if err := includeManager.ValidateIncludes(tmpl.Path); err != nil {
            return fmt.Errorf("include validation failed for %s: %w", tmpl.Path, err)
        }
        
        // Validate partials
        if err := partialManager.ValidatePartials(tmpl.Path); err != nil {
            return fmt.Errorf("partial validation failed for %s: %w", tmpl.Path, err)
        }
    }
    
    return nil
}
```
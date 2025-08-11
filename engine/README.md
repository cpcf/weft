# Engine Package

Package engine provides the core template processing engine for weft. It handles template rendering with caching, concurrent processing, error handling, and context management to support efficient code generation workflows.

## Architecture

The engine package is organized into six main components:

- **Engine Core** (`engine.go`): Main orchestrator with configurable options and rendering coordination
- **Template Cache** (`cache.go`): Thread-safe template caching with filesystem-aware keys
- **Context Management** (`context.go`): Execution context encapsulation with filesystem and output path handling
- **Renderer** (`renderer.go`): Template file processing and output generation
- **Concurrency** (`concurrency.go`): Worker pools, concurrent rendering, and async task management
- **Error Handling** (`errors.go`): Enhanced error types with path context and multi-error aggregation

## Basic Usage

### Create and Configure Engine

```go
engine := engine.New(
    engine.WithOutputRoot("./generated"),
    engine.WithFailureMode(engine.FailFast),
    engine.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, nil))),
)
```

### Render Templates from Directory

```go
ctx := engine.NewContext(templateFS, "./output", "github.com/example/package")
data := map[string]any{
    "Package": "main",
    "Version": "1.0.0",
    "Author":  "John Doe",
}

err := engine.RenderDir(ctx, "templates/", data)
if err != nil {
    log.Fatalf("Rendering failed: %v", err)
}
```

## Template Integration

Templates use standard Go template syntax and are automatically processed:

```html
<!-- In your template file: config.go.tmpl -->
package {{.Package}}

// Version represents the application version
const Version = "{{.Version}}"

// Author information
const Author = "{{.Author}}"

// Generated configuration
var Config = struct {
    Debug   bool
    Timeout int
}{
    Debug:   {{.Debug | default false}},
    Timeout: {{.Timeout | default 30}},
}
```

### Template File Naming

- Templates must end with `.tmpl` extension
- Output files strip the `.tmpl` extension: `config.go.tmpl` → `config.go`
- Directory structure is preserved in output

## Configuration

The engine supports various configuration options through functional options:

```go
engine := engine.New(
    engine.WithOutputRoot("./generated"),           // Set output directory
    engine.WithFailureMode(engine.FailAtEnd),      // Configure error handling
    engine.WithLogger(customLogger),               // Custom structured logger
)
```

### Failure Modes

Control how the engine handles errors during batch processing:

| Mode | Description |
|------|-------------|
| `FailFast` | Stop immediately on first error (default) |
| `FailAtEnd` | Process all templates, then return aggregated errors |
| `BestEffort` | Continue processing despite errors, no error returned |

### Context Configuration

```go
ctx := engine.NewContext(
    templateFS,                    // Template filesystem
    "./output",                   // Output root directory  
    "github.com/example/package", // Go package path (future use)
)
```

## Template Caching

The engine includes intelligent template caching for performance:

```go
cache := engine.NewTemplateCache()

// Templates are automatically cached by filesystem + path
template, err := cache.Get(templateFS, "user.tmpl")
if err != nil {
    return err
}

// Clear cache when needed
cache.Clear()
```

### Cache Features

- **Filesystem Aware**: Different filesystems cache separately even with same paths
- **Thread Safe**: Concurrent access with read/write locks
- **Automatic Invalidation**: Cache keys include filesystem identity
- **Memory Efficient**: Lazy loading and parsing on demand

## Failure Modes and Error Handling

### Basic Error Handling

```go
err := engine.RenderDir(ctx, "templates/", data)
if err != nil {
    // Handle single error or multi-error
    if multiErr, ok := err.(*engine.MultiError); ok {
        for _, genErr := range multiErr.Errors {
            fmt.Printf("Error in %s: %s\n", genErr.Path, genErr.Message)
        }
    }
}
```

### Error Types

```go
// Single generation error with context
genErr := &engine.GenerationError{
    Path:    "templates/config.go.tmpl",
    Message: "template parse failed",
    Err:     originalError,
}

// Multiple errors aggregated
var multiErr engine.MultiError
multiErr.Add("template1.tmpl", "parse error", parseErr)
multiErr.Add("template2.tmpl", "execution error", execErr)
```

### Failure Mode Examples

```go
// Fail fast - stop on first error
engine := engine.New(engine.WithFailureMode(engine.FailFast))

// Fail at end - collect all errors
engine := engine.New(engine.WithFailureMode(engine.FailAtEnd))

// Best effort - ignore errors and process what we can
engine := engine.New(engine.WithFailureMode(engine.BestEffort))
```

## Concurrent Rendering

### Worker Pool Management

```go
// Create concurrent renderer with worker pool
renderer := engine.NewConcurrentRenderer(4, *baseRenderer)
renderer.Start()
defer renderer.Stop()

// Submit async rendering tasks
taskID, resultChan, err := renderer.RenderAsync(
    "template.tmpl",
    "output.go", 
    templateData,
)

// Wait for completion
result := <-resultChan
if result.Success {
    fmt.Printf("Rendered %s in %v\n", taskID, result.Duration)
}
```

### Batch Processing

```go
requests := []engine.RenderRequest{
    {TemplatePath: "user.tmpl", OutputPath: "user.go", Data: userData},
    {TemplatePath: "config.tmpl", OutputPath: "config.go", Data: configData},
    {TemplatePath: "handlers.tmpl", OutputPath: "handlers.go", Data: handlerData},
}

results, err := renderer.RenderBatch(requests)
for _, result := range results {
    if !result.Success {
        fmt.Printf("Task %s failed: %s\n", result.TaskID, result.Error)
    }
}
```

### Monitoring and Stats

```go
stats := renderer.GetStats()
fmt.Printf("Workers: %d, Queue: %d/%d, Completed: %d, Failed: %d\n",
    stats.WorkerCount,
    stats.QueueLength,
    stats.QueueCapacity,
    stats.TasksCompleted,
    stats.TasksFailed,
)
```

## Performance Considerations

### Optimization Tips

1. **Template Caching**: Templates are automatically cached after first parse
2. **Concurrent Processing**: Use worker pools for large template sets
3. **Failure Modes**: Choose appropriate mode for your use case
4. **Context Reuse**: Reuse contexts when processing multiple template sets
5. **Resource Management**: Always call Stop() on concurrent renderers

### Performance Settings

```go
// Optimized for throughput
renderer := engine.NewConcurrentRenderer(
    runtime.NumCPU()*2, // Worker count
    baseRenderer,
)

// Monitor queue depth
if stats.QueueLength > stats.QueueCapacity*0.8 {
    log.Warn("Worker pool queue nearly full")
}
```

### Memory Management

```go
// Clear template cache periodically
cache.Clear()

// Stop worker pools to free resources
renderer.Stop()

// Wait for completion with timeout
err := renderer.WaitForCompletion(30 * time.Second)
if err != nil {
    log.Warn("Timeout waiting for rendering completion")
}
```

## Security Notes

### Path Security

- Output paths are resolved safely using filepath operations
- Template paths are validated against the provided filesystem
- Directory traversal attacks are prevented by filesystem boundaries
- Generated files respect the output root configuration

### Security Best Practices

1. **Validate Template Sources**: Only use trusted template filesystems
2. **Sanitize Output Paths**: Ensure output directories are under intended root
3. **Control Template Data**: Validate and sanitize template input data
4. **Monitor File Creation**: Log and audit generated file locations

### Secure Configuration

```go
// Secure output configuration
engine := engine.New(
    engine.WithOutputRoot("/safe/output/path"), // Controlled output location
    engine.WithLogger(auditLogger),            // Security event logging
)

// Validate context paths
ctx := engine.NewContext(trustedFS, safeOutputDir, packagePath)
```

## Thread Safety

All public functions and types in this package are thread-safe and can be used concurrently from multiple goroutines.

### Concurrent Usage Examples

```go
// Safe concurrent engine usage
engine := engine.New()

var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        
        ctx := engine.NewContext(templateFS, fmt.Sprintf("./output-%d", id), "example")
        data := map[string]any{"ID": id}
        
        err := engine.RenderDir(ctx, "templates/", data)
        if err != nil {
            log.Printf("Goroutine %d failed: %v", id, err)
        }
    }(i)
}
wg.Wait()
```

### Thread-Safe Components

```go
// Template cache is thread-safe
cache := engine.NewTemplateCache()
go func() { cache.Get(fs1, "template1.tmpl") }()
go func() { cache.Get(fs2, "template2.tmpl") }()

// Concurrent safe map for shared data
safeMap := engine.NewConcurrentSafeMap()
safeMap.Set("key", "value")
value, exists := safeMap.Get("key")
```

## Advanced Features

### Custom Worker Pool Configuration

```go
pool := engine.NewWorkerPool(8) // 8 workers
pool.Start()

// Submit tasks with timeout
task := &customTask{...}
err := pool.SubmitWithTimeout(task, 10*time.Second)
if err != nil {
    log.Printf("Task submission failed: %v", err)
}

pool.Stop()
```

### Safe Engine Wrapper

```go
// Wrap engine for additional thread safety
safeEngine := engine.NewSafeEngine(*regularEngine)

// Use in concurrent scenarios
go func() {
    safeEngine.RenderDir(ctx1, "templates/", data1)
}()
go func() {
    safeEngine.RenderDir(ctx2, "templates/", data2)
}()
```

### Custom Task Implementation

```go
type CustomRenderTask struct {
    id         string
    priority   int
    templateFS fs.FS
    data       any
}

func (t *CustomRenderTask) Execute(ctx context.Context) error {
    // Custom rendering logic
    return nil
}

func (t *CustomRenderTask) ID() string { return t.id }
func (t *CustomRenderTask) Priority() int { return t.priority }
```

## Integration Examples

### With HTTP Handlers

```go
func codeGenHandler(w http.ResponseWriter, r *http.Request) {
    engine := engine.New(
        engine.WithOutputRoot("./generated"),
        engine.WithFailureMode(engine.FailFast),
    )
    
    ctx := engine.NewContext(templateFS, "./output", "api")
    data := extractDataFromRequest(r)
    
    if err := engine.RenderDir(ctx, "api-templates/", data); err != nil {
        http.Error(w, fmt.Sprintf("Code generation failed: %v", err), 
                   http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
        "output": "./output",
    })
}
```

### With CLI Applications

```go
func main() {
    var (
        templateDir = flag.String("templates", "./templates", "Template directory")
        outputDir   = flag.String("output", "./generated", "Output directory") 
        configFile  = flag.String("config", "config.json", "Configuration file")
        concurrent  = flag.Bool("concurrent", false, "Enable concurrent processing")
    )
    flag.Parse()
    
    // Load configuration data
    data, err := loadConfig(*configFile)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Setup engine
    engine := engine.New(
        engine.WithOutputRoot(*outputDir),
        engine.WithFailureMode(engine.FailAtEnd),
    )
    
    ctx := engine.NewContext(os.DirFS("."), *outputDir, data.PackagePath)
    
    if *concurrent {
        // Use concurrent rendering
        renderer := engine.NewConcurrentRenderer(runtime.NumCPU(), *engine.renderer)
        renderer.Start()
        defer renderer.Stop()
        
        // Process with monitoring
        go func() {
            ticker := time.NewTicker(1 * time.Second)
            defer ticker.Stop()
            
            for range ticker.C {
                stats := renderer.GetStats()
                log.Printf("Progress: %d completed, %d failed, %d processing", 
                          stats.TasksCompleted, stats.TasksFailed, stats.TasksProcessing)
                
                if stats.TasksProcessing == 0 && stats.QueueLength == 0 {
                    break
                }
            }
        }()
        
        err = renderer.WaitForCompletion(5 * time.Minute)
    } else {
        // Standard synchronous rendering
        err = engine.RenderDir(ctx, *templateDir, data)
    }
    
    if err != nil {
        if multiErr, ok := err.(*engine.MultiError); ok {
            log.Printf("Generation completed with %d errors:", len(multiErr.Errors))
            for _, genErr := range multiErr.Errors {
                log.Printf("  %s: %s", genErr.Path, genErr.Message)
            }
            os.Exit(1)
        } else {
            log.Fatalf("Generation failed: %v", err)
        }
    }
    
    log.Println("Code generation completed successfully")
}
```

### With Build Systems

```go
// Integration with build pipelines
func generateCode(buildContext *BuildContext) error {
    engine := engine.New(
        engine.WithOutputRoot(buildContext.OutputDir),
        engine.WithFailureMode(engine.BestEffort), // Continue on errors
        engine.WithLogger(buildContext.Logger),
    )
    
    // Use embedded templates from build
    ctx := engine.NewContext(buildContext.TemplateFS, buildContext.OutputDir, buildContext.Package)
    
    // Generate with build metadata
    data := map[string]any{
        "Package":     buildContext.Package,
        "Version":     buildContext.Version,
        "BuildTime":   time.Now().Format(time.RFC3339),
        "Commit":      buildContext.GitCommit,
        "Features":    buildContext.EnabledFeatures,
    }
    
    return engine.RenderDir(ctx, ".", data)
}
```
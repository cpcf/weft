# Write Package

Package write provides file writing, composite operations, coordination, and custom writer capabilities for the gogenkit template engine. It offers configurable write strategies, atomic operations, file locking, retry mechanisms, and specialized writers for different use cases.

## Architecture

The write package is organized into four main components:

- **Core Writing** (`writer.go`): Base writer interface and implementation with backup, atomic, and directory creation features
- **Composite Operations** (`composite.go`): Multiple writer coordination with conditions, chaining, validation, and retry logic
- **File Coordination** (`coordination.go`): Thread-safe file locking and concurrent write management
- **Custom Writers** (`custom.go`): Specialized writers for skip-if-exists, segment replacement, timestamps, and dry-run operations

## Basic Usage

### Create a Basic Writer with Options

```go
writer := write.NewBaseWriter()
options := write.WriteOptions{
    CreateDirs: true,
    Backup:     true,
    Overwrite:  true,
    Atomic:     true,
}

err := writer.Write("/path/to/file.txt", []byte("content"), options)
```

### Use a Composite Writer with Multiple Strategies

```go
composite := write.NewCompositeWriter()

// Add writers with conditions and priorities
goWriter := write.NewBaseWriter()
composite.AddWriter(
    write.ExtensionCondition(".go", ".gogen"),
    goWriter,
    100, // high priority
)

jsWriter := write.NewBaseWriter()
composite.AddWriter(
    write.ExtensionCondition(".js", ".ts"),
    jsWriter,
    50, // lower priority
)

err := composite.Write("/path/to/file.go", content, options)
```

## Template Integration

The package integrates seamlessly with template engines for code generation:

```go
// Create a writer with automatic directory creation
writer := write.NewBaseWriter()
options := write.WriteOptions{
    CreateDirs: true,
    Backup:     true,
    Atomic:     true,
}

// Write generated template output
templateOutput := []byte("package main\n\nfunc main() {}")
err := writer.Write("generated/main.go", templateOutput, options)
```

### Register Composite Writers for Templates

```go
composite := write.NewCompositeWriter()

// Source files get timestamped
timestampWriter := write.NewTimestampWriter(write.NewBaseWriter())
composite.AddWriter(
    write.ExtensionCondition(".go", ".py", ".js"),
    timestampWriter,
    100,
)

// Config files replace segments
segmentWriter := write.NewReplaceSegmentWriter(write.NewBaseWriter())
segmentWriter.SetMarkers("# BEGIN CONFIG", "# END CONFIG")
composite.AddWriter(
    write.ExtensionCondition(".yml", ".yaml", ".toml"),
    segmentWriter,
    90,
)
```

## Configuration

The package supports various configuration options through WriteOptions:

```go
options := write.WriteOptions{
    CreateDirs: true,  // Create parent directories automatically
    Backup:     true,  // Create .bak files before overwriting
    BackupDir:  "/tmp/backups", // Custom backup directory
    Overwrite:  true,  // Allow overwriting existing files
    Atomic:     true,  // Use atomic writes (write to .tmp then rename)
}
```

## File Coordination

Coordinate concurrent writes with file locking:

```go
lockManager := write.NewFileLockManager(
    write.WithLockTimeout(30 * time.Second),
    write.WithCleanupInterval(5 * time.Minute),
)

writer := write.NewCoordinatedWriter(
    write.NewBaseWriter(),
    lockManager,
    "writer-1", // owner identifier
)

err := writer.Write("/path/to/shared/file.txt", content, options)
```

### Concurrent Write Management

```go
manager := write.NewConcurrentWriteManager()

// Add multiple writers with different owners
manager.AddWriter("go-gen", write.NewBaseWriter(), "go-generator")
manager.AddWriter("js-gen", write.NewBaseWriter(), "js-generator") 

// Write using specific writer
err := manager.WriteWithWriter("go-gen", "output.go", content, options)

// Get statistics
stats := manager.GetStats()
fmt.Printf("Active locks: %d, Writers: %d\n", stats.ActiveLocks, stats.WriterCount)
```

## Writer Types

The package supports multiple writer types for different scenarios:

| Writer Type | Description |
|-------------|-------------|
| `BaseWriter` | Standard file writer with backup, atomic, and directory creation |
| `CompositeWriter` | Routes files to different writers based on conditions |
| `ChainedWriter` | Executes multiple writers in sequence |
| `ConditionalWriter` | Chooses between primary and secondary writer based on condition |
| `RetryWriter` | Retries failed writes with configurable backoff |
| `ValidatingWriter` | Validates content before writing |
| `CoordinatedWriter` | Thread-safe writer with file locking |

### Writer Creation Examples

```go
// Basic writer
base := write.NewBaseWriter()

// Skip existing files
skipWriter := write.NewSkipIfExistsWriter(base)

// Replace marked segments
segmentWriter := write.NewReplaceSegmentWriter(base)
segmentWriter.SetMarkers("// BEGIN GEN", "// END GEN")

// Add timestamps
timestampWriter := write.NewTimestampWriter(base)
timestampWriter.SetFormat("// Generated: %s\n\n")

// Dry run (track changes without writing)
dryRun := write.NewDryRunWriter()

// Logging wrapper
logged := write.NewLoggingWriter(base, func(msg string, args ...any) {
    log.Printf(msg, args...)
})
```

## Writer Conditions

### Available Conditions

| Condition | Description | Example |
|-----------|-------------|---------|
| `ExtensionCondition` | Match file extensions | `ExtensionCondition(".go", ".gogen")` |
| `PrefixCondition` | Match path prefixes | `PrefixCondition("src/", "internal/")` |
| `PatternCondition` | Match glob patterns | `PatternCondition("*.test.go", "mock_*.go")` |
| `AndCondition` | All conditions must match | `AndCondition(extGo, prefixSrc)` |
| `OrCondition` | Any condition must match | `OrCondition(extGo, extJs)` |
| `NotCondition` | Condition must not match | `NotCondition(testFiles)` |

### Condition Usage Examples

```go
// Complex condition combinations
sourceFiles := write.AndCondition(
    write.ExtensionCondition(".go"),
    write.NotCondition(write.PatternCondition("*_test.go")),
)

configFiles := write.OrCondition(
    write.ExtensionCondition(".yml", ".yaml"),
    write.ExtensionCondition(".toml", ".json"),
)

// Use in composite writer
composite.AddWriter(sourceFiles, timestampWriter, 100)
composite.AddWriter(configFiles, segmentWriter, 90)
```

## Advanced Operations

### Segment Replacement

```go
writer := write.NewReplaceSegmentWriter(write.NewBaseWriter())
writer.SetMarkers("<!-- BEGIN GENERATED -->", "<!-- END GENERATED -->")

// Will replace content between markers, or append if markers don't exist
content := []byte("<div>Generated HTML content</div>")
err := writer.Write("template.html", content, options)
```

### Atomic Operations with Retry

```go
retryWriter := write.NewRetryWriter(
    write.NewBaseWriter(),
    3, // max retries
)
retryWriter.SetBackoff(func(attempt int) time.Duration {
    return time.Duration(attempt*attempt) * 100 * time.Millisecond
})

options.Atomic = true // Use atomic writes
err := retryWriter.Write(path, content, options)
```

### Content Validation

```go
validator := func(path string, content []byte) error {
    if strings.Contains(string(content), "TODO") {
        return fmt.Errorf("content contains TODO markers")
    }
    return nil
}

validatingWriter := write.NewValidatingWriter(
    write.NewBaseWriter(),
    validator,
)

err := validatingWriter.Write(path, content, options)
```

## Performance Considerations

### Optimization Tips

1. **Use Atomic Writes Judiciously**: Atomic writes are safer but slower for large files
2. **Batch Operations**: Use composite writers to reduce I/O operations
3. **Efficient Condition Checking**: Order conditions by frequency (most common first)
4. **Lock Management**: Use appropriate timeout and cleanup intervals for file locks
5. **Content Diffing**: Use `NeedsWrite()` to avoid unnecessary writes

### Performance Settings

```go
// Optimized for high-throughput scenarios
options := write.WriteOptions{
    CreateDirs: true,
    Backup:     false, // Disable for better performance
    Overwrite:  true,
    Atomic:     false, // Direct writes for speed
}

// Efficient lock settings
lockManager := write.NewFileLockManager(
    write.WithLockTimeout(5 * time.Second),    // Shorter timeout
    write.WithCleanupInterval(1 * time.Minute), // Frequent cleanup
)
```

### Memory Management

```go
// Check if write is needed to avoid unnecessary processing
needs, err := writer.NeedsWrite(path, content)
if err != nil {
    return err
}
if !needs {
    return nil // Skip unnecessary write
}

// Use dry run to preview changes
dryRun := write.NewDryRunWriter()
err = dryRun.Write(path, content, options)
changes := dryRun.GetChanges()
```

## Security Notes

### File System Security

- Atomic writes prevent partial file corruption during failures
- Backup functionality preserves original files before modification
- Path validation prevents directory traversal attacks
- File locking prevents concurrent modification conflicts

### Security Best Practices

1. **Validate File Paths**: Ensure paths don't escape intended directories
2. **Use Appropriate Permissions**: Files are created with 0644 permissions by default
3. **Secure Backup Storage**: Store backups in secure locations
4. **Lock Timeouts**: Use reasonable timeouts to prevent indefinite blocking
5. **Content Validation**: Validate generated content before writing

### Path Security

```go
// Path validation example
validator := func(path string, content []byte) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }
    
    // Ensure path is within allowed directory
    allowedDir := "/app/generated/"
    if !strings.HasPrefix(absPath, allowedDir) {
        return fmt.Errorf("path outside allowed directory: %s", path)
    }
    
    return nil
}

secureWriter := write.NewValidatingWriter(writer, validator)
```

## Thread Safety

All public functions and types in this package are thread-safe and can be used concurrently from multiple goroutines.

### Concurrent Usage Examples

```go
// Safe concurrent writer usage
manager := write.NewConcurrentWriteManager()

var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        
        writerName := fmt.Sprintf("worker-%d", id)
        manager.AddWriter(writerName, write.NewBaseWriter(), writerName)
        
        content := []byte(fmt.Sprintf("Content from worker %d", id))
        path := fmt.Sprintf("output/worker-%d.txt", id)
        
        err := manager.WriteWithWriter(writerName, path, content, options)
        if err != nil {
            log.Printf("Worker %d failed: %v", id, err)
        }
    }(i)
}
wg.Wait()
```

## Advanced Features

### Lock Information and Statistics

```go
lockManager := write.NewFileLockManager()

// Get active locks
locks := lockManager.GetActiveLocks()
for _, lock := range locks {
    fmt.Printf("Lock: %s owned by %s (refs: %d)\n", 
        lock.Path, lock.Owner, lock.Refs)
}

// Check if specific file is locked
if lockManager.IsLocked("/path/to/file.txt") {
    info, err := lockManager.GetLockInfo("/path/to/file.txt")
    if err == nil {
        fmt.Printf("File locked by: %s\n", info.Owner)
    }
}
```

### Custom Content Filters

```go
// Remove sensitive information filter
sensitiveFilter := func(path string, content []byte) ([]byte, error) {
    // Remove API keys, passwords, etc.
    filtered := regexp.MustCompile(`api_key:\s*"[^"]*"`).
        ReplaceAll(content, []byte(`api_key: "REDACTED"`))
    
    return filtered, nil
}

filterWriter := write.NewFilterWriter(
    write.NewBaseWriter(),
    sensitiveFilter,
)

err := filterWriter.Write("config.yml", content, options)
```

### Template Content Wrapping

```go
header := `// Code generated by gogenkit. DO NOT EDIT.
// Source: template.go.tmpl

package main

import "fmt"

`

footer := `
// End of generated code
`

templateWriter := write.NewTemplateWriter(
    write.NewBaseWriter(),
    header,
    footer,
)

err := templateWriter.Write("generated.go", content, options)
```

## Integration Examples

### With Template Engines

```go
func generateFiles(templates map[string]string, data any) error {
    composite := write.NewCompositeWriter()
    
    // Go files get timestamps and validation
    goValidator := func(path string, content []byte) error {
        if !bytes.HasPrefix(content, []byte("package ")) {
            return fmt.Errorf("Go file must start with package declaration")
        }
        return nil
    }
    
    goWriter := write.NewValidatingWriter(
        write.NewTimestampWriter(write.NewBaseWriter()),
        goValidator,
    )
    
    composite.AddWriter(
        write.ExtensionCondition(".go"),
        goWriter,
        100,
    )
    
    // Config files use segment replacement
    configWriter := write.NewReplaceSegmentWriter(write.NewBaseWriter())
    composite.AddWriter(
        write.ExtensionCondition(".yml", ".toml", ".json"),
        configWriter,
        90,
    )
    
    options := write.WriteOptions{
        CreateDirs: true,
        Backup:     true,
        Atomic:     true,
    }
    
    for templatePath, templateContent := range templates {
        outputPath := strings.TrimSuffix(templatePath, ".tmpl")
        
        // Execute template (pseudo-code)
        rendered, err := executeTemplate(templateContent, data)
        if err != nil {
            return fmt.Errorf("template execution failed: %w", err)
        }
        
        if err := composite.Write(outputPath, rendered, options); err != nil {
            return fmt.Errorf("write failed for %s: %w", outputPath, err)
        }
    }
    
    return nil
}
```

### With Build Systems

```go
func buildPipeline() error {
    // Create coordinated writer for concurrent builds
    lockManager := write.NewFileLockManager()
    defer lockManager.Stop()
    
    manager := write.NewConcurrentWriteManager()
    
    // Add different generators
    manager.AddWriter("proto-gen", 
        write.NewTimestampWriter(write.NewBaseWriter()), 
        "protoc-generator")
        
    manager.AddWriter("swagger-gen", 
        write.NewSkipIfExistsWriter(write.NewBaseWriter()), 
        "swagger-generator")
    
    // Concurrent generation
    var wg sync.WaitGroup
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        generateProtoFiles(manager)
    }()
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        generateSwaggerFiles(manager)
    }()
    
    wg.Wait()
    
    // Print statistics
    stats := manager.GetStats()
    fmt.Printf("Build completed: %d active locks, %d writers\n",
        stats.ActiveLocks, stats.WriterCount)
    
    return nil
}
```

### With CLI Applications

```go
func main() {
    var dryRun bool
    flag.BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")
    flag.Parse()
    
    var writer write.Writer
    
    if dryRun {
        dryRunWriter := write.NewDryRunWriter()
        defer func() {
            changes := dryRunWriter.GetChanges()
            fmt.Printf("Would make %d changes:\n", len(changes))
            for _, change := range changes {
                fmt.Printf("  %s: %s (%d bytes)\n", 
                    change.Action, change.Path, change.Size)
            }
        }()
        writer = dryRunWriter
    } else {
        writer = write.NewLoggingWriter(
            write.NewBaseWriter(),
            func(msg string, args ...any) {
                log.Printf(msg, args...)
            },
        )
    }
    
    options := write.WriteOptions{
        CreateDirs: true,
        Backup:     !dryRun, // No backups for dry runs
        Atomic:     true,
    }
    
    // Process files
    files := []struct {
        path    string
        content []byte
    }{
        {"output/main.go", []byte("package main\n\nfunc main() {}")},
        {"output/config.yml", []byte("version: 1.0")},
    }
    
    for _, file := range files {
        if err := writer.Write(file.path, file.content, options); err != nil {
            log.Fatalf("Failed to write %s: %v", file.path, err)
        }
    }
}
```
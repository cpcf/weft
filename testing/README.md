# Testing Package

Package testing provides testing utilities, performance benchmarking, memory analysis, and snapshot testing capabilities for weft. It offers configurable benchmark runners, memory profilers, filesystem mocks, snapshot management, and a rich set of testing helper functions.

## Architecture

The testing package is organized into four main components:

- **Benchmark Testing** (`benchmarks.go`): Performance testing with memory profiling and result comparison
- **Memory Testing** (`memory.go`): In-memory filesystem implementation for testing
- **Mock Testing** (`mocks.go`): Mock filesystem, logger, and renderer implementations
- **Snapshot Testing** (`snapshots.go`): Golden file testing with diff generation and update modes

## Basic Usage

### Create a Benchmark Runner

```go
runner := testing.NewBenchmarkRunner()
runner.SetMinIterations(100)
runner.SetMaxIterations(10000)
runner.SetMinTime(5 * time.Second)
```

### Run Performance Benchmarks

```go
result := runner.Benchmark("template_render", func() error {
    _, err := template.Execute(data)
    return err
})

fmt.Printf("Average time: %v\n", result.AvgDuration)
fmt.Printf("Memory per op: %d bytes\n", result.BytesPerOp)
```

## Memory Testing

Create in-memory filesystems for testing template operations:

```go
memFS := testing.NewMemoryFS()
memFS.WriteFile("templates/user.tmpl", []byte("Hello {{.Name}}!"))

// Use with template engine
file, err := memFS.Open("templates/user.tmpl")
```

### Memory Filesystem Features

- **File Operations**: Read, write, and stat files in memory
- **Directory Support**: Create and navigate directory structures
- **fs.FS Interface**: Compatible with Go's filesystem interfaces
- **Concurrent Safe**: Thread-safe operations for parallel testing

## Configuration

The package supports various configuration options:

```go
// Benchmark configuration
runner := testing.NewBenchmarkRunner()
runner.SetWarmupIterations(5)    // Warmup runs before timing
runner.SetMinIterations(50)      // Minimum benchmark iterations
runner.SetMaxIterations(100000)  // Maximum benchmark iterations
runner.SetMinTime(2 * time.Second) // Minimum benchmark duration

// Snapshot configuration
manager := testing.NewSnapshotManager("testdata/snapshots", false)
manager.SetUpdateMode(true)      // Enable snapshot updates
```

## Snapshot Testing

Test template output against saved snapshots:

```go
manager := testing.NewSnapshotManager("testdata/snapshots", false)

// Assert output matches snapshot
output := executeTemplate("user.tmpl", userData)
err := manager.AssertSnapshot("user_template", output)
if err != nil {
    t.Error(err)
}
```

### Snapshot Features

- **Golden File Testing**: Compare output against saved reference files
- **Update Mode**: Automatically update snapshots when content changes
- **Diff Generation**: Detailed line-by-line difference reporting
- **Orphan Cleanup**: Remove unused snapshot files

## Mock Testing

Use mock implementations for isolated testing:

```go
// Mock filesystem
mockFS := testing.NewMockFS()
mockFS.AddFile("template.tmpl", "Hello {{.Name}}!")

// Mock logger
mockLogger := testing.NewMockLogger()
// ... use logger in code
logs := mockLogger.GetLogs()

// Mock renderer
mockRenderer := testing.NewMockRenderer()
mockRenderer.SetRenderFunc(func(path string, data any) (string, error) {
    return "mocked output", nil
})
```

## Benchmark Results

The package provides detailed benchmark analysis:

| Metric | Description |
|--------|-------------|
| `Iterations` | Number of benchmark runs performed |
| `AvgDuration` | Average execution time per iteration |
| `MinDuration` | Fastest single iteration time |
| `MaxDuration` | Slowest single iteration time |
| `MemoryUsed` | Total memory allocated during benchmark |
| `AllocsPerOp` | Memory allocations per operation |
| `BytesPerOp` | Bytes allocated per operation |

### Performance Comparisons

```go
// Compare two benchmark results
comparison, err := runner.Compare("benchmark1", "benchmark2")
if err == nil {
    fmt.Println(comparison.String())
    // Output: "benchmark1 is 2.34x faster than benchmark2"
}
```

## Memory Profiling

Track memory usage and system resources:

```go
profiler := testing.NewPerformanceProfiler()
profiler.StartProfile("template_execution")

// ... execute templates

profiler.EndProfile("template_execution")
profile, _ := profiler.GetProfile("template_execution")
fmt.Printf("Memory delta: %d bytes\n", profile.MemoryProfile.HeapAllocDelta)
```

### Profiling Metrics

| Field | Description | Usage |
|-------|-------------|-------|
| `AllocsDelta` | Change in allocation count | Memory allocation tracking |
| `HeapAllocDelta` | Heap memory change | Memory usage analysis |
| `SysDelta` | System memory change | Overall memory impact |
| `Duration` | Total execution time | Performance measurement |

## Mock Components

### Available Mocks

| Component | Description | Example |
|-----------|-------------|---------|
| `MockFS` | Mock filesystem implementation | `mockFS.AddFile("test.txt", "content")` |
| `MockLogger` | Mock logger with call tracking | `mockLogger.Info("message")` |
| `MockRenderer` | Mock template renderer | `mockRenderer.Render("tmpl", data)` |

### Mock Usage Examples

```go
// Mock filesystem operations
mockFS := testing.NewMockFS()
mockFS.AddFile("config.json", `{"key": "value"}`)
content, _ := mockFS.ReadFile("config.json")

// Mock logger verification
mockLogger := testing.NewMockLogger()
mockLogger.Error("test error")
errorLogs := mockLogger.GetLogsByLevel("ERROR")
assert.Equal(t, 1, len(errorLogs))

// Mock renderer behavior
mockRenderer := testing.NewMockRenderer()
result, _ := mockRenderer.Render("template.tmpl", map[string]string{"name": "test"})
assert.True(t, mockRenderer.WasCalled("template.tmpl"))
```

## Performance Considerations

### Optimization Tips

1. **Benchmark Warmup**: Use warmup iterations to eliminate JIT overhead
2. **Memory Profiling**: Monitor allocations to identify memory hotspots
3. **Snapshot Efficiency**: Use update mode judiciously to avoid unnecessary file I/O
4. **Mock Performance**: Mocks are lightweight but still add overhead in tight loops

### Performance Settings

```go
// Production-optimized benchmark settings
runner := testing.NewBenchmarkRunner()
runner.SetWarmupIterations(2)  // Minimal warmup
runner.SetMinIterations(10)    // Fewer minimum iterations
runner.SetMaxIterations(1000)  // Lower maximum for faster results
runner.SetMinTime(500 * time.Millisecond) // Shorter minimum time
```

### Memory Management

```go
// Clear benchmark results to free memory
runner.Clear()

// Clear mock state
mockLogger.Clear()
mockRenderer.Clear()

// Clear profiler data
profiler.Clear()
```

## Security Notes

### Test Data Protection

- Mock components isolate test data from production systems
- In-memory filesystems prevent accidental file system modifications
- Snapshot directories should be properly secured in CI/CD environments

### Security Best Practices

1. **Isolate test environments** from production data
2. **Sanitize test data** to avoid exposing sensitive information
3. **Use temporary directories** for snapshot storage during testing
4. **Regular cleanup** of test artifacts and temporary files

### Sensitive Data Handling

```go
// Avoid storing sensitive data in snapshots
sensitiveData := map[string]string{
    "password": "secret123",
    "api_key":  "key_abc123",
}

// Use mock data instead
mockData := map[string]string{
    "password": "[REDACTED]",
    "api_key":  "[REDACTED]",
}
```

## Thread Safety

All public functions and types in this package are thread-safe and can be used concurrently from multiple goroutines.

### Concurrent Usage Examples

```go
// Safe concurrent benchmark runs
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        runner.Benchmark(fmt.Sprintf("test_%d", id), testFunc)
    }(i)
}
wg.Wait()

// Safe concurrent mock usage
go func() {
    mockLogger.Info("Goroutine 1 message")
}()

go func() {
    mockLogger.Error("Goroutine 2 message")
}()
```

## Advanced Features

### Custom Benchmark Analysis

```go
runner := testing.NewBenchmarkRunner()
// ... run benchmarks

report := runner.GenerateReport()
fmt.Printf("Success rate: %.2f%%\n", 
    float64(report.Summary.SuccessfulRuns)/float64(report.Summary.TotalBenchmarks)*100)

// Export detailed results
err := runner.ExportReport("benchmark_results.json")
```

### Snapshot Test Suites

```go
suite := testing.NewSnapshotTestSuite("testdata/snapshots", false)
suite.AddTestCase("user_profile", "templates/user.tmpl", map[string]any{
    "Name":  "John Doe",
    "Email": "john@example.com",
})

// Load test cases from file
err := suite.LoadTestCases("testdata/test_cases.json")

// Save test cases for reuse
err = suite.SaveTestCases("testdata/updated_cases.json")
```

### Performance Profiling

```go
profiler := testing.NewPerformanceProfiler()

// Profile multiple operations
profiler.StartProfile("template_parsing")
// ... parsing operations
profiler.EndProfile("template_parsing")

profiler.StartProfile("template_rendering")  
// ... rendering operations
profiler.EndProfile("template_rendering")

// Analyze all profiles
profiles := profiler.GetAllProfiles()
for name, profile := range profiles {
    fmt.Printf("%s: %v (memory: %d bytes)\n", 
        name, profile.Duration, profile.MemoryProfile.HeapAllocDelta)
}
```

## Integration Examples

### With Standard Testing

```go
func TestTemplateRendering(t *testing.T) {
    runner := testing.NewBenchmarkRunner()
    
    result := runner.Benchmark("user_template", func() error {
        return renderUserTemplate(userData)
    })
    
    if !result.Success {
        t.Errorf("Benchmark failed: %s", result.Error)
    }
    
    if result.AvgDuration > 10*time.Millisecond {
        t.Errorf("Template rendering too slow: %v", result.AvgDuration)
    }
}
```

### With Snapshot Testing

```go
func TestTemplateSnapshots(t *testing.T) {
    manager := testing.NewSnapshotManager("testdata/snapshots", 
        os.Getenv("UPDATE_SNAPSHOTS") == "true")
    
    testCases := []struct {
        name string
        tmpl string  
        data map[string]any
    }{
        {"user_card", "templates/user_card.tmpl", userData},
        {"product_list", "templates/products.tmpl", productData},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            output, err := renderTemplate(tc.tmpl, tc.data)
            if err != nil {
                t.Fatal(err)
            }
            
            if err := manager.AssertSnapshot(tc.name, output); err != nil {
                t.Error(err)
            }
        })
    }
}
```

### With Mock Testing

```go
func TestWithMocks(t *testing.T) {
    // Setup mocks
    mockFS := testing.NewMockFS()
    mockFS.AddFile("template.tmpl", "Hello {{.Name}}!")
    
    mockLogger := testing.NewMockLogger()
    mockRenderer := testing.NewMockRenderer()
    
    // Configure mock behavior
    mockRenderer.SetRenderFunc(func(path string, data any) (string, error) {
        if path == "error.tmpl" {
            return "", fmt.Errorf("template error")
        }
        return fmt.Sprintf("rendered: %s", path), nil
    })
    
    // Test with mocks
    result, err := mockRenderer.Render("template.tmpl", map[string]string{"Name": "Test"})
    
    // Verify mock interactions
    assert.NoError(t, err)
    assert.True(t, mockRenderer.WasCalled("template.tmpl"))
    assert.Equal(t, 1, mockRenderer.GetCallCount())
}
```
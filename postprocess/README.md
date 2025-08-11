# Post-Processing System

The post-processing system provides a generic framework for transforming generated content after template rendering but before writing to disk.

## Overview

Post-processors allow you to:
- Format and organize code (imports, whitespace)
- Add generated file headers
- Apply custom transformations
- Validate generated content
- Perform file-type specific optimizations

## Basic Usage

```go
import (
    "github.com/cpcf/gogenkit/engine"
    "github.com/cpcf/gogenkit/processors"
)

// Create engine and add processors
eng := engine.New()

// Add built-in processors
eng.AddPostProcessor(processors.NewGoImports())                // Fix Go imports
eng.AddPostProcessor(processors.NewTrimWhitespace())           // Clean whitespace  
eng.AddPostProcessor(processors.NewAddGeneratedHeader("myapp", ".go", ".js")) // Add headers

// Generate files
err := eng.RenderDir(ctx, "templates", data)
```

## Built-in Processors

### Go Imports (`processors.NewGoImports()`)
Fixes import statements and formats Go code using `goimports`:

```go
processor := processors.NewGoImports()
// Customize options
processor.TabWidth = 4
processor.TabIndent = false
processor.AllErrors = true
```

### Trim Whitespace (`processors.NewTrimWhitespace()`)
Removes trailing whitespace from all lines:

```go
eng.AddPostProcessor(processors.NewTrimWhitespace())
```

### Add Generated Header (`processors.NewAddGeneratedHeader()`)
Adds "Code generated" headers to files:

```go
// Add header to specific file types
eng.AddPostProcessor(processors.NewAddGeneratedHeader("myapp", ".go", ".java"))

// Add header to all files
eng.AddPostProcessor(processors.NewAddGeneratedHeader("myapp"))
```

### Regex Replace (`processors.NewRegexReplace()`)
Apply regex transformations:

```go
// Replace TODO comments with DONE
processor, err := processors.NewRegexReplace(`TODO: (.+)`, "DONE: $1")
if err != nil {
    log.Fatal(err)
}
eng.AddPostProcessor(processor)

// Limit to specific file types
processor.WithFilePattern(`\.go$`)
```

## Custom Processors

Implement the `postprocess.Processor` interface:

```go
type CustomProcessor struct{}

func (p *CustomProcessor) ProcessContent(filePath string, content []byte) ([]byte, error) {
    // Apply custom transformation
    if strings.HasSuffix(filePath, ".go") {
        // Add custom Go code transformations
        transformed := doCustomTransform(content)
        return transformed, nil
    }
    return content, nil // Leave other files unchanged
}

// Add to engine
eng.AddPostProcessor(&CustomProcessor{})
```

## Function-based Processors

For simple transformations, use function processors:

```go
eng.AddPostProcessorFunc(func(filePath string, content []byte) ([]byte, error) {
    // Convert line endings to Unix style
    return bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n")), nil
})
```

## Advanced Usage

### Chaining Multiple Processors
Processors run in the order they're added:

```go
eng.AddPostProcessor(processors.NewGoImports())           // 1. Fix imports first
eng.AddPostProcessor(processors.NewTrimWhitespace())      // 2. Clean whitespace
eng.AddPostProcessor(processors.NewAddGeneratedHeader("myapp", ".go")) // 3. Add header last
```

### Error Handling
Processors that fail log a warning but don't stop generation:

```go
// This processor might fail but won't break the build
eng.AddPostProcessor(processors.NewGoImports())
```

### File-type Specific Processing

```go
// Only process Go files
eng.AddPostProcessorFunc(func(filePath string, content []byte) ([]byte, error) {
    if !strings.HasSuffix(filePath, ".go") {
        return content, nil
    }
    // Go-specific processing here
    return processGoFile(content), nil
})
```

## Examples in Other Languages

### Java Processor
```go
type JavaFormatter struct{}

func (j *JavaFormatter) ProcessContent(filePath string, content []byte) ([]byte, error) {
    if !strings.HasSuffix(filePath, ".java") {
        return content, nil
    }
    // Format Java code using google-java-format or similar
    return formatJavaCode(content), nil
}
```

### Python Processor  
```go
type PythonFormatter struct{}

func (p *PythonFormatter) ProcessContent(filePath string, content []byte) ([]byte, error) {
    if !strings.HasSuffix(filePath, ".py") {
        return content, nil
    }
    // Format Python code using black or autopep8
    return formatPythonCode(content), nil
}
```

## Best Practices

1. **Order Matters**: Add processors in logical order (format → clean → annotate)
2. **File Type Checking**: Always check file extensions before processing
3. **Error Handling**: Return original content on errors rather than failing
4. **Performance**: Keep processors lightweight for large codebases
5. **Idempotency**: Ensure processors can run multiple times safely

## Integration with CI/CD

Post-processors integrate seamlessly with build pipelines:

```go
// In your generator
func main() {
    eng := engine.New()
    
    // Add standard processors
    eng.AddPostProcessor(processors.NewGoImports())
    eng.AddPostProcessor(processors.NewTrimWhitespace())
    eng.AddPostProcessor(processors.NewAddGeneratedHeader(os.Args[0]))
    
    // Add custom validation
    eng.AddPostProcessorFunc(validateGeneratedFiles)
    
    if err := eng.RenderDir(ctx, "templates", data); err != nil {
        log.Fatal(err)
    }
}
```

This ensures generated code is properly formatted, validated, and ready for version control.
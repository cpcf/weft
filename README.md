# gogenkit

A Go template engine for code generation with state management, debugging, and testing utilities.

## Overview

gogenkit provides a modular template processing system designed for generating code and managing output files. It includes template rendering, file state tracking, debugging capabilities, and testing utilities.

## Packages

- **[engine](engine/)** - Core template processing with caching and concurrent rendering
- **[render](render/)** - Template discovery, blocks, includes, and function registry  
- **[write](write/)** - File writing with coordination, locking, and composite operations
- **[state](state/)** - File tracking, manifest management, and cleanup operations
- **[debug](debug/)** - Debugging, error handling, and template validation
- **[testing](testing/)** - Testing utilities, benchmarks, mocks, and snapshots

## Basic Usage

```go
package main

import (
    "github.com/cpcf/gogenkit/engine"
    "github.com/cpcf/gogenkit/render"
    "github.com/cpcf/gogenkit/write"
)

func main() {
    // Create template engine
    eng := engine.New(
        engine.WithTemplateFS(templateFS),
        engine.WithOutputPath("generated/"),
    )

    // Create writers
    writer := write.NewFileWriter()
    
    // Render templates
    err := eng.RenderTemplate("template.tmpl", data, writer)
    if err != nil {
        panic(err)
    }
}
```
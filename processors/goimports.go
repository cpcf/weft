// Package processors provides built-in post-processors for common use cases.
package processors

import (
	"fmt"
	"go/format"
	"path/filepath"
	"strings"

	"golang.org/x/tools/imports"
)

// GoImports is a post-processor that fixes imports and formats Go source files.
// It uses goimports to organize imports and gofmt as a fallback.
//
// Example usage:
//
//	eng := engine.New()
//	eng.AddPostProcessor(processors.NewGoImports())
type GoImports struct {
	// TabWidth sets the tab width for formatting (default: 8)
	TabWidth int
	// TabIndent determines whether to use tabs for indentation (default: true)
	TabIndent bool
	// AllErrors determines whether to report all errors or just the first (default: false)
	AllErrors bool
	// Comments determines whether to update comments (default: true)
	Comments bool
}

// NewGoImports creates a new Go imports processor with sensible defaults.
func NewGoImports() *GoImports {
	return &GoImports{
		TabWidth:  8,
		TabIndent: true,
		AllErrors: false,
		Comments:  true,
	}
}

// ProcessContent implements the postprocess.Processor interface.
// It processes Go files through goimports and leaves other files unchanged.
func (g *GoImports) ProcessContent(filePath string, content []byte) ([]byte, error) {
	if !g.isGoFile(filePath) {
		return content, nil
	}

	// Configure goimports options
	options := &imports.Options{
		Fragment:  false,
		AllErrors: g.AllErrors,
		Comments:  g.Comments,
		TabIndent: g.TabIndent,
		TabWidth:  g.TabWidth,
	}

	// Try goimports first for full import management
	formatted, err := imports.Process(filePath, content, options)
	if err != nil {
		// Fall back to basic gofmt for syntax formatting
		formatted, fmtErr := format.Source(content)
		if fmtErr != nil {
			return nil, fmt.Errorf("failed to format Go code with goimports (%w) and gofmt (%w)", err, fmtErr)
		}
		return formatted, nil
	}

	return formatted, nil
}

// isGoFile checks if the file path represents a Go source file.
func (g *GoImports) isGoFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".go"
}

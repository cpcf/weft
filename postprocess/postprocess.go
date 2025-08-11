// Package postprocess provides a generic framework for post-processing generated content.
//
// The post-processing system allows you to apply transformations to generated files
// after template rendering but before writing to disk. This is useful for:
//
//   - Code formatting and import organization
//   - Static analysis and linting
//   - Content validation and transformation
//   - File-type specific optimizations
//
// Example usage:
//
//	import (
//		"github.com/cpcf/gogenkit/engine"
//		"github.com/cpcf/gogenkit/processors"
//	)
//
//	eng := engine.New()
//	eng.AddPostProcessor(processors.NewGoImports())
//	eng.AddPostProcessor(myCustomProcessor)
package postprocess

import "fmt"

// Processor defines the interface for content post-processors.
// Implementations should be stateless and safe for concurrent use.
type Processor interface {
	// ProcessContent processes the content of a file and returns the transformed content.
	// The filePath parameter provides context about the file being processed.
	// Processors should return the original content unchanged if they don't apply to the file type.
	ProcessContent(filePath string, content []byte) ([]byte, error)
}

// ProcessorFunc is a function adapter that implements the Processor interface.
// It allows using regular functions as processors.
type ProcessorFunc func(filePath string, content []byte) ([]byte, error)

// ProcessContent implements the Processor interface.
func (f ProcessorFunc) ProcessContent(filePath string, content []byte) ([]byte, error) {
	return f(filePath, content)
}

// Chain manages and executes multiple post-processors in sequence.
// Processors are applied in the order they were added.
type Chain struct {
	processors []Processor
}

// NewChain creates a new empty processor chain.
func NewChain() *Chain {
	return &Chain{
		processors: make([]Processor, 0),
	}
}

// Add adds a processor to the end of the chain.
func (c *Chain) Add(processor Processor) {
	c.processors = append(c.processors, processor)
}

// AddFunc adds a function as a processor to the end of the chain.
func (c *Chain) AddFunc(fn func(filePath string, content []byte) ([]byte, error)) {
	c.processors = append(c.processors, ProcessorFunc(fn))
}

// Process runs all processors in sequence on the given content.
// If any processor fails, processing stops and the error is returned.
func (c *Chain) Process(filePath string, content []byte) ([]byte, error) {
	result := content
	for i, processor := range c.processors {
		processed, err := processor.ProcessContent(filePath, result)
		if err != nil {
			return nil, fmt.Errorf("processor %d failed for %s: %w", i, filePath, err)
		}
		result = processed
	}
	return result, nil
}

// HasProcessors returns true if the chain contains any processors.
func (c *Chain) HasProcessors() bool {
	return len(c.processors) > 0
}

// Len returns the number of processors in the chain.
func (c *Chain) Len() int {
	return len(c.processors)
}

// Clear removes all processors from the chain.
func (c *Chain) Clear() {
	c.processors = c.processors[:0]
}

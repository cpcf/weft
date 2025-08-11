// Package engine provides the core template rendering and post-processing functionality.
package engine

import (
	"io"
	"log/slog"

	"github.com/cpcf/weft/postprocess"
)

type Engine struct {
	logger         *slog.Logger
	outputRoot     string
	failMode       FailureMode
	renderer       *Renderer
	cache          *TemplateCache
	postprocessors *postprocess.Chain
}

type FailureMode int

const (
	FailFast FailureMode = iota
	FailAtEnd
	BestEffort
)

func New(opts ...Option) *Engine {
	e := &Engine{
		logger:         slog.Default(),
		outputRoot:     "./out",
		failMode:       FailFast,
		cache:          NewTemplateCache(),
		postprocessors: postprocess.NewChain(),
	}

	for _, opt := range opts {
		opt(e)
	}

	e.renderer = NewRenderer(e.logger, e.cache, e.postprocessors)

	return e
}

func (e *Engine) RenderDir(ctx Context, templateDir string, data any) error {
	return e.renderer.RenderDir(ctx, e.failMode, templateDir, data)
}

func (e *Engine) SetOutput(w io.Writer) {
	// For future use with structured output
}

// AddPostProcessor adds a post-processor to the processing chain.
// Processors are applied in the order they are added.
func (e *Engine) AddPostProcessor(processor postprocess.Processor) {
	e.postprocessors.Add(processor)
}

// AddPostProcessorFunc adds a function as a post-processor to the processing chain.
// This is a convenience method for simple transformations.
func (e *Engine) AddPostProcessorFunc(fn func(filePath string, content []byte) ([]byte, error)) {
	e.postprocessors.AddFunc(fn)
}

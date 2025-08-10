package engine

import (
	"io"
	"log/slog"
)

type Engine struct {
	logger     *slog.Logger
	outputRoot string
	failMode   FailureMode
	renderer   *Renderer
	cache      *TemplateCache
}

type FailureMode int

const (
	FailFast FailureMode = iota
	FailAtEnd
	BestEffort
)

func New(opts ...Option) *Engine {
	e := &Engine{
		logger:     slog.Default(),
		outputRoot: "./out",
		failMode:   FailFast,
		cache:      NewTemplateCache(),
	}

	for _, opt := range opts {
		opt(e)
	}

	e.renderer = NewRenderer(e.logger, e.cache)

	return e
}

func (e *Engine) RenderDir(ctx Context, templateDir string, data any) error {
	return e.renderer.RenderDir(ctx, e.failMode, templateDir, data)
}

func (e *Engine) SetOutput(w io.Writer) {
	// For future use with structured output
}

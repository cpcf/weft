package engine

import "log/slog"

type Option func(*Engine)

func WithLogger(logger *slog.Logger) Option {
	return func(e *Engine) {
		e.logger = logger
	}
}

func WithOutputRoot(root string) Option {
	return func(e *Engine) {
		e.outputRoot = root
	}
}

func WithFailureMode(mode FailureMode) Option {
	return func(e *Engine) {
		e.failMode = mode
	}
}
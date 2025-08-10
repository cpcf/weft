package debug

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type DebugLevel int

const (
	LevelOff DebugLevel = iota
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
)

func (dl DebugLevel) String() string {
	switch dl {
	case LevelOff:
		return "OFF"
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARN"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	case LevelTrace:
		return "TRACE"
	default:
		return "UNKNOWN"
	}
}

type DebugMode struct {
	level          DebugLevel
	output         io.Writer
	logger         *slog.Logger
	enableProfiling bool
	enableTracing   bool
	enableMetrics   bool
	startTime      time.Time
	mu             sync.RWMutex
}

type DebugOption func(*DebugMode)

func WithLevel(level DebugLevel) DebugOption {
	return func(dm *DebugMode) {
		dm.level = level
	}
}

func WithOutput(output io.Writer) DebugOption {
	return func(dm *DebugMode) {
		dm.output = output
	}
}

func WithProfiling(enable bool) DebugOption {
	return func(dm *DebugMode) {
		dm.enableProfiling = enable
	}
}

func WithTracing(enable bool) DebugOption {
	return func(dm *DebugMode) {
		dm.enableTracing = enable
	}
}

func WithMetrics(enable bool) DebugOption {
	return func(dm *DebugMode) {
		dm.enableMetrics = enable
	}
}

func NewDebugMode(opts ...DebugOption) *DebugMode {
	dm := &DebugMode{
		level:     LevelInfo,
		output:    os.Stderr,
		startTime: time.Now(),
	}

	for _, opt := range opts {
		opt(dm)
	}

	dm.setupLogger()
	return dm
}

func (dm *DebugMode) setupLogger() {
	var level slog.Level
	switch dm.level {
	case LevelError:
		level = slog.LevelError
	case LevelWarn:
		level = slog.LevelWarn
	case LevelInfo:
		level = slog.LevelInfo
	case LevelDebug, LevelTrace:
		level = slog.LevelDebug
	default:
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: dm.level >= LevelDebug,
	}

	handler := slog.NewTextHandler(dm.output, opts)
	dm.logger = slog.New(handler)
}

func (dm *DebugMode) IsEnabled(level DebugLevel) bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.level >= level
}

func (dm *DebugMode) SetLevel(level DebugLevel) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.level = level
	dm.setupLogger()
}

func (dm *DebugMode) Error(msg string, args ...any) {
	if dm.IsEnabled(LevelError) {
		dm.logger.Error(msg, args...)
	}
}

func (dm *DebugMode) Warn(msg string, args ...any) {
	if dm.IsEnabled(LevelWarn) {
		dm.logger.Warn(msg, args...)
	}
}

func (dm *DebugMode) Info(msg string, args ...any) {
	if dm.IsEnabled(LevelInfo) {
		dm.logger.Info(msg, args...)
	}
}

func (dm *DebugMode) Debug(msg string, args ...any) {
	if dm.IsEnabled(LevelDebug) {
		dm.logger.Debug(msg, args...)
	}
}

func (dm *DebugMode) Trace(msg string, args ...any) {
	if dm.IsEnabled(LevelTrace) {
		dm.logger.Debug("[TRACE] "+msg, args...)
	}
}

func (dm *DebugMode) LogTemplateExecution(templatePath string, data any, duration time.Duration) {
	if dm.IsEnabled(LevelDebug) {
		dm.Debug("template executed",
			"path", templatePath,
			"duration", duration,
			"data_type", fmt.Sprintf("%T", data))
	}
}

func (dm *DebugMode) LogTemplateData(templatePath string, data any) {
	if dm.IsEnabled(LevelTrace) {
		dataJSON, _ := json.MarshalIndent(data, "", "  ")
		dm.Trace("template data",
			"path", templatePath,
			"data", string(dataJSON))
	}
}

func (dm *DebugMode) LogFileWrite(path string, size int, duration time.Duration) {
	if dm.IsEnabled(LevelDebug) {
		dm.Debug("file written",
			"path", path,
			"size", size,
			"duration", duration)
	}
}

func (dm *DebugMode) LogError(operation string, err error, context map[string]any) {
	if dm.IsEnabled(LevelError) {
		args := []any{"operation", operation, "error", err}
		for k, v := range context {
			args = append(args, k, v)
		}
		dm.Error("operation failed", args...)
	}
}

func (dm *DebugMode) GetStats() DebugStats {
	return DebugStats{
		Level:           dm.level,
		StartTime:       dm.startTime,
		Uptime:          time.Since(dm.startTime),
		ProfilingEnabled: dm.enableProfiling,
		TracingEnabled:  dm.enableTracing,
		MetricsEnabled:  dm.enableMetrics,
	}
}

type DebugStats struct {
	Level            DebugLevel    `json:"level"`
	StartTime        time.Time     `json:"start_time"`
	Uptime           time.Duration `json:"uptime"`
	ProfilingEnabled bool          `json:"profiling_enabled"`
	TracingEnabled   bool          `json:"tracing_enabled"`
	MetricsEnabled   bool          `json:"metrics_enabled"`
}

func (ds DebugStats) String() string {
	return fmt.Sprintf("Debug Stats: Level=%s, Uptime=%v, Profiling=%v, Tracing=%v, Metrics=%v",
		ds.Level, ds.Uptime, ds.ProfilingEnabled, ds.TracingEnabled, ds.MetricsEnabled)
}

type DebugContext struct {
	mode       *DebugMode
	operation  string
	startTime  time.Time
	attributes map[string]any
	mu         sync.RWMutex
}

func (dm *DebugMode) NewContext(operation string) *DebugContext {
	return &DebugContext{
		mode:       dm,
		operation:  operation,
		startTime:  time.Now(),
		attributes: make(map[string]any),
	}
}

func (dc *DebugContext) SetAttribute(key string, value any) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.attributes[key] = value
}

func (dc *DebugContext) GetAttribute(key string) (any, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	value, exists := dc.attributes[key]
	return value, exists
}

func (dc *DebugContext) Error(msg string, err error) {
	dc.mode.LogError(dc.operation, err, dc.attributes)
}

func (dc *DebugContext) Info(msg string, args ...any) {
	if dc.mode.IsEnabled(LevelInfo) {
		allArgs := []any{"operation", dc.operation, "duration", time.Since(dc.startTime)}
		allArgs = append(allArgs, args...)
		for k, v := range dc.attributes {
			allArgs = append(allArgs, k, v)
		}
		dc.mode.Info(msg, allArgs...)
	}
}

func (dc *DebugContext) Debug(msg string, args ...any) {
	if dc.mode.IsEnabled(LevelDebug) {
		allArgs := []any{"operation", dc.operation, "duration", time.Since(dc.startTime)}
		allArgs = append(allArgs, args...)
		for k, v := range dc.attributes {
			allArgs = append(allArgs, k, v)
		}
		dc.mode.Debug(msg, allArgs...)
	}
}

func (dc *DebugContext) Complete() {
	duration := time.Since(dc.startTime)
	dc.mode.Debug("operation completed",
		"operation", dc.operation,
		"duration", duration)
}

func (dc *DebugContext) CompleteWithError(err error) {
	duration := time.Since(dc.startTime)
	dc.mode.Error("operation failed",
		"operation", dc.operation,
		"duration", duration,
		"error", err)
}
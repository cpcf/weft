package debug

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type Config struct {
	MaxStackFrames       int `json:"max_stack_frames"`
	ErrorBufferSize      int `json:"error_buffer_size"`
	ExecutionBufferSize  int `json:"execution_buffer_size"`
	MaxStackTraceDisplay int `json:"max_stack_trace_display"`
}

func DefaultConfig() Config {
	return Config{
		MaxStackFrames:       10,
		ErrorBufferSize:      100,
		ExecutionBufferSize:  100,
		MaxStackTraceDisplay: 5,
	}
}

var globalConfig = DefaultConfig()

func SetConfig(config Config) error {
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	globalConfig = config
	return nil
}

func validateConfig(config Config) error {
	if config.MaxStackFrames < 1 || config.MaxStackFrames > 100 {
		return fmt.Errorf("MaxStackFrames must be between 1 and 100, got %d", config.MaxStackFrames)
	}
	if config.ErrorBufferSize < 1 || config.ErrorBufferSize > 10000 {
		return fmt.Errorf("ErrorBufferSize must be between 1 and 10000, got %d", config.ErrorBufferSize)
	}
	if config.ExecutionBufferSize < 1 || config.ExecutionBufferSize > 10000 {
		return fmt.Errorf("ExecutionBufferSize must be between 1 and 10000, got %d", config.ExecutionBufferSize)
	}
	if config.MaxStackTraceDisplay < 1 || config.MaxStackTraceDisplay > config.MaxStackFrames {
		return fmt.Errorf("MaxStackTraceDisplay must be between 1 and MaxStackFrames (%d), got %d", 
			config.MaxStackFrames, config.MaxStackTraceDisplay)
	}
	return nil
}

func GetConfig() Config {
	return globalConfig
}

type ErrorContext struct {
	Operation    string         `json:"operation"`
	TemplatePath string         `json:"template_path,omitempty"`
	OutputPath   string         `json:"output_path,omitempty"`
	LineNumber   int            `json:"line_number,omitempty"`
	Context      map[string]any `json:"context,omitempty"`
	Suggestions  []string       `json:"suggestions,omitempty"`
	Timestamp    time.Time      `json:"timestamp"`
	Stack        []StackFrame   `json:"stack,omitempty"`
}

type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

type EnhancedError struct {
	originalError error
	context       *ErrorContext
}

func NewEnhancedError(err error, operation string) *EnhancedError {
	if err == nil {
		return nil
	}

	context := &ErrorContext{
		Operation: operation,
		Context:   make(map[string]any),
		Timestamp: time.Now(),
	}

	context.Stack = captureStack(2)

	return &EnhancedError{
		originalError: err,
		context:       context,
	}
}

func (ee *EnhancedError) Error() string {
	return ee.originalError.Error()
}

func (ee *EnhancedError) Unwrap() error {
	return ee.originalError
}

func (ee *EnhancedError) WithTemplate(path string) *EnhancedError {
	ee.context.TemplatePath = path
	return ee
}

func (ee *EnhancedError) WithOutput(path string) *EnhancedError {
	ee.context.OutputPath = path
	return ee
}

func (ee *EnhancedError) WithLine(line int) *EnhancedError {
	ee.context.LineNumber = line
	return ee
}

func (ee *EnhancedError) WithContext(key string, value any) *EnhancedError {
	ee.context.Context[key] = value
	return ee
}

func (ee *EnhancedError) WithSuggestion(suggestion string) *EnhancedError {
	ee.context.Suggestions = append(ee.context.Suggestions, suggestion)
	return ee
}

func (ee *EnhancedError) GetContext() *ErrorContext {
	return ee.context
}

func (ee *EnhancedError) FormatDetailed() string {
	var builder strings.Builder

	// Basic error information
	ee.writeBasicInfo(&builder)

	// Optional file/location information
	ee.writeLocationInfo(&builder)

	// Context data
	ee.writeContextData(&builder)

	// Suggestions
	ee.writeSuggestions(&builder)

	// Stack trace
	ee.writeStackTrace(&builder)

	return builder.String()
}

func (ee *EnhancedError) writeBasicInfo(builder *strings.Builder) {
	builder.WriteString(fmt.Sprintf("Error: %s\n", ee.originalError.Error()))
	builder.WriteString(fmt.Sprintf("Operation: %s\n", ee.context.Operation))
	builder.WriteString(fmt.Sprintf("Timestamp: %s\n", ee.context.Timestamp.Format(time.RFC3339)))
}

func (ee *EnhancedError) writeLocationInfo(builder *strings.Builder) {
	if ee.context.TemplatePath != "" {
		builder.WriteString(fmt.Sprintf("Template: %s\n", ee.context.TemplatePath))
	}

	if ee.context.OutputPath != "" {
		builder.WriteString(fmt.Sprintf("Output: %s\n", ee.context.OutputPath))
	}

	if ee.context.LineNumber > 0 {
		builder.WriteString(fmt.Sprintf("Line: %d\n", ee.context.LineNumber))
	}
}

func (ee *EnhancedError) writeContextData(builder *strings.Builder) {
	if len(ee.context.Context) == 0 {
		return
	}

	builder.WriteString("\nContext:\n")
	keys := make([]string, 0, len(ee.context.Context))
	for k := range ee.context.Context {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("  %s: %v\n", key, ee.context.Context[key]))
	}
}

func (ee *EnhancedError) writeSuggestions(builder *strings.Builder) {
	if len(ee.context.Suggestions) == 0 {
		return
	}

	builder.WriteString("\nSuggestions:\n")
	for i, suggestion := range ee.context.Suggestions {
		builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
	}
}

func (ee *EnhancedError) writeStackTrace(builder *strings.Builder) {
	if len(ee.context.Stack) == 0 {
		return
	}

	builder.WriteString("\nStack trace:\n")
	for i, frame := range ee.context.Stack {
		if i >= globalConfig.MaxStackTraceDisplay {
			break
		}
		builder.WriteString(fmt.Sprintf("  %s:%d %s\n", frame.File, frame.Line, frame.Function))
	}
}

func captureStack(skip int) []StackFrame {
	var frames []StackFrame

	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		frames = append(frames, StackFrame{
			Function: fn.Name(),
			File:     file,
			Line:     line,
		})

		if len(frames) >= globalConfig.MaxStackFrames {
			break
		}
	}

	return frames
}

type ErrorAnalyzer struct {
	errors []EnhancedError
	mu     sync.RWMutex
}

func NewErrorAnalyzer() *ErrorAnalyzer {
	return &ErrorAnalyzer{
		errors: make([]EnhancedError, 0),
	}
}

func (ea *ErrorAnalyzer) AddError(err *EnhancedError) {
	if err == nil {
		return
	}

	ea.mu.Lock()
	defer ea.mu.Unlock()

	ea.errors = append(ea.errors, *err)

	if len(ea.errors) > globalConfig.ErrorBufferSize {
		ea.errors = ea.errors[1:]
	}
}

func (ea *ErrorAnalyzer) GetErrors() []EnhancedError {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	errors := make([]EnhancedError, len(ea.errors))
	copy(errors, ea.errors)
	return errors
}

func (ea *ErrorAnalyzer) GetErrorsByOperation(operation string) []EnhancedError {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	var filtered []EnhancedError
	for _, err := range ea.errors {
		if err.context.Operation == operation {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

func (ea *ErrorAnalyzer) GetErrorsByTemplate(templatePath string) []EnhancedError {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	var filtered []EnhancedError
	for _, err := range ea.errors {
		if err.context.TemplatePath == templatePath {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

func (ea *ErrorAnalyzer) GetStatistics() ErrorStatistics {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	stats := ErrorStatistics{
		TotalErrors:    len(ea.errors),
		OperationStats: make(map[string]int),
		TemplateStats:  make(map[string]int),
		TimeRange:      TimeRange{},
	}

	if len(ea.errors) == 0 {
		return stats
	}

	stats.TimeRange.Start = ea.errors[0].context.Timestamp
	stats.TimeRange.End = ea.errors[0].context.Timestamp

	for _, err := range ea.errors {
		stats.OperationStats[err.context.Operation]++

		if err.context.TemplatePath != "" {
			stats.TemplateStats[err.context.TemplatePath]++
		}

		if err.context.Timestamp.Before(stats.TimeRange.Start) {
			stats.TimeRange.Start = err.context.Timestamp
		}
		if err.context.Timestamp.After(stats.TimeRange.End) {
			stats.TimeRange.End = err.context.Timestamp
		}
	}

	return stats
}

func (ea *ErrorAnalyzer) Clear() {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.errors = make([]EnhancedError, 0)
}

type ErrorStatistics struct {
	TotalErrors    int            `json:"total_errors"`
	OperationStats map[string]int `json:"operation_stats"`
	TemplateStats  map[string]int `json:"template_stats"`
	TimeRange      TimeRange      `json:"time_range"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func (es ErrorStatistics) String() string {
	if es.TotalErrors == 0 {
		return "No errors recorded"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Total errors: %d\n", es.TotalErrors))

	if !es.TimeRange.Start.IsZero() && !es.TimeRange.End.IsZero() {
		duration := es.TimeRange.End.Sub(es.TimeRange.Start)
		builder.WriteString(fmt.Sprintf("Time range: %s to %s (%v)\n",
			es.TimeRange.Start.Format("15:04:05"),
			es.TimeRange.End.Format("15:04:05"),
			duration))
	}

	if len(es.OperationStats) > 0 {
		builder.WriteString("\nErrors by operation:\n")
		for op, count := range es.OperationStats {
			builder.WriteString(fmt.Sprintf("  %s: %d\n", op, count))
		}
	}

	if len(es.TemplateStats) > 0 {
		builder.WriteString("\nErrors by template:\n")
		for tmpl, count := range es.TemplateStats {
			builder.WriteString(fmt.Sprintf("  %s: %d\n", tmpl, count))
		}
	}

	return builder.String()
}

func SuggestTemplateErrors(err error, templatePath string) []string {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())
	var suggestions []string

	if strings.Contains(errMsg, "no such file") {
		suggestions = append(suggestions, fmt.Sprintf("Check if template file exists: %s", templatePath))
		suggestions = append(suggestions, "Verify the template directory path is correct")
	}

	if strings.Contains(errMsg, "parse") || strings.Contains(errMsg, "syntax") {
		suggestions = append(suggestions, "Check template syntax for unclosed braces {{ }}")
		suggestions = append(suggestions, "Verify function names are spelled correctly")
		suggestions = append(suggestions, "Check for missing quotes around string values")
	}

	if strings.Contains(errMsg, "undefined") || strings.Contains(errMsg, "function") {
		suggestions = append(suggestions, "Check if the function is available in the template function map")
		suggestions = append(suggestions, "Verify the function name spelling")
	}

	if strings.Contains(errMsg, "nil pointer") {
		suggestions = append(suggestions, "Check if template data contains nil values")
		suggestions = append(suggestions, "Use conditional checks like {{ if .Field }}...{{ end }}")
	}

	if strings.Contains(errMsg, "permission") {
		suggestions = append(suggestions, "Check file/directory permissions")
		suggestions = append(suggestions, "Ensure the output directory is writable")
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Check the template syntax and data structure")
		suggestions = append(suggestions, "Enable debug mode for more detailed error information")
	}

	return suggestions
}

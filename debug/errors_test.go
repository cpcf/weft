package debug

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewEnhancedError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		wantNil   bool
	}{
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "test",
			wantNil:   true,
		},
		{
			name:      "valid error creates enhanced error",
			err:       errors.New("test error"),
			operation: "template_parse",
			wantNil:   false,
		},
		{
			name:      "empty operation still creates enhanced error",
			err:       errors.New("test error"),
			operation: "",
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewEnhancedError(tt.err, tt.operation)

			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil enhanced error")
			}

			if result.originalError != tt.err {
				t.Errorf("Expected original error %v, got %v", tt.err, result.originalError)
			}

			if result.context.Operation != tt.operation {
				t.Errorf("Expected operation %s, got %s", tt.operation, result.context.Operation)
			}

			if result.context.Context == nil {
				t.Error("Expected context map to be initialized")
			}

			if result.context.Timestamp.IsZero() {
				t.Error("Expected timestamp to be set")
			}

			if len(result.context.Stack) == 0 {
				t.Error("Expected stack trace to be captured")
			}
		})
	}
}

func TestEnhancedError_Error(t *testing.T) {
	originalErr := errors.New("original error message")
	ee := NewEnhancedError(originalErr, "test")

	if ee.Error() != originalErr.Error() {
		t.Errorf("Expected %s, got %s", originalErr.Error(), ee.Error())
	}
}

func TestEnhancedError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	ee := NewEnhancedError(originalErr, "test")

	if ee.Unwrap() != originalErr {
		t.Errorf("Expected %v, got %v", originalErr, ee.Unwrap())
	}
}

func TestEnhancedError_WithTemplate(t *testing.T) {
	ee := NewEnhancedError(errors.New("test"), "operation")
	templatePath := "/path/to/template.tmpl"

	result := ee.WithTemplate(templatePath)

	if result != ee {
		t.Error("WithTemplate should return the same instance for chaining")
	}

	if ee.context.TemplatePath != templatePath {
		t.Errorf("Expected template path %s, got %s", templatePath, ee.context.TemplatePath)
	}
}

func TestEnhancedError_WithOutput(t *testing.T) {
	ee := NewEnhancedError(errors.New("test"), "operation")
	outputPath := "/path/to/output.txt"

	result := ee.WithOutput(outputPath)

	if result != ee {
		t.Error("WithOutput should return the same instance for chaining")
	}

	if ee.context.OutputPath != outputPath {
		t.Errorf("Expected output path %s, got %s", outputPath, ee.context.OutputPath)
	}
}

func TestEnhancedError_WithLine(t *testing.T) {
	ee := NewEnhancedError(errors.New("test"), "operation")
	lineNumber := 42

	result := ee.WithLine(lineNumber)

	if result != ee {
		t.Error("WithLine should return the same instance for chaining")
	}

	if ee.context.LineNumber != lineNumber {
		t.Errorf("Expected line number %d, got %d", lineNumber, ee.context.LineNumber)
	}
}

func TestEnhancedError_WithContext(t *testing.T) {
	ee := NewEnhancedError(errors.New("test"), "operation")
	key := "testKey"
	value := "testValue"

	result := ee.WithContext(key, value)

	if result != ee {
		t.Error("WithContext should return the same instance for chaining")
	}

	if ee.context.Context[key] != value {
		t.Errorf("Expected context[%s] = %v, got %v", key, value, ee.context.Context[key])
	}
}

func TestEnhancedError_WithSuggestion(t *testing.T) {
	ee := NewEnhancedError(errors.New("test"), "operation")
	suggestion1 := "First suggestion"
	suggestion2 := "Second suggestion"

	result1 := ee.WithSuggestion(suggestion1)
	result2 := ee.WithSuggestion(suggestion2)

	if result1 != ee || result2 != ee {
		t.Error("WithSuggestion should return the same instance for chaining")
	}

	if len(ee.context.Suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(ee.context.Suggestions))
	}

	if ee.context.Suggestions[0] != suggestion1 {
		t.Errorf("Expected first suggestion %s, got %s", suggestion1, ee.context.Suggestions[0])
	}

	if ee.context.Suggestions[1] != suggestion2 {
		t.Errorf("Expected second suggestion %s, got %s", suggestion2, ee.context.Suggestions[1])
	}
}

func TestEnhancedError_GetContext(t *testing.T) {
	ee := NewEnhancedError(errors.New("test"), "operation")

	context := ee.GetContext()

	if context != ee.context {
		t.Error("GetContext should return the same context instance")
	}
}

func TestEnhancedError_FormatDetailed(t *testing.T) {
	originalErr := errors.New("test error")
	ee := NewEnhancedError(originalErr, "test_operation")
	ee.WithTemplate("/path/to/template.tmpl").
		WithOutput("/path/to/output.txt").
		WithLine(42).
		WithContext("key1", "value1").
		WithContext("key2", 123).
		WithSuggestion("Try this").
		WithSuggestion("Or this")

	formatted := ee.FormatDetailed()

	expectedParts := []string{
		"Error: test error",
		"Operation: test_operation",
		"Template: /path/to/template.tmpl",
		"Output: /path/to/output.txt",
		"Line: 42",
		"Context:",
		"key1: value1",
		"key2: 123",
		"Suggestions:",
		"1. Try this",
		"2. Or this",
		"Stack trace:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(formatted, part) {
			t.Errorf("Expected formatted output to contain '%s', but it didn't.\nFull output:\n%s", part, formatted)
		}
	}
}

func TestEnhancedError_FormatDetailed_MinimalError(t *testing.T) {
	originalErr := errors.New("simple error")
	ee := NewEnhancedError(originalErr, "simple_operation")

	formatted := ee.FormatDetailed()

	expectedParts := []string{
		"Error: simple error",
		"Operation: simple_operation",
		"Timestamp:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(formatted, part) {
			t.Errorf("Expected formatted output to contain '%s', but it didn't.\nFull output:\n%s", part, formatted)
		}
	}

	unexpectedParts := []string{
		"Template:",
		"Output:",
		"Line:",
		"Context:",
		"Suggestions:",
	}

	for _, part := range unexpectedParts {
		if strings.Contains(formatted, part) {
			t.Errorf("Expected formatted output NOT to contain '%s', but it did.\nFull output:\n%s", part, formatted)
		}
	}
}

func TestCaptureStack(t *testing.T) {
	stack := captureStack(0)

	if len(stack) == 0 {
		t.Error("Expected stack trace to contain frames")
	}

	if len(stack) > 10 {
		t.Errorf("Expected stack trace to be limited to 10 frames, got %d", len(stack))
	}

	for i, frame := range stack {
		if frame.Function == "" {
			t.Errorf("Stack frame %d has empty function name", i)
		}
		if frame.File == "" {
			t.Errorf("Stack frame %d has empty file name", i)
		}
		if frame.Line <= 0 {
			t.Errorf("Stack frame %d has invalid line number: %d", i, frame.Line)
		}
	}
}

func TestCaptureStack_WithSkip(t *testing.T) {
	stack1 := captureStack(0)
	stack2 := captureStack(1)

	if len(stack2) >= len(stack1) {
		t.Error("Expected skip=1 to produce fewer frames than skip=0")
	}

	if len(stack1) > 1 && len(stack2) > 0 {
		if stack1[1].Function == stack2[0].Function {
			t.Log("Skip parameter working correctly - frames shifted")
		}
	}
}

func TestNewErrorAnalyzer(t *testing.T) {
	ea := NewErrorAnalyzer()

	if ea == nil {
		t.Fatal("Expected non-nil ErrorAnalyzer")
	}

	if len(ea.errors) != 0 {
		t.Errorf("Expected empty error slice, got length %d", len(ea.errors))
	}
}

func TestErrorAnalyzer_AddError(t *testing.T) {
	ea := NewErrorAnalyzer()

	t.Run("add nil error", func(t *testing.T) {
		ea.AddError(nil)
		errors := ea.GetErrors()
		if len(errors) != 0 {
			t.Errorf("Expected 0 errors after adding nil, got %d", len(errors))
		}
	})

	t.Run("add valid error", func(t *testing.T) {
		ee := NewEnhancedError(errors.New("test"), "operation")
		ea.AddError(ee)

		errors := ea.GetErrors()
		if len(errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(errors))
		}
	})

	t.Run("buffer overflow protection", func(t *testing.T) {
		ea.Clear()

		for i := range 105 {
			ee := NewEnhancedError(fmt.Errorf("error %d", i), "operation")
			ea.AddError(ee)
		}

		errors := ea.GetErrors()
		if len(errors) != 100 {
			t.Errorf("Expected buffer to be limited to 100 errors, got %d", len(errors))
		}

		if !strings.Contains(errors[0].Error(), "error 5") {
			t.Error("Expected oldest errors to be removed (FIFO behavior)")
		}
	})
}

func TestErrorAnalyzer_GetErrors(t *testing.T) {
	ea := NewErrorAnalyzer()
	ee1 := NewEnhancedError(errors.New("error1"), "op1")
	ee2 := NewEnhancedError(errors.New("error2"), "op2")

	ea.AddError(ee1)
	ea.AddError(ee2)

	errors := ea.GetErrors()

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	errors[0] = *NewEnhancedError(fmt.Errorf("modified"), "modified")

	unmodified := ea.GetErrors()
	if unmodified[0].Error() == "modified" {
		t.Error("GetErrors should return a copy, not the original slice")
	}
}

func TestErrorAnalyzer_GetErrorsByOperation(t *testing.T) {
	ea := NewErrorAnalyzer()

	ee1 := NewEnhancedError(errors.New("error1"), "template_parse")
	ee2 := NewEnhancedError(errors.New("error2"), "file_write")
	ee3 := NewEnhancedError(errors.New("error3"), "template_parse")

	ea.AddError(ee1)
	ea.AddError(ee2)
	ea.AddError(ee3)

	parseErrors := ea.GetErrorsByOperation("template_parse")
	if len(parseErrors) != 2 {
		t.Errorf("Expected 2 template_parse errors, got %d", len(parseErrors))
	}

	writeErrors := ea.GetErrorsByOperation("file_write")
	if len(writeErrors) != 1 {
		t.Errorf("Expected 1 file_write error, got %d", len(writeErrors))
	}

	noErrors := ea.GetErrorsByOperation("nonexistent")
	if len(noErrors) != 0 {
		t.Errorf("Expected 0 errors for nonexistent operation, got %d", len(noErrors))
	}
}

func TestErrorAnalyzer_GetErrorsByTemplate(t *testing.T) {
	ea := NewErrorAnalyzer()

	ee1 := NewEnhancedError(errors.New("error1"), "op1").WithTemplate("template1.tmpl")
	ee2 := NewEnhancedError(errors.New("error2"), "op2").WithTemplate("template2.tmpl")
	ee3 := NewEnhancedError(errors.New("error3"), "op3").WithTemplate("template1.tmpl")
	ee4 := NewEnhancedError(errors.New("error4"), "op4") // No template

	ea.AddError(ee1)
	ea.AddError(ee2)
	ea.AddError(ee3)
	ea.AddError(ee4)

	template1Errors := ea.GetErrorsByTemplate("template1.tmpl")
	if len(template1Errors) != 2 {
		t.Errorf("Expected 2 errors for template1.tmpl, got %d", len(template1Errors))
	}

	template2Errors := ea.GetErrorsByTemplate("template2.tmpl")
	if len(template2Errors) != 1 {
		t.Errorf("Expected 1 error for template2.tmpl, got %d", len(template2Errors))
	}

	noErrors := ea.GetErrorsByTemplate("nonexistent.tmpl")
	if len(noErrors) != 0 {
		t.Errorf("Expected 0 errors for nonexistent template, got %d", len(noErrors))
	}
}

func TestErrorAnalyzer_GetStatistics(t *testing.T) {
	ea := NewErrorAnalyzer()

	t.Run("empty analyzer", func(t *testing.T) {
		stats := ea.GetStatistics()

		if stats.TotalErrors != 0 {
			t.Errorf("Expected 0 total errors, got %d", stats.TotalErrors)
		}

		if len(stats.OperationStats) != 0 {
			t.Errorf("Expected empty operation stats, got %d entries", len(stats.OperationStats))
		}
	})

	t.Run("with errors", func(t *testing.T) {
		ea.Clear()

		start := time.Now()

		ee1 := NewEnhancedError(errors.New("error1"), "parse")
		ee1.context.Timestamp = start
		ea.AddError(ee1)

		time.Sleep(1 * time.Millisecond)

		ee2 := NewEnhancedError(errors.New("error2"), "write").WithTemplate("template1.tmpl")
		ee2.context.Timestamp = start.Add(10 * time.Millisecond)
		ea.AddError(ee2)

		ee3 := NewEnhancedError(errors.New("error3"), "parse").WithTemplate("template2.tmpl")
		ee3.context.Timestamp = start.Add(20 * time.Millisecond)
		ea.AddError(ee3)

		stats := ea.GetStatistics()

		if stats.TotalErrors != 3 {
			t.Errorf("Expected 3 total errors, got %d", stats.TotalErrors)
		}

		if stats.OperationStats["parse"] != 2 {
			t.Errorf("Expected 2 parse errors, got %d", stats.OperationStats["parse"])
		}

		if stats.OperationStats["write"] != 1 {
			t.Errorf("Expected 1 write error, got %d", stats.OperationStats["write"])
		}

		if stats.TemplateStats["template1.tmpl"] != 1 {
			t.Errorf("Expected 1 error for template1.tmpl, got %d", stats.TemplateStats["template1.tmpl"])
		}

		if stats.TemplateStats["template2.tmpl"] != 1 {
			t.Errorf("Expected 1 error for template2.tmpl, got %d", stats.TemplateStats["template2.tmpl"])
		}

		if stats.TimeRange.Start != start {
			t.Errorf("Expected start time %v, got %v", start, stats.TimeRange.Start)
		}

		expectedEnd := start.Add(20 * time.Millisecond)
		if !stats.TimeRange.End.Equal(expectedEnd) {
			t.Errorf("Expected end time %v, got %v", expectedEnd, stats.TimeRange.End)
		}
	})
}

func TestErrorAnalyzer_Clear(t *testing.T) {
	ea := NewErrorAnalyzer()

	ee := NewEnhancedError(errors.New("test"), "operation")
	ea.AddError(ee)

	if len(ea.GetErrors()) != 1 {
		t.Error("Expected 1 error before clear")
	}

	ea.Clear()

	if len(ea.GetErrors()) != 0 {
		t.Error("Expected 0 errors after clear")
	}
}

func TestErrorAnalyzer_ConcurrentAccess(t *testing.T) {
	ea := NewErrorAnalyzer()

	var wg sync.WaitGroup
	numGoroutines := 10
	errorsPerGoroutine := 10

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range errorsPerGoroutine {
				ee := NewEnhancedError(fmt.Errorf("error %d-%d", id, j), fmt.Sprintf("operation_%d", id))
				ea.AddError(ee)
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 50 {
			_ = ea.GetErrors()
			_ = ea.GetStatistics()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	wg.Wait()

	errors := ea.GetErrors()
	if len(errors) != numGoroutines*errorsPerGoroutine {
		t.Errorf("Expected %d errors, got %d", numGoroutines*errorsPerGoroutine, len(errors))
	}
}

func TestErrorStatistics_String(t *testing.T) {
	t.Run("empty statistics", func(t *testing.T) {
		stats := ErrorStatistics{}
		result := stats.String()

		if result != "No errors recorded" {
			t.Errorf("Expected 'No errors recorded', got '%s'", result)
		}
	})

	t.Run("with statistics", func(t *testing.T) {
		start := time.Now()
		end := start.Add(5 * time.Minute)

		stats := ErrorStatistics{
			TotalErrors: 3,
			OperationStats: map[string]int{
				"parse": 2,
				"write": 1,
			},
			TemplateStats: map[string]int{
				"template1.tmpl": 2,
				"template2.tmpl": 1,
			},
			TimeRange: TimeRange{
				Start: start,
				End:   end,
			},
		}

		result := stats.String()

		expectedParts := []string{
			"Total errors: 3",
			"parse: 2",
			"write: 1",
			"template1.tmpl: 2",
			"template2.tmpl: 1",
			"5m0s",
		}

		for _, part := range expectedParts {
			if !strings.Contains(result, part) {
				t.Errorf("Expected result to contain '%s', but it didn't.\nFull result:\n%s", part, result)
			}
		}
	})
}

func TestSuggestTemplateErrors(t *testing.T) {
	tests := []struct {
		name                string
		err                 error
		templatePath        string
		expectedSuggestions []string
	}{
		{
			name:                "nil error",
			err:                 nil,
			templatePath:        "test.tmpl",
			expectedSuggestions: nil,
		},
		{
			name:         "file not found error",
			err:          errors.New("no such file or directory"),
			templatePath: "missing.tmpl",
			expectedSuggestions: []string{
				"Check if template file exists: missing.tmpl",
				"Verify the template directory path is correct",
			},
		},
		{
			name:         "parse error",
			err:          errors.New("template: parse error at line 5"),
			templatePath: "syntax.tmpl",
			expectedSuggestions: []string{
				"Check template syntax for unclosed braces {{ }}",
				"Verify function names are spelled correctly",
				"Check for missing quotes around string values",
			},
		},
		{
			name:         "undefined function error",
			err:          errors.New("function \"unknownFunc\" not defined"),
			templatePath: "func.tmpl",
			expectedSuggestions: []string{
				"Check if the function is available in the template function map",
				"Verify the function name spelling",
			},
		},
		{
			name:         "nil pointer error",
			err:          errors.New("runtime error: invalid memory address or nil pointer dereference"),
			templatePath: "nil.tmpl",
			expectedSuggestions: []string{
				"Check if template data contains nil values",
				"Use conditional checks like {{ if .Field }}...{{ end }}",
			},
		},
		{
			name:         "permission error",
			err:          errors.New("permission denied"),
			templatePath: "restricted.tmpl",
			expectedSuggestions: []string{
				"Check file/directory permissions",
				"Ensure the output directory is writable",
			},
		},
		{
			name:         "unknown error",
			err:          errors.New("some unknown error"),
			templatePath: "unknown.tmpl",
			expectedSuggestions: []string{
				"Check the template syntax and data structure",
				"Enable debug mode for more detailed error information",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := SuggestTemplateErrors(tt.err, tt.templatePath)

			if tt.expectedSuggestions == nil {
				if suggestions != nil {
					t.Errorf("Expected nil suggestions, got %v", suggestions)
				}
				return
			}

			if len(suggestions) != len(tt.expectedSuggestions) {
				t.Errorf("Expected %d suggestions, got %d", len(tt.expectedSuggestions), len(suggestions))
			}

			for i, expected := range tt.expectedSuggestions {
				if i >= len(suggestions) {
					t.Errorf("Missing expected suggestion: %s", expected)
					continue
				}
				if suggestions[i] != expected {
					t.Errorf("Expected suggestion %d to be '%s', got '%s'", i, expected, suggestions[i])
				}
			}
		})
	}
}

func TestSuggestTemplateErrors_CombinedErrors(t *testing.T) {
	err := errors.New("template: parse error: function \"unknownFunc\" not defined")
	suggestions := SuggestTemplateErrors(err, "combined.tmpl")

	expectedParts := []string{
		"Check template syntax",
		"Check if the function is available",
	}

	for _, part := range expectedParts {
		found := false
		for _, suggestion := range suggestions {
			if strings.Contains(suggestion, part) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected suggestions to contain text '%s', but they didn't.\nSuggestions: %v", part, suggestions)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxStackFrames != 10 {
		t.Errorf("Expected MaxStackFrames to be 10, got %d", config.MaxStackFrames)
	}
	if config.ErrorBufferSize != 100 {
		t.Errorf("Expected ErrorBufferSize to be 100, got %d", config.ErrorBufferSize)
	}
	if config.ExecutionBufferSize != 100 {
		t.Errorf("Expected ExecutionBufferSize to be 100, got %d", config.ExecutionBufferSize)
	}
	if config.MaxStackTraceDisplay != 5 {
		t.Errorf("Expected MaxStackTraceDisplay to be 5, got %d", config.MaxStackTraceDisplay)
	}
}

func TestSetConfig(t *testing.T) {
	originalConfig := GetConfig()
	defer func() {
		_ = SetConfig(originalConfig)
	}()

	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				MaxStackFrames:       20,
				ErrorBufferSize:      200,
				ExecutionBufferSize:  150,
				MaxStackTraceDisplay: 10,
			},
			expectError: false,
		},
		{
			name: "MaxStackFrames too small",
			config: Config{
				MaxStackFrames:       0,
				ErrorBufferSize:      100,
				ExecutionBufferSize:  100,
				MaxStackTraceDisplay: 5,
			},
			expectError: true,
		},
		{
			name: "MaxStackFrames too large",
			config: Config{
				MaxStackFrames:       101,
				ErrorBufferSize:      100,
				ExecutionBufferSize:  100,
				MaxStackTraceDisplay: 5,
			},
			expectError: true,
		},
		{
			name: "ErrorBufferSize too small",
			config: Config{
				MaxStackFrames:       10,
				ErrorBufferSize:      0,
				ExecutionBufferSize:  100,
				MaxStackTraceDisplay: 5,
			},
			expectError: true,
		},
		{
			name: "ErrorBufferSize too large",
			config: Config{
				MaxStackFrames:       10,
				ErrorBufferSize:      10001,
				ExecutionBufferSize:  100,
				MaxStackTraceDisplay: 5,
			},
			expectError: true,
		},
		{
			name: "ExecutionBufferSize too small",
			config: Config{
				MaxStackFrames:       10,
				ErrorBufferSize:      100,
				ExecutionBufferSize:  0,
				MaxStackTraceDisplay: 5,
			},
			expectError: true,
		},
		{
			name: "ExecutionBufferSize too large",
			config: Config{
				MaxStackFrames:       10,
				ErrorBufferSize:      100,
				ExecutionBufferSize:  10001,
				MaxStackTraceDisplay: 5,
			},
			expectError: true,
		},
		{
			name: "MaxStackTraceDisplay too small",
			config: Config{
				MaxStackFrames:       10,
				ErrorBufferSize:      100,
				ExecutionBufferSize:  100,
				MaxStackTraceDisplay: 0,
			},
			expectError: true,
		},
		{
			name: "MaxStackTraceDisplay larger than MaxStackFrames",
			config: Config{
				MaxStackFrames:       10,
				ErrorBufferSize:      100,
				ExecutionBufferSize:  100,
				MaxStackTraceDisplay: 15,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetConfig(tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				config := GetConfig()
				if config.MaxStackFrames != tt.config.MaxStackFrames {
					t.Errorf("Expected MaxStackFrames %d, got %d", tt.config.MaxStackFrames, config.MaxStackFrames)
				}
				if config.ErrorBufferSize != tt.config.ErrorBufferSize {
					t.Errorf("Expected ErrorBufferSize %d, got %d", tt.config.ErrorBufferSize, config.ErrorBufferSize)
				}
			}
		})
	}
}

func TestConfigurableStackFrames(t *testing.T) {
	originalConfig := GetConfig()
	defer func() {
		_ = SetConfig(originalConfig)
	}()

	// Test with different stack frame limits
	err := SetConfig(Config{
		MaxStackFrames:       3,
		ErrorBufferSize:      100,
		ExecutionBufferSize:  100,
		MaxStackTraceDisplay: 2,
	})
	if err != nil {
		t.Fatalf("Unexpected error setting config: %v", err)
	}

	stack := captureStack(0)
	if len(stack) > 3 {
		t.Errorf("Expected stack to be limited to 3 frames, got %d", len(stack))
	}
}

func TestConfigurableErrorBuffer(t *testing.T) {
	originalConfig := GetConfig()
	defer func() {
		_ = SetConfig(originalConfig)
	}()

	// Test with smaller buffer size
	err := SetConfig(Config{
		MaxStackFrames:       10,
		ErrorBufferSize:      3,
		ExecutionBufferSize:  100,
		MaxStackTraceDisplay: 5,
	})
	if err != nil {
		t.Fatalf("Unexpected error setting config: %v", err)
	}

	analyzer := NewErrorAnalyzer()

	// Add more errors than buffer size
	for i := 0; i < 5; i++ {
		testErr := NewEnhancedError(fmt.Errorf("test error %d", i), "test_operation")
		analyzer.AddError(testErr)
	}

	errors := analyzer.GetErrors()
	if len(errors) > 3 {
		t.Errorf("Expected error buffer to be limited to 3, got %d", len(errors))
	}
}

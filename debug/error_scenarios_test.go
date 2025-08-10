package debug

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"
	"text/template"
)

// TestErrorScenarios tests various error conditions and edge cases
// to ensure robust error handling and recovery mechanisms.
func TestErrorScenarios(t *testing.T) {
	t.Run("EnhancedError Edge Cases", func(t *testing.T) {
		testEnhancedErrorEdgeCases(t)
	})

	t.Run("DebugMode Error Handling", func(t *testing.T) {
		testDebugModeErrorHandling(t)
	})

	t.Run("Template Validation Errors", func(t *testing.T) {
		testTemplateValidationErrors(t)
	})

	t.Run("Concurrent Error Handling", func(t *testing.T) {
		testConcurrentErrorHandling(t)
	})

	t.Run("Memory Pressure Scenarios", func(t *testing.T) {
		testMemoryPressureScenarios(t)
	})
}

func testEnhancedErrorEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() error
		expectError bool
		expectPanic bool
	}{
		{
			name: "nested enhanced errors",
			setup: func() error {
				inner := NewEnhancedError(errors.New("inner error"), "inner_op")
				return NewEnhancedError(inner, "outer_op")
			},
			expectError: true,
		},
		{
			name: "very long error message",
			setup: func() error {
				longMsg := strings.Repeat("error ", 10000)
				return NewEnhancedError(errors.New(longMsg), "long_op")
			},
			expectError: true,
		},
		{
			name: "error with special characters",
			setup: func() error {
				specialMsg := "error with \n\t\r\x00 special chars 🚨"
				return NewEnhancedError(errors.New(specialMsg), "special_op")
			},
			expectError: true,
		},
		{
			name: "circular error reference",
			setup: func() error {
				// Create a scenario that could lead to circular references
				err1 := NewEnhancedError(errors.New("base error"), "op1")
				err2 := NewEnhancedError(err1, "op2")
				// This shouldn't cause infinite recursion
				return NewEnhancedError(err2, "op3")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic but none occurred")
					}
				}()
			}

			err := tt.setup()

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if err != nil {
				// Test that error formatting doesn't panic
				_ = err.Error()
				if enhanced, ok := err.(*EnhancedError); ok {
					_ = enhanced.FormatDetailed()
				}
			}
		})
	}
}

func testDebugModeErrorHandling(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *DebugMode
		test  func(*DebugMode) error
	}{
		{
			name: "invalid debug level",
			setup: func() *DebugMode {
				return NewDebugMode()
			},
			test: func(dm *DebugMode) error {
				// Test setting invalid debug level
				err := dm.SetLevel(DebugLevel(999))
				if err == nil {
					t.Error("Expected error for invalid debug level")
				}
				return nil
			},
		},
		{
			name: "logging at off level",
			setup: func() *DebugMode {
				dm := NewDebugMode()
				dm.SetLevel(LevelOff)
				return dm
			},
			test: func(dm *DebugMode) error {
				// Should not panic even when level is off
				dm.Error("test error", "key", "value")
				dm.Info("test info", "key", "value")
				dm.Debug("test debug", "key", "value")
				return nil
			},
		},
		{
			name: "concurrent level changes",
			setup: func() *DebugMode {
				return NewDebugMode()
			},
			test: func(dm *DebugMode) error {
				// Test concurrent level setting operations
				done := make(chan bool, 2)

				go func() {
					for range 100 {
						dm.SetLevel(LevelOff)
						dm.SetLevel(LevelInfo)
					}
					done <- true
				}()

				go func() {
					for range 100 {
						dm.SetLevel(LevelDebug)
						dm.SetLevel(LevelError)
					}
					done <- true
				}()

				<-done
				<-done
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := tt.setup()
			if err := tt.test(dm); err != nil {
				t.Errorf("Test failed: %v", err)
			}
		})
	}
}

func testTemplateValidationErrors(t *testing.T) {
	// Create test filesystem with various problematic templates
	testFS := fstest.MapFS{
		"good.tmpl":           {Data: []byte("{{ .Name }}")},
		"syntax_error.tmpl":   {Data: []byte("{{ .Name")}, // Missing closing brace
		"unknown_func.tmpl":   {Data: []byte("{{ unknownFunc .Name }}")},
		"path_traversal.tmpl": {Data: []byte("{{ template \"../../../etc/passwd\" . }}")},
		"deep_access.tmpl":    {Data: []byte("{{ .User.Profile.Settings.Display.Theme.Color.Primary }}")},
		"malformed.tmpl":      {Data: []byte("{{ if .Name }}{{ .Name }}")}, // Missing end
	}

	validator := NewTemplateValidator(testFS, template.FuncMap{}, nil)

	tests := []struct {
		name           string
		templatePath   string
		expectValid    bool
		expectErrors   int
		expectWarnings int
		errorTypes     []string
	}{
		{
			name:         "good template",
			templatePath: "good.tmpl",
			expectValid:  true,
		},
		{
			name:         "syntax error",
			templatePath: "syntax_error.tmpl",
			expectValid:  false,
			expectErrors: 2, // Syntax error + brace mismatch
			errorTypes:   []string{"syntax_error", "brace_mismatch"},
		},
		{
			name:         "unknown function",
			templatePath: "unknown_func.tmpl",
			expectValid:  false, // Actually generates error
			expectErrors: 1,
			errorTypes:   []string{"syntax_error"},
		},
		{
			name:         "path traversal attempt",
			templatePath: "path_traversal.tmpl",
			expectValid:  false,
			expectErrors: 1,
			errorTypes:   []string{"security_error"},
		},
		{
			name:           "deep variable access",
			templatePath:   "deep_access.tmpl",
			expectValid:    true, // Warning, not error
			expectWarnings: 2,    // Deep access + potentially unknown function warnings
		},
		{
			name:         "malformed template",
			templatePath: "malformed.tmpl",
			expectValid:  false,
			expectErrors: 1,
			errorTypes:   []string{"syntax_error"},
		},
		{
			name:         "nonexistent template",
			templatePath: "nonexistent.tmpl",
			expectValid:  false,
			expectErrors: 1,
			errorTypes:   []string{"file_error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateTemplate(tt.templatePath)

			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, result.Valid)
			}

			if tt.expectErrors > 0 && len(result.Errors) != tt.expectErrors {
				t.Errorf("Expected %d errors, got %d", tt.expectErrors, len(result.Errors))
			}

			if tt.expectWarnings > 0 && len(result.Warnings) != tt.expectWarnings {
				t.Errorf("Expected %d warnings, got %d", tt.expectWarnings, len(result.Warnings))
			}

			// Check error types
			for _, expectedType := range tt.errorTypes {
				found := false
				for _, err := range result.Errors {
					if err.Type == expectedType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error type %s not found", expectedType)
				}
			}
		})
	}
}

func testConcurrentErrorHandling(t *testing.T) {
	// Test concurrent error creation and analysis
	t.Run("concurrent error creation", func(t *testing.T) {
		const numGoroutines = 50
		const errorsPerGoroutine = 100

		done := make(chan bool, numGoroutines)

		for i := range numGoroutines {
			go func(id int) {
				defer func() { done <- true }()

				for range errorsPerGoroutine {
					err := NewEnhancedError(
						errors.New("concurrent error"),
						"concurrent_op",
					)
					if err == nil {
						t.Errorf("Goroutine %d: Expected error but got nil", id)
						return
					}

					// Test error formatting under concurrency
					_ = err.Error()
					_ = err.FormatDetailed()
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for range numGoroutines {
			<-done
		}
	})

	// Test concurrent validation
	t.Run("concurrent validation", func(t *testing.T) {
		testFS := fstest.MapFS{
			"test.tmpl": {Data: []byte("{{ .Name }}")},
		}

		validator := NewTemplateValidator(testFS, nil, nil)
		const numGoroutines = 20
		done := make(chan bool, numGoroutines)

		for range numGoroutines {
			go func() {
				defer func() { done <- true }()

				for range 50 {
					result := validator.ValidateTemplate("test.tmpl")
					if !result.Valid {
						t.Error("Expected valid template")
						return
					}
				}
			}()
		}

		for range numGoroutines {
			<-done
		}
	})
}

func testMemoryPressureScenarios(t *testing.T) {
	// Test behavior under memory pressure conditions
	t.Run("large error buffer", func(t *testing.T) {
		// Create many errors to test buffer management
		const numErrors = 1000

		for i := range numErrors {
			err := NewEnhancedError(
				errors.New("memory pressure test"),
				"pressure_test",
			)
			if err == nil {
				t.Fatal("Expected error but got nil")
			}

			// Force garbage collection periodically
			if i%100 == 0 {
				// runtime.GC() // Commented out to avoid importing runtime
			}
		}
	})

	t.Run("large template validation", func(t *testing.T) {
		// Create a very large template to test memory handling
		largeTemplate := strings.Repeat("{{ .Field }} ", 10000)

		testFS := fstest.MapFS{
			"large.tmpl": {Data: []byte(largeTemplate)},
		}

		validator := NewTemplateValidator(testFS, nil, nil)
		result := validator.ValidateTemplate("large.tmpl")

		// Should handle large templates gracefully
		if !result.Valid {
			t.Error("Large template validation failed")
		}
	})
}

// TestErrorRecovery tests error recovery mechanisms
func TestErrorRecovery(t *testing.T) {
	tests := []struct {
		name     string
		scenario func() error
		recovery func(error) bool
	}{
		{
			name: "recover from stack overflow simulation",
			scenario: func() error {
				// Simulate deep recursion scenario
				var createNestedError func(int) error
				createNestedError = func(depth int) error {
					if depth > 100 { // Prevent actual stack overflow
						return errors.New("max depth reached")
					}
					return NewEnhancedError(createNestedError(depth+1), "nested")
				}
				return createNestedError(0)
			},
			recovery: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "max depth")
			},
		},
		{
			name: "recover from invalid filesystem",
			scenario: func() error {
				// Test with invalid filesystem
				validator := NewTemplateValidator(nil, nil, nil)
				result := validator.ValidateTemplate("test.tmpl")
				if result.Valid {
					return errors.New("expected validation to fail with nil fs")
				}
				// Should have filesystem_error
				if len(result.Errors) == 0 || result.Errors[0].Type != "filesystem_error" {
					return errors.New("expected filesystem_error")
				}
				return nil
			},
			recovery: func(err error) bool {
				return err == nil // Should handle gracefully without panicking
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scenario()

			if !tt.recovery(err) {
				t.Errorf("Recovery check failed for error: %v", err)
			}
		})
	}
}

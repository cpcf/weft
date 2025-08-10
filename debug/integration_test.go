package debug

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
	"text/template"
	"time"
)

func TestEndToEndDebuggingWorkflow(t *testing.T) {
	// Set up a complete debugging environment
	var logBuffer bytes.Buffer
	
	// Create debug mode with all features enabled
	debugMode := NewDebugMode(
		WithLevel(LevelTrace),
		WithOutput(&logBuffer),
		WithProfiling(true),
		WithTracing(true),
		WithMetrics(true),
	)

	// Create error analyzer
	errorAnalyzer := NewErrorAnalyzer()

	// Create template debugger
	templateDebugger := NewTemplateDebugger(debugMode)

	// Create debug function map
	debugFuncMap := CreateDebugFuncMap(debugMode)

	// Add custom functions to the function map
	customFuncMap := template.FuncMap{
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"multiply":  func(a, b int) int { return a * b },
		"failing":   func() (string, error) { return "", errors.New("intentional failure") },
		"dangerous": func() string { panic("intentional panic") },
	}

	// Merge debug and custom function maps
	allFuncs := make(template.FuncMap)
	for name, fn := range debugFuncMap {
		allFuncs[name] = fn
	}
	for name, fn := range customFuncMap {
		allFuncs[name] = fn
	}

	// Create filesystem with templates
	templateFS := fstest.MapFS{
		"main.tmpl": &fstest.MapFile{
			Data: []byte(`
Debug Mode: {{debugContext}}
Current Time: {{debugTime}}
User Info: {{debug .User}}
{{debugLog "Processing user: %s" .User.Name}}

{{if .User.Active}}
	Hello {{upper .User.Name}}!
	You have {{multiply .User.Points 2}} bonus points.
	
	{{/* This will cause an error */}}
	{{if .TestFailure}}
		{{failing}}
	{{end}}
{{else}}
	User is not active: {{lower .User.Status}}
{{end}}

Template execution complete.
`),
		},
		"error.tmpl": &fstest.MapFile{
			Data: []byte(`
This template has syntax errors {{.Name
{{end}}
`),
		},
		"panic.tmpl": &fstest.MapFile{
			Data: []byte(`
This will panic: {{dangerous}}
`),
		},
	}

	// Create template validator
	validator := NewTemplateValidator(templateFS, allFuncs, debugMode)
	validator.SetStrict(true)

	t.Run("successful template execution with full debugging", func(t *testing.T) {
		logBuffer.Reset()

		// Validate template first
		validation := validator.ValidateTemplate("main.tmpl")
		if !validation.Valid {
			t.Logf("Template validation issues: %s", validation.Summary())
		}

		// Create template
		tmpl, err := template.New("main").Funcs(allFuncs).ParseFS(templateFS, "main.tmpl")
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		// Test data
		data := map[string]interface{}{
			"User": map[string]interface{}{
				"Name":    "John Doe",
				"Active":  true,
				"Points":  50,
				"Status":  "ACTIVE",
			},
			"TestFailure": false,
		}

		// Execute template with debugging
		output, err := templateDebugger.ExecuteWithDebug("main", tmpl, data)
		if err != nil {
			t.Fatalf("Template execution failed: %v", err)
		}

		// Verify output contains expected content
		expectedContent := []string{
			"Debug Mode:",
			"Hello JOHN DOE!",
			"You have 100 bonus points",
			"Template execution complete",
			"<!-- DEBUG: Processing user: John Doe -->",
		}

		for _, content := range expectedContent {
			if !strings.Contains(output, content) {
				t.Errorf("Expected output to contain '%s', got:\n%s", content, output)
			}
		}

		// Check debug logs
		logOutput := logBuffer.String()
		expectedLogs := []string{
			"template executed successfully",
			"template debug",
			"Processing user: John Doe",
		}

		for _, logContent := range expectedLogs {
			if !strings.Contains(logOutput, logContent) {
				t.Errorf("Expected log to contain '%s', got:\n%s", logContent, logOutput)
			}
		}

		// Check execution statistics
		executions := templateDebugger.GetExecutions()
		if len(executions) != 1 {
			t.Errorf("Expected 1 execution, got %d", len(executions))
		}

		execution := executions[0]
		if execution.Name != "main" {
			t.Errorf("Expected execution name 'main', got '%s'", execution.Name)
		}
		if execution.Error != "" {
			t.Errorf("Expected no execution error, got '%s'", execution.Error)
		}
		if execution.Duration <= 0 {
			t.Error("Expected positive execution duration")
		}

		stats := templateDebugger.GetExecutionStats()
		if stats["success_count"] != 1 {
			t.Errorf("Expected 1 successful execution, got %v", stats["success_count"])
		}
	})

	t.Run("template execution with controlled error", func(t *testing.T) {
		logBuffer.Reset()
		templateDebugger.ClearExecutions()

		// Create template
		tmpl, err := template.New("main").Funcs(allFuncs).ParseFS(templateFS, "main.tmpl")
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		// Test data that will trigger the failing function
		data := map[string]interface{}{
			"User": map[string]interface{}{
				"Name":    "Jane Doe",
				"Active":  true,
				"Points":  25,
				"Status":  "ACTIVE",
			},
			"TestFailure": true,
		}

		// Execute template - should fail
		_, err = templateDebugger.ExecuteWithDebug("main_error", tmpl, data)
		if err == nil {
			t.Error("Expected template execution to fail")
		}

		// Create enhanced error
		enhancedErr := NewEnhancedError(err, "template_execution").
			WithTemplate("main.tmpl").
			WithContext("user", "Jane Doe").
			WithSuggestion("Check the failing function implementation")

		// Add error to analyzer
		errorAnalyzer.AddError(enhancedErr)

		// Check execution was recorded as failed
		executions := templateDebugger.GetExecutions()
		if len(executions) != 1 {
			t.Errorf("Expected 1 execution, got %d", len(executions))
		}

		execution := executions[0]
		if execution.Error == "" {
			t.Error("Expected execution to have error recorded")
		}

		// Check error analysis
		errors := errorAnalyzer.GetErrors()
		if len(errors) != 1 {
			t.Errorf("Expected 1 error in analyzer, got %d", len(errors))
		}

		templateErrors := errorAnalyzer.GetErrorsByTemplate("main.tmpl")
		if len(templateErrors) != 1 {
			t.Errorf("Expected 1 template error, got %d", len(templateErrors))
		}

		// Check error suggestions
		suggestions := SuggestTemplateErrors(err, "main.tmpl")
		if len(suggestions) == 0 {
			t.Error("Expected error suggestions to be provided")
		}

		// Check error statistics
		stats := errorAnalyzer.GetStatistics()
		if stats.TotalErrors != 1 {
			t.Errorf("Expected 1 total error, got %d", stats.TotalErrors)
		}
	})

	t.Run("template syntax validation and error handling", func(t *testing.T) {
		logBuffer.Reset()

		// Validate syntax error template
		validation := validator.ValidateTemplate("error.tmpl")
		if validation.Valid {
			t.Error("Expected template with syntax errors to be invalid")
		}

		if len(validation.Errors) == 0 {
			t.Error("Expected syntax errors to be detected")
		}

		// Try to parse and execute the broken template
		tmpl := template.New("error").Funcs(allFuncs)
		content, _ := templateFS.ReadFile("error.tmpl")
		
		_, parseErr := tmpl.Parse(string(content))
		if parseErr == nil {
			t.Error("Expected template parsing to fail")
		}

		// Create enhanced error for parse failure
		enhancedErr := NewEnhancedError(parseErr, "template_parsing").
			WithTemplate("error.tmpl").
			WithLine(2).
			WithContext("validation_type", "syntax_check")

		errorAnalyzer.AddError(enhancedErr)

		// Check that error was properly categorized
		parseErrors := errorAnalyzer.GetErrorsByOperation("template_parsing")
		if len(parseErrors) != 1 {
			t.Errorf("Expected 1 parse error, got %d", len(parseErrors))
		}
	})

	t.Run("debug context and performance tracking", func(t *testing.T) {
		// Create debug context for operation tracking
		ctx := debugMode.NewContext("complex_operation")
		ctx.SetAttribute("operation_type", "template_processing")
		ctx.SetAttribute("batch_size", 100)

		// Simulate some work
		time.Sleep(1 * time.Millisecond)

		ctx.Info("Operation in progress", "step", "validation")

		// Add some more attributes during execution
		ctx.SetAttribute("templates_processed", 3)
		ctx.SetAttribute("errors_found", 1)

		ctx.Debug("Processing details", "current_template", "main.tmpl")

		// Complete the operation
		ctx.Complete()

		// Verify context attributes were captured in logs
		logOutput := logBuffer.String()
		expectedContextLogs := []string{
			"complex_operation",
			"operation_type",
			"template_processing",
			"templates_processed",
			"operation completed",
		}

		for _, logContent := range expectedContextLogs {
			if !strings.Contains(logOutput, logContent) {
				t.Errorf("Expected context log to contain '%s', got:\n%s", logContent, logOutput)
			}
		}

		// Check debug mode statistics
		debugStats := debugMode.GetStats()
		if debugStats.Level != LevelTrace {
			t.Errorf("Expected debug level %v, got %v", LevelTrace, debugStats.Level)
		}
		if !debugStats.ProfilingEnabled {
			t.Error("Expected profiling to be enabled")
		}
		if !debugStats.TracingEnabled {
			t.Error("Expected tracing to be enabled")
		}
		if !debugStats.MetricsEnabled {
			t.Error("Expected metrics to be enabled")
		}
	})

	t.Run("comprehensive error analysis and reporting", func(t *testing.T) {
		// Add various types of errors
		errors := []struct {
			err       error
			operation string
			template  string
		}{
			{fmt.Errorf("file not found"), "file_read", "missing.tmpl"},
			{fmt.Errorf("permission denied"), "file_write", "output.txt"},
			{fmt.Errorf("syntax error at line 5"), "template_parse", "broken.tmpl"},
			{fmt.Errorf("undefined function 'badFunc'"), "template_execute", "functions.tmpl"},
			{fmt.Errorf("nil pointer dereference"), "template_execute", "data.tmpl"},
		}

		for _, errorCase := range errors {
			enhancedErr := NewEnhancedError(errorCase.err, errorCase.operation).
				WithTemplate(errorCase.template).
				WithContext("timestamp", time.Now()).
				WithContext("user", "test_user")

			// Add suggestions based on error type
			suggestions := SuggestTemplateErrors(errorCase.err, errorCase.template)
			for _, suggestion := range suggestions {
				enhancedErr.WithSuggestion(suggestion)
			}

			errorAnalyzer.AddError(enhancedErr)
		}

		// Analyze error patterns
		allErrors := errorAnalyzer.GetErrors()
		if len(allErrors) < len(errors) {
			t.Errorf("Expected at least %d errors, got %d", len(errors), len(allErrors))
		}

		// Check error categorization
		parseErrors := errorAnalyzer.GetErrorsByOperation("template_parse")
		if len(parseErrors) == 0 {
			t.Error("Expected parse errors to be categorized")
		}

		executeErrors := errorAnalyzer.GetErrorsByOperation("template_execute")
		if len(executeErrors) == 0 {
			t.Error("Expected execute errors to be categorized")
		}

		// Check error statistics and reporting
		statistics := errorAnalyzer.GetStatistics()
		if statistics.TotalErrors == 0 {
			t.Error("Expected total errors to be counted")
		}

		if len(statistics.OperationStats) == 0 {
			t.Error("Expected operation statistics")
		}

		if len(statistics.TemplateStats) == 0 {
			t.Error("Expected template statistics")
		}

		// Generate human-readable report
		report := statistics.String()
		if !strings.Contains(report, "Total errors:") {
			t.Error("Expected error report to contain total count")
		}

		if !strings.Contains(report, "Errors by operation:") {
			t.Error("Expected error report to contain operation breakdown")
		}
	})

	t.Run("directory-wide validation workflow", func(t *testing.T) {
		// Add more templates to filesystem for directory validation
		extendedFS := fstest.MapFS{
			"templates/main.tmpl": &fstest.MapFile{
				Data: []byte("Valid template: {{.Name}}"),
			},
			"templates/header.tmpl": &fstest.MapFile{
				Data: []byte("Header: {{upper .Title}}"),
			},
			"templates/broken.tmpl": &fstest.MapFile{
				Data: []byte("Broken template: {{.Name"),
			},
			"templates/nested/deep.tmpl": &fstest.MapFile{
				Data: []byte("Deep template: {{.User.Profile.Settings.Theme.Color}}"),
			},
			"templates/functions.tmpl": &fstest.MapFile{
				Data: []byte("Functions: {{unknownFunc .Data}} {{upper .Text}}"),
			},
			"static/readme.txt": &fstest.MapFile{
				Data: []byte("This is not a template"),
			},
		}

		directoryValidator := NewTemplateValidator(extendedFS, allFuncs, debugMode)
		directoryValidator.SetStrict(true)

		// Validate entire directory
		results := directoryValidator.ValidateDirectory("templates")

		// Should find all template files
		expectedTemplates := []string{
			"templates/main.tmpl",
			"templates/header.tmpl", 
			"templates/broken.tmpl",
			"templates/nested/deep.tmpl",
			"templates/functions.tmpl",
		}

		for _, tmpl := range expectedTemplates {
			if _, found := results[tmpl]; !found {
				t.Errorf("Expected results for template %s", tmpl)
			}
		}

		// Should not include non-template files
		if _, found := results["static/readme.txt"]; found {
			t.Error("Expected non-template files to be excluded")
		}

		// Check specific validation results
		if result, found := results["templates/main.tmpl"]; found {
			if !result.Valid {
				t.Errorf("Expected main.tmpl to be valid, got: %s", result.Summary())
			}
		}

		if result, found := results["templates/broken.tmpl"]; found {
			if result.Valid {
				t.Error("Expected broken.tmpl to be invalid")
			}
		}

		if result, found := results["templates/nested/deep.tmpl"]; found {
			// Should have deep access warning
			foundDeepWarning := false
			for _, warning := range result.Warnings {
				if warning.Type == "deep_access" {
					foundDeepWarning = true
					break
				}
			}
			if !foundDeepWarning {
				t.Error("Expected deep access warning for deeply nested template")
			}
		}

		if result, found := results["templates/functions.tmpl"]; found {
			// Should have unknown function warning
			foundUnknownFunc := false
			for _, warning := range result.Warnings {
				if warning.Type == "unknown_function" && strings.Contains(warning.Message, "unknownFunc") {
					foundUnknownFunc = true
					break
				}
			}
			if !foundUnknownFunc {
				t.Error("Expected unknown function warning")
			}
		}
	})

	t.Run("full workflow integration", func(t *testing.T) {
		logBuffer.Reset()
		errorAnalyzer.Clear()
		templateDebugger.ClearExecutions()

		// Complete workflow: validate -> parse -> execute -> analyze
		templatePath := "main.tmpl"
		
		// Step 1: Validate
		validation := validator.ValidateTemplate(templatePath)
		debugMode.Info("Template validation completed", 
			"template", templatePath,
			"valid", validation.Valid,
			"errors", len(validation.Errors),
			"warnings", len(validation.Warnings))

		// Step 2: Parse
		tmpl, parseErr := template.New("workflow").Funcs(allFuncs).ParseFS(templateFS, templatePath)
		if parseErr != nil {
			enhancedErr := NewEnhancedError(parseErr, "parse_phase").
				WithTemplate(templatePath)
			errorAnalyzer.AddError(enhancedErr)
			debugMode.Error("Template parsing failed", "error", parseErr, "template", templatePath)
		}

		// Step 3: Execute with various data scenarios
		testScenarios := []struct {
			name string
			data interface{}
			expectError bool
		}{
			{
				name: "valid_data",
				data: map[string]interface{}{
					"User": map[string]interface{}{
						"Name": "Alice",
						"Active": true,
						"Points": 75,
					},
					"TestFailure": false,
				},
				expectError: false,
			},
			{
				name: "nil_data",
				data: nil,
				expectError: true,
			},
			{
				name: "partial_data",
				data: map[string]interface{}{
					"User": map[string]interface{}{
						"Name": "Bob",
						// Missing other fields
					},
				},
				expectError: false, // Template should handle missing fields gracefully
			},
		}

		for _, scenario := range testScenarios {
			scenarioCtx := debugMode.NewContext(fmt.Sprintf("execution_%s", scenario.name))
			scenarioCtx.SetAttribute("scenario", scenario.name)
			scenarioCtx.SetAttribute("expect_error", scenario.expectError)

			if tmpl != nil {
				output, execErr := templateDebugger.ExecuteWithDebug(scenario.name, tmpl, scenario.data)
				
				if scenario.expectError && execErr == nil {
					t.Errorf("Expected error for scenario %s", scenario.name)
				} else if !scenario.expectError && execErr != nil {
					t.Errorf("Unexpected error for scenario %s: %v", scenario.name, execErr)
				}

				if execErr != nil {
					enhancedErr := NewEnhancedError(execErr, "execution_phase").
						WithTemplate(templatePath).
						WithContext("scenario", scenario.name).
						WithContext("data_type", fmt.Sprintf("%T", scenario.data))
					errorAnalyzer.AddError(enhancedErr)
					scenarioCtx.CompleteWithError(execErr)
				} else {
					scenarioCtx.SetAttribute("output_length", len(output))
					scenarioCtx.Complete()
				}
			}
		}

		// Step 4: Analysis and Reporting
		finalStats := debugMode.GetStats()
		execStats := templateDebugger.GetExecutionStats()
		errorStats := errorAnalyzer.GetStatistics()

		debugMode.Info("Workflow completed",
			"debug_uptime", finalStats.Uptime,
			"total_executions", execStats["total_executions"],
			"success_rate", execStats["success_rate"], 
			"total_errors", errorStats.TotalErrors)

		// Verify comprehensive logging occurred
		finalLogOutput := logBuffer.String()
		expectedWorkflowLogs := []string{
			"Template validation completed",
			"execution_valid_data", 
			"execution_nil_data",
			"execution_partial_data",
			"Workflow completed",
		}

		for _, expectedLog := range expectedWorkflowLogs {
			if !strings.Contains(finalLogOutput, expectedLog) {
				t.Errorf("Expected workflow log to contain '%s'", expectedLog)
			}
		}

		// Verify all components worked together
		if templateDebugger.GetExecutions() == nil {
			t.Error("Expected template debugger to have recorded executions")
		}

		if errorAnalyzer.GetErrors() == nil && errorStats.TotalErrors > 0 {
			t.Error("Expected error analyzer to have recorded errors")
		}

		if finalStats.Uptime <= 0 {
			t.Error("Expected debug mode to track uptime")
		}
	})
}
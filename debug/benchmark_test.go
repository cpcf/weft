package debug

import (
	"bytes"
	"fmt"
	"testing"
	"testing/fstest"
	"text/template"
	"time"
)

// BenchmarkDebugMode_Creation benchmarks creating debug modes with different configurations
func BenchmarkDebugMode_Creation(b *testing.B) {
	b.Run("basic", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dm := NewDebugMode()
			_ = dm
		}
	})

	b.Run("with_options", func(b *testing.B) {
		var buf bytes.Buffer
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dm := NewDebugMode(
				WithLevel(LevelDebug),
				WithOutput(&buf),
				WithProfiling(true),
				WithTracing(true),
				WithMetrics(true),
			)
			_ = dm
		}
	})
}

// BenchmarkDebugMode_Logging benchmarks logging operations at different levels
func BenchmarkDebugMode_Logging(b *testing.B) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))

	b.Run("enabled_info", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dm.Info("benchmark message", "iteration", i, "data", "test")
		}
	})

	b.Run("enabled_debug", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dm.Debug("benchmark debug", "iteration", i, "complex", map[string]int{"count": i})
		}
	})

	b.Run("disabled_trace", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dm.Trace("benchmark trace", "iteration", i, "data", "test")
		}
	})

	dmOff := NewDebugMode(WithLevel(LevelOff), WithOutput(&buf))

	b.Run("disabled_all", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dmOff.Error("error message", "iteration", i)
			dmOff.Info("info message", "iteration", i)
			dmOff.Debug("debug message", "iteration", i)
		}
	})
}

// BenchmarkDebugContext benchmarks debug context operations
func BenchmarkDebugContext_Operations(b *testing.B) {
	dm := NewDebugMode(WithLevel(LevelOff)) // Disable logging for pure context ops

	b.Run("create_context", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := dm.NewContext(fmt.Sprintf("operation_%d", i))
			_ = ctx
		}
	})

	b.Run("set_attributes", func(b *testing.B) {
		ctx := dm.NewContext("benchmark")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx.SetAttribute(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
		}
	})

	b.Run("get_attributes", func(b *testing.B) {
		ctx := dm.NewContext("benchmark")
		for i := 0; i < 100; i++ {
			ctx.SetAttribute(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = ctx.GetAttribute(fmt.Sprintf("key_%d", i%100))
		}
	})
}

// BenchmarkDebugValue benchmarks the reflection-heavy debug value function
func BenchmarkDebugValue_ReflectionUsage(b *testing.B) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	debugFunc := debugValue(dm)

	testData := []struct {
		name  string
		value interface{}
	}{
		{"string", "hello world"},
		{"int", 42},
		{"float", 3.14159},
		{"bool", true},
		{"slice", []int{1, 2, 3, 4, 5}},
		{"map", map[string]int{"a": 1, "b": 2, "c": 3}},
		{"struct", struct{ Name string; Age int }{"John", 30}},
		{"pointer", func() *int { i := 42; return &i }()},
		{"nil", nil},
	}

	for _, td := range testData {
		b.Run(td.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := debugFunc(td.value)
				_ = result
			}
		})
	}

	// Benchmark with debug disabled to measure overhead
	dmOff := NewDebugMode(WithLevel(LevelOff))
	debugFuncOff := debugValue(dmOff)

	b.Run("disabled_overhead", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := debugFuncOff(map[string]interface{}{"complex": "data"})
			_ = result
		}
	})
}

// BenchmarkDebugHelpers benchmarks all debug helper functions
func BenchmarkDebugHelpers_AllFunctions(b *testing.B) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	funcMap := CreateDebugFuncMap(dm)

	testData := map[string]interface{}{
		"Name":    "John Doe",
		"Age":     30,
		"Active":  true,
		"Scores":  []int{95, 87, 92},
		"Profile": map[string]string{"role": "admin", "department": "engineering"},
	}

	b.Run("debug", func(b *testing.B) {
		debugFunc := funcMap["debug"].(func(interface{}) string)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = debugFunc(testData)
		}
	})

	b.Run("debugType", func(b *testing.B) {
		typeFunc := funcMap["debugType"].(func(interface{}) string)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = typeFunc(testData)
		}
	})

	b.Run("debugKeys", func(b *testing.B) {
		keysFunc := funcMap["debugKeys"].(func(interface{}) []string)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = keysFunc(testData)
		}
	})

	b.Run("debugJSON", func(b *testing.B) {
		jsonFunc := funcMap["debugJSON"].(func(interface{}) string)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = jsonFunc(testData)
		}
	})

	b.Run("debugPretty", func(b *testing.B) {
		prettyFunc := funcMap["debugPretty"].(func(interface{}) string)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = prettyFunc(testData)
		}
	})

	b.Run("debugTime", func(b *testing.B) {
		timeFunc := funcMap["debugTime"].(func() string)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = timeFunc()
		}
	})

	b.Run("debugContext", func(b *testing.B) {
		contextFunc := funcMap["debugContext"].(func() map[string]interface{})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = contextFunc()
		}
	})
}

// BenchmarkTemplateDebugger benchmarks template execution with debugging
func BenchmarkTemplateDebugger_Execution(b *testing.B) {
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&bytes.Buffer{}))
	td := NewTemplateDebugger(dm)
	funcMap := CreateDebugFuncMap(dm)

	templateText := `
Name: {{.Name}}
Debug Info: {{debug .}}
Type: {{debugType .Age}}
JSON: {{debugJSON .Scores}}
Time: {{debugTime}}
{{debugLog "Processing user: %s" .Name}}
`

	tmpl, err := template.New("benchmark").Funcs(funcMap).Parse(templateText)
	if err != nil {
		b.Fatalf("Failed to parse template: %v", err)
	}

	testData := map[string]interface{}{
		"Name":   "Alice Johnson",
		"Age":    28,
		"Scores": []int{88, 92, 85, 90},
		"Active": true,
	}

	b.Run("with_debugging", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = td.ExecuteWithDebug(fmt.Sprintf("bench_%d", i), tmpl, testData)
		}
	})

	// Compare with normal template execution
	normalTemplate := template.Must(template.New("normal").Parse("Name: {{.Name}}, Age: {{.Age}}"))

	b.Run("without_debugging", func(b *testing.B) {
		var buf bytes.Buffer
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			_ = normalTemplate.Execute(&buf, testData)
		}
	})

	// Benchmark with debugging disabled
	dmOff := NewDebugMode(WithLevel(LevelOff))
	funcMapOff := CreateDebugFuncMap(dmOff)
	tmplOff := template.Must(template.New("off").Funcs(funcMapOff).Parse(templateText))

	b.Run("debugging_disabled", func(b *testing.B) {
		var buf bytes.Buffer
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			_ = tmplOff.Execute(&buf, testData)
		}
	})
}

// BenchmarkErrorAnalyzer benchmarks error tracking and analysis
func BenchmarkErrorAnalyzer_Operations(b *testing.B) {
	ea := NewErrorAnalyzer()

	// Pre-populate with some errors
	for i := 0; i < 50; i++ {
		err := NewEnhancedError(fmt.Errorf("error %d", i), fmt.Sprintf("operation_%d", i%5))
		err.WithTemplate(fmt.Sprintf("template_%d.tmpl", i%10))
		err.WithContext("iteration", i)
		ea.AddError(err)
	}

	b.Run("add_error", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewEnhancedError(fmt.Errorf("benchmark error %d", i), "benchmark_op")
			ea.AddError(err)
		}
	})

	b.Run("get_errors", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ea.GetErrors()
		}
	})

	b.Run("get_errors_by_operation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ea.GetErrorsByOperation("operation_1")
		}
	})

	b.Run("get_statistics", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ea.GetStatistics()
		}
	})
}

// BenchmarkTemplateValidator benchmarks template validation operations
func BenchmarkTemplateValidator_Validation(b *testing.B) {
	testFS := fstest.MapFS{
		"simple.tmpl": &fstest.MapFile{
			Data: []byte("Hello {{.Name}}! You are {{.Age}} years old."),
		},
		"complex.tmpl": &fstest.MapFile{
			Data: []byte(`
{{/* Complex template with various constructs */}}
{{range .Items}}
	{{if .Active}}
		{{upper .Name}} - {{.Description}}
		{{template "partial" .}}
		{{include "shared"}}
		{{debugLog "Processing: %s" .Name}}
	{{end}}
{{end}}

{{with .User}}
	Profile: {{.Profile.Name}}
	Settings: {{.Settings.Theme.Color}}
{{end}}
`),
		},
		"broken.tmpl": &fstest.MapFile{
			Data: []byte("Broken template: {{.Name {{.Age}}"),
		},
	}

	funcMap := template.FuncMap{
		"upper": func(s string) string { return s },
		"debugLog": func(format string, args ...interface{}) string { 
			return fmt.Sprintf("<!-- %s -->", fmt.Sprintf(format, args...))
		},
	}

	validator := NewTemplateValidator(testFS, funcMap, NewDebugMode(WithLevel(LevelOff)))

	b.Run("simple_template", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateTemplate("simple.tmpl")
		}
	})

	b.Run("complex_template", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateTemplate("complex.tmpl")
		}
	})

	b.Run("broken_template", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateTemplate("broken.tmpl")
		}
	})

	// Benchmark strict vs non-strict validation
	validatorStrict := NewTemplateValidator(testFS, funcMap, NewDebugMode(WithLevel(LevelOff)))
	validatorStrict.SetStrict(true)

	b.Run("strict_validation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validatorStrict.ValidateTemplate("complex.tmpl")
		}
	})
}

// BenchmarkEnhancedError benchmarks error creation and formatting
func BenchmarkEnhancedError_Operations(b *testing.B) {
	baseError := fmt.Errorf("base error message")

	b.Run("create_enhanced_error", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewEnhancedError(baseError, fmt.Sprintf("operation_%d", i))
			_ = err
		}
	})

	b.Run("build_enhanced_error", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewEnhancedError(baseError, "operation").
				WithTemplate(fmt.Sprintf("template_%d.tmpl", i)).
				WithOutput(fmt.Sprintf("/output/file_%d.txt", i)).
				WithLine(i + 1).
				WithContext("iteration", i).
				WithContext("timestamp", time.Now()).
				WithSuggestion("Check your template syntax").
				WithSuggestion("Verify your data structure")
			_ = err
		}
	})

	// Pre-create a fully built error for formatting benchmarks
	complexError := NewEnhancedError(baseError, "complex_operation").
		WithTemplate("complex_template.tmpl").
		WithOutput("/path/to/output.txt").
		WithLine(42).
		WithContext("user", "john_doe").
		WithContext("session_id", "abc123").
		WithContext("request_id", "req_456").
		WithContext("data", map[string]int{"items": 10, "processed": 8}).
		WithSuggestion("Check the input data format").
		WithSuggestion("Verify template syntax").
		WithSuggestion("Enable debug mode for more details")

	b.Run("format_detailed", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = complexError.FormatDetailed()
		}
	})

	b.Run("capture_stack", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = captureStack(2)
		}
	})
}

// BenchmarkConcurrentAccess benchmarks thread safety under concurrent load
func BenchmarkConcurrentAccess(b *testing.B) {
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&bytes.Buffer{}))
	ea := NewErrorAnalyzer()
	td := NewTemplateDebugger(dm)

	b.Run("debug_mode_concurrent", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				dm.Info("concurrent message", "goroutine", i)
				dm.SetLevel(LevelDebug)
				_ = dm.IsEnabled(LevelInfo)
				_ = dm.GetStats()
				i++
			}
		})
	})

	b.Run("error_analyzer_concurrent", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				err := NewEnhancedError(fmt.Errorf("concurrent error %d", i), "concurrent_op")
				ea.AddError(err)
				_ = ea.GetErrors()
				_ = ea.GetStatistics()
				i++
			}
		})
	})

	b.Run("template_debugger_concurrent", func(b *testing.B) {
		tmpl := template.Must(template.New("concurrent").Parse("Concurrent {{.ID}}"))
		
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				data := map[string]interface{}{"ID": i}
				_, _ = td.ExecuteWithDebug(fmt.Sprintf("concurrent_%d", i), tmpl, data)
				_ = td.GetExecutions()
				_ = td.GetExecutionStats()
				i++
			}
		})
	})
}

// BenchmarkMemoryUsage provides insights into memory allocation patterns
func BenchmarkMemoryUsage_Allocations(b *testing.B) {
	b.Run("debug_mode_creation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dm := NewDebugMode(WithLevel(LevelDebug))
			_ = dm
		}
	})

	b.Run("enhanced_error_creation", func(b *testing.B) {
		baseErr := fmt.Errorf("test error")
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewEnhancedError(baseErr, "operation").
				WithTemplate("test.tmpl").
				WithContext("key", "value")
			_ = err
		}
	})

	b.Run("debug_func_execution", func(b *testing.B) {
		dm := NewDebugMode(WithLevel(LevelDebug))
		debugFunc := debugValue(dm)
		testData := map[string]interface{}{"name": "test", "count": 42}
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = debugFunc(testData)
		}
	})

	b.Run("template_execution_with_debug", func(b *testing.B) {
		dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&bytes.Buffer{}))
		td := NewTemplateDebugger(dm)
		funcMap := CreateDebugFuncMap(dm)
		
		tmpl := template.Must(template.New("alloc").Funcs(funcMap).Parse(
			"{{debug .}} {{debugJSON .}} {{debugTime}}"))
		data := map[string]interface{}{"test": true, "value": 123}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = td.ExecuteWithDebug("alloc_test", tmpl, data)
		}
	})
}
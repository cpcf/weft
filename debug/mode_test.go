package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDebugLevel_String(t *testing.T) {
	tests := []struct {
		level    DebugLevel
		expected string
	}{
		{LevelOff, "OFF"},
		{LevelError, "ERROR"},
		{LevelWarn, "WARN"},
		{LevelInfo, "INFO"},
		{LevelDebug, "DEBUG"},
		{LevelTrace, "TRACE"},
		{DebugLevel(999), "UNKNOWN"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			if result := test.level.String(); result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestWithLevel(t *testing.T) {
	dm := &DebugMode{}
	opt := WithLevel(LevelDebug)
	opt(dm)

	if dm.level != LevelDebug {
		t.Errorf("Expected level %v, got %v", LevelDebug, dm.level)
	}
}

func TestWithOutput(t *testing.T) {
	var buf bytes.Buffer
	dm := &DebugMode{}
	opt := WithOutput(&buf)
	opt(dm)

	if dm.output != &buf {
		t.Error("Expected output to be set to buffer")
	}
}

func TestWithProfiling(t *testing.T) {
	dm := &DebugMode{}
	opt := WithProfiling(true)
	opt(dm)

	if !dm.enableProfiling {
		t.Error("Expected profiling to be enabled")
	}
}

func TestWithTracing(t *testing.T) {
	dm := &DebugMode{}
	opt := WithTracing(true)
	opt(dm)

	if !dm.enableTracing {
		t.Error("Expected tracing to be enabled")
	}
}

func TestWithMetrics(t *testing.T) {
	dm := &DebugMode{}
	opt := WithMetrics(true)
	opt(dm)

	if !dm.enableMetrics {
		t.Error("Expected metrics to be enabled")
	}
}

func TestNewDebugMode(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		dm := NewDebugMode()

		if dm.level != LevelInfo {
			t.Errorf("Expected default level %v, got %v", LevelInfo, dm.level)
		}

		if dm.logger == nil {
			t.Error("Expected logger to be initialized")
		}

		if dm.startTime.IsZero() {
			t.Error("Expected start time to be set")
		}
	})

	t.Run("with options", func(t *testing.T) {
		var buf bytes.Buffer
		dm := NewDebugMode(
			WithLevel(LevelDebug),
			WithOutput(&buf),
			WithProfiling(true),
			WithTracing(true),
			WithMetrics(true),
		)

		if dm.level != LevelDebug {
			t.Errorf("Expected level %v, got %v", LevelDebug, dm.level)
		}

		if dm.output != &buf {
			t.Error("Expected output to be buffer")
		}

		if !dm.enableProfiling {
			t.Error("Expected profiling to be enabled")
		}

		if !dm.enableTracing {
			t.Error("Expected tracing to be enabled")
		}

		if !dm.enableMetrics {
			t.Error("Expected metrics to be enabled")
		}
	})
}

func TestDebugMode_IsEnabled(t *testing.T) {
	tests := []struct {
		modeLevel DebugLevel
		checkLevel DebugLevel
		expected bool
	}{
		{LevelOff, LevelError, false},
		{LevelError, LevelError, true},
		{LevelError, LevelWarn, false},
		{LevelWarn, LevelError, true},
		{LevelWarn, LevelWarn, true},
		{LevelWarn, LevelInfo, false},
		{LevelInfo, LevelInfo, true},
		{LevelInfo, LevelDebug, false},
		{LevelDebug, LevelDebug, true},
		{LevelDebug, LevelTrace, false},
		{LevelTrace, LevelTrace, true},
		{LevelTrace, LevelDebug, true},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", test.modeLevel, test.checkLevel), func(t *testing.T) {
			dm := NewDebugMode(WithLevel(test.modeLevel))
			result := dm.IsEnabled(test.checkLevel)

			if result != test.expected {
				t.Errorf("Expected IsEnabled(%v) = %v for mode level %v, got %v", 
					test.checkLevel, test.expected, test.modeLevel, result)
			}
		})
	}
}

func TestDebugMode_SetLevel(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelInfo))

	if dm.level != LevelInfo {
		t.Error("Expected initial level to be Info")
	}

	dm.SetLevel(LevelDebug)

	if dm.level != LevelDebug {
		t.Errorf("Expected level to be %v after SetLevel, got %v", LevelDebug, dm.level)
	}

	if !dm.IsEnabled(LevelDebug) {
		t.Error("Expected debug level to be enabled after SetLevel")
	}
}

func TestDebugMode_LoggingMethods(t *testing.T) {
	tests := []struct {
		name string
		level DebugLevel
		logFunc func(*DebugMode, *bytes.Buffer)
		expectOutput bool
	}{
		{
			name: "Error at Error level",
			level: LevelError,
			logFunc: func(dm *DebugMode, buf *bytes.Buffer) {
				dm.Error("test error", "key", "value")
			},
			expectOutput: true,
		},
		{
			name: "Error at Off level",
			level: LevelOff,
			logFunc: func(dm *DebugMode, buf *bytes.Buffer) {
				dm.Error("test error", "key", "value")
			},
			expectOutput: false,
		},
		{
			name: "Warn at Warn level",
			level: LevelWarn,
			logFunc: func(dm *DebugMode, buf *bytes.Buffer) {
				dm.Warn("test warning", "key", "value")
			},
			expectOutput: true,
		},
		{
			name: "Warn at Error level",
			level: LevelError,
			logFunc: func(dm *DebugMode, buf *bytes.Buffer) {
				dm.Warn("test warning", "key", "value")
			},
			expectOutput: false,
		},
		{
			name: "Info at Info level",
			level: LevelInfo,
			logFunc: func(dm *DebugMode, buf *bytes.Buffer) {
				dm.Info("test info", "key", "value")
			},
			expectOutput: true,
		},
		{
			name: "Debug at Debug level",
			level: LevelDebug,
			logFunc: func(dm *DebugMode, buf *bytes.Buffer) {
				dm.Debug("test debug", "key", "value")
			},
			expectOutput: true,
		},
		{
			name: "Debug at Info level",
			level: LevelInfo,
			logFunc: func(dm *DebugMode, buf *bytes.Buffer) {
				dm.Debug("test debug", "key", "value")
			},
			expectOutput: false,
		},
		{
			name: "Trace at Trace level",
			level: LevelTrace,
			logFunc: func(dm *DebugMode, buf *bytes.Buffer) {
				dm.Trace("test trace", "key", "value")
			},
			expectOutput: true,
		},
		{
			name: "Trace at Debug level",
			level: LevelDebug,
			logFunc: func(dm *DebugMode, buf *bytes.Buffer) {
				dm.Trace("test trace", "key", "value")
			},
			expectOutput: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			dm := NewDebugMode(WithLevel(test.level), WithOutput(&buf))

			test.logFunc(dm, &buf)

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != test.expectOutput {
				t.Errorf("Expected output=%v, got output=%v (len=%d)", test.expectOutput, hasOutput, len(output))
			}

			if test.expectOutput && strings.Contains(test.name, "Trace") {
				if !strings.Contains(output, "[TRACE]") {
					t.Error("Expected trace output to contain [TRACE] prefix")
				}
			}
		})
	}
}

func TestDebugMode_LogTemplateExecution(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))

	templatePath := "/path/to/template.tmpl"
	data := map[string]interface{}{"key": "value"}
	duration := 100 * time.Millisecond

	dm.LogTemplateExecution(templatePath, data, duration)

	output := buf.String()
	if !strings.Contains(output, "template executed") {
		t.Error("Expected output to contain 'template executed'")
	}
	if !strings.Contains(output, templatePath) {
		t.Error("Expected output to contain template path")
	}
	if !strings.Contains(output, "100ms") {
		t.Error("Expected output to contain duration")
	}
}

func TestDebugMode_LogTemplateData(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelTrace), WithOutput(&buf))

	templatePath := "/path/to/template.tmpl"
	data := map[string]interface{}{"key": "value", "number": 42}

	dm.LogTemplateData(templatePath, data)

	output := buf.String()
	if !strings.Contains(output, "[TRACE]") {
		t.Error("Expected output to contain [TRACE] prefix")
	}
	if !strings.Contains(output, "template data") {
		t.Error("Expected output to contain 'template data'")
	}
	if !strings.Contains(output, templatePath) {
		t.Error("Expected output to contain template path")
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Error("Expected output to contain serialized data")
	}
}

func TestDebugMode_LogFileWrite(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))

	path := "/path/to/output.txt"
	size := 1024
	duration := 50 * time.Millisecond

	dm.LogFileWrite(path, size, duration)

	output := buf.String()
	if !strings.Contains(output, "file written") {
		t.Error("Expected output to contain 'file written'")
	}
	if !strings.Contains(output, path) {
		t.Error("Expected output to contain file path")
	}
	if !strings.Contains(output, "1024") {
		t.Error("Expected output to contain file size")
	}
	if !strings.Contains(output, "50ms") {
		t.Error("Expected output to contain duration")
	}
}

func TestDebugMode_LogError(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelError), WithOutput(&buf))

	operation := "file_write"
	err := fmt.Errorf("permission denied")
	context := map[string]any{
		"file": "/protected/file.txt",
		"size": 512,
	}

	dm.LogError(operation, err, context)

	output := buf.String()
	if !strings.Contains(output, "operation failed") {
		t.Error("Expected output to contain 'operation failed'")
	}
	if !strings.Contains(output, operation) {
		t.Error("Expected output to contain operation")
	}
	if !strings.Contains(output, "permission denied") {
		t.Error("Expected output to contain error message")
	}
	if !strings.Contains(output, "/protected/file.txt") {
		t.Error("Expected output to contain context values")
	}
}

func TestDebugMode_GetStats(t *testing.T) {
	start := time.Now()
	dm := NewDebugMode(
		WithLevel(LevelDebug),
		WithProfiling(true),
		WithTracing(false),
		WithMetrics(true),
	)

	time.Sleep(1 * time.Millisecond) // Ensure some uptime

	stats := dm.GetStats()

	if stats.Level != LevelDebug {
		t.Errorf("Expected level %v, got %v", LevelDebug, stats.Level)
	}

	if stats.StartTime.Before(start) {
		t.Error("Expected start time to be after test start")
	}

	if stats.Uptime <= 0 {
		t.Error("Expected positive uptime")
	}

	if !stats.ProfilingEnabled {
		t.Error("Expected profiling to be enabled in stats")
	}

	if stats.TracingEnabled {
		t.Error("Expected tracing to be disabled in stats")
	}

	if !stats.MetricsEnabled {
		t.Error("Expected metrics to be enabled in stats")
	}
}

func TestDebugStats_String(t *testing.T) {
	stats := DebugStats{
		Level:            LevelInfo,
		StartTime:        time.Now(),
		Uptime:           5 * time.Minute,
		ProfilingEnabled: true,
		TracingEnabled:   false,
		MetricsEnabled:   true,
	}

	result := stats.String()

	expectedParts := []string{
		"Debug Stats:",
		"Level=INFO",
		"Uptime=5m0s",
		"Profiling=true",
		"Tracing=false",
		"Metrics=true",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected stats string to contain '%s', got: %s", part, result)
		}
	}
}

func TestDebugMode_NewContext(t *testing.T) {
	dm := NewDebugMode()
	operation := "template_parse"

	ctx := dm.NewContext(operation)

	if ctx == nil {
		t.Fatal("Expected non-nil debug context")
	}

	if ctx.mode != dm {
		t.Error("Expected context to reference the debug mode")
	}

	if ctx.operation != operation {
		t.Errorf("Expected operation %s, got %s", operation, ctx.operation)
	}

	if ctx.startTime.IsZero() {
		t.Error("Expected start time to be set")
	}

	if ctx.attributes == nil {
		t.Error("Expected attributes map to be initialized")
	}
}

func TestDebugContext_SetAttribute(t *testing.T) {
	dm := NewDebugMode()
	ctx := dm.NewContext("test")

	key := "testKey"
	value := "testValue"

	ctx.SetAttribute(key, value)

	if ctx.attributes[key] != value {
		t.Errorf("Expected attribute[%s] = %v, got %v", key, value, ctx.attributes[key])
	}
}

func TestDebugContext_GetAttribute(t *testing.T) {
	dm := NewDebugMode()
	ctx := dm.NewContext("test")

	key := "testKey"
	value := 42

	ctx.SetAttribute(key, value)

	result, exists := ctx.GetAttribute(key)
	if !exists {
		t.Error("Expected attribute to exist")
	}
	if result != value {
		t.Errorf("Expected attribute value %v, got %v", value, result)
	}

	_, exists = ctx.GetAttribute("nonexistent")
	if exists {
		t.Error("Expected nonexistent attribute to not exist")
	}
}

func TestDebugContext_Error(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelError), WithOutput(&buf))
	ctx := dm.NewContext("test_operation")

	ctx.SetAttribute("file", "test.txt")
	ctx.SetAttribute("size", 1024)

	err := fmt.Errorf("test error")
	ctx.Error("operation failed", err)

	output := buf.String()
	if !strings.Contains(output, "operation failed") {
		t.Error("Expected output to contain 'operation failed'")
	}
	if !strings.Contains(output, "test_operation") {
		t.Error("Expected output to contain operation name")
	}
	if !strings.Contains(output, "test error") {
		t.Error("Expected output to contain error message")
	}
	if !strings.Contains(output, "test.txt") {
		t.Error("Expected output to contain context attributes")
	}
}

func TestDebugContext_Info(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelInfo), WithOutput(&buf))
	ctx := dm.NewContext("test_operation")

	ctx.SetAttribute("file", "test.txt")

	ctx.Info("processing file", "processed", true)

	output := buf.String()
	if !strings.Contains(output, "processing file") {
		t.Error("Expected output to contain message")
	}
	if !strings.Contains(output, "test_operation") {
		t.Error("Expected output to contain operation name")
	}
	if !strings.Contains(output, "duration") {
		t.Error("Expected output to contain duration")
	}
	if !strings.Contains(output, "test.txt") {
		t.Error("Expected output to contain context attributes")
	}
	if !strings.Contains(output, "processed") {
		t.Error("Expected output to contain additional args")
	}
}

func TestDebugContext_Debug(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))
	ctx := dm.NewContext("test_operation")

	ctx.SetAttribute("step", 1)

	ctx.Debug("processing step", "success", true)

	output := buf.String()
	if !strings.Contains(output, "processing step") {
		t.Error("Expected output to contain message")
	}
	if !strings.Contains(output, "test_operation") {
		t.Error("Expected output to contain operation name")
	}
	if !strings.Contains(output, "step") {
		t.Error("Expected output to contain context attributes")
	}
}

func TestDebugContext_Complete(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))
	ctx := dm.NewContext("test_operation")

	time.Sleep(1 * time.Millisecond) // Ensure some duration

	ctx.Complete()

	output := buf.String()
	if !strings.Contains(output, "operation completed") {
		t.Error("Expected output to contain 'operation completed'")
	}
	if !strings.Contains(output, "test_operation") {
		t.Error("Expected output to contain operation name")
	}
	if !strings.Contains(output, "duration") {
		t.Error("Expected output to contain duration")
	}
}

func TestDebugContext_CompleteWithError(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelError), WithOutput(&buf))
	ctx := dm.NewContext("test_operation")

	time.Sleep(1 * time.Millisecond) // Ensure some duration

	err := fmt.Errorf("operation failed")
	ctx.CompleteWithError(err)

	output := buf.String()
	if !strings.Contains(output, "operation failed") {
		t.Error("Expected output to contain 'operation failed'")
	}
	if !strings.Contains(output, "test_operation") {
		t.Error("Expected output to contain operation name")
	}
	if !strings.Contains(output, "duration") {
		t.Error("Expected output to contain duration")
	}
	if !strings.Contains(output, "operation failed") {
		t.Error("Expected output to contain error message")
	}
}

func TestDebugMode_ConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				dm.SetLevel(LevelDebug)
				dm.Info("concurrent test", "goroutine", id, "operation", j)
				dm.Debug("debug message", "goroutine", id, "operation", j)
				_ = dm.IsEnabled(LevelInfo)
				_ = dm.GetStats()
			}
		}(i)
	}

	wg.Wait()

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected some output from concurrent operations")
	}
}

func TestDebugContext_ConcurrentAccess(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelOff))
	ctx := dm.NewContext("concurrent_test")

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				ctx.SetAttribute(key, value)

				retrieved, exists := ctx.GetAttribute(key)
				if exists && retrieved != value {
					t.Errorf("Concurrent access corruption: expected %v, got %v", value, retrieved)
				}
			}
		}(i)
	}

	wg.Wait()

	expectedKeys := numGoroutines * operationsPerGoroutine
	actualKeys := len(ctx.attributes)

	if actualKeys != expectedKeys {
		t.Errorf("Expected %d attributes, got %d", expectedKeys, actualKeys)
	}
}

func TestDebugMode_LogTemplateData_NonSerializableData(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelTrace), WithOutput(&buf))

	nonSerializable := make(chan int)

	dm.LogTemplateData("test.tmpl", nonSerializable)

	output := buf.String()
	if !strings.Contains(output, "[TRACE]") {
		t.Error("Expected output even for non-serializable data")
	}
}

func TestDebugMode_SetupLoggerLevelMapping(t *testing.T) {
	tests := []struct {
		debugLevel DebugLevel
		name       string
	}{
		{LevelOff, "off"},
		{LevelError, "error"},
		{LevelWarn, "warn"},
		{LevelInfo, "info"},
		{LevelDebug, "debug"},
		{LevelTrace, "trace"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			dm := NewDebugMode(WithLevel(test.debugLevel), WithOutput(&buf))

			if dm.logger == nil {
				t.Error("Expected logger to be initialized regardless of level")
			}

			dm.Error("error test")
			dm.Info("info test")
			dm.Debug("debug test")

			if test.debugLevel >= LevelDebug {
				output := buf.String()
				if !strings.Contains(output, "debug/mode.go") {
					t.Errorf("Expected source information to be included at debug level and above. Got output: %q", output)
				}
			}
		})
	}
}

func TestDebugStats_JSONSerialization(t *testing.T) {
	stats := DebugStats{
		Level:            LevelDebug,
		StartTime:        time.Now(),
		Uptime:           5 * time.Minute,
		ProfilingEnabled: true,
		TracingEnabled:   false,
		MetricsEnabled:   true,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal stats: %v", err)
	}

	var unmarshaled DebugStats
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal stats: %v", err)
	}

	if unmarshaled.Level != stats.Level {
		t.Errorf("Level not preserved: expected %v, got %v", stats.Level, unmarshaled.Level)
	}

	if unmarshaled.ProfilingEnabled != stats.ProfilingEnabled {
		t.Errorf("ProfilingEnabled not preserved: expected %v, got %v", stats.ProfilingEnabled, unmarshaled.ProfilingEnabled)
	}
}
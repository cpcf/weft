package debug

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"
)

func TestCreateDebugFuncMap(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	funcMap := CreateDebugFuncMap(dm)

	expectedFunctions := []string{
		"debug",
		"debugType",
		"debugKeys",
		"debugSize",
		"debugJSON",
		"debugPretty",
		"debugLog",
		"debugTime",
		"debugStack",
		"debugContext",
	}

	for _, funcName := range expectedFunctions {
		if _, exists := funcMap[funcName]; !exists {
			t.Errorf("Expected function '%s' to exist in debug func map", funcName)
		}
	}

	if len(funcMap) != len(expectedFunctions) {
		t.Errorf("Expected %d functions, got %d", len(expectedFunctions), len(funcMap))
	}
}

func TestDebugValue(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	debugFunc := debugValue(dm)

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: "<nil>",
		},
		{
			name:     "string value",
			input:    "hello",
			expected: `string(5): "hello"`,
		},
		{
			name:     "int value",
			input:    42,
			expected: "int: 42",
		},
		{
			name:     "uint value",
			input:    uint(42),
			expected: "uint: 42",
		},
		{
			name:     "float value",
			input:    3.14,
			expected: "float: 3.14",
		},
		{
			name:     "bool value",
			input:    true,
			expected: "bool: true",
		},
		{
			name:     "slice value",
			input:    []int{1, 2, 3},
			expected: "[]int[3]: [1 2 3]",
		},
		{
			name:     "map value",
			input:    map[string]int{"a": 1, "b": 2},
			expected: "map[string]int{2 keys}:",
		},
		{
			name:     "struct value",
			input:    struct{ Name string }{"test"},
			expected: "struct { Name string }: {Name:test}",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := debugFunc(test.input)
			if !strings.Contains(result, test.expected) {
				t.Errorf("Expected result to contain '%s', got '%s'", test.expected, result)
			}
		})
	}

	t.Run("nil pointer", func(t *testing.T) {
		var p *int = nil
		result := debugFunc(p)
		if !strings.Contains(result, "<nil>") {
			t.Errorf("Expected nil pointer to contain '<nil>', got '%s'", result)
		}
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		value := 42
		p := &value
		result := debugFunc(p)
		if !strings.Contains(result, "*int:") || !strings.Contains(result, "int: 42") {
			t.Errorf("Expected pointer dereference, got '%s'", result)
		}
	})

	t.Run("debug disabled", func(t *testing.T) {
		dmOff := NewDebugMode(WithLevel(LevelOff))
		debugFuncOff := debugValue(dmOff)
		result := debugFuncOff("test")
		if result != "" {
			t.Errorf("Expected empty result when debug disabled, got '%s'", result)
		}
	})
}

func TestDebugType(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	typeFunc := debugType(dm)

	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: []string{"<nil>"},
		},
		{
			name:     "string type",
			input:    "hello",
			expected: []string{"Type: string", "Kind: string"},
		},
		{
			name:     "int type",
			input:    42,
			expected: []string{"Type: int", "Kind: int"},
		},
		{
			name:     "slice type",
			input:    []int{1, 2, 3},
			expected: []string{"Type: []int", "Kind: slice"},
		},
		{
			name:     "struct type",
			input:    struct{ Name string }{"test"},
			expected: []string{"Kind: struct"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := typeFunc(test.input)
			for _, expected := range test.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got '%s'", expected, result)
				}
			}
		})
	}

	t.Run("nil pointer", func(t *testing.T) {
		var p *int = nil
		result := typeFunc(p)
		if !strings.Contains(result, "nil pointer") {
			t.Errorf("Expected nil pointer information, got '%s'", result)
		}
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		value := 42
		p := &value
		result := typeFunc(p)
		if !strings.Contains(result, "*int") || !strings.Contains(result, "-> Type: int") {
			t.Errorf("Expected pointer type information, got '%s'", result)
		}
	})

	t.Run("debug disabled", func(t *testing.T) {
		dmOff := NewDebugMode(WithLevel(LevelOff))
		typeFuncOff := debugType(dmOff)
		result := typeFuncOff("test")
		if result != "" {
			t.Errorf("Expected empty result when debug disabled, got '%s'", result)
		}
	})
}

func TestDebugKeys(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	keysFunc := debugKeys(dm)

	t.Run("nil value", func(t *testing.T) {
		result := keysFunc(nil)
		if result != nil {
			t.Errorf("Expected nil for nil input, got %v", result)
		}
	})

	t.Run("map keys", func(t *testing.T) {
		input := map[string]int{"zebra": 1, "apple": 2, "banana": 3}
		result := keysFunc(input)
		
		expected := []string{"apple", "banana", "zebra"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("struct fields", func(t *testing.T) {
		input := struct {
			Name    string
			Age     int
			private string
		}{"test", 25, "hidden"}
		
		result := keysFunc(input)
		
		expected := []string{"Age", "Name"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("non-map/struct", func(t *testing.T) {
		result := keysFunc("string")
		if result != nil {
			t.Errorf("Expected nil for non-map/struct input, got %v", result)
		}
	})

	t.Run("debug disabled", func(t *testing.T) {
		dmOff := NewDebugMode(WithLevel(LevelOff))
		keysFuncOff := debugKeys(dmOff)
		result := keysFuncOff(map[string]int{"key": 1})
		if result != nil {
			t.Errorf("Expected nil when debug disabled, got %v", result)
		}
	})
}

func TestDebugSize(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	sizeFunc := debugSize(dm)

	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: 0,
		},
		{
			name:     "string size",
			input:    "hello",
			expected: 5,
		},
		{
			name:     "slice size",
			input:    []int{1, 2, 3, 4},
			expected: 4,
		},
		{
			name:     "map size",
			input:    map[string]int{"a": 1, "b": 2, "c": 3},
			expected: 3,
		},
		{
			name:     "array size",
			input:    [5]int{1, 2, 3, 4, 5},
			expected: 5,
		},
		{
			name:     "other type size",
			input:    42,
			expected: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := sizeFunc(test.input)
			if result != test.expected {
				t.Errorf("Expected size %d, got %d", test.expected, result)
			}
		})
	}

	t.Run("debug disabled", func(t *testing.T) {
		dmOff := NewDebugMode(WithLevel(LevelOff))
		sizeFuncOff := debugSize(dmOff)
		result := sizeFuncOff("test")
		if result != 0 {
			t.Errorf("Expected 0 when debug disabled, got %d", result)
		}
	})
}

func TestDebugJSON(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	jsonFunc := debugJSON(dm)

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: "null",
		},
		{
			name:     "string value",
			input:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "number value",
			input:    42,
			expected: "42",
		},
		{
			name:     "bool value",
			input:    true,
			expected: "true",
		},
		{
			name:     "map value",
			input:    map[string]int{"key": 123},
			expected: `{"key":123}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := jsonFunc(test.input)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}

	t.Run("non-serializable value", func(t *testing.T) {
		ch := make(chan int)
		result := jsonFunc(ch)
		if !strings.Contains(result, "JSON Error:") {
			t.Errorf("Expected JSON error for non-serializable value, got '%s'", result)
		}
	})

	t.Run("debug disabled", func(t *testing.T) {
		dmOff := NewDebugMode(WithLevel(LevelOff))
		jsonFuncOff := debugJSON(dmOff)
		result := jsonFuncOff("test")
		if result != "" {
			t.Errorf("Expected empty result when debug disabled, got '%s'", result)
		}
	})
}

func TestDebugPretty(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	prettyFunc := debugPretty(dm)

	t.Run("pretty JSON formatting", func(t *testing.T) {
		input := map[string]interface{}{
			"name": "test",
			"age":  25,
			"settings": map[string]bool{
				"active": true,
			},
		}

		result := prettyFunc(input)

		if !strings.Contains(result, "  ") {
			t.Error("Expected pretty formatting with indentation")
		}
		if !strings.Contains(result, "name") || !strings.Contains(result, "test") {
			t.Error("Expected formatted JSON to contain input data")
		}
	})

	t.Run("nil value", func(t *testing.T) {
		result := prettyFunc(nil)
		if result != "null" {
			t.Errorf("Expected 'null', got '%s'", result)
		}
	})

	t.Run("debug disabled", func(t *testing.T) {
		dmOff := NewDebugMode(WithLevel(LevelOff))
		prettyFuncOff := debugPretty(dmOff)
		result := prettyFuncOff(map[string]int{"key": 1})
		if result != "" {
			t.Errorf("Expected empty result when debug disabled, got '%s'", result)
		}
	})
}

func TestDebugLog(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))
	logFunc := debugLog(dm)

	t.Run("basic logging", func(t *testing.T) {
		result := logFunc("Hello %s", "world")
		
		if result != "<!-- DEBUG: Hello world -->" {
			t.Errorf("Expected HTML comment format, got '%s'", result)
		}

		output := buf.String()
		if !strings.Contains(output, "template debug") {
			t.Error("Expected debug log to be written to logger")
		}
		if !strings.Contains(output, "Hello world") {
			t.Error("Expected debug log to contain formatted message")
		}
	})

	t.Run("debug disabled", func(t *testing.T) {
		dmOff := NewDebugMode(WithLevel(LevelOff))
		logFuncOff := debugLog(dmOff)
		result := logFuncOff("test")
		if result != "" {
			t.Errorf("Expected empty result when debug disabled, got '%s'", result)
		}
	})
}

func TestDebugTime(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	timeFunc := debugTime(dm)

	result := timeFunc()
	
	if len(result) != len("2006-01-02 15:04:05.000") {
		t.Errorf("Expected time format length %d, got %d", len("2006-01-02 15:04:05.000"), len(result))
	}

	if !strings.Contains(result, "-") || !strings.Contains(result, ":") || !strings.Contains(result, ".") {
		t.Errorf("Expected time format YYYY-MM-DD HH:MM:SS.mmm, got '%s'", result)
	}

	t.Run("debug disabled", func(t *testing.T) {
		dmOff := NewDebugMode(WithLevel(LevelOff))
		timeFuncOff := debugTime(dmOff)
		result := timeFuncOff()
		if result != "" {
			t.Errorf("Expected empty result when debug disabled, got '%s'", result)
		}
	})
}

func TestDebugStack(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelTrace))
	stackFunc := debugStack(dm)

	result := stackFunc()
	if !strings.Contains(result, "Stack trace not implemented") {
		t.Errorf("Expected placeholder message, got '%s'", result)
	}

	t.Run("trace disabled", func(t *testing.T) {
		dmDebug := NewDebugMode(WithLevel(LevelDebug))
		stackFuncDebug := debugStack(dmDebug)
		result := stackFuncDebug()
		if result != "" {
			t.Errorf("Expected empty result when trace disabled, got '%s'", result)
		}
	})
}

func TestDebugContext(t *testing.T) {
	dm := NewDebugMode(
		WithLevel(LevelDebug),
		WithProfiling(true),
		WithTracing(false),
		WithMetrics(true),
	)
	contextFunc := debugContext(dm)

	result := contextFunc()

	if result == nil {
		t.Fatal("Expected non-nil context map")
	}

	expectedKeys := []string{
		"debug_level",
		"uptime",
		"start_time",
		"profiling",
		"tracing",
		"metrics",
	}

	for _, key := range expectedKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("Expected context to contain key '%s'", key)
		}
	}

	if result["debug_level"] != "DEBUG" {
		t.Errorf("Expected debug_level to be 'DEBUG', got '%v'", result["debug_level"])
	}

	if result["profiling"] != true {
		t.Errorf("Expected profiling to be true, got %v", result["profiling"])
	}

	if result["tracing"] != false {
		t.Errorf("Expected tracing to be false, got %v", result["tracing"])
	}

	t.Run("debug disabled", func(t *testing.T) {
		dmOff := NewDebugMode(WithLevel(LevelOff))
		contextFuncOff := debugContext(dmOff)
		result := contextFuncOff()
		if result != nil {
			t.Errorf("Expected nil when debug disabled, got %v", result)
		}
	})
}

func TestNewTemplateDebugger(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	td := NewTemplateDebugger(dm)

	if td == nil {
		t.Fatal("Expected non-nil TemplateDebugger")
	}

	if td.debugMode != dm {
		t.Error("Expected debugger to reference the debug mode")
	}

	if td.templates == nil {
		t.Error("Expected templates map to be initialized")
	}

	if td.executions == nil {
		t.Error("Expected executions slice to be initialized")
	}

	if len(td.executions) != 0 {
		t.Errorf("Expected empty executions, got %d", len(td.executions))
	}
}

func TestTemplateDebugger_RegisterTemplate(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	td := NewTemplateDebugger(dm)

	tmpl, err := template.New("test").Parse("Hello {{.Name}}")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	td.RegisterTemplate("test", tmpl)

	if registered, exists := td.templates["test"]; !exists {
		t.Error("Expected template to be registered")
	} else if registered != tmpl {
		t.Error("Expected registered template to match original")
	}
}

func TestTemplateDebugger_ExecuteWithDebug(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))
	td := NewTemplateDebugger(dm)

	tmpl, err := template.New("test").Parse("Hello {{.Name}}!")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	data := map[string]interface{}{"Name": "World"}

	t.Run("successful execution", func(t *testing.T) {
		output, err := td.ExecuteWithDebug("test", tmpl, data)
		
		if err != nil {
			t.Errorf("Expected successful execution, got error: %v", err)
		}

		if output != "Hello World!" {
			t.Errorf("Expected 'Hello World!', got '%s'", output)
		}

		executions := td.GetExecutions()
		if len(executions) != 1 {
			t.Errorf("Expected 1 execution, got %d", len(executions))
		}

		exec := executions[0]
		if exec.Name != "test" {
			t.Errorf("Expected execution name 'test', got '%s'", exec.Name)
		}

		if exec.Error != "" {
			t.Errorf("Expected no error, got '%s'", exec.Error)
		}

		if exec.Output != "Hello World!" {
			t.Errorf("Expected output 'Hello World!', got '%s'", exec.Output)
		}

		logOutput := buf.String()
		if !strings.Contains(logOutput, "template executed successfully") {
			t.Error("Expected success log message")
		}
	})

	t.Run("failed execution", func(t *testing.T) {
		buf.Reset()
		td.ClearExecutions()

		badTmpl, err := template.New("bad").Parse("Hello {{.NonExistent.Field}}!")
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}

		_, err = td.ExecuteWithDebug("bad", badTmpl, data)

		if err == nil {
			t.Error("Expected execution to fail")
		}

		executions := td.GetExecutions()
		if len(executions) != 1 {
			t.Errorf("Expected 1 execution, got %d", len(executions))
		}

		exec := executions[0]
		if exec.Error == "" {
			t.Error("Expected error to be recorded")
		}

		logOutput := buf.String()
		if !strings.Contains(logOutput, "template execution failed") {
			t.Error("Expected failure log message")
		}
	})

	t.Run("trace level data capture", func(t *testing.T) {
		dmTrace := NewDebugMode(WithLevel(LevelTrace))
		tdTrace := NewTemplateDebugger(dmTrace)

		_, err := tdTrace.ExecuteWithDebug("trace", tmpl, data)
		if err != nil {
			t.Errorf("Expected successful execution, got error: %v", err)
		}

		executions := tdTrace.GetExecutions()
		if len(executions) != 1 {
			t.Fatal("Expected 1 execution")
		}

		exec := executions[0]
		if exec.Data == nil {
			t.Error("Expected data to be captured at trace level")
		}

		if exec.Data["Name"] != "World" {
			t.Errorf("Expected captured data to contain Name=World, got %v", exec.Data)
		}
	})

	t.Run("execution buffer overflow", func(t *testing.T) {
		td.ClearExecutions()

		for i := 0; i < 105; i++ {
			tmpl, _ := template.New(fmt.Sprintf("test%d", i)).Parse(fmt.Sprintf("Test %d", i))
			td.ExecuteWithDebug(fmt.Sprintf("test%d", i), tmpl, nil)
		}

		executions := td.GetExecutions()
		if len(executions) != 100 {
			t.Errorf("Expected buffer to be limited to 100 executions, got %d", len(executions))
		}

		if executions[0].Name == "test0" {
			t.Error("Expected oldest executions to be removed (FIFO behavior)")
		}
	})
}

func TestTemplateDebugger_GetExecutions(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	td := NewTemplateDebugger(dm)

	tmpl, _ := template.New("test").Parse("Hello {{.Name}}")
	data := map[string]interface{}{"Name": "World"}

	td.ExecuteWithDebug("test1", tmpl, data)
	td.ExecuteWithDebug("test2", tmpl, data)

	executions := td.GetExecutions()

	if len(executions) != 2 {
		t.Errorf("Expected 2 executions, got %d", len(executions))
	}

	executions[0].Name = "modified"

	unmodified := td.GetExecutions()
	if unmodified[0].Name == "modified" {
		t.Error("GetExecutions should return a copy, not the original slice")
	}
}

func TestTemplateDebugger_GetExecutionStats(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	td := NewTemplateDebugger(dm)

	t.Run("empty stats", func(t *testing.T) {
		stats := td.GetExecutionStats()
		
		if stats["total_executions"] != 0 {
			t.Errorf("Expected 0 total executions, got %v", stats["total_executions"])
		}
	})

	t.Run("with executions", func(t *testing.T) {
		successTmpl, _ := template.New("success").Parse("Success {{.Name}}")
		failTmpl, _ := template.New("fail").Parse("Fail {{.NonExistent}}")
		data := map[string]interface{}{"Name": "Test"}

		td.ExecuteWithDebug("success1", successTmpl, data)
		td.ExecuteWithDebug("success2", successTmpl, data)
		td.ExecuteWithDebug("fail1", failTmpl, data)

		stats := td.GetExecutionStats()

		if stats["total_executions"] != 3 {
			t.Errorf("Expected 3 total executions, got %v", stats["total_executions"])
		}

		if stats["success_count"] != 2 {
			t.Errorf("Expected 2 successful executions, got %v", stats["success_count"])
		}

		if stats["error_count"] != 1 {
			t.Errorf("Expected 1 failed execution, got %v", stats["error_count"])
		}

		successRate, ok := stats["success_rate"].(float64)
		if !ok || successRate < 0.66 || successRate > 0.67 {
			t.Errorf("Expected success rate around 0.667, got %v", successRate)
		}

		if stats["avg_duration"] == nil {
			t.Error("Expected avg_duration to be set")
		}

		if stats["total_duration"] == nil {
			t.Error("Expected total_duration to be set")
		}

		templateStats, ok := stats["template_stats"].(map[string]int)
		if !ok {
			t.Error("Expected template_stats to be map[string]int")
		} else {
			if templateStats["success1"] != 1 {
				t.Errorf("Expected success1 count to be 1, got %d", templateStats["success1"])
			}
			if templateStats["success2"] != 1 {
				t.Errorf("Expected success2 count to be 1, got %d", templateStats["success2"])
			}
			if templateStats["fail1"] != 1 {
				t.Errorf("Expected fail1 count to be 1, got %d", templateStats["fail1"])
			}
		}
	})
}

func TestTemplateDebugger_ClearExecutions(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	td := NewTemplateDebugger(dm)

	tmpl, _ := template.New("test").Parse("Test")
	td.ExecuteWithDebug("test", tmpl, nil)

	if len(td.GetExecutions()) != 1 {
		t.Error("Expected 1 execution before clear")
	}

	td.ClearExecutions()

	if len(td.GetExecutions()) != 0 {
		t.Error("Expected 0 executions after clear")
	}

	stats := td.GetExecutionStats()
	if stats["total_executions"] != 0 {
		t.Error("Expected stats to show 0 executions after clear")
	}
}

func TestTemplateDebugger_ConcurrentAccess(t *testing.T) {
	dm := NewDebugMode(WithLevel(LevelDebug))
	td := NewTemplateDebugger(dm)

	var wg sync.WaitGroup
	numGoroutines := 10
	executionsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tmpl, _ := template.New("concurrent").Parse(fmt.Sprintf("Goroutine %d: {{.Value}}", id))
			for j := 0; j < executionsPerGoroutine; j++ {
				data := map[string]interface{}{"Value": j}
				td.ExecuteWithDebug(fmt.Sprintf("test_%d_%d", id, j), tmpl, data)
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = td.GetExecutions()
			_ = td.GetExecutionStats()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	wg.Wait()

	executions := td.GetExecutions()
	if len(executions) != numGoroutines*executionsPerGoroutine {
		t.Errorf("Expected %d executions, got %d", numGoroutines*executionsPerGoroutine, len(executions))
	}

	stats := td.GetExecutionStats()
	if stats["total_executions"] != numGoroutines*executionsPerGoroutine {
		t.Errorf("Expected %d total executions in stats, got %v", numGoroutines*executionsPerGoroutine, stats["total_executions"])
	}
}

func TestDebugFuncMapIntegration(t *testing.T) {
	var buf bytes.Buffer
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))
	funcMap := CreateDebugFuncMap(dm)

	templateText := `
Debug Value: {{ debug .Name }}
Debug Type: {{ debugType .Age }}
Debug Size: {{ debugSize .Items }}
Debug JSON: {{ debugJSON .Config }}
Debug Keys: {{ debugKeys .Config }}
Current Time: {{ debugTime }}
{{ debugLog "Processing item: %s" .Name }}
`

	tmpl, err := template.New("test").Funcs(funcMap).Parse(templateText)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := map[string]interface{}{
		"Name":  "Test User",
		"Age":   25,
		"Items": []string{"item1", "item2", "item3"},
		"Config": map[string]bool{
			"enabled": true,
			"debug":   false,
		},
	}

	var output strings.Builder
	err = tmpl.Execute(&output, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	result := output.String()

	expectedContent := []string{
		`string(9): "Test User"`,
		"Type: int",
		"3",
		`{"debug":false,"enabled":true}`,
		"<!-- DEBUG: Processing item: Test User -->",
	}

	for _, content := range expectedContent {
		if !strings.Contains(result, content) {
			t.Errorf("Expected output to contain '%s', got:\n%s", content, result)
		}
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "template debug") {
		t.Error("Expected debugLog to write to logger")
	}
}

func TestDebugFuncMapWithDisabledDebug(t *testing.T) {
	dmOff := NewDebugMode(WithLevel(LevelOff))
	funcMap := CreateDebugFuncMap(dmOff)

	templateText := `
{{ debug .Name }}{{ debugType .Age }}{{ debugJSON .Items }}{{ debugLog "test" }}{{ debugTime }}
`

	tmpl, err := template.New("test").Funcs(funcMap).Parse(templateText)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := map[string]interface{}{
		"Name":  "Test",
		"Age":   25,
		"Items": []string{"a", "b"},
	}

	var output strings.Builder
	err = tmpl.Execute(&output, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	result := strings.TrimSpace(output.String())
	if result != "" {
		t.Errorf("Expected empty output when debug disabled, got '%s'", result)
	}
}
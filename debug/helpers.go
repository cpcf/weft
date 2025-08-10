package debug

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"
)

func CreateDebugFuncMap(debugMode *DebugMode) template.FuncMap {
	return template.FuncMap{
		"debug":        debugValue(debugMode),
		"debugType":    debugType(debugMode),
		"debugKeys":    debugKeys(debugMode),
		"debugSize":    debugSize(debugMode),
		"debugJSON":    debugJSON(debugMode),
		"debugPretty":  debugPretty(debugMode),
		"debugLog":     debugLog(debugMode),
		"debugTime":    debugTime(debugMode),
		"debugStack":   debugStack(debugMode),
		"debugContext": debugContext(debugMode),
	}
}

func debugValue(debugMode *DebugMode) func(any) string {
	return func(value any) string {
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		if value == nil {
			return "<nil>"
		}

		v := reflect.ValueOf(value)
		return formatValueByKind(v, value, debugMode)
	}
}

type valueFormatter func(reflect.Value, any, *DebugMode) string

var intKinds = map[reflect.Kind]bool{
	reflect.Int: true, reflect.Int8: true, reflect.Int16: true,
	reflect.Int32: true, reflect.Int64: true,
}

var uintKinds = map[reflect.Kind]bool{
	reflect.Uint: true, reflect.Uint8: true, reflect.Uint16: true,
	reflect.Uint32: true, reflect.Uint64: true,
}

var floatKinds = map[reflect.Kind]bool{
	reflect.Float32: true, reflect.Float64: true,
}

func getTypeFormatters() map[reflect.Kind]valueFormatter {
	return map[reflect.Kind]valueFormatter{
		reflect.String: formatStringValueWithContext,
		reflect.Bool:   formatBoolValueWithContext,
		reflect.Slice:  formatSliceArrayValueWithContext,
		reflect.Array:  formatSliceArrayValueWithContext,
		reflect.Map:    formatMapValueWithContext,
		reflect.Struct: formatStructValueWithContext,
		reflect.Ptr:    formatPtrValueWithContext,
	}
}

func formatValueByKind(v reflect.Value, value any, debugMode *DebugMode) string {
	kind := v.Kind()

	// Check specific type formatters first
	typeFormatters := getTypeFormatters()
	if formatter, exists := typeFormatters[kind]; exists {
		return formatter(v, value, debugMode)
	}

	// Handle numeric types with maps for better performance
	if intKinds[kind] {
		return formatIntValue(v)
	}
	if uintKinds[kind] {
		return formatUintValue(v)
	}
	if floatKinds[kind] {
		return formatFloatValue(v)
	}

	return formatDefaultValue(v, value)
}

func formatStringValue(v reflect.Value) string {
	return fmt.Sprintf("string(%d): %q", len(v.String()), v.String())
}

func formatIntValue(v reflect.Value) string {
	return fmt.Sprintf("int: %d", v.Int())
}

func formatUintValue(v reflect.Value) string {
	return fmt.Sprintf("uint: %d", v.Uint())
}

func formatFloatValue(v reflect.Value) string {
	return fmt.Sprintf("float: %g", v.Float())
}

func formatBoolValue(v reflect.Value) string {
	return fmt.Sprintf("bool: %t", v.Bool())
}

func formatSliceArrayValue(v reflect.Value, value any) string {
	return fmt.Sprintf("%s[%d]: %v", v.Type(), v.Len(), value)
}

func formatMapValue(v reflect.Value, value any) string {
	return fmt.Sprintf("%s{%d keys}: %v", v.Type(), v.Len(), value)
}

func formatStructValue(v reflect.Value, value any) string {
	return fmt.Sprintf("%s: %+v", v.Type(), value)
}

func formatPtrValue(v reflect.Value, debugMode *DebugMode) string {
	if v.IsNil() {
		return fmt.Sprintf("%s: <nil>", v.Type())
	}
	return fmt.Sprintf("%s: -> %s", v.Type(), debugValue(debugMode)(v.Elem().Interface()))
}

// Context-aware formatters that match the valueFormatter signature
func formatStringValueWithContext(v reflect.Value, value any, debugMode *DebugMode) string {
	return formatStringValue(v)
}

func formatBoolValueWithContext(v reflect.Value, value any, debugMode *DebugMode) string {
	return formatBoolValue(v)
}

func formatSliceArrayValueWithContext(v reflect.Value, value any, debugMode *DebugMode) string {
	return formatSliceArrayValue(v, value)
}

func formatMapValueWithContext(v reflect.Value, value any, debugMode *DebugMode) string {
	return formatMapValue(v, value)
}

func formatStructValueWithContext(v reflect.Value, value any, debugMode *DebugMode) string {
	return formatStructValue(v, value)
}

func formatPtrValueWithContext(v reflect.Value, value any, debugMode *DebugMode) string {
	return formatPtrValue(v, debugMode)
}

func formatDefaultValue(v reflect.Value, value any) string {
	return fmt.Sprintf("%s: %v", v.Type(), value)
}

func debugType(debugMode *DebugMode) func(any) string {
	return func(value any) string {
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		if value == nil {
			return "<nil>"
		}

		t := reflect.TypeOf(value)
		v := reflect.ValueOf(value)

		info := fmt.Sprintf("Type: %s, Kind: %s", t.String(), v.Kind().String())

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				info += " (nil pointer)"
			} else {
				elem := v.Elem()
				info += fmt.Sprintf(" -> Type: %s, Kind: %s", elem.Type().String(), elem.Kind().String())
			}
		}

		return info
	}
}

func debugKeys(debugMode *DebugMode) func(any) []string {
	return func(value any) []string {
		if !debugMode.IsEnabled(LevelDebug) {
			return nil
		}

		if value == nil {
			return nil
		}

		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Map:
			keys := make([]string, 0, v.Len())
			for _, key := range v.MapKeys() {
				keys = append(keys, fmt.Sprintf("%v", key.Interface()))
			}
			sort.Strings(keys)
			return keys
		case reflect.Struct:
			t := v.Type()
			fields := make([]string, 0, t.NumField())
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				if field.IsExported() {
					fields = append(fields, field.Name)
				}
			}
			sort.Strings(fields)
			return fields
		default:
			return nil
		}
	}
}

func debugSize(debugMode *DebugMode) func(any) int {
	return func(value any) int {
		if !debugMode.IsEnabled(LevelDebug) {
			return 0
		}

		if value == nil {
			return 0
		}

		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.String:
			return len(v.String())
		case reflect.Slice, reflect.Array, reflect.Map:
			return v.Len()
		default:
			return 1
		}
	}
}

func debugJSON(debugMode *DebugMode) func(any) string {
	return func(value any) string {
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		if value == nil {
			return "null"
		}

		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("JSON Error: %v", err)
		}

		return string(data)
	}
}

func debugPretty(debugMode *DebugMode) func(any) string {
	return func(value any) string {
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		if value == nil {
			return "null"
		}

		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Sprintf("JSON Error: %v", err)
		}

		return string(data)
	}
}

func debugLog(debugMode *DebugMode) func(string, ...any) string {
	return func(message string, args ...any) string {
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		formattedMessage := fmt.Sprintf(message, args...)
		debugMode.Debug("template debug", "message", formattedMessage)
		return fmt.Sprintf("<!-- DEBUG: %s -->", formattedMessage)
	}
}

func debugTime(debugMode *DebugMode) func() string {
	return func() string {
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		return time.Now().Format("2006-01-02 15:04:05.000")
	}
}

func debugStack(debugMode *DebugMode) func() string {
	return func() string {
		if !debugMode.IsEnabled(LevelTrace) {
			return ""
		}

		return "<!-- Stack trace not implemented -->"
	}
}

func debugContext(debugMode *DebugMode) func() map[string]any {
	return func() map[string]any {
		if !debugMode.IsEnabled(LevelDebug) {
			return nil
		}

		stats := debugMode.GetStats()
		return map[string]any{
			"debug_level": stats.Level.String(),
			"uptime":      stats.Uptime.String(),
			"start_time":  stats.StartTime.Format(time.RFC3339),
			"profiling":   stats.ProfilingEnabled,
			"tracing":     stats.TracingEnabled,
			"metrics":     stats.MetricsEnabled,
		}
	}
}

type TemplateDebugger struct {
	debugMode  *DebugMode
	templates  map[string]*template.Template
	executions []TemplateExecution
	mu         sync.RWMutex
}

type TemplateExecution struct {
	Name      string         `json:"name"`
	StartTime time.Time      `json:"start_time"`
	Duration  time.Duration  `json:"duration"`
	Data      map[string]any `json:"data"`
	Error     string         `json:"error,omitempty"`
	Output    string         `json:"output,omitempty"`
}

func NewTemplateDebugger(debugMode *DebugMode) *TemplateDebugger {
	return &TemplateDebugger{
		debugMode:  debugMode,
		templates:  make(map[string]*template.Template),
		executions: make([]TemplateExecution, 0),
	}
}

func (td *TemplateDebugger) RegisterTemplate(name string, tmpl *template.Template) {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.templates[name] = tmpl
}

func (td *TemplateDebugger) ExecuteWithDebug(name string, tmpl *template.Template, data any) (string, error) {
	startTime := time.Now()

	execution := TemplateExecution{
		Name:      name,
		StartTime: startTime,
	}

	if td.debugMode.IsEnabled(LevelTrace) {
		if dataMap, ok := data.(map[string]any); ok {
			execution.Data = dataMap
		} else {
			dataJSON, _ := json.Marshal(data)
			var dataMap map[string]any
			json.Unmarshal(dataJSON, &dataMap)
			execution.Data = dataMap
		}
	}

	var output strings.Builder
	err := tmpl.Execute(&output, data)

	execution.Duration = time.Since(startTime)
	execution.Output = output.String()

	if err != nil {
		execution.Error = err.Error()
		td.debugMode.Error("template execution failed",
			"name", name,
			"duration", execution.Duration,
			"error", err)
	} else {
		td.debugMode.Debug("template executed successfully",
			"name", name,
			"duration", execution.Duration,
			"output_size", len(execution.Output))
	}

	td.mu.Lock()
	td.executions = append(td.executions, execution)
	if len(td.executions) > GetConfig().ExecutionBufferSize {
		td.executions = td.executions[1:]
	}
	td.mu.Unlock()

	return output.String(), err
}

func (td *TemplateDebugger) GetExecutions() []TemplateExecution {
	td.mu.RLock()
	defer td.mu.RUnlock()

	executions := make([]TemplateExecution, len(td.executions))
	copy(executions, td.executions)
	return executions
}

func (td *TemplateDebugger) GetExecutionStats() map[string]any {
	td.mu.RLock()
	defer td.mu.RUnlock()

	if len(td.executions) == 0 {
		return map[string]any{
			"total_executions": 0,
		}
	}

	var totalDuration time.Duration
	var successCount, errorCount int
	templateStats := make(map[string]int)

	for _, exec := range td.executions {
		totalDuration += exec.Duration
		templateStats[exec.Name]++

		if exec.Error != "" {
			errorCount++
		} else {
			successCount++
		}
	}

	avgDuration := totalDuration / time.Duration(len(td.executions))

	return map[string]any{
		"total_executions": len(td.executions),
		"success_count":    successCount,
		"error_count":      errorCount,
		"success_rate":     float64(successCount) / float64(len(td.executions)),
		"avg_duration":     avgDuration.String(),
		"total_duration":   totalDuration.String(),
		"template_stats":   templateStats,
	}
}

func (td *TemplateDebugger) ClearExecutions() {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.executions = make([]TemplateExecution, 0)
}

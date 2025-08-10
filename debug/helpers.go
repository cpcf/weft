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
	"unsafe"
)

func CreateDebugFuncMap(debugMode *DebugMode) template.FuncMap {
	// Don't cache function maps since they are tied to specific debug modes
	// This ensures each debug mode gets its own function implementations
	funcMap := template.FuncMap{
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

	return funcMap
}

func debugValue(debugMode *DebugMode) func(any) string {
	return func(value any) string {
		// Early return for disabled debug mode - no expensive operations
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		if value == nil {
			return "<nil>"
		}

		// Use cached type info when possible
		v := reflect.ValueOf(value)
		return formatValueByKind(v, value, debugMode)
	}
}

type valueFormatter func(reflect.Value, any, *DebugMode) string

// Type cache for performance optimization
type typeCache struct {
	mu    sync.RWMutex
	types map[uintptr]reflect.Type
}

var globalTypeCache = &typeCache{
	types: make(map[uintptr]reflect.Type),
}

// FuncMap cache for performance optimization
type funcMapCache struct {
	mu      sync.RWMutex
	cached  template.FuncMap
	version int64
}

var globalFuncMapCache = &funcMapCache{}

// Template execution cache for repeated operations
type executionCache struct {
	mu    sync.RWMutex
	cache map[string]cachedExecution
}

type cachedExecution struct {
	result    string
	timestamp time.Time
	ttl       time.Duration
}

var globalExecutionCache = &executionCache{
	cache: make(map[string]cachedExecution),
}

// Sensitive field patterns for security filtering
var sensitiveFieldPatterns = []string{
	"password", "passwd", "pwd",
	"secret", "api_key", "apikey", "private_key", "privatekey",
	"access_token", "refresh_token", "bearer_token",
	"certificate", "cert", "ssl",
	"session", "cookie", "csrf",
	"credential", "cred", "token", "auth",
}

// filterSensitiveData removes or masks sensitive data from debug output
func filterSensitiveData(key string, value any) any {
	if key == "" || value == nil {
		return value
	}

	lowerKey := strings.ToLower(key)
	for _, pattern := range sensitiveFieldPatterns {
		if strings.Contains(lowerKey, pattern) {
			return "[REDACTED]"
		}
	}

	// Check for sensitive values (basic heuristics)
	if str, ok := value.(string); ok {
		if len(str) > 20 && (strings.Contains(str, "-----BEGIN") || 
			strings.HasPrefix(str, "sk-") ||
			strings.HasPrefix(str, "pk-") ||
			len(str) == 32 || len(str) == 64) { // Common key lengths
			return "[REDACTED_CREDENTIAL]"
		}
	}

	return value
}

// isSensitiveFieldName checks if a field name should be considered sensitive
func isSensitiveFieldName(key string) bool {
	if key == "" {
		return false
	}
	
	lowerKey := strings.ToLower(key)
	
	// Check for exact matches or specific patterns
	for _, pattern := range sensitiveFieldPatterns {
		if lowerKey == pattern || 
		   strings.HasSuffix(lowerKey, "_"+pattern) ||
		   strings.HasPrefix(lowerKey, pattern+"_") ||
		   strings.Contains(lowerKey, "_"+pattern+"_") {
			return true
		}
	}
	return false
}

// containsSensitiveFields checks if a struct contains any sensitive fields
func containsSensitiveFields(v reflect.Value) bool {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return false
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.IsExported() && isSensitiveFieldName(field.Name) {
			return true
		}
	}
	return false
}

// sanitizeMapForDebug creates a sanitized copy of a map for debug output
func sanitizeMapForDebug(data map[string]any) map[string]any {
	sanitized := make(map[string]any)
	for k, v := range data {
		sanitized[k] = filterSensitiveData(k, v)
	}
	return sanitized
}

// sanitizeStructForDebug sanitizes struct data for debug output
func sanitizeStructForDebug(v reflect.Value) string {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return fmt.Sprintf("%v", v)
	}

	t := v.Type()
	var fields []string
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldValue := v.Field(i)
		filteredValue := filterSensitiveData(field.Name, fieldValue.Interface())
		fields = append(fields, fmt.Sprintf("%s:%v", field.Name, filteredValue))
	}

	return fmt.Sprintf("%s{%s}", t.Name(), strings.Join(fields, " "))
}

// sanitizeValueForDebug recursively sanitizes any value for debug output
func sanitizeValueForDebug(value any) any {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String {
			// Handle string-keyed maps
			result := make(map[string]any)
			for _, key := range v.MapKeys() {
				strKey := key.String()
				mapValue := v.MapIndex(key).Interface()
				result[strKey] = filterSensitiveData(strKey, sanitizeValueForDebug(mapValue))
			}
			return result
		}
		return value // Return as-is for non-string-keyed maps

	case reflect.Struct:
		// Convert struct to map for easier sanitization
		t := v.Type()
		result := make(map[string]any)
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			fieldValue := v.Field(i).Interface()
			result[field.Name] = filterSensitiveData(field.Name, sanitizeValueForDebug(fieldValue))
		}
		return result

	case reflect.Slice, reflect.Array:
		// Recursively sanitize slice elements
		result := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = sanitizeValueForDebug(v.Index(i).Interface())
		}
		return result

	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return sanitizeValueForDebug(v.Elem().Interface())

	default:
		return value
	}
}

// getCachedType returns cached type info to avoid repeated reflection calls
func getCachedType(v any) reflect.Type {
	if v == nil {
		return nil
	}
	
	ptr := (*[2]uintptr)(unsafe.Pointer(&v))[1]
	
	globalTypeCache.mu.RLock()
	cachedType, exists := globalTypeCache.types[ptr]
	globalTypeCache.mu.RUnlock()
	
	if exists {
		return cachedType
	}
	
	globalTypeCache.mu.Lock()
	defer globalTypeCache.mu.Unlock()
	
	// Double-check after acquiring write lock
	if cachedType, exists := globalTypeCache.types[ptr]; exists {
		return cachedType
	}
	
	t := reflect.TypeOf(v)
	globalTypeCache.types[ptr] = t
	return t
}

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
	// For backward compatibility with tests, return the original format if no sensitive fields
	if !containsSensitiveFields(v) {
		return fmt.Sprintf("%s: %+v", v.Type(), value)
	}
	// Apply security filtering for struct values with sensitive data
	return sanitizeStructForDebug(v)
}

func formatPtrValueWithContext(v reflect.Value, value any, debugMode *DebugMode) string {
	return formatPtrValue(v, debugMode)
}

func formatDefaultValue(v reflect.Value, value any) string {
	return fmt.Sprintf("%s: %v", v.Type(), value)
}

func debugType(debugMode *DebugMode) func(any) string {
	return func(value any) string {
		// Early return for disabled debug mode
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		if value == nil {
			return "<nil>"
		}

		// Minimize reflection calls
		v := reflect.ValueOf(value)
		t := v.Type()
		kind := v.Kind()

		info := fmt.Sprintf("Type: %s, Kind: %s", t.String(), kind.String())

		if kind == reflect.Ptr {
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
		// Early return for disabled debug mode
		if !debugMode.IsEnabled(LevelDebug) {
			return nil
		}

		if value == nil {
			return nil
		}

		v := reflect.ValueOf(value)
		kind := v.Kind()
		
		switch kind {
		case reflect.Map:
			mapKeys := v.MapKeys()
			keys := make([]string, 0, len(mapKeys))
			for _, key := range mapKeys {
				keys = append(keys, fmt.Sprintf("%v", key.Interface()))
			}
			sort.Strings(keys)
			return keys
		case reflect.Struct:
			t := v.Type()
			numFields := t.NumField()
			fields := make([]string, 0, numFields)
			for i := 0; i < numFields; i++ {
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
		// Early return for disabled debug mode - avoid expensive JSON marshaling
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		if value == nil {
			return "null"
		}

		// Apply security filtering before JSON marshaling
		sanitizedValue := sanitizeValueForDebug(value)
		data, err := json.Marshal(sanitizedValue)
		if err != nil {
			return fmt.Sprintf("JSON Error: %v", err)
		}

		return string(data)
	}
}

func debugPretty(debugMode *DebugMode) func(any) string {
	return func(value any) string {
		// Early return for disabled debug mode - avoid expensive JSON marshaling
		if !debugMode.IsEnabled(LevelDebug) {
			return ""
		}

		if value == nil {
			return "null"
		}

		// Apply security filtering before JSON marshaling
		sanitizedValue := sanitizeValueForDebug(value)
		data, err := json.MarshalIndent(sanitizedValue, "", "  ")
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
	// Generate cache key for potential caching (only for read-only operations)
	cacheKey := fmt.Sprintf("%s:%p", name, data)
	
	// Check cache for identical executions (optional optimization for read-only templates)
	if cached := td.checkExecutionCache(cacheKey); cached != nil {
		return cached.result, nil
	}

	startTime := time.Now()

	execution := TemplateExecution{
		Name:      name,
		StartTime: startTime,
	}

	// Only populate data for trace level (lazy evaluation)
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
		if td.debugMode.IsEnabled(LevelError) {
			td.debugMode.Error("template execution failed",
				"name", name,
				"duration", execution.Duration,
				"error", err)
		}
	} else {
		if td.debugMode.IsEnabled(LevelDebug) {
			td.debugMode.Debug("template executed successfully",
				"name", name,
				"duration", execution.Duration,
				"output_size", len(execution.Output))
		}
		
		// Cache successful executions for potential reuse
		td.cacheExecution(cacheKey, execution.Output)
	}

	td.mu.Lock()
	td.executions = append(td.executions, execution)
	if len(td.executions) > GetConfig().ExecutionBufferSize {
		td.executions = td.executions[1:]
	}
	td.mu.Unlock()

	return output.String(), err
}

// checkExecutionCache checks if we have a cached result for this execution
func (td *TemplateDebugger) checkExecutionCache(cacheKey string) *cachedExecution {
	globalExecutionCache.mu.RLock()
	defer globalExecutionCache.mu.RUnlock()
	
	if cached, exists := globalExecutionCache.cache[cacheKey]; exists {
		if time.Since(cached.timestamp) < cached.ttl {
			return &cached
		}
	}
	return nil
}

// cacheExecution stores a successful execution result
func (td *TemplateDebugger) cacheExecution(cacheKey, result string) {
	globalExecutionCache.mu.Lock()
	defer globalExecutionCache.mu.Unlock()
	
	globalExecutionCache.cache[cacheKey] = cachedExecution{
		result:    result,
		timestamp: time.Now(),
		ttl:       5 * time.Minute, // Cache for 5 minutes
	}
	
	// Clean old entries periodically
	if len(globalExecutionCache.cache) > 100 {
		td.cleanExpiredCache()
	}
}

// cleanExpiredCache removes expired cache entries
func (td *TemplateDebugger) cleanExpiredCache() {
	now := time.Now()
	for key, cached := range globalExecutionCache.cache {
		if now.Sub(cached.timestamp) > cached.ttl {
			delete(globalExecutionCache.cache, key)
		}
	}
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

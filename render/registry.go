package render

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"
)

type FunctionRegistry struct {
	mu        sync.RWMutex
	functions map[string]interface{}
	metadata  map[string]FunctionMetadata
}

type FunctionMetadata struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Parameters  []ParamInfo `json:"parameters"`
	ReturnType  string    `json:"return_type"`
	Examples    []string  `json:"examples"`
	Since       string    `json:"since"`
	Deprecated  bool      `json:"deprecated"`
	AddedAt     time.Time `json:"added_at"`
}

type ParamInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

type FunctionOption func(*FunctionMetadata)

func NewFunctionRegistry() *FunctionRegistry {
	return &FunctionRegistry{
		functions: make(map[string]interface{}),
		metadata:  make(map[string]FunctionMetadata),
	}
}

func WithDescription(description string) FunctionOption {
	return func(meta *FunctionMetadata) {
		meta.Description = description
	}
}

func WithCategory(category string) FunctionOption {
	return func(meta *FunctionMetadata) {
		meta.Category = category
	}
}

func WithParameters(params ...ParamInfo) FunctionOption {
	return func(meta *FunctionMetadata) {
		meta.Parameters = params
	}
}

func WithReturnType(returnType string) FunctionOption {
	return func(meta *FunctionMetadata) {
		meta.ReturnType = returnType
	}
}

func WithExamples(examples ...string) FunctionOption {
	return func(meta *FunctionMetadata) {
		meta.Examples = examples
	}
}

func WithSince(version string) FunctionOption {
	return func(meta *FunctionMetadata) {
		meta.Since = version
	}
}

func WithDeprecated() FunctionOption {
	return func(meta *FunctionMetadata) {
		meta.Deprecated = true
	}
}

func (fr *FunctionRegistry) Register(name string, fn interface{}, opts ...FunctionOption) error {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	if fn == nil {
		return fmt.Errorf("function cannot be nil")
	}

	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return fmt.Errorf("expected function, got %T", fn)
	}

	meta := FunctionMetadata{
		Name:       name,
		Category:   "general",
		ReturnType: "interface{}",
		AddedAt:    time.Now(),
	}

	for _, opt := range opts {
		opt(&meta)
	}

	if meta.Parameters == nil {
		meta.Parameters = fr.inferParameters(fnValue)
	}

	fr.functions[name] = fn
	fr.metadata[name] = meta

	return nil
}

func (fr *FunctionRegistry) Unregister(name string) {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	delete(fr.functions, name)
	delete(fr.metadata, name)
}

func (fr *FunctionRegistry) Get(name string) (interface{}, bool) {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	fn, exists := fr.functions[name]
	return fn, exists
}

func (fr *FunctionRegistry) GetMetadata(name string) (FunctionMetadata, bool) {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	meta, exists := fr.metadata[name]
	return meta, exists
}

func (fr *FunctionRegistry) List() []string {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	names := make([]string, 0, len(fr.functions))
	for name := range fr.functions {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

func (fr *FunctionRegistry) ListByCategory() map[string][]string {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	categories := make(map[string][]string)
	
	for name, meta := range fr.metadata {
		category := meta.Category
		if category == "" {
			category = "general"
		}
		categories[category] = append(categories[category], name)
	}

	for category := range categories {
		sort.Strings(categories[category])
	}

	return categories
}

func (fr *FunctionRegistry) GetFuncMap() template.FuncMap {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	funcMap := make(template.FuncMap)
	for name, fn := range fr.functions {
		funcMap[name] = fn
	}

	return funcMap
}

func (fr *FunctionRegistry) MergeFuncMap(external template.FuncMap) template.FuncMap {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	funcMap := make(template.FuncMap)
	
	for name, fn := range fr.functions {
		funcMap[name] = fn
	}

	for name, fn := range external {
		funcMap[name] = fn
	}

	return funcMap
}

func (fr *FunctionRegistry) RegisterDefaults() {
	defaultFuncs := DefaultFuncMap()
	
	fr.Register("snake", defaultFuncs["snake"],
		WithDescription("Convert string to snake_case"),
		WithCategory("string"),
		WithParameters(ParamInfo{Name: "input", Type: "string", Required: true}),
		WithReturnType("string"),
		WithExamples(`{{ "HelloWorld" | snake }} // hello_world`),
		WithSince("1.0.0"))

	fr.Register("camel", defaultFuncs["camel"],
		WithDescription("Convert string to camelCase"),
		WithCategory("string"),
		WithParameters(ParamInfo{Name: "input", Type: "string", Required: true}),
		WithReturnType("string"),
		WithExamples(`{{ "hello_world" | camel }} // helloWorld`),
		WithSince("1.0.0"))

	fr.Register("pascal", defaultFuncs["pascal"],
		WithDescription("Convert string to PascalCase"),
		WithCategory("string"),
		WithParameters(ParamInfo{Name: "input", Type: "string", Required: true}),
		WithReturnType("string"),
		WithExamples(`{{ "hello_world" | pascal }} // HelloWorld`),
		WithSince("1.0.0"))

	fr.Register("kebab", defaultFuncs["kebab"],
		WithDescription("Convert string to kebab-case"),
		WithCategory("string"),
		WithParameters(ParamInfo{Name: "input", Type: "string", Required: true}),
		WithReturnType("string"),
		WithExamples(`{{ "HelloWorld" | kebab }} // hello-world`),
		WithSince("1.0.0"))

	fr.Register("formatSlice", defaultFuncs["formatSlice"],
		WithDescription("Format slice elements with separator and format string"),
		WithCategory("collection"),
		WithParameters(
			ParamInfo{Name: "slice", Type: "[]interface{}", Required: true},
			ParamInfo{Name: "separator", Type: "string", Required: true},
			ParamInfo{Name: "format", Type: "string", Required: false},
		),
		WithReturnType("string"),
		WithExamples(`{{ formatSlice .Items ", " "%s" }}`),
		WithSince("1.0.0"))

	fr.Register("filter", defaultFuncs["filter"],
		WithDescription("Filter slice elements by predicate function"),
		WithCategory("collection"),
		WithParameters(
			ParamInfo{Name: "slice", Type: "[]interface{}", Required: true},
			ParamInfo{Name: "predicate", Type: "func(interface{}) bool", Required: true},
		),
		WithReturnType("[]interface{}"),
		WithSince("1.0.0"))

	fr.Register("plural", defaultFuncs["plural"],
		WithDescription("Convert word to plural form"),
		WithCategory("string"),
		WithParameters(ParamInfo{Name: "word", Type: "string", Required: true}),
		WithReturnType("string"),
		WithExamples(`{{ "person" | plural }} // people`),
		WithSince("1.0.0"))

	fr.Register("singular", defaultFuncs["singular"],
		WithDescription("Convert word to singular form"),
		WithCategory("string"),
		WithParameters(ParamInfo{Name: "word", Type: "string", Required: true}),
		WithReturnType("string"),
		WithExamples(`{{ "people" | singular }} // person`),
		WithSince("1.0.0"))

	fr.Register("default", defaultFuncs["default"],
		WithDescription("Return default value if given value is nil or empty"),
		WithCategory("utility"),
		WithParameters(
			ParamInfo{Name: "default", Type: "interface{}", Required: true},
			ParamInfo{Name: "value", Type: "interface{}", Required: true},
		),
		WithReturnType("interface{}"),
		WithExamples(`{{ default "unknown" .Name }}`),
		WithSince("1.0.0"))

	fr.Register("add", defaultFuncs["add"],
		WithDescription("Add two numbers"),
		WithCategory("math"),
		WithParameters(
			ParamInfo{Name: "a", Type: "number", Required: true},
			ParamInfo{Name: "b", Type: "number", Required: true},
		),
		WithReturnType("number"),
		WithExamples(`{{ add 5 3 }} // 8`),
		WithSince("1.0.0"))

	fr.Register("now", defaultFuncs["now"],
		WithDescription("Get current time"),
		WithCategory("time"),
		WithReturnType("time.Time"),
		WithExamples(`{{ now.Format "2006-01-02" }}`),
		WithSince("1.0.0"))
}

func (fr *FunctionRegistry) RegisterExtended() {
	extendedFuncs := ExtendedFuncMap()
	
	fr.Register("uuid", extendedFuncs["uuid"],
		WithDescription("Generate a random UUID"),
		WithCategory("utility"),
		WithReturnType("string"),
		WithExamples(`{{ uuid }} // 550e8400-e29b-41d4-a716-446655440000`),
		WithSince("1.1.0"))

	fr.Register("md5", extendedFuncs["md5"],
		WithDescription("Calculate MD5 hash of string"),
		WithCategory("crypto"),
		WithParameters(ParamInfo{Name: "input", Type: "string", Required: true}),
		WithReturnType("string"),
		WithExamples(`{{ "hello" | md5 }}`),
		WithSince("1.1.0"))

	fr.Register("base64", extendedFuncs["base64"],
		WithDescription("Encode string to base64"),
		WithCategory("encoding"),
		WithParameters(ParamInfo{Name: "input", Type: "string", Required: true}),
		WithReturnType("string"),
		WithExamples(`{{ "hello" | base64 }}`),
		WithSince("1.1.0"))

	fr.Register("env", extendedFuncs["env"],
		WithDescription("Get environment variable value"),
		WithCategory("system"),
		WithParameters(ParamInfo{Name: "name", Type: "string", Required: true}),
		WithReturnType("string"),
		WithExamples(`{{ env "HOME" }}`),
		WithSince("1.1.0"))

	fr.Register("regexMatch", extendedFuncs["regexMatch"],
		WithDescription("Test if string matches regex pattern"),
		WithCategory("regex"),
		WithParameters(
			ParamInfo{Name: "pattern", Type: "string", Required: true},
			ParamInfo{Name: "text", Type: "string", Required: true},
		),
		WithReturnType("bool"),
		WithExamples(`{{ regexMatch "^[a-z]+$" "hello" }}`),
		WithSince("1.1.0"))
}

func (fr *FunctionRegistry) inferParameters(fnValue reflect.Value) []ParamInfo {
	fnType := fnValue.Type()
	numIn := fnType.NumIn()
	
	var params []ParamInfo
	for i := 0; i < numIn; i++ {
		paramType := fnType.In(i)
		param := ParamInfo{
			Name:     fmt.Sprintf("arg%d", i+1),
			Type:     paramType.String(),
			Required: true,
		}
		params = append(params, param)
	}
	
	return params
}

func (fr *FunctionRegistry) ValidateFunction(name string, fn interface{}) error {
	if name == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	if fn == nil {
		return fmt.Errorf("function cannot be nil")
	}

	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return fmt.Errorf("expected function, got %T", fn)
	}

	return nil
}

func (fr *FunctionRegistry) GetDocumentation() string {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	var doc strings.Builder
	doc.WriteString("# Template Functions\n\n")

	categories := fr.ListByCategory()
	categoryOrder := []string{"string", "collection", "math", "time", "utility", "crypto", "encoding", "system", "regex", "general"}

	for _, category := range categoryOrder {
		if functions, exists := categories[category]; exists {
			doc.WriteString(fmt.Sprintf("## %s Functions\n\n", strings.Title(category)))
			
			for _, name := range functions {
				meta := fr.metadata[name]
				doc.WriteString(fmt.Sprintf("### %s\n\n", name))
				
				if meta.Description != "" {
					doc.WriteString(fmt.Sprintf("%s\n\n", meta.Description))
				}

				if meta.Deprecated {
					doc.WriteString("**⚠️ Deprecated**\n\n")
				}

				if len(meta.Parameters) > 0 {
					doc.WriteString("**Parameters:**\n")
					for _, param := range meta.Parameters {
						required := ""
						if param.Required {
							required = " (required)"
						}
						doc.WriteString(fmt.Sprintf("- `%s` (%s)%s: %s\n", 
							param.Name, param.Type, required, param.Description))
					}
					doc.WriteString("\n")
				}

				if meta.ReturnType != "" {
					doc.WriteString(fmt.Sprintf("**Returns:** %s\n\n", meta.ReturnType))
				}

				if len(meta.Examples) > 0 {
					doc.WriteString("**Examples:**\n")
					for _, example := range meta.Examples {
						doc.WriteString(fmt.Sprintf("```\n%s\n```\n\n", example))
					}
				}

				if meta.Since != "" {
					doc.WriteString(fmt.Sprintf("**Since:** %s\n\n", meta.Since))
				}

				doc.WriteString("---\n\n")
			}
		}
	}

	return doc.String()
}

func (fr *FunctionRegistry) ExportJSON() map[string]FunctionMetadata {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	result := make(map[string]FunctionMetadata)
	for name, meta := range fr.metadata {
		result[name] = meta
	}

	return result
}

func (fr *FunctionRegistry) ImportJSON(data map[string]FunctionMetadata) {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	for name, meta := range data {
		fr.metadata[name] = meta
	}
}

func (fr *FunctionRegistry) Count() int {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	return len(fr.functions)
}

func (fr *FunctionRegistry) HasFunction(name string) bool {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	_, exists := fr.functions[name]
	return exists
}

func (fr *FunctionRegistry) GetFunctionSignature(name string) string {
	meta, exists := fr.GetMetadata(name)
	if !exists {
		return ""
	}

	var sig strings.Builder
	sig.WriteString(name)
	sig.WriteString("(")

	for i, param := range meta.Parameters {
		if i > 0 {
			sig.WriteString(", ")
		}
		sig.WriteString(param.Name)
		if !param.Required {
			sig.WriteString("?")
		}
		sig.WriteString(" ")
		sig.WriteString(param.Type)
	}

	sig.WriteString(")")
	if meta.ReturnType != "" {
		sig.WriteString(" -> ")
		sig.WriteString(meta.ReturnType)
	}

	return sig.String()
}
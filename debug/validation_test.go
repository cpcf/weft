package debug

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"text/template"
)

func TestNewTemplateValidator(t *testing.T) {
	testFS := fstest.MapFS{}
	funcMap := template.FuncMap{"testFunc": func() string { return "test" }}
	dm := NewDebugMode(WithLevel(LevelDebug))

	validator := NewTemplateValidator(testFS, funcMap, dm)

	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}

	if validator.fs == nil {
		t.Error("Expected fs to be set")
	}

	if validator.funcMap == nil {
		t.Error("Expected funcMap to be set")
	}

	if validator.debugMode != dm {
		t.Error("Expected debugMode to be set")
	}

	if validator.strict {
		t.Error("Expected strict mode to be disabled by default")
	}
}

func TestTemplateValidator_SetStrict(t *testing.T) {
	validator := NewTemplateValidator(fstest.MapFS{}, nil, nil)

	if validator.strict {
		t.Error("Expected strict to be false initially")
	}

	validator.SetStrict(true)

	if !validator.strict {
		t.Error("Expected strict to be true after SetStrict(true)")
	}

	validator.SetStrict(false)

	if validator.strict {
		t.Error("Expected strict to be false after SetStrict(false)")
	}
}

func TestTemplateValidator_ValidateTemplate_FileErrors(t *testing.T) {
	testFS := fstest.MapFS{}
	validator := NewTemplateValidator(testFS, nil, nil)

	result := validator.ValidateTemplate("nonexistent.tmpl")

	if result.Valid {
		t.Error("Expected validation to fail for nonexistent file")
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	if result.Errors[0].Type != "file_error" {
		t.Errorf("Expected file_error type, got %s", result.Errors[0].Type)
	}

	if !strings.Contains(result.Errors[0].Message, "Cannot read template file") {
		t.Errorf("Expected file error message, got %s", result.Errors[0].Message)
	}
}

func TestTemplateValidator_ValidateSyntax(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantValid bool
		errorType string
		errorMsg  string
	}{
		{
			name:     "valid template",
			content:  "Hello {{.Name}}!",
			wantValid: true,
		},
		{
			name:     "syntax error - unclosed brace",
			content:  "Hello {{.Name!",
			wantValid: false,
			errorType: "syntax_error",
		},
		{
			name:     "syntax error - invalid function",
			content:  "Hello {{badfunction}}!",
			wantValid: false,
			errorType: "syntax_error",
		},
		{
			name:     "brace mismatch - extra closing",
			content:  "Hello {{.Name}}}}!",
			wantValid: false,
			errorType: "brace_mismatch",
		},
		{
			name:     "brace mismatch - extra opening",
			content:  "Hello {{{{.Name}}!",
			wantValid: false,
			errorType: "brace_mismatch",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFS := fstest.MapFS{
				"test.tmpl": &fstest.MapFile{
					Data: []byte(test.content),
				},
			}

			validator := NewTemplateValidator(testFS, nil, nil)
			result := validator.ValidateTemplate("test.tmpl")

			if result.Valid != test.wantValid {
				t.Errorf("Expected valid=%v, got valid=%v", test.wantValid, result.Valid)
			}

			if !test.wantValid {
				found := false
				for _, err := range result.Errors {
					if err.Type == test.errorType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error type %s, got errors: %v", test.errorType, result.Errors)
				}
			}
		})
	}
}

func TestTemplateValidator_ValidateBraceBalance(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectErrors int
		errorType    string
	}{
		{
			name:         "balanced braces",
			content:      "{{.Name}} and {{.Age}}",
			expectErrors: 0,
		},
		{
			name:         "unmatched opening",
			content:      "{{.Name}} and {{.Age",
			expectErrors: 1,
			errorType:    "brace_mismatch",
		},
		{
			name:         "unmatched closing",
			content:      "{{.Name}} and .Age}}",
			expectErrors: 1,
			errorType:    "brace_mismatch",
		},
		{
			name:         "multiple unmatched opening",
			content:      "{{{{.Name}} {{.Age",
			expectErrors: 1,
			errorType:    "brace_mismatch",
		},
		{
			name:         "multiple unmatched closing",
			content:      "{{.Name}}}} {{.Age}}}}",
			expectErrors: 2,
			errorType:    "brace_mismatch",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFS := fstest.MapFS{
				"test.tmpl": &fstest.MapFile{
					Data: []byte(test.content),
				},
			}

			validator := NewTemplateValidator(testFS, nil, nil)
			result := validator.ValidateTemplate("test.tmpl")

			errorCount := 0
			for _, err := range result.Errors {
				if err.Type == test.errorType {
					errorCount++
				}
			}

			if errorCount != test.expectErrors {
				t.Errorf("Expected %d %s errors, got %d", test.expectErrors, test.errorType, errorCount)
			}
		})
	}
}

func TestTemplateValidator_ValidateWhitespace(t *testing.T) {
	content := "{{.Name}}  \nHello\t\n{{.Age}}"

	testFS := fstest.MapFS{
		"test.tmpl": &fstest.MapFile{
			Data: []byte(content),
		},
	}

	t.Run("non-strict mode", func(t *testing.T) {
		validator := NewTemplateValidator(testFS, nil, nil)
		validator.SetStrict(false)
		result := validator.ValidateTemplate("test.tmpl")

		// Should not have whitespace warnings in non-strict mode
		whitespaceWarnings := 0
		for _, warning := range result.Warnings {
			if warning.Type == "whitespace_warning" {
				whitespaceWarnings++
			}
		}

		if whitespaceWarnings != 0 {
			t.Errorf("Expected 0 whitespace warnings in non-strict mode, got %d", whitespaceWarnings)
		}
	})

	t.Run("strict mode", func(t *testing.T) {
		validator := NewTemplateValidator(testFS, nil, nil)
		validator.SetStrict(true)
		result := validator.ValidateTemplate("test.tmpl")

		// Should have whitespace warnings in strict mode
		whitespaceWarnings := 0
		for _, warning := range result.Warnings {
			if warning.Type == "whitespace_warning" {
				whitespaceWarnings++
			}
		}

		if whitespaceWarnings != 2 {
			t.Errorf("Expected 2 whitespace warnings in strict mode, got %d", whitespaceWarnings)
		}
	})
}

func TestTemplateValidator_ValidateFunctions(t *testing.T) {
	funcMap := template.FuncMap{
		"upper":   strings.ToUpper,
		"myFunc": func() string { return "test" },
	}

	tests := []struct {
		name           string
		content        string
		expectedWarnings int
		expectedFuncName string
	}{
		{
			name:             "known function",
			content:          "{{upper .Name}}",
			expectedWarnings: 0,
		},
		{
			name:             "builtin function",
			content:          "{{printf \"%s\" .Name}}",
			expectedWarnings: 0,
		},
		{
			name:             "control structure",
			content:          "{{if .Active}}Yes{{end}}",
			expectedWarnings: 0,
		},
		{
			name:             "unknown function",
			content:          "{{unknownFunc .Name}}",
			expectedWarnings: 1,
			expectedFuncName: "unknownFunc",
		},
		{
			name:             "mixed functions",
			content:          "{{upper .Name}} {{unknownFunc .Age}} {{printf \"%d\" .Count}}",
			expectedWarnings: 1,
			expectedFuncName: "unknownFunc",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFS := fstest.MapFS{
				"test.tmpl": &fstest.MapFile{
					Data: []byte(test.content),
				},
			}

			validator := NewTemplateValidator(testFS, funcMap, nil)
			result := validator.ValidateTemplate("test.tmpl")

			unknownFuncWarnings := 0
			for _, warning := range result.Warnings {
				if warning.Type == "unknown_function" {
					unknownFuncWarnings++
					if test.expectedFuncName != "" && !strings.Contains(warning.Message, test.expectedFuncName) {
						t.Errorf("Expected warning to mention function '%s', got: %s", test.expectedFuncName, warning.Message)
					}
				}
			}

			if unknownFuncWarnings != test.expectedWarnings {
				t.Errorf("Expected %d unknown function warnings, got %d", test.expectedWarnings, unknownFuncWarnings)
			}
		})
	}
}

func TestTemplateValidator_ValidateVariableAccess(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedWarnings int
	}{
		{
			name:             "simple access",
			content:          "{{.Name}}",
			expectedWarnings: 0,
		},
		{
			name:             "nested access",
			content:          "{{.User.Name}}",
			expectedWarnings: 0,
		},
		{
			name:             "moderately deep access",
			content:          "{{.User.Profile.Settings.Theme}}",
			expectedWarnings: 0,
		},
		{
			name:             "very deep access",
			content:          "{{.User.Profile.Settings.Theme.Colors.Background}}",
			expectedWarnings: 1,
		},
		{
			name:             "extremely deep access",
			content:          "{{.Level1.Level2.Level3.Level4.Level5.Level6.Level7}}",
			expectedWarnings: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFS := fstest.MapFS{
				"test.tmpl": &fstest.MapFile{
					Data: []byte(test.content),
				},
			}

			validator := NewTemplateValidator(testFS, nil, nil)
			result := validator.ValidateTemplate("test.tmpl")

			deepAccessWarnings := 0
			for _, warning := range result.Warnings {
				if warning.Type == "deep_access" {
					deepAccessWarnings++
				}
			}

			if deepAccessWarnings != test.expectedWarnings {
				t.Errorf("Expected %d deep access warnings, got %d", test.expectedWarnings, deepAccessWarnings)
			}
		})
	}
}

func TestTemplateValidator_ValidatePartials(t *testing.T) {
	testFS := fstest.MapFS{
		"templates/main.tmpl": &fstest.MapFile{
			Data: []byte(`{{template "header"}} Content {{template "footer"}}`),
		},
		"templates/_header.tmpl": &fstest.MapFile{
			Data: []byte("<header>Header</header>"),
		},
		"_footer.tpl": &fstest.MapFile{
			Data: []byte("<footer>Footer</footer>"),
		},
		"missing_partial.tmpl": &fstest.MapFile{
			Data: []byte(`{{template "nonexistent"}}`),
		},
	}

	tests := []struct {
		name           string
		templatePath   string
		expectErrors   int
		expectValid    bool
	}{
		{
			name:         "existing partials",
			templatePath: "templates/main.tmpl",
			expectErrors: 2, // both header and footer will be missing in this simple test
			expectValid:  false,
		},
		{
			name:         "missing partial",
			templatePath: "missing_partial.tmpl",
			expectErrors: 1,
			expectValid:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewTemplateValidator(testFS, nil, nil)
			result := validator.ValidateTemplate(test.templatePath)

			if result.Valid != test.expectValid {
				t.Errorf("Expected valid=%v, got valid=%v", test.expectValid, result.Valid)
			}

			missingPartialErrors := 0
			for _, err := range result.Errors {
				if err.Type == "missing_partial" {
					missingPartialErrors++
				}
			}

			if missingPartialErrors != test.expectErrors {
				t.Errorf("Expected %d missing partial errors, got %d", test.expectErrors, missingPartialErrors)
			}
		})
	}
}

func TestTemplateValidator_ValidateIncludes(t *testing.T) {
	testFS := fstest.MapFS{
		"main.tmpl": &fstest.MapFile{
			Data: []byte(`{{include "existing.tmpl"}} {{include "nonexistent.tmpl"}}`),
		},
		"existing.tmpl": &fstest.MapFile{
			Data: []byte("Existing content"),
		},
		"includes/shared.tmpl": &fstest.MapFile{
			Data: []byte("Shared content"),
		},
		"with_includes.tmpl": &fstest.MapFile{
			Data: []byte(`{{include "shared"}}`),
		},
	}

	tests := []struct {
		name           string
		templatePath   string
		expectErrors   int
		expectValid    bool
	}{
		{
			name:         "mixed includes",
			templatePath: "main.tmpl",
			expectErrors: 1,
			expectValid:  false,
		},
		{
			name:         "include in subdirectory",
			templatePath: "with_includes.tmpl",
			expectErrors: 0,
			expectValid:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewTemplateValidator(testFS, nil, nil)
			result := validator.ValidateTemplate(test.templatePath)

			if result.Valid != test.expectValid {
				t.Errorf("Expected valid=%v, got valid=%v", test.expectValid, result.Valid)
			}

			missingIncludeErrors := 0
			for _, err := range result.Errors {
				if err.Type == "missing_include" {
					missingIncludeErrors++
				}
			}

			if missingIncludeErrors != test.expectErrors {
				t.Errorf("Expected %d missing include errors, got %d", test.expectErrors, missingIncludeErrors)
			}
		})
	}
}

func TestTemplateValidator_ResolvePartialPath(t *testing.T) {
	testFS := fstest.MapFS{
		"templates/_header.tmpl":     &fstest.MapFile{Data: []byte("header")},
		"templates/_footer.tpl":      &fstest.MapFile{Data: []byte("footer")},
		"_global.tmpl":               &fstest.MapFile{Data: []byte("global")},
		"_global.tpl":                &fstest.MapFile{Data: []byte("global2")},
	}

	validator := NewTemplateValidator(testFS, nil, nil)

	tests := []struct {
		templatePath string
		partialName  string
		expected     string
	}{
		{
			templatePath: "templates/main.tmpl",
			partialName:  "header",
			expected:     "templates/_header.tmpl",
		},
		{
			templatePath: "templates/main.tmpl",
			partialName:  "footer",
			expected:     "templates/_footer.tpl",
		},
		{
			templatePath: "main.tmpl",
			partialName:  "global",
			expected:     "_global.tmpl",
		},
		{
			templatePath: "main.tmpl",
			partialName:  "nonexistent",
			expected:     "",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s-%s", test.templatePath, test.partialName), func(t *testing.T) {
			result := validator.resolvePartialPath(test.templatePath, test.partialName)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestTemplateValidator_ResolveIncludePath(t *testing.T) {
	testFS := fstest.MapFS{
		"shared.tmpl":           &fstest.MapFile{Data: []byte("shared")},
		"templates/local.tmpl":  &fstest.MapFile{Data: []byte("local")},
		"includes/common.tmpl":  &fstest.MapFile{Data: []byte("common")},
		"templates/nested.tpl":  &fstest.MapFile{Data: []byte("nested")},
	}

	validator := NewTemplateValidator(testFS, nil, nil)

	tests := []struct {
		templatePath string
		includePath  string
		expected     string
	}{
		{
			templatePath: "main.tmpl",
			includePath:  "shared",
			expected:     "shared.tmpl",
		},
		{
			templatePath: "templates/main.tmpl",
			includePath:  "local",
			expected:     "templates/local.tmpl",
		},
		{
			templatePath: "main.tmpl",
			includePath:  "common",
			expected:     "includes/common.tmpl",
		},
		{
			templatePath: "templates/main.tmpl",
			includePath:  "nested.tpl",
			expected:     "templates/nested.tpl",
		},
		{
			templatePath: "main.tmpl",
			includePath:  "nonexistent",
			expected:     "",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s-%s", test.templatePath, test.includePath), func(t *testing.T) {
			result := validator.resolveIncludePath(test.templatePath, test.includePath)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestTemplateValidator_IsControlStructure(t *testing.T) {
	validator := NewTemplateValidator(fstest.MapFS{}, nil, nil)

	controlStructures := []string{"if", "else", "else if", "end", "range", "with", "define", "block"}
	nonControlStructures := []string{"printf", "upper", "len", "myFunc", "unknown"}

	for _, cs := range controlStructures {
		t.Run("control_"+cs, func(t *testing.T) {
			if !validator.isControlStructure(cs) {
				t.Errorf("Expected '%s' to be recognized as control structure", cs)
			}
		})
	}

	for _, ncs := range nonControlStructures {
		t.Run("non_control_"+ncs, func(t *testing.T) {
			if validator.isControlStructure(ncs) {
				t.Errorf("Expected '%s' to NOT be recognized as control structure", ncs)
			}
		})
	}
}

func TestTemplateValidator_IsBuiltinFunction(t *testing.T) {
	validator := NewTemplateValidator(fstest.MapFS{}, nil, nil)

	builtinFunctions := []string{"and", "or", "not", "eq", "ne", "lt", "le", "gt", "ge", "printf", "print", "println", "len", "index", "slice", "call", "html", "js", "urlquery"}
	nonBuiltinFunctions := []string{"upper", "lower", "myFunc", "unknown", "custom"}

	for _, bf := range builtinFunctions {
		t.Run("builtin_"+bf, func(t *testing.T) {
			if !validator.isBuiltinFunction(bf) {
				t.Errorf("Expected '%s' to be recognized as builtin function", bf)
			}
		})
	}

	for _, nbf := range nonBuiltinFunctions {
		t.Run("non_builtin_"+nbf, func(t *testing.T) {
			if validator.isBuiltinFunction(nbf) {
				t.Errorf("Expected '%s' to NOT be recognized as builtin function", nbf)
			}
		})
	}
}

func TestTemplateValidator_ExtractLineColumn(t *testing.T) {
	validator := NewTemplateValidator(fstest.MapFS{}, nil, nil)

	tests := []struct {
		errorMsg     string
		expectedLine int
		expectedCol  int
	}{
		{
			errorMsg:     "error at line 5:10 in template",
			expectedLine: 5,
			expectedCol:  10,
		},
		{
			errorMsg:     "error at line 15 in template",
			expectedLine: 15,
			expectedCol:  0,
		},
		{
			errorMsg:     "template error at line 7:3",
			expectedLine: 7,
			expectedCol:  3,
		},
		{
			errorMsg:     "parsing failed at line 20",
			expectedLine: 20,
			expectedCol:  0,
		},
		{
			errorMsg:     "generic error message",
			expectedLine: 0,
			expectedCol:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.errorMsg, func(t *testing.T) {
			line, col := validator.extractLineColumn(test.errorMsg)
			if line != test.expectedLine {
				t.Errorf("Expected line %d, got %d", test.expectedLine, line)
			}
			if col != test.expectedCol {
				t.Errorf("Expected column %d, got %d", test.expectedCol, col)
			}
		})
	}
}

func TestTemplateValidator_SuggestSyntaxFix(t *testing.T) {
	validator := NewTemplateValidator(fstest.MapFS{}, nil, nil)

	tests := []struct {
		errorMsg     string
		expectedText string
	}{
		{
			errorMsg:     "unexpected {{ in template",
			expectedText: "unmatched braces",
		},
		{
			errorMsg:     "unexpected }} in template",
			expectedText: "unmatched braces",
		},
		{
			errorMsg:     "unexpected token in template",
			expectedText: "template syntax",
		},
		{
			errorMsg:     "unterminated string in template",
			expectedText: "missing closing quotes",
		},
		{
			errorMsg:     "function not found in template",
			expectedText: "function name spelling",
		},
		{
			errorMsg:     "generic error message",
			expectedText: "template syntax documentation",
		},
	}

	for _, test := range tests {
		t.Run(test.errorMsg, func(t *testing.T) {
			suggestion := validator.suggestSyntaxFix(test.errorMsg)
			if !strings.Contains(strings.ToLower(suggestion), strings.ToLower(test.expectedText)) {
				t.Errorf("Expected suggestion to contain '%s', got '%s'", test.expectedText, suggestion)
			}
		})
	}
}

func TestTemplateValidator_ValidateDirectory(t *testing.T) {
	testFS := fstest.MapFS{
		"templates/valid.tmpl": &fstest.MapFile{
			Data: []byte("Hello {{.Name}}!"),
		},
		"templates/invalid.tmpl": &fstest.MapFile{
			Data: []byte("Hello {{.Name!"),
		},
		"templates/nested/deep.tpl": &fstest.MapFile{
			Data: []byte("Deep {{printf \"%s\" .Value}}"),
		},
		"templates/readme.txt": &fstest.MapFile{
			Data: []byte("This is not a template file"),
		},
		"templates/subdir/another.tmpl": &fstest.MapFile{
			Data: []byte("Another {{.Item}}"),
		},
	}

	var buf strings.Builder
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))
	validator := NewTemplateValidator(testFS, nil, dm)

	results := validator.ValidateDirectory("templates")

	expectedFiles := []string{
		"templates/valid.tmpl",
		"templates/invalid.tmpl",
		"templates/nested/deep.tpl",
		"templates/subdir/another.tmpl",
	}

	if len(results) != len(expectedFiles) {
		t.Errorf("Expected %d results, got %d", len(expectedFiles), len(results))
	}

	for _, file := range expectedFiles {
		if _, exists := results[file]; !exists {
			t.Errorf("Expected result for file %s", file)
		}
	}

	// Check that non-template files were ignored
	if _, exists := results["templates/readme.txt"]; exists {
		t.Error("Expected non-template files to be ignored")
	}

	// Check that valid template is marked as valid
	if validResult, exists := results["templates/valid.tmpl"]; exists {
		if !validResult.Valid {
			t.Error("Expected valid template to be marked as valid")
		}
	}

	// Check that invalid template is marked as invalid
	if invalidResult, exists := results["templates/invalid.tmpl"]; exists {
		if invalidResult.Valid {
			t.Error("Expected invalid template to be marked as invalid")
		}
		if len(invalidResult.Errors) == 0 {
			t.Error("Expected invalid template to have errors")
		}
	}
}

func TestTemplateValidator_ValidateDirectory_WithError(t *testing.T) {
	// Create a filesystem that will cause WalkDir to fail
	testFS := &errorFS{}
	
	var buf strings.Builder
	dm := NewDebugMode(WithLevel(LevelDebug), WithOutput(&buf))
	validator := NewTemplateValidator(testFS, nil, dm)

	results := validator.ValidateDirectory("nonexistent")

	// Should return empty results when directory walk fails
	if len(results) != 0 {
		t.Errorf("Expected empty results on directory walk failure, got %d", len(results))
	}

	// Should log error
	logOutput := buf.String()
	if !strings.Contains(logOutput, "Directory validation failed") {
		t.Error("Expected error to be logged when directory validation fails")
	}
}

// errorFS is a test filesystem that always returns errors
type errorFS struct{}

func (e *errorFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func (e *errorFS) ReadFile(name string) ([]byte, error) {
	return nil, fs.ErrNotExist
}

func (e *errorFS) Stat(name string) (fs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func TestValidationResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		result   ValidationResult
		expected bool
	}{
		{
			name: "no errors, valid",
			result: ValidationResult{
				Valid:  true,
				Errors: []ValidationError{},
			},
			expected: false,
		},
		{
			name: "no errors, invalid",
			result: ValidationResult{
				Valid:  false,
				Errors: []ValidationError{},
			},
			expected: true,
		},
		{
			name: "has errors, valid",
			result: ValidationResult{
				Valid: true,
				Errors: []ValidationError{
					{Type: "test", Message: "test error"},
				},
			},
			expected: true,
		},
		{
			name: "has errors, invalid",
			result: ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Type: "test", Message: "test error"},
				},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.result.HasErrors() != test.expected {
				t.Errorf("Expected HasErrors()=%v, got HasErrors()=%v", test.expected, test.result.HasErrors())
			}
		})
	}
}

func TestValidationResult_Summary(t *testing.T) {
	tests := []struct {
		name     string
		result   ValidationResult
		expected string
	}{
		{
			name: "valid with no errors or warnings",
			result: ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationError{},
			},
			expected: "Valid",
		},
		{
			name: "errors only",
			result: ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Type: "error1", Message: "Error 1"},
					{Type: "error2", Message: "Error 2"},
				},
				Warnings: []ValidationError{},
			},
			expected: "2 error(s)",
		},
		{
			name: "warnings only",
			result: ValidationResult{
				Valid:  true,
				Errors: []ValidationError{},
				Warnings: []ValidationError{
					{Type: "warning1", Message: "Warning 1"},
				},
			},
			expected: "1 warning(s)",
		},
		{
			name: "errors and warnings",
			result: ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Type: "error1", Message: "Error 1"},
				},
				Warnings: []ValidationError{
					{Type: "warning1", Message: "Warning 1"},
					{Type: "warning2", Message: "Warning 2"},
				},
			},
			expected: "1 error(s), 2 warning(s)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			summary := test.result.Summary()
			if summary != test.expected {
				t.Errorf("Expected summary '%s', got '%s'", test.expected, summary)
			}
		})
	}
}

func TestTemplateValidator_ComprehensiveValidation(t *testing.T) {
	funcMap := template.FuncMap{
		"upper": strings.ToUpper,
	}

	testFS := fstest.MapFS{
		"complex.tmpl": &fstest.MapFile{
			Data: []byte(`
{{/* This is a complex template with various issues */}}
Hello {{.User.Name}}!

{{if .User.Active}}
  {{upper .User.Name}} is active
  {{unknownFunc .User.Email}}
  {{.Very.Deep.Nested.Variable.Chain.That.Is.Too.Long}}
{{end}}

{{template "header"}}
{{include "footer"}}

{{/* Syntax issues */}}
{{.Name
`),
		},
		"simple.tmpl": &fstest.MapFile{
			Data: []byte("Hello {{.Name}}!"),
		},
	}

	validator := NewTemplateValidator(testFS, funcMap, nil)
	validator.SetStrict(true)

	result := validator.ValidateTemplate("complex.tmpl")

	if result.Valid {
		t.Error("Expected complex template to be invalid")
	}

	// Check for various types of errors and warnings
	errorTypes := make(map[string]int)
	warningTypes := make(map[string]int)

	for _, err := range result.Errors {
		errorTypes[err.Type]++
	}

	for _, warning := range result.Warnings {
		warningTypes[warning.Type]++
	}

	// Should have syntax errors
	if errorTypes["syntax_error"] == 0 {
		t.Error("Expected syntax errors in complex template")
	}

	// Should have unknown function warnings
	if warningTypes["unknown_function"] == 0 {
		t.Error("Expected unknown function warnings in complex template")
	}

	// Should have deep access warnings
	if warningTypes["deep_access"] == 0 {
		t.Error("Expected deep access warnings in complex template")
	}

	// Should have missing partial/include errors
	if errorTypes["missing_partial"] == 0 && errorTypes["missing_include"] == 0 {
		t.Error("Expected missing partial or include errors in complex template")
	}
}
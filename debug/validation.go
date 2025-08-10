package debug

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
)

type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
	Info     []string          `json:"info"`
}

type ValidationError struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

type TemplateValidator struct {
	fs        fs.FS
	funcMap   template.FuncMap
	strict    bool
	debugMode *DebugMode
}

func NewTemplateValidator(templateFS fs.FS, funcMap template.FuncMap, debugMode *DebugMode) *TemplateValidator {
	return &TemplateValidator{
		fs:        templateFS,
		funcMap:   funcMap,
		strict:    false,
		debugMode: debugMode,
	}
}

func (tv *TemplateValidator) SetStrict(strict bool) {
	tv.strict = strict
}

func (tv *TemplateValidator) ValidateTemplate(templatePath string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationError, 0),
		Info:     make([]string, 0),
	}

	content, err := fs.ReadFile(tv.fs, templatePath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    "file_error",
			Message: fmt.Sprintf("Cannot read template file: %v", err),
			File:    templatePath,
		})
		return result
	}

	templateContent := string(content)

	tv.validateSyntax(templatePath, templateContent, &result)
	tv.validateFunctions(templatePath, templateContent, &result)
	tv.validateVariableAccess(templatePath, templateContent, &result)
	tv.validatePartials(templatePath, templateContent, &result)
	tv.validateIncludes(templatePath, templateContent, &result)

	return result
}

func (tv *TemplateValidator) validateSyntax(templatePath, content string, result *ValidationResult) {
	tmpl := template.New(templatePath)
	if tv.funcMap != nil {
		tmpl = tmpl.Funcs(tv.funcMap)
	}

	_, err := tmpl.Parse(content)
	if err != nil {
		result.Valid = false

		errorMsg := err.Error()
		line, col := tv.extractLineColumn(errorMsg)

		result.Errors = append(result.Errors, ValidationError{
			Type:       "syntax_error",
			Message:    errorMsg,
			File:       templatePath,
			Line:       line,
			Column:     col,
			Suggestion: tv.suggestSyntaxFix(errorMsg),
		})
	}

	tv.validateBraceBalance(templatePath, content, result)
	tv.validateWhitespace(templatePath, content, result)
}

func (tv *TemplateValidator) validateBraceBalance(templatePath, content string, result *ValidationResult) {
	lines := strings.Split(content, "\n")
	openBraces := 0

	for lineNum, line := range lines {
		for i := 0; i < len(line); i++ {
			if i+1 < len(line) && line[i] == '{' && line[i+1] == '{' {
				openBraces++
				i++ // skip the next character to avoid double counting
			} else if i+1 < len(line) && line[i] == '}' && line[i+1] == '}' {
				openBraces--
				if openBraces < 0 {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Type:       "brace_mismatch",
						Message:    "Unmatched closing braces }}",
						File:       templatePath,
						Line:       lineNum + 1,
						Column:     i + 1,
						Suggestion: "Check for missing opening braces {{",
					})
					openBraces = 0
				}
				i++ // skip the next character to avoid double counting
			}
		}
	}

	if openBraces > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:       "brace_mismatch",
			Message:    fmt.Sprintf("Unclosed braces: %d opening braces without matching closing braces", openBraces),
			File:       templatePath,
			Suggestion: "Add missing closing braces }}",
		})
	}
}

func (tv *TemplateValidator) validateWhitespace(templatePath, content string, result *ValidationResult) {
	if tv.strict {
		lines := strings.Split(content, "\n")
		for lineNum, line := range lines {
			if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
				result.Warnings = append(result.Warnings, ValidationError{
					Type:    "whitespace_warning",
					Message: "Line has trailing whitespace",
					File:    templatePath,
					Line:    lineNum + 1,
				})
			}
		}
	}
}

func (tv *TemplateValidator) validateFunctions(templatePath, content string, result *ValidationResult) {
	functionPattern := regexp.MustCompile(`{{\s*([^}\s]+)`)
	matches := functionPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		funcName := strings.Split(match[1], " ")[0]
		funcName = strings.Split(funcName, "|")[0]

		if tv.isControlStructure(funcName) {
			continue
		}

		if tv.isBuiltinFunction(funcName) {
			continue
		}

		if tv.funcMap != nil {
			if _, exists := tv.funcMap[funcName]; !exists {
				result.Warnings = append(result.Warnings, ValidationError{
					Type:       "unknown_function",
					Message:    fmt.Sprintf("Function '%s' is not defined", funcName),
					File:       templatePath,
					Suggestion: fmt.Sprintf("Check if '%s' is spelled correctly or add it to the function map", funcName),
				})
			}
		}
	}
}

func (tv *TemplateValidator) validateVariableAccess(templatePath, content string, result *ValidationResult) {
	variablePattern := regexp.MustCompile(`{{\s*\.([^}\s|]+)`)
	matches := variablePattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		varPath := match[1]
		if strings.Contains(varPath, " ") {
			continue
		}

		parts := strings.Split(varPath, ".")
		if len(parts) > 5 {
			result.Warnings = append(result.Warnings, ValidationError{
				Type:       "deep_access",
				Message:    fmt.Sprintf("Variable access chain '%s' is very deep", varPath),
				File:       templatePath,
				Suggestion: "Consider simplifying the data structure or using intermediate variables",
			})
		}
	}
}

func (tv *TemplateValidator) validatePartials(templatePath, content string, result *ValidationResult) {
	partialPattern := regexp.MustCompile(`{{\s*template\s+"([^"]+)"`)
	matches := partialPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		partialName := match[1]

		partialPath := tv.resolvePartialPath(templatePath, partialName)
		if partialPath == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:       "missing_partial",
				Message:    fmt.Sprintf("Partial template '%s' not found", partialName),
				File:       templatePath,
				Suggestion: fmt.Sprintf("Create partial file or check the name '%s'", partialName),
			})
		}
	}
}

func (tv *TemplateValidator) validateIncludes(templatePath, content string, result *ValidationResult) {
	includePattern := regexp.MustCompile(`{{\s*include\s+"([^"]+)"`)
	matches := includePattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		includePath := match[1]

		resolvedPath := tv.resolveIncludePath(templatePath, includePath)
		if resolvedPath == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:       "missing_include",
				Message:    fmt.Sprintf("Include file '%s' not found", includePath),
				File:       templatePath,
				Suggestion: fmt.Sprintf("Create include file or check the path '%s'", includePath),
			})
		}
	}
}

func (tv *TemplateValidator) resolvePartialPath(templatePath, partialName string) string {
	baseDir := filepath.Dir(templatePath)

	candidates := []string{
		filepath.Join(baseDir, "_"+partialName+".tmpl"),
		filepath.Join(baseDir, "_"+partialName+".tpl"),
		"_" + partialName + ".tmpl",
		"_" + partialName + ".tpl",
	}

	for _, candidate := range candidates {
		if _, err := fs.Stat(tv.fs, candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func (tv *TemplateValidator) resolveIncludePath(templatePath, includePath string) string {
	baseDir := filepath.Dir(templatePath)

	candidates := []string{
		includePath,
		includePath + ".tmpl",
		includePath + ".tpl",
		filepath.Join(baseDir, includePath),
		filepath.Join(baseDir, includePath+".tmpl"),
		filepath.Join(baseDir, includePath+".tpl"),
		filepath.Join("includes", includePath),
		filepath.Join("includes", includePath+".tmpl"),
		filepath.Join("includes", includePath+".tpl"),
	}

	for _, candidate := range candidates {
		if _, err := fs.Stat(tv.fs, candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func (tv *TemplateValidator) isControlStructure(name string) bool {
	controlStructures := []string{
		"if", "else", "else if", "end",
		"range", "with", "define", "block",
	}

	return slices.Contains(controlStructures, name)
}

func (tv *TemplateValidator) isBuiltinFunction(name string) bool {
	builtins := []string{
		"and", "or", "not", "eq", "ne", "lt", "le", "gt", "ge",
		"printf", "print", "println", "len", "index", "slice",
		"call", "html", "js", "urlquery",
	}

	return slices.Contains(builtins, name)
}

func (tv *TemplateValidator) extractLineColumn(errorMsg string) (int, int) {
	lineColPattern := regexp.MustCompile(`line (\d+):(\d+)`)
	matches := lineColPattern.FindStringSubmatch(errorMsg)

	if len(matches) >= 3 {
		line := 0
		col := 0
		fmt.Sscanf(matches[1], "%d", &line)
		fmt.Sscanf(matches[2], "%d", &col)
		return line, col
	}

	linePattern := regexp.MustCompile(`line (\d+)`)
	matches = linePattern.FindStringSubmatch(errorMsg)

	if len(matches) >= 2 {
		line := 0
		fmt.Sscanf(matches[1], "%d", &line)
		return line, 0
	}

	return 0, 0
}

func (tv *TemplateValidator) suggestSyntaxFix(errorMsg string) string {
	errorMsg = strings.ToLower(errorMsg)

	if strings.Contains(errorMsg, "unexpected") {
		if strings.Contains(errorMsg, "{{") || strings.Contains(errorMsg, "}}") {
			return "Check for unmatched braces {{ }}"
		}
		return "Check template syntax near the error location"
	}

	if strings.Contains(errorMsg, "unterminated") {
		return "Check for missing closing quotes or braces"
	}

	if strings.Contains(errorMsg, "function") {
		return "Check function name spelling and availability"
	}

	return "Review template syntax documentation"
}

func (tv *TemplateValidator) ValidateDirectory(templateDir string) map[string]ValidationResult {
	results := make(map[string]ValidationResult)

	err := fs.WalkDir(tv.fs, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".tmpl") && !strings.HasSuffix(path, ".tpl") {
			return nil
		}

		result := tv.ValidateTemplate(path)
		results[path] = result

		return nil
	})

	if err != nil && tv.debugMode != nil {
		tv.debugMode.Error("Directory validation failed", "error", err, "directory", templateDir)
	}

	return results
}

func (vr ValidationResult) HasErrors() bool {
	return !vr.Valid || len(vr.Errors) > 0
}

func (vr ValidationResult) Summary() string {
	if vr.Valid && len(vr.Errors) == 0 && len(vr.Warnings) == 0 {
		return "Valid"
	}

	parts := []string{}
	if len(vr.Errors) > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", len(vr.Errors)))
	}
	if len(vr.Warnings) > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s)", len(vr.Warnings)))
	}

	return strings.Join(parts, ", ")
}

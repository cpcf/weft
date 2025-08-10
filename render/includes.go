package render

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type IncludeManager struct {
	templateFS   fs.FS
	cache        map[string]string
	funcMap      template.FuncMap
	maxDepth     int
	includeStack []string
}

type IncludeOptions struct {
	MaxDepth     int
	AllowedPaths []string
	Preprocessor func(string) string
}

func NewIncludeManager(templateFS fs.FS, funcMap template.FuncMap) *IncludeManager {
	im := &IncludeManager{
		templateFS:   templateFS,
		cache:        make(map[string]string),
		funcMap:      funcMap,
		maxDepth:     10,
		includeStack: make([]string, 0),
	}

	if im.funcMap == nil {
		im.funcMap = make(template.FuncMap)
	}

	im.funcMap["include"] = im.includeFunc
	im.funcMap["includeWith"] = im.includeWithFunc

	return im
}

func (im *IncludeManager) SetMaxDepth(depth int) {
	im.maxDepth = depth
}

func (im *IncludeManager) includeFunc(path string) (string, error) {
	return im.processInclude(path, nil)
}

func (im *IncludeManager) includeWithFunc(path string, data interface{}) (string, error) {
	return im.processInclude(path, data)
}

func (im *IncludeManager) processInclude(includePath string, data interface{}) (string, error) {
	if len(im.includeStack) >= im.maxDepth {
		return "", fmt.Errorf("include depth limit exceeded (%d)", im.maxDepth)
	}

	for _, stackPath := range im.includeStack {
		if stackPath == includePath {
			return "", fmt.Errorf("circular include detected: %s", includePath)
		}
	}

	if cached, exists := im.cache[includePath]; exists {
		return im.renderInclude(cached, data, includePath)
	}

	resolvedPath := im.resolveIncludePath(includePath)
	if resolvedPath == "" {
		return "", fmt.Errorf("include file not found: %s", includePath)
	}

	content, err := fs.ReadFile(im.templateFS, resolvedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read include file %s: %w", resolvedPath, err)
	}

	processedContent := string(content)
	processedContent = im.preprocessIncludes(processedContent)

	im.cache[includePath] = processedContent
	return im.renderInclude(processedContent, data, includePath)
}

func (im *IncludeManager) renderInclude(content string, data interface{}, includePath string) (string, error) {
	im.includeStack = append(im.includeStack, includePath)
	defer func() {
		im.includeStack = im.includeStack[:len(im.includeStack)-1]
	}()

	tmpl, err := template.New(includePath).Funcs(im.funcMap).Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse include template %s: %w", includePath, err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to execute include template %s: %w", includePath, err)
	}

	return result.String(), nil
}

func (im *IncludeManager) resolveIncludePath(includePath string) string {
	if strings.HasPrefix(includePath, "/") {
		includePath = strings.TrimPrefix(includePath, "/")
	}

	candidates := []string{
		includePath,
		includePath + ".tmpl",
		includePath + ".tpl",
		filepath.Join("includes", includePath),
		filepath.Join("includes", includePath+".tmpl"),
		filepath.Join("includes", includePath+".tpl"),
	}

	for _, candidate := range candidates {
		if _, err := fs.Stat(im.templateFS, candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func (im *IncludeManager) preprocessIncludes(content string) string {
	includeRegex := regexp.MustCompile(`{{\s*include\s+"([^"]+)"\s*}}`)
	includeWithRegex := regexp.MustCompile(`{{\s*includeWith\s+"([^"]+)"\s+([^}]+)\s*}}`)

	content = includeRegex.ReplaceAllStringFunc(content, func(match string) string {
		groups := includeRegex.FindStringSubmatch(match)
		if len(groups) > 1 {
			return fmt.Sprintf(`{{ include "%s" }}`, groups[1])
		}
		return match
	})

	content = includeWithRegex.ReplaceAllStringFunc(content, func(match string) string {
		groups := includeWithRegex.FindStringSubmatch(match)
		if len(groups) > 2 {
			return fmt.Sprintf(`{{ includeWith "%s" %s }}`, groups[1], groups[2])
		}
		return match
	})

	return content
}

func (im *IncludeManager) GetFuncMap() template.FuncMap {
	return im.funcMap
}

func (im *IncludeManager) ValidateIncludes(templatePath string) error {
	content, err := fs.ReadFile(im.templateFS, templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	includes := im.extractIncludeReferences(string(content))
	
	for _, includePath := range includes {
		resolvedPath := im.resolveIncludePath(includePath)
		if resolvedPath == "" {
			return fmt.Errorf("include file not found: %s", includePath)
		}

		if err := im.validateIncludeContent(resolvedPath); err != nil {
			return fmt.Errorf("invalid include %s: %w", includePath, err)
		}
	}

	return nil
}

func (im *IncludeManager) validateIncludeContent(includePath string) error {
	content, err := fs.ReadFile(im.templateFS, includePath)
	if err != nil {
		return err
	}

	tmpl, err := template.New("validate").Funcs(im.funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("template syntax error: %w", err)
	}

	_ = tmpl
	return nil
}

func (im *IncludeManager) extractIncludeReferences(content string) []string {
	var includes []string
	
	includeRegex := regexp.MustCompile(`{{\s*include(?:With)?\s+"([^"]+)"`)
	matches := includeRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			includes = append(includes, match[1])
		}
	}

	return includes
}

func (im *IncludeManager) ListIncludes(templatePath string) ([]string, error) {
	content, err := fs.ReadFile(im.templateFS, templatePath)
	if err != nil {
		return nil, err
	}

	return im.extractIncludeReferences(string(content)), nil
}

func (im *IncludeManager) GetIncludeContent(includePath string) (string, error) {
	resolvedPath := im.resolveIncludePath(includePath)
	if resolvedPath == "" {
		return "", fmt.Errorf("include file not found: %s", includePath)
	}

	content, err := fs.ReadFile(im.templateFS, resolvedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read include file: %w", err)
	}

	return string(content), nil
}

func (im *IncludeManager) ClearCache() {
	im.cache = make(map[string]string)
}

func (im *IncludeManager) PreloadIncludes(templatePaths []string) error {
	for _, templatePath := range templatePaths {
		includes, err := im.ListIncludes(templatePath)
		if err != nil {
			return fmt.Errorf("failed to list includes for %s: %w", templatePath, err)
		}

		for _, includePath := range includes {
			if _, exists := im.cache[includePath]; !exists {
				content, err := im.GetIncludeContent(includePath)
				if err != nil {
					return fmt.Errorf("failed to preload include %s: %w", includePath, err)
				}
				im.cache[includePath] = content
			}
		}
	}

	return nil
}

func (im *IncludeManager) GetIncludeGraph(templatePath string, visited map[string]bool) (map[string][]string, error) {
	if visited == nil {
		visited = make(map[string]bool)
	}

	if visited[templatePath] {
		return nil, fmt.Errorf("circular dependency detected: %s", templatePath)
	}

	visited[templatePath] = true
	defer func() { visited[templatePath] = false }()

	includes, err := im.ListIncludes(templatePath)
	if err != nil {
		return nil, err
	}

	graph := make(map[string][]string)
	graph[templatePath] = includes

	for _, includePath := range includes {
		resolvedPath := im.resolveIncludePath(includePath)
		if resolvedPath == "" {
			continue
		}

		subGraph, err := im.GetIncludeGraph(resolvedPath, visited)
		if err != nil {
			return nil, err
		}

		for k, v := range subGraph {
			graph[k] = v
		}
	}

	return graph, nil
}
package render

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type PartialManager struct {
	templateFS fs.FS
	cache      map[string]*template.Template
	funcMap    template.FuncMap
}

func NewPartialManager(templateFS fs.FS, funcMap template.FuncMap) *PartialManager {
	return &PartialManager{
		templateFS: templateFS,
		cache:      make(map[string]*template.Template),
		funcMap:    funcMap,
	}
}

func (pm *PartialManager) LoadPartials(rootTemplate *template.Template, templatePath string) error {
	dir := filepath.Dir(templatePath)
	
	err := fs.WalkDir(pm.templateFS, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !pm.isPartialFile(path) {
			return nil
		}

		return pm.loadPartialFile(rootTemplate, path)
	})

	return err
}

func (pm *PartialManager) isPartialFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, "_") && (strings.HasSuffix(base, ".tmpl") || strings.HasSuffix(base, ".tpl"))
}

func (pm *PartialManager) loadPartialFile(rootTemplate *template.Template, partialPath string) error {
	content, err := fs.ReadFile(pm.templateFS, partialPath)
	if err != nil {
		return fmt.Errorf("failed to read partial file %s: %w", partialPath, err)
	}

	partialName := pm.getPartialName(partialPath)
	
	_, err = rootTemplate.New(partialName).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse partial %s: %w", partialPath, err)
	}

	return nil
}

func (pm *PartialManager) getPartialName(partialPath string) string {
	base := filepath.Base(partialPath)
	
	base = strings.TrimPrefix(base, "_")
	
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	
	return base
}

func (pm *PartialManager) RegisterPartials(rootTemplate *template.Template, partialPaths []string) error {
	for _, partialPath := range partialPaths {
		if err := pm.loadPartialFile(rootTemplate, partialPath); err != nil {
			return err
		}
	}
	return nil
}

func (pm *PartialManager) FindPartials(templateDir string) ([]string, error) {
	var partials []string
	
	err := fs.WalkDir(pm.templateFS, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if pm.isPartialFile(path) {
			partials = append(partials, path)
		}

		return nil
	})

	return partials, err
}

func (pm *PartialManager) CreateTemplate(name string, content string) (*template.Template, error) {
	tmpl := template.New(name).Funcs(pm.funcMap)
	
	parsed, err := tmpl.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	templateDir := filepath.Dir(name)
	if templateDir != "." && templateDir != "" {
		if err := pm.LoadPartials(parsed, name); err != nil {
			return nil, fmt.Errorf("failed to load partials for template %s: %w", name, err)
		}
	}

	pm.cache[name] = parsed
	return parsed, nil
}

func (pm *PartialManager) GetTemplate(name string) (*template.Template, bool) {
	tmpl, exists := pm.cache[name]
	return tmpl, exists
}

func (pm *PartialManager) ClearCache() {
	pm.cache = make(map[string]*template.Template)
}

func (pm *PartialManager) ParseTemplateWithPartials(templatePath string) (*template.Template, error) {
	content, err := fs.ReadFile(pm.templateFS, templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	if cached, exists := pm.GetTemplate(templatePath); exists {
		return cached, nil
	}

	return pm.CreateTemplate(templatePath, string(content))
}

func (pm *PartialManager) ValidatePartials(templatePath string) error {
	tmpl, err := pm.ParseTemplateWithPartials(templatePath)
	if err != nil {
		return err
	}

	partialNames := pm.extractPartialReferences(string(tmpl.Root.Tree.Root))
	
	for _, partialName := range partialNames {
		if tmpl.Lookup(partialName) == nil {
			return fmt.Errorf("partial template '%s' not found", partialName)
		}
	}

	return nil
}

func (pm *PartialManager) extractPartialReferences(content string) []string {
	var partials []string
	
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(line, "template") {
			if start := strings.Index(line, `"`); start != -1 {
				if end := strings.Index(line[start+1:], `"`); end != -1 {
					partialName := line[start+1 : start+1+end]
					partials = append(partials, partialName)
				}
			}
		}
	}

	return partials
}

func (pm *PartialManager) ListTemplateNames(rootTemplate *template.Template) []string {
	var names []string
	
	if rootTemplate != nil {
		for _, tmpl := range rootTemplate.Templates() {
			names = append(names, tmpl.Name())
		}
	}
	
	return names
}

func (pm *PartialManager) GetPartialContent(partialName string) (string, error) {
	for cachedPath, tmpl := range pm.cache {
		if pm.getPartialName(cachedPath) == partialName {
			if partial := tmpl.Lookup(partialName); partial != nil {
				return partial.Root.String(), nil
			}
		}
	}

	partialPath := "_" + partialName + ".tmpl"
	content, err := fs.ReadFile(pm.templateFS, partialPath)
	if err != nil {
		partialPath = "_" + partialName + ".tpl"
		content, err = fs.ReadFile(pm.templateFS, partialPath)
		if err != nil {
			return "", fmt.Errorf("partial %s not found", partialName)
		}
	}

	return string(content), nil
}

func (pm *PartialManager) ResolvePartialPath(basePath, partialName string) string {
	baseDir := filepath.Dir(basePath)
	
	candidates := []string{
		path.Join(baseDir, "_"+partialName+".tmpl"),
		path.Join(baseDir, "_"+partialName+".tpl"),
		"_" + partialName + ".tmpl",
		"_" + partialName + ".tpl",
	}

	for _, candidate := range candidates {
		if _, err := fs.Stat(pm.templateFS, candidate); err == nil {
			return candidate
		}
	}

	return ""
}
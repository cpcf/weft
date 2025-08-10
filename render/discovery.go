package render

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type DiscoveryRule struct {
	Name        string   `json:"name"`
	Patterns    []string `json:"patterns"`
	Exclude     []string `json:"exclude"`
	Extensions  []string `json:"extensions"`
	Directories []string `json:"directories"`
	Recursive   bool     `json:"recursive"`
	Priority    int      `json:"priority"`
}

type DiscoveredTemplate struct {
	Path         string            `json:"path"`
	Name         string            `json:"name"`
	Extension    string            `json:"extension"`
	Directory    string            `json:"directory"`
	Size         int64             `json:"size"`
	IsPartial    bool              `json:"is_partial"`
	IsInclude    bool              `json:"is_include"`
	RuleName     string            `json:"rule_name"`
	Priority     int               `json:"priority"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type TemplateDiscovery struct {
	templateFS fs.FS
	rules      []DiscoveryRule
	cache      map[string][]DiscoveredTemplate
}

func NewTemplateDiscovery(templateFS fs.FS) *TemplateDiscovery {
	return &TemplateDiscovery{
		templateFS: templateFS,
		rules:      []DiscoveryRule{},
		cache:      make(map[string][]DiscoveredTemplate),
	}
}

func (td *TemplateDiscovery) AddRule(rule DiscoveryRule) {
	td.rules = append(td.rules, rule)
	td.sortRules()
	td.clearCache()
}

func (td *TemplateDiscovery) AddDefaultRules() {
	defaultRules := []DiscoveryRule{
		{
			Name:        "standard-templates",
			Patterns:    []string{"*.tmpl", "*.tpl"},
			Exclude:     []string{"_*"},
			Extensions:  []string{".tmpl", ".tpl"},
			Directories: []string{},
			Recursive:   true,
			Priority:    10,
		},
		{
			Name:        "partials",
			Patterns:    []string{"_*.tmpl", "_*.tpl"},
			Extensions:  []string{".tmpl", ".tpl"},
			Directories: []string{"partials", "_partials"},
			Recursive:   true,
			Priority:    20,
		},
		{
			Name:        "includes",
			Patterns:    []string{"*.tmpl", "*.tpl"},
			Extensions:  []string{".tmpl", ".tpl"},
			Directories: []string{"includes", "_includes"},
			Recursive:   true,
			Priority:    15,
		},
	}

	for _, rule := range defaultRules {
		td.AddRule(rule)
	}
}

func (td *TemplateDiscovery) sortRules() {
	sort.Slice(td.rules, func(i, j int) bool {
		return td.rules[i].Priority > td.rules[j].Priority
	})
}

func (td *TemplateDiscovery) clearCache() {
	td.cache = make(map[string][]DiscoveredTemplate)
}

func (td *TemplateDiscovery) DiscoverTemplates(rootPath string) ([]DiscoveredTemplate, error) {
	if cached, exists := td.cache[rootPath]; exists {
		return cached, nil
	}

	var discovered []DiscoveredTemplate
	seen := make(map[string]bool)

	for _, rule := range td.rules {
		templates, err := td.discoverByRule(rootPath, rule)
		if err != nil {
			return nil, fmt.Errorf("discovery failed for rule %s: %w", rule.Name, err)
		}

		for _, tmpl := range templates {
			if !seen[tmpl.Path] {
				discovered = append(discovered, tmpl)
				seen[tmpl.Path] = true
			}
		}
	}

	td.sortDiscoveredTemplates(discovered)
	td.cache[rootPath] = discovered
	return discovered, nil
}

func (td *TemplateDiscovery) discoverByRule(rootPath string, rule DiscoveryRule) ([]DiscoveredTemplate, error) {
	var discovered []DiscoveredTemplate

	searchDirs := []string{rootPath}
	if len(rule.Directories) > 0 {
		searchDirs = rule.Directories
		for i, dir := range searchDirs {
			if !filepath.IsAbs(dir) {
				searchDirs[i] = filepath.Join(rootPath, dir)
			}
		}
	}

	for _, dir := range searchDirs {
		if rule.Recursive {
			err := fs.WalkDir(td.templateFS, dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() {
					return nil
				}

				if td.matchesRule(path, rule) {
					template, err := td.createDiscoveredTemplate(path, rule)
					if err != nil {
						return err
					}
					discovered = append(discovered, template)
				}

				return nil
			})
			
			if err != nil {
				return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
			}
		} else {
			entries, err := fs.ReadDir(td.templateFS, dir)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				path := filepath.Join(dir, entry.Name())
				if td.matchesRule(path, rule) {
					template, err := td.createDiscoveredTemplate(path, rule)
					if err != nil {
						return nil, err
					}
					discovered = append(discovered, template)
				}
			}
		}
	}

	return discovered, nil
}

func (td *TemplateDiscovery) matchesRule(path string, rule DiscoveryRule) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(path)

	for _, excludePattern := range rule.Exclude {
		if matched, _ := filepath.Match(excludePattern, base); matched {
			return false
		}
	}

	if len(rule.Extensions) > 0 {
		matchesExt := false
		for _, allowedExt := range rule.Extensions {
			if ext == allowedExt {
				matchesExt = true
				break
			}
		}
		if !matchesExt {
			return false
		}
	}

	if len(rule.Patterns) > 0 {
		for _, pattern := range rule.Patterns {
			if matched, _ := filepath.Match(pattern, base); matched {
				return true
			}
		}
		return false
	}

	return true
}

func (td *TemplateDiscovery) createDiscoveredTemplate(path string, rule DiscoveryRule) (DiscoveredTemplate, error) {
	info, err := fs.Stat(td.templateFS, path)
	if err != nil {
		return DiscoveredTemplate{}, err
	}

	base := filepath.Base(path)
	ext := filepath.Ext(path)
	dir := filepath.Dir(path)
	name := strings.TrimSuffix(base, ext)

	template := DiscoveredTemplate{
		Path:      path,
		Name:      name,
		Extension: ext,
		Directory: dir,
		Size:      info.Size(),
		IsPartial: td.isPartial(base),
		IsInclude: td.isInclude(dir),
		RuleName:  rule.Name,
		Priority:  rule.Priority,
		Metadata:  make(map[string]string),
	}

	template.Metadata["rule"] = rule.Name
	template.Metadata["base"] = base
	template.Metadata["dir"] = dir

	return template, nil
}

func (td *TemplateDiscovery) isPartial(filename string) bool {
	return strings.HasPrefix(filename, "_")
}

func (td *TemplateDiscovery) isInclude(dir string) bool {
	base := filepath.Base(dir)
	return base == "includes" || base == "_includes"
}

func (td *TemplateDiscovery) sortDiscoveredTemplates(templates []DiscoveredTemplate) {
	sort.Slice(templates, func(i, j int) bool {
		if templates[i].Priority != templates[j].Priority {
			return templates[i].Priority > templates[j].Priority
		}
		return templates[i].Path < templates[j].Path
	})
}

func (td *TemplateDiscovery) GetTemplatesByType(rootPath string, templateType string) ([]DiscoveredTemplate, error) {
	all, err := td.DiscoverTemplates(rootPath)
	if err != nil {
		return nil, err
	}

	var filtered []DiscoveredTemplate
	for _, tmpl := range all {
		switch templateType {
		case "partial":
			if tmpl.IsPartial {
				filtered = append(filtered, tmpl)
			}
		case "include":
			if tmpl.IsInclude {
				filtered = append(filtered, tmpl)
			}
		case "standard":
			if !tmpl.IsPartial && !tmpl.IsInclude {
				filtered = append(filtered, tmpl)
			}
		default:
			filtered = append(filtered, tmpl)
		}
	}

	return filtered, nil
}

func (td *TemplateDiscovery) GetTemplateByName(rootPath, name string) (DiscoveredTemplate, error) {
	all, err := td.DiscoverTemplates(rootPath)
	if err != nil {
		return DiscoveredTemplate{}, err
	}

	for _, tmpl := range all {
		if tmpl.Name == name {
			return tmpl, nil
		}
	}

	return DiscoveredTemplate{}, fmt.Errorf("template not found: %s", name)
}

func (td *TemplateDiscovery) GetTemplatesByPattern(rootPath, pattern string) ([]DiscoveredTemplate, error) {
	all, err := td.DiscoverTemplates(rootPath)
	if err != nil {
		return nil, err
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var matched []DiscoveredTemplate
	for _, tmpl := range all {
		if regex.MatchString(tmpl.Path) || regex.MatchString(tmpl.Name) {
			matched = append(matched, tmpl)
		}
	}

	return matched, nil
}

func (td *TemplateDiscovery) ValidateDiscovery(rootPath string) error {
	templates, err := td.DiscoverTemplates(rootPath)
	if err != nil {
		return err
	}

	for _, tmpl := range templates {
		if _, err := fs.Stat(td.templateFS, tmpl.Path); err != nil {
			return fmt.Errorf("discovered template does not exist: %s", tmpl.Path)
		}
	}

	return nil
}

func (td *TemplateDiscovery) GetDiscoveryStats(rootPath string) (map[string]int, error) {
	templates, err := td.DiscoverTemplates(rootPath)
	if err != nil {
		return nil, err
	}

	stats := map[string]int{
		"total":     len(templates),
		"partials":  0,
		"includes":  0,
		"standard":  0,
		"templates": 0,
	}

	ruleStats := make(map[string]int)

	for _, tmpl := range templates {
		if tmpl.IsPartial {
			stats["partials"]++
		} else if tmpl.IsInclude {
			stats["includes"]++
		} else {
			stats["standard"]++
		}

		if strings.HasSuffix(tmpl.Extension, "tmpl") || strings.HasSuffix(tmpl.Extension, "tpl") {
			stats["templates"]++
		}

		ruleStats[tmpl.RuleName]++
	}

	for rule, count := range ruleStats {
		stats["rule_"+rule] = count
	}

	return stats, nil
}

func (td *TemplateDiscovery) GetRules() []DiscoveryRule {
	return td.rules
}

func (td *TemplateDiscovery) RemoveRule(name string) {
	for i, rule := range td.rules {
		if rule.Name == name {
			td.rules = append(td.rules[:i], td.rules[i+1:]...)
			break
		}
	}
	td.clearCache()
}
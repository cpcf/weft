package engine

import (
	"io/fs"
	"sync"
	"text/template"
)

type TemplateCache struct {
	mu        sync.RWMutex
	templates map[string]*template.Template
}

func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		templates: make(map[string]*template.Template),
	}
}

func (c *TemplateCache) Get(fsys fs.FS, path string) (*template.Template, error) {
	c.mu.RLock()
	if tmpl, exists := c.templates[path]; exists {
		c.mu.RUnlock()
		return tmpl, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if tmpl, exists := c.templates[path]; exists {
		return tmpl, nil
	}

	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(path).Parse(string(content))
	if err != nil {
		return nil, err
	}

	c.templates[path] = tmpl
	return tmpl, nil
}

func (c *TemplateCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.templates = make(map[string]*template.Template)
}
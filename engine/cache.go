package engine

import (
	"io/fs"
	"sync"
	"text/template"
	"unsafe"
)

type cacheKey struct {
	fsID uintptr
	path string
}

type TemplateCache struct {
	mu        sync.RWMutex
	templates map[cacheKey]*template.Template
}

func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		templates: make(map[cacheKey]*template.Template),
	}
}

func (c *TemplateCache) Get(fsys fs.FS, path string) (*template.Template, error) {
	key := cacheKey{
		fsID: uintptr(unsafe.Pointer(&fsys)),
		path: path,
	}
	
	c.mu.RLock()
	if tmpl, exists := c.templates[key]; exists {
		c.mu.RUnlock()
		return tmpl, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if tmpl, exists := c.templates[key]; exists {
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

	c.templates[key] = tmpl
	return tmpl, nil
}

func (c *TemplateCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.templates = make(map[cacheKey]*template.Template)
}

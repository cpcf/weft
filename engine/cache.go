package engine

import (
	"fmt"
	"io/fs"
	"sync"
	"text/template"

	"github.com/cpcf/weft/render"
)

type cacheKey struct {
	fsIdentifier string
	path         string
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
	fsID, err := c.getFSIdentifier(fsys)
	if err != nil {
		return nil, fmt.Errorf("failed to create fs identifier: %w", err)
	}

	key := cacheKey{
		fsIdentifier: fsID,
		path:         path,
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

	tmpl, err := template.New(path).Funcs(render.DefaultFuncMap()).Parse(string(content))
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

// getFSIdentifier creates a stable filesystem identifier
func (c *TemplateCache) getFSIdentifier(fsys fs.FS) (string, error) {
	// For embed.FS and other deterministic filesystems, use type info
	if stringer, ok := fsys.(fmt.Stringer); ok {
		return fmt.Sprintf("%T:%s", fsys, stringer.String()), nil
	}

	// For os.DirFS-like filesystems, use type name with pointer
	typeName := fmt.Sprintf("%T", fsys)
	if typeName == "*os.dirFS" {
		return fmt.Sprintf("os.DirFS:%p", fsys), nil
	}

	// For other types, use type name (less efficient but safe)
	return fmt.Sprintf("%T:%p", fsys, fsys), nil
}

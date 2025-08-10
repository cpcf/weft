package engine

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type Renderer struct {
	logger *slog.Logger
	cache  *TemplateCache
}

func NewRenderer(logger *slog.Logger, cache *TemplateCache) *Renderer {
	return &Renderer{
		logger: logger,
		cache:  cache,
	}
}

func (r *Renderer) RenderDir(ctx Context, templateDir string, data any) error {
	return fs.WalkDir(ctx.TmplFS, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		return r.renderFile(ctx, path, data)
	})
}

func (r *Renderer) renderFile(ctx Context, templatePath string, data any) error {
	r.logger.Debug("rendering template", "path", templatePath)

	tmpl, err := r.cache.Get(ctx.TmplFS, templatePath)
	if err != nil {
		return fmt.Errorf("failed to get template %s: %w", templatePath, err)
	}

	outputPath := r.resolveOutputPath(ctx, templatePath)
	if err := r.ensureOutputDir(outputPath); err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	r.logger.Info("rendered template", "template", templatePath, "output", outputPath)
	return nil
}

func (r *Renderer) resolveOutputPath(ctx Context, templatePath string) string {
	outputName := strings.TrimSuffix(filepath.Base(templatePath), ".tmpl")
	outputDir := filepath.Join(ctx.OutputRoot, filepath.Dir(templatePath))
	return filepath.Join(outputDir, outputName)
}

func (r *Renderer) ensureOutputDir(outputPath string) error {
	dir := filepath.Dir(outputPath)
	return os.MkdirAll(dir, 0755)
}
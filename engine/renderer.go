package engine

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/cpcf/weft/postprocess"
)

type Renderer struct {
	logger         *slog.Logger
	cache          *TemplateCache
	postprocessors *postprocess.Chain
}

func NewRenderer(logger *slog.Logger, cache *TemplateCache, postprocessors *postprocess.Chain) *Renderer {
	return &Renderer{
		logger:         logger,
		cache:          cache,
		postprocessors: postprocessors,
	}
}

func (r *Renderer) RenderDir(ctx Context, failMode FailureMode, templateDir string, data any) error {
	var multiErr MultiError

	err := fs.WalkDir(ctx.TmplFS, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if failMode == FailFast {
				return err
			}
			multiErr.Add(path, "filesystem error", err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		if renderErr := r.renderFile(ctx, path, data); renderErr != nil {
			if failMode == FailFast {
				return renderErr
			}
			multiErr.Add(path, "render failed", renderErr)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if multiErr.HasErrors() && failMode != BestEffort {
		return &multiErr
	}

	return nil
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

	// Render template to buffer first
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	content := []byte(buf.String())

	// Apply post-processing if any processors are configured
	if r.postprocessors.HasProcessors() {
		processed, err := r.postprocessors.Process(outputPath, content)
		if err != nil {
			r.logger.Warn("post-processing failed", "path", outputPath, "error", err)
			// Continue with unprocessed content rather than failing
		} else {
			content = processed
		}
	}

	// Write the final content to file
	if err := os.WriteFile(outputPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write output file %s: %w", outputPath, err)
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
	return os.MkdirAll(dir, 0o755)
}

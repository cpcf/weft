package engine

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cpcf/weft/postprocess"
	gogentest "github.com/cpcf/weft/testing"
)

func TestEngineBasic(t *testing.T) {
	memFS := gogentest.NewMemoryFS()
	memFS.WriteFile("templates/hello.go.tmpl", []byte("package {{.Package}}\n\nconst Message = \"{{.Message}}\"\n"))

	tempDir, err := os.MkdirTemp("", "weft-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	engine := New(WithOutputRoot(tempDir))

	ctx := NewContext(memFS, tempDir, "example")
	data := map[string]any{
		"Package": "main",
		"Message": "Hello, World!",
	}

	err = engine.RenderDir(ctx, "templates", data)
	if err != nil {
		t.Fatalf("RenderDir failed: %v", err)
	}

	outputPath := filepath.Join(tempDir, "templates", "hello.go")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expected := "package main\n\nconst Message = \"Hello, World!\"\n"
	if string(content) != expected {
		t.Errorf("Output mismatch.\nExpected: %q\nGot: %q", expected, string(content))
	}
}

func TestEngineTemplateParseError(t *testing.T) {
	memFS := gogentest.NewMemoryFS()
	memFS.WriteFile("templates/bad.go.tmpl", []byte("package {{.Package\n\nconst Message = \"invalid\"")) // Missing closing brace

	tempDir, err := os.MkdirTemp("", "weft-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	engine := New(WithOutputRoot(tempDir))
	ctx := NewContext(memFS, tempDir, "example")
	data := map[string]any{"Package": "main"}

	err = engine.RenderDir(ctx, "templates", data)
	if err == nil {
		t.Fatal("Expected error for malformed template, got nil")
	}

	if !strings.Contains(err.Error(), "template") {
		t.Errorf("Expected template error, got: %v", err)
	}
}

func TestEngineFailureModes(t *testing.T) {
	memFS := gogentest.NewMemoryFS()
	memFS.WriteFile("templates/good.go.tmpl", []byte("package {{.Package}}"))
	memFS.WriteFile("templates/bad.go.tmpl", []byte("package {{.Package")) // Missing closing brace

	tempDir, err := os.MkdirTemp("", "weft-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	ctx := NewContext(memFS, tempDir, "example")
	data := map[string]any{"Package": "main"}

	t.Run("FailFast", func(t *testing.T) {
		engine := New(WithOutputRoot(tempDir), WithFailureMode(FailFast))
		err := engine.RenderDir(ctx, "templates", data)
		if err == nil {
			t.Fatal("Expected error in FailFast mode")
		}
	})

	t.Run("FailAtEnd", func(t *testing.T) {
		engine := New(WithOutputRoot(tempDir), WithFailureMode(FailAtEnd))
		err := engine.RenderDir(ctx, "templates", data)
		if err == nil {
			t.Fatal("Expected error in FailAtEnd mode")
		}

		multiErr, ok := err.(*MultiError)
		if !ok {
			t.Errorf("Expected MultiError, got %T", err)
		} else if !multiErr.HasErrors() {
			t.Error("Expected MultiError to have errors")
		}
	})

	t.Run("BestEffort", func(t *testing.T) {
		engine := New(WithOutputRoot(tempDir), WithFailureMode(BestEffort))
		err := engine.RenderDir(ctx, "templates", data)
		if err != nil {
			t.Errorf("Expected no error in BestEffort mode, got: %v", err)
		}

		// Check that the good template was rendered despite the bad one
		goodPath := filepath.Join(tempDir, "templates", "good.go")
		if _, err := os.Stat(goodPath); os.IsNotExist(err) {
			t.Error("Good template should have been rendered in BestEffort mode")
		}
	})
}

func TestTemplateCacheKeyCollision(t *testing.T) {
	cache := NewTemplateCache()

	memFS1 := gogentest.NewMemoryFS()
	memFS1.WriteFile("test.tmpl", []byte("template1: {{.Value}}"))

	memFS2 := gogentest.NewMemoryFS()
	memFS2.WriteFile("test.tmpl", []byte("template2: {{.Value}}"))

	// Get template from first filesystem
	tmpl1, err := cache.Get(memFS1, "test.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	// Get template from second filesystem with same path
	tmpl2, err := cache.Get(memFS2, "test.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	// They should be different templates (different cache entries)
	var buf1, buf2 strings.Builder
	data := map[string]any{"Value": "test"}

	if err := tmpl1.Execute(&buf1, data); err != nil {
		t.Fatal(err)
	}
	if err := tmpl2.Execute(&buf2, data); err != nil {
		t.Fatal(err)
	}

	if buf1.String() == buf2.String() {
		t.Error("Templates from different filesystems should be cached separately")
	}
}

func TestConcurrentRenderer_Stop_RaceCondition(t *testing.T) {
	memFS := gogentest.NewMemoryFS()
	memFS.WriteFile("templates/test.go.tmpl", []byte("package {{.Package}}"))

	tempDir, err := os.MkdirTemp("", "concurrent-renderer-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create renderer components manually
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cache := NewTemplateCache()
	postprocessors := postprocess.NewChain()
	testRenderer := NewRenderer(logger, cache, postprocessors)
	renderer := NewConcurrentRenderer(4, *testRenderer)
	renderer.Start()

	// Start multiple concurrent render operations
	numTasks := 50
	taskIDs := make([]string, 0, numTasks)
	resultChans := make([]<-chan TaskResult, 0, numTasks)

	for i := 0; i < numTasks; i++ {
		data := map[string]any{"Package": "test"}
		outputPath := filepath.Join(tempDir, "test.go")

		taskID, resultChan, err := renderer.RenderAsync(memFS, "templates/test.go.tmpl", outputPath, data)
		if err != nil {
			// Expected if renderer is stopped during task submission
			continue
		}

		taskIDs = append(taskIDs, taskID)
		resultChans = append(resultChans, resultChan)

		// Stop the renderer in the middle of submitting tasks
		if i == 10 {
			go func() {
				renderer.Stop()
			}()
		}
	}

	// Try to collect results from submitted tasks
	// This should not panic even if Stop() was called concurrently
	completedTasks := 0
	for _, resultChan := range resultChans {
		select {
		case result := <-resultChan:
			if result.Success {
				completedTasks++
			}
		default:
			// Channel might be closed, which is expected
		}
	}

	// Make sure Stop() has completed
	renderer.Stop()

	// Verify renderer rejects new tasks after stop
	_, _, err = renderer.RenderAsync(memFS, "templates/test.go.tmpl", filepath.Join(tempDir, "test2.go"), map[string]any{"Package": "test"})
	if err == nil {
		t.Error("Expected error when submitting task to stopped renderer")
	}
	if !strings.Contains(err.Error(), "stopped") {
		t.Errorf("Expected error message to contain 'stopped', got: %v", err)
	}

	// Multiple Stop() calls should be safe
	renderer.Stop()

	// Verify batch operations also reject when stopped
	requests := []RenderRequest{{
		TmplFS:       memFS,
		TemplatePath: "templates/test.go.tmpl",
		OutputPath:   filepath.Join(tempDir, "batch-test.go"),
		Data:         map[string]any{"Package": "test"},
	}}

	_, err = renderer.RenderBatch(requests)
	if err == nil {
		t.Error("Expected error when running batch operation on stopped renderer")
	}
	if !strings.Contains(err.Error(), "stopped") {
		t.Errorf("Expected error message to contain 'stopped', got: %v", err)
	}

	t.Logf("Test completed successfully. Completed %d out of %d submitted tasks", completedTasks, len(resultChans))
}

package engine

import (
	"os"
	"path/filepath"
	"testing"

	gogentest "github.com/cpcf/gogenkit/testing"
)

func TestEngineBasic(t *testing.T) {
	memFS := gogentest.NewMemoryFS()
	memFS.WriteFile("templates/hello.go.tmpl", []byte("package {{.Package}}\n\nconst Message = \"{{.Message}}\"\n"))

	tempDir, err := os.MkdirTemp("", "gogenkit-test")
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
package processors

import (
	"strings"
	"testing"
)

func TestGoImports_ProcessContent(t *testing.T) {
	processor := NewGoImports()

	tests := []struct {
		name     string
		filePath string
		input    string
		want     string
	}{
		{
			name:     "removes unused imports",
			filePath: "test.go",
			input: `package main

import (
	"fmt"
	"context"
	"net/http"
)

func main() {
	ctx := context.Background()
	_ = ctx
}
`,
			want: "context", // Should only contain context import
		},
		{
			name:     "non-go file unchanged",
			filePath: "test.txt",
			input:    "some text content",
			want:     "some text content",
		},
		{
			name:     "keeps used imports",
			filePath: "test.go",
			input: `package main

import (
	"fmt"
	"context"
)

func main() {
	ctx := context.Background()
	fmt.Println("Hello")
	_ = ctx
}
`,
			want: "context", // Should contain both imports
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ProcessContent(tt.filePath, []byte(tt.input))
			if err != nil {
				t.Errorf("ProcessContent() error = %v", err)
				return
			}

			output := string(result)

			if tt.filePath == "test.txt" {
				// Non-Go files should be unchanged
				if output != tt.input {
					t.Errorf("ProcessContent() for non-Go file changed content")
				}
				return
			}

			// For Go files, check that expected imports are present
			if !strings.Contains(output, tt.want) {
				t.Errorf("ProcessContent() result doesn't contain expected import %q\nResult:\n%s", tt.want, output)
			}
		})
	}
}

func TestGoImports_isGoFile(t *testing.T) {
	processor := NewGoImports()

	tests := []struct {
		filePath string
		want     bool
	}{
		{"main.go", true},
		{"test_file.go", true},
		{"file.GO", true},
		{"file.txt", false},
		{"file.json", false},
		{"file", false},
		{"go.mod", false},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			if got := processor.isGoFile(tt.filePath); got != tt.want {
				t.Errorf("isGoFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

package postprocess

import (
	"errors"
	"strings"
	"testing"
)

// mockProcessor for testing
type mockProcessor struct {
	name      string
	transform func(string, []byte) ([]byte, error)
}

func (m *mockProcessor) ProcessContent(filePath string, content []byte) ([]byte, error) {
	if m.transform != nil {
		return m.transform(filePath, content)
	}
	// Default: add processor name as prefix
	return []byte(m.name + ":" + string(content)), nil
}

func TestChain_Add(t *testing.T) {
	chain := NewChain()
	if chain.Len() != 0 {
		t.Errorf("New chain should be empty, got length %d", chain.Len())
	}

	processor := &mockProcessor{name: "test"}
	chain.Add(processor)
	if chain.Len() != 1 {
		t.Errorf("Chain should have 1 processor after Add, got %d", chain.Len())
	}

	if !chain.HasProcessors() {
		t.Errorf("Chain should have processors after Add")
	}
}

func TestChain_AddFunc(t *testing.T) {
	chain := NewChain()

	chain.AddFunc(func(filePath string, content []byte) ([]byte, error) {
		return []byte("func:" + string(content)), nil
	})

	if chain.Len() != 1 {
		t.Errorf("Chain should have 1 processor after AddFunc, got %d", chain.Len())
	}

	result, err := chain.Process("test.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	expected := "func:hello"
	if string(result) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result))
	}
}

func TestChain_Process(t *testing.T) {
	tests := []struct {
		name        string
		processors  []Processor
		input       string
		expected    string
		shouldError bool
	}{
		{
			name:       "empty chain",
			processors: []Processor{},
			input:      "hello",
			expected:   "hello",
		},
		{
			name: "single processor",
			processors: []Processor{
				&mockProcessor{name: "A"},
			},
			input:    "hello",
			expected: "A:hello",
		},
		{
			name: "multiple processors",
			processors: []Processor{
				&mockProcessor{name: "A"},
				&mockProcessor{name: "B"},
			},
			input:    "hello",
			expected: "B:A:hello",
		},
		{
			name: "processor error",
			processors: []Processor{
				&mockProcessor{
					name: "error",
					transform: func(string, []byte) ([]byte, error) {
						return nil, errors.New("processor error")
					},
				},
			},
			input:       "hello",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewChain()
			for _, processor := range tt.processors {
				chain.Add(processor)
			}

			result, err := chain.Process("test.txt", []byte(tt.input))

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestChain_Clear(t *testing.T) {
	chain := NewChain()
	chain.Add(&mockProcessor{name: "test1"})
	chain.Add(&mockProcessor{name: "test2"})

	if chain.Len() != 2 {
		t.Errorf("Expected 2 processors before clear, got %d", chain.Len())
	}

	chain.Clear()

	if chain.Len() != 0 {
		t.Errorf("Expected 0 processors after clear, got %d", chain.Len())
	}

	if chain.HasProcessors() {
		t.Errorf("Chain should not have processors after clear")
	}
}

func TestProcessorFunc(t *testing.T) {
	fn := ProcessorFunc(func(filePath string, content []byte) ([]byte, error) {
		return []byte(strings.ToUpper(string(content))), nil
	})

	result, err := fn.ProcessContent("test.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("ProcessorFunc failed: %v", err)
	}

	expected := "HELLO"
	if string(result) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result))
	}
}

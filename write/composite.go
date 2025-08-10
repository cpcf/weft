package write

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type CompositeWriter struct {
	writers []WriterRule
}

type WriterRule struct {
	Condition WriterCondition
	Writer    Writer
	Priority  int
}

type WriterCondition func(path string) bool

func NewCompositeWriter() *CompositeWriter {
	return &CompositeWriter{
		writers: make([]WriterRule, 0),
	}
}

func (cw *CompositeWriter) AddWriter(condition WriterCondition, writer Writer, priority int) {
	rule := WriterRule{
		Condition: condition,
		Writer:    writer,
		Priority:  priority,
	}

	inserted := false
	for i, existing := range cw.writers {
		if priority > existing.Priority {
			cw.writers = append(cw.writers[:i], append([]WriterRule{rule}, cw.writers[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		cw.writers = append(cw.writers, rule)
	}
}

func (cw *CompositeWriter) Write(path string, content []byte, options WriteOptions) error {
	writer := cw.selectWriter(path)
	if writer == nil {
		return fmt.Errorf("no suitable writer found for path: %s", path)
	}

	return writer.Write(path, content, options)
}

func (cw *CompositeWriter) CanWrite(path string) bool {
	writer := cw.selectWriter(path)
	if writer == nil {
		return false
	}
	return writer.CanWrite(path)
}

func (cw *CompositeWriter) NeedsWrite(path string, content []byte) (bool, error) {
	writer := cw.selectWriter(path)
	if writer == nil {
		return false, fmt.Errorf("no suitable writer found for path: %s", path)
	}

	return writer.NeedsWrite(path, content)
}

func (cw *CompositeWriter) selectWriter(path string) Writer {
	for _, rule := range cw.writers {
		if rule.Condition(path) {
			return rule.Writer
		}
	}
	return nil
}

func (cw *CompositeWriter) GetMatchingWriters(path string) []Writer {
	var writers []Writer
	for _, rule := range cw.writers {
		if rule.Condition(path) {
			writers = append(writers, rule.Writer)
		}
	}
	return writers
}

func ExtensionCondition(extensions ...string) WriterCondition {
	return func(path string) bool {
		for _, ext := range extensions {
			if strings.HasSuffix(path, ext) {
				return true
			}
		}
		return false
	}
}

func PrefixCondition(prefixes ...string) WriterCondition {
	return func(path string) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		return false
	}
}

func PatternCondition(patterns ...string) WriterCondition {
	return func(path string) bool {
		for _, pattern := range patterns {
			if matched, _ := filepath.Match(pattern, path); matched {
				return true
			}
		}
		return false
	}
}

func AndCondition(conditions ...WriterCondition) WriterCondition {
	return func(path string) bool {
		for _, condition := range conditions {
			if !condition(path) {
				return false
			}
		}
		return true
	}
}

func OrCondition(conditions ...WriterCondition) WriterCondition {
	return func(path string) bool {
		for _, condition := range conditions {
			if condition(path) {
				return true
			}
		}
		return false
	}
}

func NotCondition(condition WriterCondition) WriterCondition {
	return func(path string) bool {
		return !condition(path)
	}
}

type ChainedWriter struct {
	writers []Writer
}

func NewChainedWriter(writers ...Writer) *ChainedWriter {
	return &ChainedWriter{
		writers: writers,
	}
}

func (cw *ChainedWriter) Write(path string, content []byte, options WriteOptions) error {
	for i, writer := range cw.writers {
		if err := writer.Write(path, content, options); err != nil {
			return fmt.Errorf("writer %d failed: %w", i, err)
		}
	}
	return nil
}

func (cw *ChainedWriter) CanWrite(path string) bool {
	for _, writer := range cw.writers {
		if !writer.CanWrite(path) {
			return false
		}
	}
	return true
}

func (cw *ChainedWriter) NeedsWrite(path string, content []byte) (bool, error) {
	for _, writer := range cw.writers {
		needs, err := writer.NeedsWrite(path, content)
		if err != nil {
			return false, err
		}
		if needs {
			return true, nil
		}
	}
	return false, nil
}

type ConditionalWriter struct {
	condition WriterCondition
	primary   Writer
	secondary Writer
}

func NewConditionalWriter(condition WriterCondition, primary, secondary Writer) *ConditionalWriter {
	return &ConditionalWriter{
		condition: condition,
		primary:   primary,
		secondary: secondary,
	}
}

func (cw *ConditionalWriter) Write(path string, content []byte, options WriteOptions) error {
	if cw.condition(path) {
		return cw.primary.Write(path, content, options)
	}
	if cw.secondary != nil {
		return cw.secondary.Write(path, content, options)
	}
	return fmt.Errorf("no writer available for path: %s", path)
}

func (cw *ConditionalWriter) CanWrite(path string) bool {
	if cw.condition(path) {
		return cw.primary.CanWrite(path)
	}
	if cw.secondary != nil {
		return cw.secondary.CanWrite(path)
	}
	return false
}

func (cw *ConditionalWriter) NeedsWrite(path string, content []byte) (bool, error) {
	if cw.condition(path) {
		return cw.primary.NeedsWrite(path, content)
	}
	if cw.secondary != nil {
		return cw.secondary.NeedsWrite(path, content)
	}
	return false, fmt.Errorf("no writer available for path: %s", path)
}

type RetryWriter struct {
	writer     Writer
	maxRetries int
	backoff    func(int) time.Duration
}

func NewRetryWriter(writer Writer, maxRetries int) *RetryWriter {
	return &RetryWriter{
		writer:     writer,
		maxRetries: maxRetries,
		backoff: func(attempt int) time.Duration {
			return time.Duration(attempt) * 100 * time.Millisecond
		},
	}
}

func (rw *RetryWriter) SetBackoff(backoff func(int) time.Duration) {
	rw.backoff = backoff
}

func (rw *RetryWriter) Write(path string, content []byte, options WriteOptions) error {
	var lastErr error

	for attempt := 0; attempt <= rw.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(rw.backoff(attempt))
		}

		if err := rw.writer.Write(path, content, options); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", rw.maxRetries+1, lastErr)
}

func (rw *RetryWriter) CanWrite(path string) bool {
	return rw.writer.CanWrite(path)
}

func (rw *RetryWriter) NeedsWrite(path string, content []byte) (bool, error) {
	return rw.writer.NeedsWrite(path, content)
}

type ValidatingWriter struct {
	writer    Writer
	validator func(path string, content []byte) error
}

func NewValidatingWriter(writer Writer, validator func(path string, content []byte) error) *ValidatingWriter {
	return &ValidatingWriter{
		writer:    writer,
		validator: validator,
	}
}

func (vw *ValidatingWriter) Write(path string, content []byte, options WriteOptions) error {
	if err := vw.validator(path, content); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return vw.writer.Write(path, content, options)
}

func (vw *ValidatingWriter) CanWrite(path string) bool {
	return vw.writer.CanWrite(path)
}

func (vw *ValidatingWriter) NeedsWrite(path string, content []byte) (bool, error) {
	return vw.writer.NeedsWrite(path, content)
}

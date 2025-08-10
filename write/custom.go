package write

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

type SkipIfExistsWriter struct {
	baseWriter Writer
}

func NewSkipIfExistsWriter(baseWriter Writer) *SkipIfExistsWriter {
	if baseWriter == nil {
		baseWriter = NewBaseWriter()
	}
	return &SkipIfExistsWriter{
		baseWriter: baseWriter,
	}
}

func (siw *SkipIfExistsWriter) Write(path string, content []byte, options WriteOptions) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return siw.baseWriter.Write(path, content, options)
}

func (siw *SkipIfExistsWriter) CanWrite(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return false
	}
	return siw.baseWriter.CanWrite(path)
}

func (siw *SkipIfExistsWriter) NeedsWrite(path string, content []byte) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	return siw.baseWriter.NeedsWrite(path, content)
}

type ReplaceSegmentWriter struct {
	baseWriter  Writer
	beginMarker string
	endMarker   string
}

func NewReplaceSegmentWriter(baseWriter Writer) *ReplaceSegmentWriter {
	if baseWriter == nil {
		baseWriter = NewBaseWriter()
	}
	return &ReplaceSegmentWriter{
		baseWriter:  baseWriter,
		beginMarker: "// BEGIN GENERATED",
		endMarker:   "// END GENERATED",
	}
}

func (rsw *ReplaceSegmentWriter) SetMarkers(begin, end string) {
	rsw.beginMarker = begin
	rsw.endMarker = end
}

func (rsw *ReplaceSegmentWriter) Write(path string, content []byte, options WriteOptions) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return rsw.baseWriter.Write(path, content, options)
	}

	existing, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	updated, err := rsw.replaceSegment(string(existing), string(content))
	if err != nil {
		return fmt.Errorf("failed to replace segment: %w", err)
	}

	return rsw.baseWriter.Write(path, []byte(updated), options)
}

func (rsw *ReplaceSegmentWriter) CanWrite(path string) bool {
	return rsw.baseWriter.CanWrite(path)
}

func (rsw *ReplaceSegmentWriter) NeedsWrite(path string, content []byte) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return true, nil
	}

	existing, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read existing file: %w", err)
	}

	updated, err := rsw.replaceSegment(string(existing), string(content))
	if err != nil {
		return false, fmt.Errorf("failed to replace segment: %w", err)
	}

	return string(existing) != updated, nil
}

func (rsw *ReplaceSegmentWriter) replaceSegment(existing, newContent string) (string, error) {
	beginPattern := regexp.QuoteMeta(rsw.beginMarker)
	endPattern := regexp.QuoteMeta(rsw.endMarker)

	pattern := fmt.Sprintf(`(?s)%s.*?%s`, beginPattern, endPattern)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %w", err)
	}

	replacement := fmt.Sprintf("%s\n%s\n%s", rsw.beginMarker, strings.TrimSpace(newContent), rsw.endMarker)

	if re.MatchString(existing) {
		return re.ReplaceAllString(existing, replacement), nil
	}

	return existing + "\n" + replacement + "\n", nil
}

type AppendSectionWriter struct {
	baseWriter Writer
	anchor     string
}

func NewAppendSectionWriter(baseWriter Writer, anchor string) *AppendSectionWriter {
	if baseWriter == nil {
		baseWriter = NewBaseWriter()
	}
	return &AppendSectionWriter{
		baseWriter: baseWriter,
		anchor:     anchor,
	}
}

func (asw *AppendSectionWriter) Write(path string, content []byte, options WriteOptions) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return asw.baseWriter.Write(path, content, options)
	}

	existing, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	updated, err := asw.appendSection(string(existing), string(content))
	if err != nil {
		return fmt.Errorf("failed to append section: %w", err)
	}

	return asw.baseWriter.Write(path, []byte(updated), options)
}

func (asw *AppendSectionWriter) CanWrite(path string) bool {
	return asw.baseWriter.CanWrite(path)
}

func (asw *AppendSectionWriter) NeedsWrite(path string, content []byte) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return true, nil
	}

	existing, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read existing file: %w", err)
	}

	return !strings.Contains(string(existing), string(content)), nil
}

func (asw *AppendSectionWriter) appendSection(existing, newContent string) (string, error) {
	if asw.anchor == "" {
		return existing + "\n" + newContent, nil
	}

	lines := strings.Split(existing, "\n")
	anchorIndex := -1

	for i, line := range lines {
		if strings.Contains(line, asw.anchor) {
			anchorIndex = i
			break
		}
	}

	if anchorIndex == -1 {
		return existing + "\n" + newContent, nil
	}

	before := lines[:anchorIndex+1]
	after := lines[anchorIndex+1:]

	result := strings.Join(before, "\n") + "\n" + newContent
	if len(after) > 0 {
		result += "\n" + strings.Join(after, "\n")
	}

	return result, nil
}

type TimestampWriter struct {
	baseWriter Writer
	format     string
}

func NewTimestampWriter(baseWriter Writer) *TimestampWriter {
	if baseWriter == nil {
		baseWriter = NewBaseWriter()
	}
	return &TimestampWriter{
		baseWriter: baseWriter,
		format:     "// Generated at: 2006-01-02 15:04:05\n\n",
	}
}

func (tw *TimestampWriter) SetFormat(format string) {
	tw.format = format
}

func (tw *TimestampWriter) Write(path string, content []byte, options WriteOptions) error {
	timestamp := fmt.Sprintf(tw.format, time.Now().Format("2006-01-02 15:04:05"))
	prefixed := timestamp + string(content)
	return tw.baseWriter.Write(path, []byte(prefixed), options)
}

func (tw *TimestampWriter) CanWrite(path string) bool {
	return tw.baseWriter.CanWrite(path)
}

func (tw *TimestampWriter) NeedsWrite(path string, content []byte) (bool, error) {
	timestamp := fmt.Sprintf(tw.format, time.Now().Format("2006-01-02 15:04:05"))
	prefixed := timestamp + string(content)
	return tw.baseWriter.NeedsWrite(path, []byte(prefixed))
}

type TemplateWriter struct {
	baseWriter Writer
	header     string
	footer     string
}

func NewTemplateWriter(baseWriter Writer, header, footer string) *TemplateWriter {
	if baseWriter == nil {
		baseWriter = NewBaseWriter()
	}
	return &TemplateWriter{
		baseWriter: baseWriter,
		header:     header,
		footer:     footer,
	}
}

func (tw *TemplateWriter) Write(path string, content []byte, options WriteOptions) error {
	wrapped := tw.header + string(content) + tw.footer
	return tw.baseWriter.Write(path, []byte(wrapped), options)
}

func (tw *TemplateWriter) CanWrite(path string) bool {
	return tw.baseWriter.CanWrite(path)
}

func (tw *TemplateWriter) NeedsWrite(path string, content []byte) (bool, error) {
	wrapped := tw.header + string(content) + tw.footer
	return tw.baseWriter.NeedsWrite(path, []byte(wrapped))
}

type DryRunWriter struct {
	changes []Change
}

type Change struct {
	Path      string    `json:"path"`
	Action    string    `json:"action"`
	Size      int       `json:"size"`
	Timestamp time.Time `json:"timestamp"`
}

func NewDryRunWriter() *DryRunWriter {
	return &DryRunWriter{
		changes: make([]Change, 0),
	}
}

func (drw *DryRunWriter) Write(path string, content []byte, options WriteOptions) error {
	action := "create"
	if _, err := os.Stat(path); err == nil {
		action = "update"
	}

	change := Change{
		Path:      path,
		Action:    action,
		Size:      len(content),
		Timestamp: time.Now(),
	}

	drw.changes = append(drw.changes, change)
	return nil
}

func (drw *DryRunWriter) CanWrite(path string) bool {
	return true
}

func (drw *DryRunWriter) NeedsWrite(path string, content []byte) (bool, error) {
	return true, nil
}

func (drw *DryRunWriter) GetChanges() []Change {
	return drw.changes
}

func (drw *DryRunWriter) Reset() {
	drw.changes = make([]Change, 0)
}

type LoggingWriter struct {
	baseWriter Writer
	logFunc    func(string, ...any)
}

func NewLoggingWriter(baseWriter Writer, logFunc func(string, ...any)) *LoggingWriter {
	if baseWriter == nil {
		baseWriter = NewBaseWriter()
	}
	if logFunc == nil {
		logFunc = func(format string, args ...any) {
			fmt.Printf(format+"\n", args...)
		}
	}
	return &LoggingWriter{
		baseWriter: baseWriter,
		logFunc:    logFunc,
	}
}

func (lw *LoggingWriter) Write(path string, content []byte, options WriteOptions) error {
	lw.logFunc("Writing %d bytes to %s", len(content), path)

	start := time.Now()
	err := lw.baseWriter.Write(path, content, options)
	duration := time.Since(start)

	if err != nil {
		lw.logFunc("Failed to write %s: %v (took %v)", path, err, duration)
	} else {
		lw.logFunc("Successfully wrote %s (took %v)", path, duration)
	}

	return err
}

func (lw *LoggingWriter) CanWrite(path string) bool {
	can := lw.baseWriter.CanWrite(path)
	lw.logFunc("CanWrite %s: %v", path, can)
	return can
}

func (lw *LoggingWriter) NeedsWrite(path string, content []byte) (bool, error) {
	needs, err := lw.baseWriter.NeedsWrite(path, content)
	if err != nil {
		lw.logFunc("Error checking if %s needs write: %v", path, err)
	} else {
		lw.logFunc("NeedsWrite %s: %v", path, needs)
	}
	return needs, err
}

type FilterWriter struct {
	baseWriter Writer
	filter     func(path string, content []byte) ([]byte, error)
}

func NewFilterWriter(baseWriter Writer, filter func(path string, content []byte) ([]byte, error)) *FilterWriter {
	if baseWriter == nil {
		baseWriter = NewBaseWriter()
	}
	return &FilterWriter{
		baseWriter: baseWriter,
		filter:     filter,
	}
}

func (fw *FilterWriter) Write(path string, content []byte, options WriteOptions) error {
	if fw.filter != nil {
		filtered, err := fw.filter(path, content)
		if err != nil {
			return fmt.Errorf("filter failed: %w", err)
		}
		content = filtered
	}

	return fw.baseWriter.Write(path, content, options)
}

func (fw *FilterWriter) CanWrite(path string) bool {
	return fw.baseWriter.CanWrite(path)
}

func (fw *FilterWriter) NeedsWrite(path string, content []byte) (bool, error) {
	if fw.filter != nil {
		filtered, err := fw.filter(path, content)
		if err != nil {
			return false, fmt.Errorf("filter failed: %w", err)
		}
		content = filtered
	}

	return fw.baseWriter.NeedsWrite(path, content)
}

package testing

import (
	"fmt"
	"io/fs"
	"sync"
	"time"
)

type MockFS struct {
	files map[string]MockFile
	dirs  map[string]MockDir
	mu    sync.RWMutex
}

type MockFile struct {
	Name    string
	Content []byte
	ModTime time.Time
	Mode    fs.FileMode
}

type MockDir struct {
	Name    string
	Entries []fs.DirEntry
	ModTime time.Time
	Mode    fs.FileMode
}

type MockDirEntry struct {
	name    string
	isDir   bool
	modTime time.Time
	mode    fs.FileMode
	size    int64
}

type MockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func NewMockFS() *MockFS {
	return &MockFS{
		files: make(map[string]MockFile),
		dirs:  make(map[string]MockDir),
	}
}

func (mfs *MockFS) AddFile(path string, content string) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	mfs.files[path] = MockFile{
		Name:    path,
		Content: []byte(content),
		ModTime: time.Now(),
		Mode:    0o644,
	}
}

func (mfs *MockFS) AddFileWithMode(path string, content string, mode fs.FileMode) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	mfs.files[path] = MockFile{
		Name:    path,
		Content: []byte(content),
		ModTime: time.Now(),
		Mode:    mode,
	}
}

func (mfs *MockFS) AddDir(path string) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	mfs.dirs[path] = MockDir{
		Name:    path,
		Entries: make([]fs.DirEntry, 0),
		ModTime: time.Now(),
		Mode:    0o755,
	}
}

func (mfs *MockFS) RemoveFile(path string) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	delete(mfs.files, path)
}

func (mfs *MockFS) RemoveDir(path string) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	delete(mfs.dirs, path)
}

func (mfs *MockFS) Open(name string) (fs.File, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	if file, exists := mfs.files[name]; exists {
		return &MockFileHandle{
			file:   file,
			offset: 0,
		}, nil
	}

	if dir, exists := mfs.dirs[name]; exists {
		return &MockDirHandle{
			dir: dir,
		}, nil
	}

	return nil, fmt.Errorf("file not found: %s", name)
}

func (mfs *MockFS) Stat(name string) (fs.FileInfo, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	if file, exists := mfs.files[name]; exists {
		return MockFileInfo{
			name:    file.Name,
			size:    int64(len(file.Content)),
			mode:    file.Mode,
			modTime: file.ModTime,
			isDir:   false,
		}, nil
	}

	if dir, exists := mfs.dirs[name]; exists {
		return MockFileInfo{
			name:    dir.Name,
			size:    0,
			mode:    dir.Mode,
			modTime: dir.ModTime,
			isDir:   true,
		}, nil
	}

	return nil, fmt.Errorf("file not found: %s", name)
}

func (mfs *MockFS) ReadFile(name string) ([]byte, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	if file, exists := mfs.files[name]; exists {
		content := make([]byte, len(file.Content))
		copy(content, file.Content)
		return content, nil
	}

	return nil, fmt.Errorf("file not found: %s", name)
}

func (mfs *MockFS) ReadDir(name string) ([]fs.DirEntry, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	if dir, exists := mfs.dirs[name]; exists {
		entries := make([]fs.DirEntry, len(dir.Entries))
		copy(entries, dir.Entries)
		return entries, nil
	}

	return nil, fmt.Errorf("directory not found: %s", name)
}

func (mfs *MockFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	visited := make(map[string]bool)

	var walk func(path string) error
	walk = func(path string) error {
		if visited[path] {
			return nil
		}
		visited[path] = true

		if file, exists := mfs.files[path]; exists {
			entry := MockDirEntry{
				name:    file.Name,
				isDir:   false,
				modTime: file.ModTime,
				mode:    file.Mode,
				size:    int64(len(file.Content)),
			}
			return fn(path, entry, nil)
		}

		if dir, exists := mfs.dirs[path]; exists {
			entry := MockDirEntry{
				name:    dir.Name,
				isDir:   true,
				modTime: dir.ModTime,
				mode:    dir.Mode,
				size:    0,
			}
			if err := fn(path, entry, nil); err != nil {
				return err
			}

			for _, childEntry := range dir.Entries {
				childPath := path + "/" + childEntry.Name()
				if err := walk(childPath); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return walk(root)
}

type MockFileHandle struct {
	file   MockFile
	offset int
}

func (mfh *MockFileHandle) Stat() (fs.FileInfo, error) {
	return MockFileInfo{
		name:    mfh.file.Name,
		size:    int64(len(mfh.file.Content)),
		mode:    mfh.file.Mode,
		modTime: mfh.file.ModTime,
		isDir:   false,
	}, nil
}

func (mfh *MockFileHandle) Read(b []byte) (int, error) {
	if mfh.offset >= len(mfh.file.Content) {
		return 0, fmt.Errorf("EOF")
	}

	n := copy(b, mfh.file.Content[mfh.offset:])
	mfh.offset += n
	return n, nil
}

func (mfh *MockFileHandle) Close() error {
	return nil
}

type MockDirHandle struct {
	dir MockDir
}

func (mdh *MockDirHandle) Stat() (fs.FileInfo, error) {
	return MockFileInfo{
		name:    mdh.dir.Name,
		size:    0,
		mode:    mdh.dir.Mode,
		modTime: mdh.dir.ModTime,
		isDir:   true,
	}, nil
}

func (mdh *MockDirHandle) Read([]byte) (int, error) {
	return 0, fmt.Errorf("is a directory")
}

func (mdh *MockDirHandle) Close() error {
	return nil
}

func (mde MockDirEntry) Name() string {
	return mde.name
}

func (mde MockDirEntry) IsDir() bool {
	return mde.isDir
}

func (mde MockDirEntry) Type() fs.FileMode {
	return mde.mode
}

func (mde MockDirEntry) Info() (fs.FileInfo, error) {
	return MockFileInfo{
		name:    mde.name,
		size:    mde.size,
		mode:    mde.mode,
		modTime: mde.modTime,
		isDir:   mde.isDir,
	}, nil
}

func (mfi MockFileInfo) Name() string {
	return mfi.name
}

func (mfi MockFileInfo) Size() int64 {
	return mfi.size
}

func (mfi MockFileInfo) Mode() fs.FileMode {
	return mfi.mode
}

func (mfi MockFileInfo) ModTime() time.Time {
	return mfi.modTime
}

func (mfi MockFileInfo) IsDir() bool {
	return mfi.isDir
}

func (mfi MockFileInfo) Sys() any {
	return nil
}

type MockLogger struct {
	logs []MockLogEntry
	mu   sync.RWMutex
}

type MockLogEntry struct {
	Level   string    `json:"level"`
	Message string    `json:"message"`
	Args    []any     `json:"args"`
	Time    time.Time `json:"time"`
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs: make([]MockLogEntry, 0),
	}
}

func (ml *MockLogger) Debug(msg string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.logs = append(ml.logs, MockLogEntry{
		Level:   "DEBUG",
		Message: msg,
		Args:    args,
		Time:    time.Now(),
	})
}

func (ml *MockLogger) Info(msg string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.logs = append(ml.logs, MockLogEntry{
		Level:   "INFO",
		Message: msg,
		Args:    args,
		Time:    time.Now(),
	})
}

func (ml *MockLogger) Warn(msg string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.logs = append(ml.logs, MockLogEntry{
		Level:   "WARN",
		Message: msg,
		Args:    args,
		Time:    time.Now(),
	})
}

func (ml *MockLogger) Error(msg string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.logs = append(ml.logs, MockLogEntry{
		Level:   "ERROR",
		Message: msg,
		Args:    args,
		Time:    time.Now(),
	})
}

func (ml *MockLogger) GetLogs() []MockLogEntry {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	logs := make([]MockLogEntry, len(ml.logs))
	copy(logs, ml.logs)
	return logs
}

func (ml *MockLogger) GetLogsByLevel(level string) []MockLogEntry {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	var filtered []MockLogEntry
	for _, log := range ml.logs {
		if log.Level == level {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

func (ml *MockLogger) Clear() {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.logs = make([]MockLogEntry, 0)
}

func (ml *MockLogger) HasMessage(message string) bool {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	for _, log := range ml.logs {
		if log.Message == message {
			return true
		}
	}
	return false
}

func (ml *MockLogger) CountByLevel(level string) int {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	count := 0
	for _, log := range ml.logs {
		if log.Level == level {
			count++
		}
	}
	return count
}

type MockRenderer struct {
	renderFunc func(templatePath string, data any) (string, error)
	calls      []MockRenderCall
	mu         sync.RWMutex
}

type MockRenderCall struct {
	TemplatePath string    `json:"template_path"`
	Data         any       `json:"data"`
	Result       string    `json:"result"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

func NewMockRenderer() *MockRenderer {
	return &MockRenderer{
		calls: make([]MockRenderCall, 0),
		renderFunc: func(templatePath string, data any) (string, error) {
			return fmt.Sprintf("rendered: %s with %+v", templatePath, data), nil
		},
	}
}

func (mr *MockRenderer) SetRenderFunc(fn func(templatePath string, data any) (string, error)) {
	mr.renderFunc = fn
}

func (mr *MockRenderer) Render(templatePath string, data any) (string, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	call := MockRenderCall{
		TemplatePath: templatePath,
		Data:         data,
		Timestamp:    time.Now(),
	}

	result, err := mr.renderFunc(templatePath, data)
	call.Result = result
	if err != nil {
		call.Error = err.Error()
	}

	mr.calls = append(mr.calls, call)
	return result, err
}

func (mr *MockRenderer) GetCalls() []MockRenderCall {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	calls := make([]MockRenderCall, len(mr.calls))
	copy(calls, mr.calls)
	return calls
}

func (mr *MockRenderer) GetCallCount() int {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	return len(mr.calls)
}

func (mr *MockRenderer) GetCallsForTemplate(templatePath string) []MockRenderCall {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	var filtered []MockRenderCall
	for _, call := range mr.calls {
		if call.TemplatePath == templatePath {
			filtered = append(filtered, call)
		}
	}
	return filtered
}

func (mr *MockRenderer) Clear() {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.calls = make([]MockRenderCall, 0)
}

func (mr *MockRenderer) WasCalled(templatePath string) bool {
	return len(mr.GetCallsForTemplate(templatePath)) > 0
}

package testing

import (
	"io"
	"io/fs"
	"path"
	"sort"
	"time"
)

type MemoryFS struct {
	files map[string]*MemoryFile
}

type MemoryFile struct {
	name    string
	content []byte
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func NewMemoryFS() *MemoryFS {
	return &MemoryFS{
		files: make(map[string]*MemoryFile),
	}
}

func (mfs *MemoryFS) WriteFile(name string, data []byte) {
	name = path.Clean(name)
	mfs.files[name] = &MemoryFile{
		name:    name,
		content: data,
		mode:    0o644,
		modTime: time.Now(),
		isDir:   false,
	}

	mfs.ensureDir(path.Dir(name))
}

func (mfs *MemoryFS) ensureDir(dir string) {
	if dir == "." || dir == "/" {
		return
	}

	if _, exists := mfs.files[dir]; !exists {
		mfs.files[dir] = &MemoryFile{
			name:    dir,
			mode:    0o755 | fs.ModeDir,
			modTime: time.Now(),
			isDir:   true,
		}
		mfs.ensureDir(path.Dir(dir))
	}
}

func (mfs *MemoryFS) Open(name string) (fs.File, error) {
	name = path.Clean(name)
	file, exists := mfs.files[name]
	if !exists {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &memoryFileHandle{file: file, mfs: mfs, path: name}, nil
}

func (mfs *MemoryFS) ReadDir(name string) ([]fs.DirEntry, error) {
	name = path.Clean(name)
	if name == "." {
		name = ""
	}

	var entries []fs.DirEntry
	for filePath, file := range mfs.files {
		if path.Dir(filePath) == name {
			entries = append(entries, &memoryDirEntry{file})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

func (mfs *MemoryFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(mfs, root, fn)
}

type memoryFileHandle struct {
	file   *MemoryFile
	mfs    *MemoryFS
	path   string
	offset int
}

func (f *memoryFileHandle) Read(b []byte) (int, error) {
	if f.file.isDir {
		return 0, &fs.PathError{Op: "read", Path: f.path, Err: fs.ErrInvalid}
	}

	if f.offset >= len(f.file.content) {
		return 0, io.EOF
	}

	n := copy(b, f.file.content[f.offset:])
	f.offset += n

	if f.offset >= len(f.file.content) {
		return n, io.EOF
	}

	return n, nil
}

func (f *memoryFileHandle) Stat() (fs.FileInfo, error) {
	return f.file, nil
}

func (f *memoryFileHandle) Close() error {
	return nil
}

func (f *memoryFileHandle) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.file.isDir {
		return nil, &fs.PathError{Op: "readdir", Path: f.path, Err: fs.ErrInvalid}
	}

	entries, err := f.mfs.ReadDir(f.path)
	if err != nil {
		return nil, err
	}

	if n <= 0 {
		return entries, nil
	}

	if n > len(entries) {
		n = len(entries)
	}

	return entries[:n], nil
}

type memoryDirEntry struct {
	file *MemoryFile
}

func (e *memoryDirEntry) Name() string {
	return path.Base(e.file.name)
}

func (e *memoryDirEntry) IsDir() bool {
	return e.file.isDir
}

func (e *memoryDirEntry) Type() fs.FileMode {
	return e.file.mode.Type()
}

func (e *memoryDirEntry) Info() (fs.FileInfo, error) {
	return e.file, nil
}

func (f *MemoryFile) Name() string {
	return path.Base(f.name)
}

func (f *MemoryFile) Size() int64 {
	return int64(len(f.content))
}

func (f *MemoryFile) Mode() fs.FileMode {
	return f.mode
}

func (f *MemoryFile) ModTime() time.Time {
	return f.modTime
}

func (f *MemoryFile) IsDir() bool {
	return f.isDir
}

func (f *MemoryFile) Sys() any {
	return nil
}

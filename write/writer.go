// Package write provides a generic interface for writing files.
package write

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Writer interface {
	Write(path string, content []byte, options WriteOptions) error
	CanWrite(path string) bool
	NeedsWrite(path string, content []byte) (bool, error)
}

type WriteOptions struct {
	CreateDirs bool
	Backup     bool
	BackupDir  string
	Overwrite  bool
	Atomic     bool
}

type BaseWriter struct{}

func NewBaseWriter() *BaseWriter {
	return &BaseWriter{}
}

func (bw *BaseWriter) Write(path string, content []byte, options WriteOptions) error {
	if options.CreateDirs {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}
	}

	if options.Backup {
		if err := bw.createBackup(path, options.BackupDir); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	if options.Atomic {
		return bw.atomicWrite(path, content)
	}

	return bw.directWrite(path, content, options.Overwrite)
}

func (bw *BaseWriter) CanWrite(path string) bool {
	return true
}

func (bw *BaseWriter) NeedsWrite(path string, content []byte) (bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	return string(existing) != string(content), nil
}

func (bw *BaseWriter) createBackup(path, backupDir string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	if backupDir == "" {
		backupDir = filepath.Dir(path)
	}

	backupPath := filepath.Join(backupDir, filepath.Base(path)+".bak")

	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()

	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}

	output, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	return err
}

func (bw *BaseWriter) atomicWrite(path string, content []byte) error {
	tempPath := path + ".tmp"

	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	if _, err := file.Write(content); err != nil {
		file.Close()
		os.Remove(tempPath)
		return err
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}

	return os.Rename(tempPath, path)
}

func (bw *BaseWriter) directWrite(path string, content []byte, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file already exists and overwrite is false: %s", path)
		}
	}

	return os.WriteFile(path, content, 0o644)
}

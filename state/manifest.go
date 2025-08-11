// Package state provides a manifest for tracking the state of generated files.
package state

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type ManifestEntry struct {
	Path         string            `json:"path"`
	Hash         string            `json:"hash"`
	Size         int64             `json:"size"`
	ModTime      time.Time         `json:"mod_time"`
	GeneratedBy  string            `json:"generated_by"`
	TemplatePath string            `json:"template_path"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type Manifest struct {
	Version    string                   `json:"version"`
	Generated  time.Time                `json:"generated"`
	Generator  string                   `json:"generator"`
	OutputRoot string                   `json:"output_root"`
	Entries    map[string]ManifestEntry `json:"entries"`
	Metadata   map[string]string        `json:"metadata,omitempty"`
}

type ManifestManager struct {
	outputRoot   string
	manifestPath string
}

func NewManifestManager(outputRoot string) *ManifestManager {
	return &ManifestManager{
		outputRoot:   outputRoot,
		manifestPath: filepath.Join(outputRoot, ".weft.manifest.json"),
	}
}

func (mm *ManifestManager) LoadManifest() (*Manifest, error) {
	if _, err := os.Stat(mm.manifestPath); os.IsNotExist(err) {
		return mm.createEmptyManifest(), nil
	}

	file, err := os.Open(mm.manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest file: %w", err)
	}
	defer file.Close()

	var manifest Manifest
	if err := json.NewDecoder(file).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	return &manifest, nil
}

func (mm *ManifestManager) SaveManifest(manifest *Manifest) error {
	if err := os.MkdirAll(filepath.Dir(mm.manifestPath), 0o755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	tmpPath := mm.manifestPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary manifest file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temporary manifest file: %w", err)
	}

	if err := os.Rename(tmpPath, mm.manifestPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to move manifest file: %w", err)
	}

	return nil
}

func (mm *ManifestManager) AddEntry(manifest *Manifest, path, templatePath string, metadata map[string]string) error {
	fullPath := filepath.Join(mm.outputRoot, path)
	stat, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", fullPath, err)
	}

	hash, err := mm.calculateFileHash(fullPath)
	if err != nil {
		return fmt.Errorf("failed to calculate hash for %s: %w", fullPath, err)
	}

	entry := ManifestEntry{
		Path:         path,
		Hash:         hash,
		Size:         stat.Size(),
		ModTime:      stat.ModTime(),
		GeneratedBy:  "weft",
		TemplatePath: templatePath,
		Metadata:     metadata,
	}

	if manifest.Entries == nil {
		manifest.Entries = make(map[string]ManifestEntry)
	}
	manifest.Entries[path] = entry
	manifest.Generated = time.Now()

	return nil
}

func (mm *ManifestManager) RemoveEntry(manifest *Manifest, path string) {
	if manifest.Entries != nil {
		delete(manifest.Entries, path)
		manifest.Generated = time.Now()
	}
}

func (mm *ManifestManager) GetEntry(manifest *Manifest, path string) (ManifestEntry, bool) {
	if manifest.Entries == nil {
		return ManifestEntry{}, false
	}
	entry, exists := manifest.Entries[path]
	return entry, exists
}

func (mm *ManifestManager) ListEntries(manifest *Manifest) []ManifestEntry {
	if manifest.Entries == nil {
		return nil
	}

	entries := make([]ManifestEntry, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		entries = append(entries, entry)
	}
	return entries
}

func (mm *ManifestManager) HasChanged(manifest *Manifest, path string) (bool, error) {
	entry, exists := mm.GetEntry(manifest, path)
	if !exists {
		return true, nil
	}

	fullPath := filepath.Join(mm.outputRoot, path)
	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("failed to stat file %s: %w", fullPath, err)
	}

	if stat.Size() != entry.Size || !stat.ModTime().Equal(entry.ModTime) {
		return true, nil
	}

	hash, err := mm.calculateFileHash(fullPath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate hash for %s: %w", fullPath, err)
	}

	return hash != entry.Hash, nil
}

func (mm *ManifestManager) createEmptyManifest() *Manifest {
	return &Manifest{
		Version:    "1.0",
		Generated:  time.Now(),
		Generator:  "weft",
		OutputRoot: mm.outputRoot,
		Entries:    make(map[string]ManifestEntry),
		Metadata:   make(map[string]string),
	}
}

func (mm *ManifestManager) calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := make([]byte, 0, 1024)
	buf := make([]byte, 4096)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}
		hash = append(hash, buf[:n]...)
	}

	return fmt.Sprintf("%x", hash), nil
}

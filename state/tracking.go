package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TrackingMode int

const (
	TrackingModeDisabled TrackingMode = iota
	TrackingModeEnabled
	TrackingModeStrict
)

type FileState int

const (
	FileStateUnknown FileState = iota
	FileStateGenerated
	FileStateModified
	FileStateDeleted
	FileStateOrphan
)

func (fs FileState) String() string {
	switch fs {
	case FileStateGenerated:
		return "generated"
	case FileStateModified:
		return "modified"
	case FileStateDeleted:
		return "deleted"
	case FileStateOrphan:
		return "orphan"
	default:
		return "unknown"
	}
}

type TrackedFile struct {
	Path         string            `json:"path"`
	State        FileState         `json:"state"`
	LastSeen     time.Time         `json:"last_seen"`
	TemplatePath string            `json:"template_path"`
	Hash         string            `json:"hash"`
	Size         int64             `json:"size"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type StateTracker struct {
	manifestManager *ManifestManager
	mode            TrackingMode
}

func NewStateTracker(outputRoot string, mode TrackingMode) *StateTracker {
	return &StateTracker{
		manifestManager: NewManifestManager(outputRoot),
		mode:            mode,
	}
}

func (st *StateTracker) TrackFile(path, templatePath string, metadata map[string]string) error {
	if st.mode == TrackingModeDisabled {
		return nil
	}

	manifest, err := st.manifestManager.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if err := st.manifestManager.AddEntry(manifest, path, templatePath, metadata); err != nil {
		return fmt.Errorf("failed to add entry to manifest: %w", err)
	}

	return st.manifestManager.SaveManifest(manifest)
}

func (st *StateTracker) UntrackFile(path string) error {
	if st.mode == TrackingModeDisabled {
		return nil
	}

	manifest, err := st.manifestManager.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	st.manifestManager.RemoveEntry(manifest, path)
	return st.manifestManager.SaveManifest(manifest)
}

func (st *StateTracker) GetFileState(path string) (FileState, error) {
	if st.mode == TrackingModeDisabled {
		return FileStateUnknown, nil
	}

	manifest, err := st.manifestManager.LoadManifest()
	if err != nil {
		return FileStateUnknown, fmt.Errorf("failed to load manifest: %w", err)
	}

	entry, exists := st.manifestManager.GetEntry(manifest, path)
	if !exists {
		fullPath := filepath.Join(st.manifestManager.outputRoot, path)
		if _, err := os.Stat(fullPath); err != nil {
			if os.IsNotExist(err) {
				return FileStateUnknown, nil
			}
			return FileStateUnknown, fmt.Errorf("failed to stat file: %w", err)
		}
		return FileStateOrphan, nil
	}

	fullPath := filepath.Join(st.manifestManager.outputRoot, path)
	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileStateDeleted, nil
		}
		return FileStateUnknown, fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.Size() != entry.Size || !stat.ModTime().Equal(entry.ModTime) {
		return FileStateModified, nil
	}

	hash, err := st.manifestManager.calculateFileHash(fullPath)
	if err != nil {
		return FileStateUnknown, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	if hash != entry.Hash {
		return FileStateModified, nil
	}

	return FileStateGenerated, nil
}

func (st *StateTracker) GetTrackedFiles() ([]TrackedFile, error) {
	if st.mode == TrackingModeDisabled {
		return nil, nil
	}

	manifest, err := st.manifestManager.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	var files []TrackedFile
	for _, entry := range manifest.Entries {
		state, err := st.GetFileState(entry.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to get state for file %s: %w", entry.Path, err)
		}

		files = append(files, TrackedFile{
			Path:         entry.Path,
			State:        state,
			LastSeen:     entry.ModTime,
			TemplatePath: entry.TemplatePath,
			Hash:         entry.Hash,
			Size:         entry.Size,
			Metadata:     entry.Metadata,
		})
	}

	return files, nil
}

func (st *StateTracker) GetOrphanedFiles() ([]string, error) {
	if st.mode == TrackingModeDisabled {
		return nil, nil
	}

	var orphans []string

	err := filepath.Walk(st.manifestManager.outputRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(st.manifestManager.outputRoot, path)
		if err != nil {
			return err
		}

		if filepath.Base(relPath) == ".gogenkit.manifest.json" {
			return nil
		}

		state, err := st.GetFileState(relPath)
		if err != nil {
			return err
		}

		if state == FileStateOrphan {
			orphans = append(orphans, relPath)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk output directory: %w", err)
	}

	return orphans, nil
}

func (st *StateTracker) GetModifiedFiles() ([]string, error) {
	if st.mode == TrackingModeDisabled {
		return nil, nil
	}

	files, err := st.GetTrackedFiles()
	if err != nil {
		return nil, err
	}

	var modified []string
	for _, file := range files {
		if file.State == FileStateModified {
			modified = append(modified, file.Path)
		}
	}

	return modified, nil
}

func (st *StateTracker) GetDeletedFiles() ([]string, error) {
	if st.mode == TrackingModeDisabled {
		return nil, nil
	}

	files, err := st.GetTrackedFiles()
	if err != nil {
		return nil, err
	}

	var deleted []string
	for _, file := range files {
		if file.State == FileStateDeleted {
			deleted = append(deleted, file.Path)
		}
	}

	return deleted, nil
}

func (st *StateTracker) IsFileTracked(path string) (bool, error) {
	if st.mode == TrackingModeDisabled {
		return false, nil
	}

	manifest, err := st.manifestManager.LoadManifest()
	if err != nil {
		return false, fmt.Errorf("failed to load manifest: %w", err)
	}

	_, exists := st.manifestManager.GetEntry(manifest, path)
	return exists, nil
}

func (st *StateTracker) RefreshManifest() error {
	if st.mode == TrackingModeDisabled {
		return nil
	}

	manifest, err := st.manifestManager.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	for path, entry := range manifest.Entries {
		fullPath := filepath.Join(st.manifestManager.outputRoot, path)
		stat, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to stat file %s: %w", fullPath, err)
		}

		if stat.Size() != entry.Size || !stat.ModTime().Equal(entry.ModTime) {
			hash, err := st.manifestManager.calculateFileHash(fullPath)
			if err != nil {
				return fmt.Errorf("failed to calculate hash for %s: %w", fullPath, err)
			}

			entry.Hash = hash
			entry.Size = stat.Size()
			entry.ModTime = stat.ModTime()
			manifest.Entries[path] = entry
		}
	}

	manifest.Generated = time.Now()
	return st.manifestManager.SaveManifest(manifest)
}

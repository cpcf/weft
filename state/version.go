package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	CurrentManifestVersion = "1.0"
	ManifestVersionKey     = "version"
)

type VersionInfo struct {
	Version     string            `json:"version"`
	CreatedAt   time.Time         `json:"created_at"`
	Generator   string            `json:"generator"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type MigrationResult struct {
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	BackupPath  string `json:"backup_path,omitempty"`
}

type VersionManager struct {
	outputRoot string
	backupDir  string
}

func NewVersionManager(outputRoot, backupDir string) *VersionManager {
	if backupDir == "" {
		backupDir = filepath.Join(outputRoot, ".gogenkit", "backups")
	}

	return &VersionManager{
		outputRoot: outputRoot,
		backupDir:  backupDir,
	}
}

func (vm *VersionManager) GetManifestVersion(manifest *Manifest) string {
	if manifest.Version == "" {
		return "0.0"
	}
	return manifest.Version
}

func (vm *VersionManager) IsVersionSupported(version string) bool {
	switch version {
	case "0.0", "1.0":
		return true
	default:
		return false
	}
}

func (vm *VersionManager) RequiresMigration(manifest *Manifest) bool {
	currentVersion := vm.GetManifestVersion(manifest)
	return currentVersion != CurrentManifestVersion
}

func (vm *VersionManager) MigrateManifest(manifest *Manifest) (*MigrationResult, error) {
	currentVersion := vm.GetManifestVersion(manifest)
	result := &MigrationResult{
		FromVersion: currentVersion,
		ToVersion:   CurrentManifestVersion,
	}

	if !vm.IsVersionSupported(currentVersion) {
		result.Error = fmt.Sprintf("unsupported manifest version: %s", currentVersion)
		return result, fmt.Errorf("%s", result.Error)
	}

	if currentVersion == CurrentManifestVersion {
		result.Success = true
		return result, nil
	}

	backupPath, err := vm.backupManifest()
	if err != nil {
		result.Error = fmt.Sprintf("failed to backup manifest: %v", err)
		return result, err
	}
	result.BackupPath = backupPath

	if err := vm.performMigration(manifest, currentVersion); err != nil {
		result.Error = fmt.Sprintf("migration failed: %v", err)
		return result, err
	}

	manifest.Version = CurrentManifestVersion
	result.Success = true
	return result, nil
}

func (vm *VersionManager) performMigration(manifest *Manifest, fromVersion string) error {
	switch fromVersion {
	case "0.0":
		return vm.migrateFrom0_0(manifest)
	default:
		return fmt.Errorf("no migration path from version %s", fromVersion)
	}
}

func (vm *VersionManager) migrateFrom0_0(manifest *Manifest) error {
	if manifest.Entries == nil {
		manifest.Entries = make(map[string]ManifestEntry)
	}
	if manifest.Metadata == nil {
		manifest.Metadata = make(map[string]string)
	}

	if manifest.Generator == "" {
		manifest.Generator = "gogenkit"
	}

	for path, entry := range manifest.Entries {
		if entry.GeneratedBy == "" {
			entry.GeneratedBy = "gogenkit"
			manifest.Entries[path] = entry
		}
	}

	return nil
}

func (vm *VersionManager) backupManifest() (string, error) {
	manifestPath := filepath.Join(vm.outputRoot, ".gogenkit.manifest.json")

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return "", nil
	}

	if err := os.MkdirAll(vm.backupDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(vm.backupDir, fmt.Sprintf("manifest-%s.json", timestamp))

	input, err := os.Open(manifestPath)
	if err != nil {
		return "", fmt.Errorf("failed to open manifest: %w", err)
	}
	defer input.Close()

	output, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}
	defer output.Close()

	buf := make([]byte, 4096)
	for {
		n, err := input.Read(buf)
		if err != nil && err.Error() != "EOF" {
			return "", fmt.Errorf("failed to read manifest: %w", err)
		}
		if n == 0 {
			break
		}

		if _, err := output.Write(buf[:n]); err != nil {
			return "", fmt.Errorf("failed to write backup: %w", err)
		}
	}

	return backupPath, nil
}

func (vm *VersionManager) GetVersionInfo() (*VersionInfo, error) {
	manifestPath := filepath.Join(vm.outputRoot, ".gogenkit.manifest.json")

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return &VersionInfo{
			Version:   "0.0",
			CreatedAt: time.Time{},
			Generator: "unknown",
		}, nil
	}

	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer file.Close()

	var manifest map[string]any
	if err := json.NewDecoder(file).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	version := "0.0"
	if v, exists := manifest["version"]; exists {
		if vStr, ok := v.(string); ok {
			version = vStr
		}
	}

	generator := "unknown"
	if g, exists := manifest["generator"]; exists {
		if gStr, ok := g.(string); ok {
			generator = gStr
		}
	}

	var createdAt time.Time
	if g, exists := manifest["generated"]; exists {
		if gStr, ok := g.(string); ok {
			if parsed, err := time.Parse(time.RFC3339, gStr); err == nil {
				createdAt = parsed
			}
		}
	}

	return &VersionInfo{
		Version:   version,
		CreatedAt: createdAt,
		Generator: generator,
	}, nil
}

func (vm *VersionManager) ListBackups() ([]string, error) {
	if _, err := os.Stat(vm.backupDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(vm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "manifest-") && strings.HasSuffix(entry.Name(), ".json") {
			backups = append(backups, entry.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(backups)))
	return backups, nil
}

func (vm *VersionManager) RestoreBackup(backupName string) error {
	backupPath := filepath.Join(vm.backupDir, backupName)
	manifestPath := filepath.Join(vm.outputRoot, ".gogenkit.manifest.json")

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupName)
	}

	currentBackupPath := manifestPath + ".pre-restore"
	if _, err := os.Stat(manifestPath); err == nil {
		if err := os.Rename(manifestPath, currentBackupPath); err != nil {
			return fmt.Errorf("failed to backup current manifest: %w", err)
		}
	}

	input, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup: %w", err)
	}
	defer input.Close()

	output, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to create manifest: %w", err)
	}
	defer output.Close()

	buf := make([]byte, 4096)
	for {
		n, err := input.Read(buf)
		if err != nil && err.Error() != "EOF" {
			return fmt.Errorf("failed to read backup: %w", err)
		}
		if n == 0 {
			break
		}

		if _, err := output.Write(buf[:n]); err != nil {
			return fmt.Errorf("failed to write manifest: %w", err)
		}
	}

	return nil
}

func (vm *VersionManager) CleanupOldBackups(keepCount int) error {
	backups, err := vm.ListBackups()
	if err != nil {
		return err
	}

	if len(backups) <= keepCount {
		return nil
	}

	toDelete := backups[keepCount:]
	for _, backup := range toDelete {
		backupPath := filepath.Join(vm.backupDir, backup)
		if err := os.Remove(backupPath); err != nil {
			return fmt.Errorf("failed to delete backup %s: %w", backup, err)
		}
	}

	return nil
}

func (vm *VersionManager) ValidateManifest(manifest *Manifest) error {
	if manifest.Version == "" {
		return fmt.Errorf("manifest version is missing")
	}

	if !vm.IsVersionSupported(manifest.Version) {
		return fmt.Errorf("unsupported manifest version: %s", manifest.Version)
	}

	if manifest.Generator == "" {
		return fmt.Errorf("manifest generator is missing")
	}

	if manifest.Entries == nil {
		return fmt.Errorf("manifest entries are missing")
	}

	for path, entry := range manifest.Entries {
		if path == "" {
			return fmt.Errorf("manifest entry has empty path")
		}

		if entry.Hash == "" {
			return fmt.Errorf("manifest entry %s has empty hash", path)
		}

		if entry.GeneratedBy == "" {
			return fmt.Errorf("manifest entry %s has empty generated_by", path)
		}
	}

	return nil
}

func CompareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := max(len(parts2), len(parts1))

	for i := range maxLen {
		var p1, p2 int
		var err error

		if i < len(parts1) {
			p1, err = strconv.Atoi(parts1[i])
			if err != nil {
				p1 = 0
			}
		}

		if i < len(parts2) {
			p2, err = strconv.Atoi(parts2[i])
			if err != nil {
				p2 = 0
			}
		}

		if p1 < p2 {
			return -1
		} else if p1 > p2 {
			return 1
		}
	}

	return 0
}

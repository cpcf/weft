package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CleanupMode int

const (
	CleanupModeAuto CleanupMode = iota
	CleanupModeInteractive
	CleanupModeReport
	CleanupModeDisabled
)

func (cm CleanupMode) String() string {
	switch cm {
	case CleanupModeAuto:
		return "auto"
	case CleanupModeInteractive:
		return "interactive"
	case CleanupModeReport:
		return "report"
	case CleanupModeDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

type CleanupAction int

const (
	CleanupActionDelete CleanupAction = iota
	CleanupActionSkip
	CleanupActionBackup
)

func (ca CleanupAction) String() string {
	switch ca {
	case CleanupActionDelete:
		return "delete"
	case CleanupActionSkip:
		return "skip"
	case CleanupActionBackup:
		return "backup"
	default:
		return "unknown"
	}
}

type OrphanFile struct {
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	IsDirectory bool      `json:"is_directory"`
	Reason      string    `json:"reason"`
}

type CleanupResult struct {
	Action     CleanupAction `json:"action"`
	File       OrphanFile    `json:"file"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	BackupPath string        `json:"backup_path,omitempty"`
}

type CleanupSummary struct {
	Mode           CleanupMode     `json:"mode"`
	OrphansFound   int             `json:"orphans_found"`
	FilesDeleted   int             `json:"files_deleted"`
	FilesSkipped   int             `json:"files_skipped"`
	FilesBackedUp  int             `json:"files_backed_up"`
	Errors         int             `json:"errors"`
	Results        []CleanupResult `json:"results"`
	TotalSizeFreed int64           `json:"total_size_freed"`
	ExecutionTime  time.Duration   `json:"execution_time"`
}

type CleanupManager struct {
	stateTracker *StateTracker
	mode         CleanupMode
	backupDir    string
	patterns     []string
}

type CleanupOption func(*CleanupManager)

func WithCleanupMode(mode CleanupMode) CleanupOption {
	return func(cm *CleanupManager) {
		cm.mode = mode
	}
}

func WithBackupDirectory(dir string) CleanupOption {
	return func(cm *CleanupManager) {
		cm.backupDir = dir
	}
}

func WithIgnorePatterns(patterns []string) CleanupOption {
	return func(cm *CleanupManager) {
		cm.patterns = patterns
	}
}

func NewCleanupManager(stateTracker *StateTracker, opts ...CleanupOption) *CleanupManager {
	cm := &CleanupManager{
		stateTracker: stateTracker,
		mode:         CleanupModeReport,
		patterns:     []string{},
	}

	for _, opt := range opts {
		opt(cm)
	}

	return cm
}

func (cm *CleanupManager) FindOrphans() ([]OrphanFile, error) {
	orphanPaths, err := cm.stateTracker.GetOrphanedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned files: %w", err)
	}

	var orphans []OrphanFile
	for _, path := range orphanPaths {
		if cm.shouldIgnore(path) {
			continue
		}

		orphan, err := cm.createOrphanFile(path)
		if err != nil {
			continue
		}

		orphans = append(orphans, orphan)
	}

	sort.Slice(orphans, func(i, j int) bool {
		return orphans[i].Path < orphans[j].Path
	})

	return orphans, nil
}

func (cm *CleanupManager) CleanupOrphans() (*CleanupSummary, error) {
	startTime := time.Now()

	orphans, err := cm.FindOrphans()
	if err != nil {
		return nil, fmt.Errorf("failed to find orphans: %w", err)
	}

	summary := &CleanupSummary{
		Mode:         cm.mode,
		OrphansFound: len(orphans),
		Results:      make([]CleanupResult, 0, len(orphans)),
	}

	if cm.mode == CleanupModeDisabled {
		return summary, nil
	}

	for _, orphan := range orphans {
		result := cm.processOrphan(orphan)
		summary.Results = append(summary.Results, result)

		switch result.Action {
		case CleanupActionDelete:
			if result.Success {
				summary.FilesDeleted++
				summary.TotalSizeFreed += orphan.Size
			} else {
				summary.Errors++
			}
		case CleanupActionSkip:
			summary.FilesSkipped++
		case CleanupActionBackup:
			if result.Success {
				summary.FilesBackedUp++
				summary.TotalSizeFreed += orphan.Size
			} else {
				summary.Errors++
			}
		}
	}

	summary.ExecutionTime = time.Since(startTime)
	return summary, nil
}

func (cm *CleanupManager) processOrphan(orphan OrphanFile) CleanupResult {
	result := CleanupResult{
		File: orphan,
	}

	switch cm.mode {
	case CleanupModeAuto:
		result.Action = CleanupActionDelete
	case CleanupModeInteractive:
		result.Action = cm.promptUserAction(orphan)
	case CleanupModeReport:
		result.Action = CleanupActionSkip
		result.Success = true
		return result
	default:
		result.Action = CleanupActionSkip
		result.Success = true
		return result
	}

	result.Success, result.Error, result.BackupPath = cm.executeAction(orphan, result.Action)
	return result
}

func (cm *CleanupManager) executeAction(orphan OrphanFile, action CleanupAction) (bool, string, string) {
	fullPath := filepath.Join(cm.stateTracker.manifestManager.outputRoot, orphan.Path)

	switch action {
	case CleanupActionDelete:
		if err := cm.deleteFile(fullPath); err != nil {
			return false, err.Error(), ""
		}
		return true, "", ""

	case CleanupActionBackup:
		backupPath, err := cm.backupFile(fullPath)
		if err != nil {
			return false, err.Error(), ""
		}
		return true, "", backupPath

	case CleanupActionSkip:
		return true, "", ""

	default:
		return false, "unknown action", ""
	}
}

func (cm *CleanupManager) deleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", path, err)
	}
	return nil
}

func (cm *CleanupManager) backupFile(path string) (string, error) {
	if cm.backupDir == "" {
		return "", fmt.Errorf("backup directory not configured")
	}

	if err := os.MkdirAll(cm.backupDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	filename := filepath.Base(path)
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(cm.backupDir, fmt.Sprintf("%s.%s.bak", filename, timestamp))

	input, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer input.Close()

	output, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer output.Close()

	buf := make([]byte, 4096)
	for {
		n, err := input.Read(buf)
		if err != nil && err.Error() != "EOF" {
			return "", fmt.Errorf("failed to read source file: %w", err)
		}
		if n == 0 {
			break
		}

		if _, err := output.Write(buf[:n]); err != nil {
			return "", fmt.Errorf("failed to write backup file: %w", err)
		}
	}

	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("failed to remove original file: %w", err)
	}

	return backupPath, nil
}

func (cm *CleanupManager) promptUserAction(orphan OrphanFile) CleanupAction {
	fmt.Printf("Orphan file found: %s\n", orphan.Path)
	fmt.Printf("  Size: %d bytes\n", orphan.Size)
	fmt.Printf("  Modified: %s\n", orphan.ModTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Reason: %s\n", orphan.Reason)
	fmt.Print("Action? (d)elete, (s)kip, (b)ackup: ")

	var input string
	fmt.Scanln(&input)

	switch strings.ToLower(strings.TrimSpace(input)) {
	case "d", "delete":
		return CleanupActionDelete
	case "b", "backup":
		return CleanupActionBackup
	default:
		return CleanupActionSkip
	}
}

func (cm *CleanupManager) shouldIgnore(path string) bool {
	for _, pattern := range cm.patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}

		matched, err = filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func (cm *CleanupManager) createOrphanFile(path string) (OrphanFile, error) {
	fullPath := filepath.Join(cm.stateTracker.manifestManager.outputRoot, path)
	stat, err := os.Stat(fullPath)
	if err != nil {
		return OrphanFile{}, err
	}

	reason := "File not tracked in manifest"
	if stat.IsDir() {
		reason = "Directory not tracked in manifest"
	}

	return OrphanFile{
		Path:        path,
		Size:        stat.Size(),
		ModTime:     stat.ModTime(),
		IsDirectory: stat.IsDir(),
		Reason:      reason,
	}, nil
}

func (cm *CleanupManager) PrintSummary(summary *CleanupSummary) {
	fmt.Printf("\nCleanup Summary (%s mode):\n", summary.Mode)
	fmt.Printf("  Orphans found: %d\n", summary.OrphansFound)
	fmt.Printf("  Files deleted: %d\n", summary.FilesDeleted)
	fmt.Printf("  Files skipped: %d\n", summary.FilesSkipped)
	fmt.Printf("  Files backed up: %d\n", summary.FilesBackedUp)
	fmt.Printf("  Errors: %d\n", summary.Errors)
	fmt.Printf("  Space freed: %d bytes\n", summary.TotalSizeFreed)
	fmt.Printf("  Execution time: %v\n", summary.ExecutionTime)

	if summary.Errors > 0 {
		fmt.Printf("\nErrors:\n")
		for _, result := range summary.Results {
			if result.Error != "" {
				fmt.Printf("  %s: %s\n", result.File.Path, result.Error)
			}
		}
	}
}

func (cm *CleanupManager) GetOrphanReport() (string, error) {
	orphans, err := cm.FindOrphans()
	if err != nil {
		return "", err
	}

	if len(orphans) == 0 {
		return "No orphan files found.", nil
	}

	var report strings.Builder
	report.WriteString(fmt.Sprintf("Found %d orphan files:\n\n", len(orphans)))

	var totalSize int64
	for _, orphan := range orphans {
		report.WriteString(fmt.Sprintf("  %s\n", orphan.Path))
		report.WriteString(fmt.Sprintf("    Size: %d bytes\n", orphan.Size))
		report.WriteString(fmt.Sprintf("    Modified: %s\n", orphan.ModTime.Format("2006-01-02 15:04:05")))
		report.WriteString(fmt.Sprintf("    Reason: %s\n\n", orphan.Reason))
		totalSize += orphan.Size
	}

	report.WriteString(fmt.Sprintf("Total size: %d bytes\n", totalSize))
	return report.String(), nil
}

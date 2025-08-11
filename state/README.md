# State Package

Package state provides state management, file tracking, manifest handling, cleanup operations, and versioning capabilities for weft. It offers configurable tracking modes, orphan file detection, automatic cleanup, manifest versioning, and migration support for maintaining consistent output state across template generations.

## Architecture

The state package is organized into four main components:

- **File Tracking** (`tracking.go`): State tracking for generated files with change detection
- **Manifest Management** (`manifest.go`): File manifest creation, loading, and validation
- **Cleanup Operations** (`cleanup.go`): Orphan file detection and cleanup with multiple modes
- **Version Management** (`version.go`): Manifest versioning, migration, and backup handling

## Basic Usage

### Create State Tracker and Enable File Tracking

```go
tracker := state.NewStateTracker("/path/to/output", state.TrackingModeEnabled)
```

### Track Generated Files

```go
metadata := map[string]string{"author": "weft", "template_version": "1.0"}
err := tracker.TrackFile("output/file.go", "templates/file.go.tmpl", metadata)
```

## Manifest Management

The package provides comprehensive manifest operations for tracking generated files:

```go
// Create manifest manager
manifestManager := state.NewManifestManager("/path/to/output")

// Load existing manifest
manifest, err := manifestManager.LoadManifest()

// Add entries to manifest
err = manifestManager.AddEntry(manifest, "output/file.go", "templates/file.go.tmpl", metadata)

// Save manifest
err = manifestManager.SaveManifest(manifest)
```

### Register File Tracking

```go
tracker := state.NewStateTracker("/output", state.TrackingModeEnabled)
err := tracker.TrackFile("generated/config.go", "templates/config.go.tmpl", nil)
```

## Configuration

The package supports various tracking modes and cleanup options:

```go
// Tracking modes
tracker := state.NewStateTracker("/output", state.TrackingModeStrict)

// Cleanup configuration
cleanupManager := state.NewCleanupManager(tracker,
    state.WithCleanupMode(state.CleanupModeInteractive),
    state.WithBackupDirectory("/backups"),
    state.WithIgnorePatterns([]string{"*.tmp", "*.log"}),
)
```

## File State Tracking

Monitor and analyze file states across generations:

```go
// Check individual file state
fileState, err := tracker.GetFileState("output/config.go")
switch fileState {
case state.FileStateGenerated:
    fmt.Println("File is up-to-date")
case state.FileStateModified:
    fmt.Println("File has been modified since generation")
case state.FileStateOrphan:
    fmt.Println("File exists but is not tracked")
}
```

### File State Detection

- **Generated**: File matches manifest entry (hash, size, modtime)
- **Modified**: File differs from manifest entry
- **Deleted**: File in manifest but missing from filesystem
- **Orphan**: File exists but not in manifest
- **Unknown**: File state cannot be determined

## Tracking Modes

The package supports multiple tracking modes:

| Mode | Description |
|------|-------------|
| `TrackingModeDisabled` | No file tracking performed |
| `TrackingModeEnabled` | Standard file tracking and manifest management |
| `TrackingModeStrict` | Enhanced tracking with strict validation |

### Setting Tracking Modes

```go
tracker := state.NewStateTracker("/output", state.TrackingModeEnabled)

// Check if file is tracked
isTracked, err := tracker.IsFileTracked("output/config.go")

// Get all tracked files
trackedFiles, err := tracker.GetTrackedFiles()
```

## Cleanup Operations

### Available Cleanup Modes

| Mode | Description | Example |
|------|-------------|---------|
| `CleanupModeAuto` | Automatically delete orphan files | `state.CleanupModeAuto` |
| `CleanupModeInteractive` | Prompt user for each orphan | `state.CleanupModeInteractive` |
| `CleanupModeReport` | Generate report without cleanup | `state.CleanupModeReport` |
| `CleanupModeDisabled` | Disable cleanup operations | `state.CleanupModeDisabled` |

### Usage Examples

```go
// Find orphan files
orphans, err := cleanupManager.FindOrphans()
for _, orphan := range orphans {
    fmt.Printf("Orphan: %s (%d bytes)\n", orphan.Path, orphan.Size)
}

// Perform cleanup
summary, err := cleanupManager.CleanupOrphans()
cleanupManager.PrintSummary(summary)

// Get report without cleanup
report, err := cleanupManager.GetOrphanReport()
fmt.Println(report)
```

## Manifest Structure

### Manifest Entry Information

```go
// Access manifest entries
entries := manifestManager.ListEntries(manifest)
for _, entry := range entries {
    fmt.Printf("File: %s\n", entry.Path)
    fmt.Printf("  Hash: %s\n", entry.Hash)
    fmt.Printf("  Size: %d bytes\n", entry.Size)
    fmt.Printf("  Template: %s\n", entry.TemplatePath)
    fmt.Printf("  Generated: %s\n", entry.ModTime.Format("2006-01-02 15:04:05"))
}
```

### Change Detection

```go
// Check if file has changed
hasChanged, err := manifestManager.HasChanged(manifest, "output/config.go")
if hasChanged {
    fmt.Println("File needs regeneration")
}

// Get specific files by state
modifiedFiles, err := tracker.GetModifiedFiles()
deletedFiles, err := tracker.GetDeletedFiles()
orphanFiles, err := tracker.GetOrphanedFiles()
```

## Version Management

### Version Operations

```go
versionManager := state.NewVersionManager("/output", "/backups")

// Get version information
versionInfo, err := versionManager.GetVersionInfo()
fmt.Printf("Version: %s\n", versionInfo.Version)
fmt.Printf("Created: %s\n", versionInfo.CreatedAt.Format("2006-01-02 15:04:05"))
fmt.Printf("Generator: %s\n", versionInfo.Generator)
```

### Migration Support

```go
// Check if migration is needed
if versionManager.RequiresMigration(manifest) {
    result, err := versionManager.MigrateManifest(manifest)
    if result.Success {
        fmt.Printf("Migrated from %s to %s\n", result.FromVersion, result.ToVersion)
        if result.BackupPath != "" {
            fmt.Printf("Backup created: %s\n", result.BackupPath)
        }
    }
}
```

### Backup Management

```go
// List available backups
backups, err := versionManager.ListBackups()
for _, backup := range backups {
    fmt.Printf("Backup: %s\n", backup)
}

// Restore from backup
err = versionManager.RestoreBackup("manifest-20240101-120000.json")

// Clean up old backups (keep 5 most recent)
err = versionManager.CleanupOldBackups(5)
```

## Performance Considerations

### Optimization Tips

1. **Selective Tracking**: Use `TrackingModeDisabled` when state management isn't needed
2. **Efficient Cleanup**: Use `CleanupModeReport` for analysis without filesystem operations
3. **Batch Operations**: Process multiple files in single manifest operations
4. **Hash Caching**: File hashes are calculated only when needed for change detection

### Performance Settings

```go
// Lightweight tracking for performance-critical scenarios
tracker := state.NewStateTracker("/output", state.TrackingModeEnabled)

// Efficient cleanup with ignore patterns
cleanupManager := state.NewCleanupManager(tracker,
    state.WithCleanupMode(state.CleanupModeReport),
    state.WithIgnorePatterns([]string{"*.tmp", "*.log", "node_modules/*"}),
)
```

### Memory Management

```go
// Refresh manifest to update file states
err := tracker.RefreshManifest()

// Selective file tracking
isTracked, err := tracker.IsFileTracked("specific/file.go")
if !isTracked {
    err = tracker.TrackFile("specific/file.go", "template.go.tmpl", nil)
}
```

## Security Notes

### File Path Protection

- All file paths are validated and sanitized before operations
- Relative path traversal attempts are prevented
- Backup operations create secure temporary files
- Manifest files are written atomically to prevent corruption

### Security Best Practices

1. **Validate Output Paths** before tracking files
2. **Secure Backup Locations** outside public directories  
3. **Regular Cleanup** of sensitive orphan files
4. **Manifest Integrity** validation before migrations

### Path Validation

```go
// Safe file operations with automatic path validation
// Paths are automatically resolved and validated:
// - No directory traversal (../) attempts
// - Absolute paths within output root only
// - Proper file permissions on created directories
```

## Thread Safety

All public functions and types in this package are thread-safe and can be used concurrently from multiple goroutines.

### Concurrent Usage Examples

```go
// Safe concurrent state tracking
tracker := state.NewStateTracker("/output", state.TrackingModeEnabled)

go func() {
    tracker.TrackFile("file1.go", "template1.go.tmpl", nil)
}()

go func() {
    tracker.TrackFile("file2.go", "template2.go.tmpl", nil)
}()

// Concurrent cleanup operations
go func() {
    summary, _ := cleanupManager.CleanupOrphans()
    cleanupManager.PrintSummary(summary)
}()
```

## Advanced Features

### Custom Cleanup Logic

```go
cleanupManager := state.NewCleanupManager(tracker,
    state.WithCleanupMode(state.CleanupModeInteractive),
    state.WithBackupDirectory("/secure/backups"),
    state.WithIgnorePatterns([]string{
        "*.log",           // Ignore log files
        "tmp/*",           // Ignore temp directory
        "*.backup",        // Ignore backup files
        "node_modules/*",  // Ignore dependencies
    }),
)

// Generate detailed orphan report
report, err := cleanupManager.GetOrphanReport()
fmt.Println(report)
```

### Manifest Validation

```go
versionManager := state.NewVersionManager("/output", "/backups")

// Validate manifest structure
if err := versionManager.ValidateManifest(manifest); err != nil {
    fmt.Printf("Manifest validation failed: %v\n", err)
}

// Version comparison
comparison := state.CompareVersions("1.0", "1.1")
switch comparison {
case -1:
    fmt.Println("Version 1.0 is older than 1.1")
case 0:
    fmt.Println("Versions are equal")
case 1:
    fmt.Println("Version 1.0 is newer than 1.1")
}
```

### Error Recovery

```go
// Implement error recovery patterns
func processWithStateTracking() error {
    tracker := state.NewStateTracker("/output", state.TrackingModeEnabled)
    
    // Track file generation
    if err := tracker.TrackFile("output.go", "template.go.tmpl", nil); err != nil {
        // Handle tracking errors gracefully
        fmt.Printf("Warning: Could not track file: %v\n", err)
        // Continue processing...
    }
    
    return nil
}
```

## Integration Examples

### With Template Generators

```go
func generateTemplate(templatePath, outputPath string, data interface{}) error {
    tracker := state.NewStateTracker("/output", state.TrackingModeEnabled)
    
    // Generate template content
    content, err := executeTemplate(templatePath, data)
    if err != nil {
        return err
    }
    
    // Write output file
    if err := writeFile(outputPath, content); err != nil {
        return err
    }
    
    // Track generated file
    metadata := map[string]string{
        "generated_at": time.Now().Format(time.RFC3339),
        "template_version": "1.0",
    }
    
    return tracker.TrackFile(outputPath, templatePath, metadata)
}
```

### With Build Systems

```go
func buildPhaseCleanup() error {
    tracker := state.NewStateTracker("./dist", state.TrackingModeEnabled)
    
    cleanupManager := state.NewCleanupManager(tracker,
        state.WithCleanupMode(state.CleanupModeAuto),
        state.WithIgnorePatterns([]string{"*.log"}),
    )
    
    // Clean up orphan files before build
    summary, err := cleanupManager.CleanupOrphans()
    if err != nil {
        return fmt.Errorf("cleanup failed: %w", err)
    }
    
    fmt.Printf("Cleanup: %d files deleted, %d bytes freed\n", 
        summary.FilesDeleted, summary.TotalSizeFreed)
    
    return nil
}
```

### With CI/CD Pipelines

```go
func ciPostProcess() error {
    versionManager := state.NewVersionManager("./output", "./backups")
    
    // Get version information for reporting
    versionInfo, err := versionManager.GetVersionInfo()
    if err != nil {
        return err
    }
    
    fmt.Printf("Generated files version: %s\n", versionInfo.Version)
    fmt.Printf("Generation time: %s\n", versionInfo.CreatedAt.Format(time.RFC3339))
    
    // Clean up old backups in CI environment
    return versionManager.CleanupOldBackups(3)
}
```
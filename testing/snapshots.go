package testing

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SnapshotManager struct {
	snapshotDir string
	updateMode  bool
	mu          sync.RWMutex
	results     map[string]SnapshotResult
}

type SnapshotResult struct {
	TestName     string    `json:"test_name"`
	Expected     string    `json:"expected"`
	Actual       string    `json:"actual"`
	Passed       bool      `json:"passed"`
	Hash         string    `json:"hash"`
	Timestamp    time.Time `json:"timestamp"`
	FilePath     string    `json:"file_path"`
	UpdatedCount int       `json:"updated_count"`
}

type SnapshotTestCase struct {
	Name     string         `json:"name"`
	Template string         `json:"template"`
	Data     map[string]any `json:"data"`
	Expected string         `json:"expected"`
}

func NewSnapshotManager(snapshotDir string, updateMode bool) *SnapshotManager {
	return &SnapshotManager{
		snapshotDir: snapshotDir,
		updateMode:  updateMode,
		results:     make(map[string]SnapshotResult),
	}
}

func (sm *SnapshotManager) SetUpdateMode(update bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.updateMode = update
}

func (sm *SnapshotManager) AssertSnapshot(testName, actual string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	snapshotPath := filepath.Join(sm.snapshotDir, testName+".snapshot")

	result := SnapshotResult{
		TestName:  testName,
		Actual:    actual,
		Timestamp: time.Now(),
		FilePath:  snapshotPath,
		Hash:      sm.calculateHash(actual),
	}

	if err := os.MkdirAll(sm.snapshotDir, 0o755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	expected, err := sm.loadSnapshot(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			if sm.updateMode {
				if err := sm.saveSnapshot(snapshotPath, actual); err != nil {
					return err
				}
				result.Expected = actual
				result.Passed = true
				result.UpdatedCount = 1
			} else {
				result.Passed = false
				return fmt.Errorf("snapshot does not exist: %s (run with update mode to create)", snapshotPath)
			}
		} else {
			return fmt.Errorf("failed to load snapshot: %w", err)
		}
	} else {
		result.Expected = expected
		result.Passed = (actual == expected)

		if !result.Passed {
			if sm.updateMode {
				if err := sm.saveSnapshot(snapshotPath, actual); err != nil {
					return err
				}
				result.Expected = actual
				result.Passed = true
				result.UpdatedCount = 1
			} else {
				return fmt.Errorf("snapshot mismatch for %s:\nExpected:\n%s\n\nActual:\n%s\n\nDiff:\n%s",
					testName, expected, actual, sm.generateDiff(expected, actual))
			}
		}
	}

	sm.results[testName] = result
	return nil
}

func (sm *SnapshotManager) loadSnapshot(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (sm *SnapshotManager) saveSnapshot(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func (sm *SnapshotManager) calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)[:16]
}

func (sm *SnapshotManager) generateDiff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var diff strings.Builder
	maxLines := max(len(actualLines), len(expectedLines))

	for i := range maxLines {
		var expectedLine, actualLine string

		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}

		if expectedLine != actualLine {
			diff.WriteString(fmt.Sprintf("Line %d:\n", i+1))
			if i < len(expectedLines) {
				diff.WriteString(fmt.Sprintf("  - %s\n", expectedLine))
			}
			if i < len(actualLines) {
				diff.WriteString(fmt.Sprintf("  + %s\n", actualLine))
			}
		}
	}

	return diff.String()
}

func (sm *SnapshotManager) GetResults() map[string]SnapshotResult {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	results := make(map[string]SnapshotResult)
	maps.Copy(results, sm.results)
	return results
}

func (sm *SnapshotManager) GetSummary() SnapshotSummary {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	summary := SnapshotSummary{
		TotalTests:   len(sm.results),
		PassedTests:  0,
		FailedTests:  0,
		UpdatedTests: 0,
		UpdateMode:   sm.updateMode,
	}

	for _, result := range sm.results {
		if result.Passed {
			summary.PassedTests++
		} else {
			summary.FailedTests++
		}
		if result.UpdatedCount > 0 {
			summary.UpdatedTests++
		}
	}

	return summary
}

func (sm *SnapshotManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.results = make(map[string]SnapshotResult)
}

func (sm *SnapshotManager) CleanOrphanSnapshots(activeTests []string) error {
	activeSet := make(map[string]bool)
	for _, test := range activeTests {
		activeSet[test+".snapshot"] = true
	}

	entries, err := os.ReadDir(sm.snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	var orphans []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(entry.Name(), ".snapshot") {
			if !activeSet[entry.Name()] {
				orphans = append(orphans, entry.Name())
			}
		}
	}

	for _, orphan := range orphans {
		orphanPath := filepath.Join(sm.snapshotDir, orphan)
		if err := os.Remove(orphanPath); err != nil {
			return fmt.Errorf("failed to remove orphan snapshot %s: %w", orphan, err)
		}
	}

	return nil
}

type SnapshotSummary struct {
	TotalTests   int  `json:"total_tests"`
	PassedTests  int  `json:"passed_tests"`
	FailedTests  int  `json:"failed_tests"`
	UpdatedTests int  `json:"updated_tests"`
	UpdateMode   bool `json:"update_mode"`
}

func (ss SnapshotSummary) String() string {
	if ss.TotalTests == 0 {
		return "No snapshot tests run"
	}

	status := "PASS"
	if ss.FailedTests > 0 {
		status = "FAIL"
	}

	result := fmt.Sprintf("%s: %d/%d tests passed", status, ss.PassedTests, ss.TotalTests)
	if ss.UpdatedTests > 0 {
		result += fmt.Sprintf(" (%d updated)", ss.UpdatedTests)
	}

	return result
}

type SnapshotTestSuite struct {
	manager   *SnapshotManager
	testCases []SnapshotTestCase
}

func NewSnapshotTestSuite(snapshotDir string, updateMode bool) *SnapshotTestSuite {
	return &SnapshotTestSuite{
		manager:   NewSnapshotManager(snapshotDir, updateMode),
		testCases: make([]SnapshotTestCase, 0),
	}
}

func (sts *SnapshotTestSuite) AddTestCase(name, template string, data map[string]any) {
	testCase := SnapshotTestCase{
		Name:     name,
		Template: template,
		Data:     data,
	}
	sts.testCases = append(sts.testCases, testCase)
}

func (sts *SnapshotTestSuite) LoadTestCases(testFile string) error {
	data, err := os.ReadFile(testFile)
	if err != nil {
		return fmt.Errorf("failed to read test file: %w", err)
	}

	var testCases []SnapshotTestCase
	if err := json.Unmarshal(data, &testCases); err != nil {
		return fmt.Errorf("failed to parse test file: %w", err)
	}

	sts.testCases = append(sts.testCases, testCases...)
	return nil
}

func (sts *SnapshotTestSuite) SaveTestCases(testFile string) error {
	data, err := json.MarshalIndent(sts.testCases, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal test cases: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
		return fmt.Errorf("failed to create test directory: %w", err)
	}

	return os.WriteFile(testFile, data, 0o644)
}

func (sts *SnapshotTestSuite) GetManager() *SnapshotManager {
	return sts.manager
}

func (sts *SnapshotTestSuite) GetTestCases() []SnapshotTestCase {
	return sts.testCases
}

type SnapshotReporter struct {
	results []SnapshotResult
	mu      sync.RWMutex
}

func NewSnapshotReporter() *SnapshotReporter {
	return &SnapshotReporter{
		results: make([]SnapshotResult, 0),
	}
}

func (sr *SnapshotReporter) AddResult(result SnapshotResult) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.results = append(sr.results, result)
}

func (sr *SnapshotReporter) GenerateReport() string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	if len(sr.results) == 0 {
		return "No snapshot test results to report"
	}

	var report strings.Builder
	report.WriteString("Snapshot Test Report\n")
	report.WriteString("===================\n\n")

	passed := 0
	failed := 0
	updated := 0

	for _, result := range sr.results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
		if result.UpdatedCount > 0 {
			updated++
		}

		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}

		report.WriteString(fmt.Sprintf("%s %s", status, result.TestName))
		if result.UpdatedCount > 0 {
			report.WriteString(" (updated)")
		}
		report.WriteString("\n")

		if !result.Passed && result.Expected != "" && result.Actual != "" {
			report.WriteString("  Expected length: ")
			report.WriteString(fmt.Sprintf("%d\n", len(result.Expected)))
			report.WriteString("  Actual length: ")
			report.WriteString(fmt.Sprintf("%d\n", len(result.Actual)))
			report.WriteString("  Hash: ")
			report.WriteString(result.Hash)
			report.WriteString("\n")
		}
		report.WriteString("\n")
	}

	report.WriteString(fmt.Sprintf("Summary: %d passed, %d failed", passed, failed))
	if updated > 0 {
		report.WriteString(fmt.Sprintf(", %d updated", updated))
	}
	report.WriteString("\n")

	return report.String()
}

func (sr *SnapshotReporter) ExportJSON(filename string) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	data, err := json.MarshalIndent(sr.results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	return os.WriteFile(filename, data, 0o644)
}

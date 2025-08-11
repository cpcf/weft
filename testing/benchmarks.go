// Package testing provides utilities for testing and benchmarking.
package testing

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

type BenchmarkResult struct {
	Name          string        `json:"name"`
	Iterations    int           `json:"iterations"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgDuration   time.Duration `json:"avg_duration"`
	MinDuration   time.Duration `json:"min_duration"`
	MaxDuration   time.Duration `json:"max_duration"`
	MemoryUsed    int64         `json:"memory_used"`
	AllocsPerOp   int64         `json:"allocs_per_op"`
	BytesPerOp    int64         `json:"bytes_per_op"`
	Timestamp     time.Time     `json:"timestamp"`
	Success       bool          `json:"success"`
	Error         string        `json:"error,omitempty"`
}

type BenchmarkRunner struct {
	results     []BenchmarkResult
	warmupIters int
	minIters    int
	maxIters    int
	minTime     time.Duration
	mu          sync.RWMutex
}

type BenchmarkFunc func() error

func NewBenchmarkRunner() *BenchmarkRunner {
	return &BenchmarkRunner{
		results:     make([]BenchmarkResult, 0),
		warmupIters: 3,
		minIters:    10,
		maxIters:    10000,
		minTime:     1 * time.Second,
	}
}

func (br *BenchmarkRunner) SetWarmupIterations(n int) {
	br.warmupIters = n
}

func (br *BenchmarkRunner) SetMinIterations(n int) {
	br.minIters = n
}

func (br *BenchmarkRunner) SetMaxIterations(n int) {
	br.maxIters = n
}

func (br *BenchmarkRunner) SetMinTime(d time.Duration) {
	br.minTime = d
}

func (br *BenchmarkRunner) Benchmark(name string, fn BenchmarkFunc) BenchmarkResult {
	result := BenchmarkResult{
		Name:        name,
		Timestamp:   time.Now(),
		MinDuration: time.Hour,
		MaxDuration: 0,
	}

	for i := 0; i < br.warmupIters; i++ {
		fn()
	}

	runtime.GC()

	var memStatsBefore, memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	startTime := time.Now()
	iterations := 0
	var durations []time.Duration
	var totalDuration time.Duration

	for iterations < br.minIters || (time.Since(startTime) < br.minTime && iterations < br.maxIters) {
		iterStart := time.Now()

		if err := fn(); err != nil {
			result.Error = err.Error()
			result.Success = false
			br.mu.Lock()
			br.results = append(br.results, result)
			br.mu.Unlock()
			return result
		}

		iterDuration := time.Since(iterStart)
		durations = append(durations, iterDuration)
		totalDuration += iterDuration

		if iterDuration < result.MinDuration {
			result.MinDuration = iterDuration
		}
		if iterDuration > result.MaxDuration {
			result.MaxDuration = iterDuration
		}

		iterations++
	}

	runtime.ReadMemStats(&memStatsAfter)

	result.Iterations = iterations
	result.TotalDuration = totalDuration
	result.AvgDuration = totalDuration / time.Duration(iterations)
	result.MemoryUsed = int64(memStatsAfter.TotalAlloc - memStatsBefore.TotalAlloc)
	result.AllocsPerOp = int64(memStatsAfter.Mallocs-memStatsBefore.Mallocs) / int64(iterations)
	result.BytesPerOp = result.MemoryUsed / int64(iterations)
	result.Success = true

	br.mu.Lock()
	br.results = append(br.results, result)
	br.mu.Unlock()

	return result
}

func (br *BenchmarkRunner) GetResults() []BenchmarkResult {
	br.mu.RLock()
	defer br.mu.RUnlock()

	results := make([]BenchmarkResult, len(br.results))
	copy(results, br.results)
	return results
}

func (br *BenchmarkRunner) GetResult(name string) (BenchmarkResult, bool) {
	br.mu.RLock()
	defer br.mu.RUnlock()

	for _, result := range br.results {
		if result.Name == name {
			return result, true
		}
	}
	return BenchmarkResult{}, false
}

func (br *BenchmarkRunner) Clear() {
	br.mu.Lock()
	defer br.mu.Unlock()
	br.results = make([]BenchmarkResult, 0)
}

func (br *BenchmarkRunner) Compare(name1, name2 string) (BenchmarkComparison, error) {
	result1, ok1 := br.GetResult(name1)
	if !ok1 {
		return BenchmarkComparison{}, fmt.Errorf("benchmark result not found: %s", name1)
	}

	result2, ok2 := br.GetResult(name2)
	if !ok2 {
		return BenchmarkComparison{}, fmt.Errorf("benchmark result not found: %s", name2)
	}

	return BenchmarkComparison{
		Name1:               result1.Name,
		Name2:               result2.Name,
		SpeedRatio:          float64(result1.AvgDuration) / float64(result2.AvgDuration),
		MemoryRatio:         float64(result1.BytesPerOp) / float64(result2.BytesPerOp),
		AllocRatio:          float64(result1.AllocsPerOp) / float64(result2.AllocsPerOp),
		Result1Faster:       result1.AvgDuration < result2.AvgDuration,
		Result1MemEfficient: result1.BytesPerOp < result2.BytesPerOp,
	}, nil
}

type BenchmarkComparison struct {
	Name1               string  `json:"name1"`
	Name2               string  `json:"name2"`
	SpeedRatio          float64 `json:"speed_ratio"`
	MemoryRatio         float64 `json:"memory_ratio"`
	AllocRatio          float64 `json:"alloc_ratio"`
	Result1Faster       bool    `json:"result1_faster"`
	Result1MemEfficient bool    `json:"result1_mem_efficient"`
}

func (bc BenchmarkComparison) String() string {
	fasterName := bc.Name1
	slowerName := bc.Name2
	speedImprovement := bc.SpeedRatio

	if !bc.Result1Faster {
		fasterName = bc.Name2
		slowerName = bc.Name1
		speedImprovement = 1.0 / bc.SpeedRatio
	}

	memEfficientName := bc.Name1
	memHeavierName := bc.Name2
	memImprovement := bc.MemoryRatio

	if !bc.Result1MemEfficient {
		memEfficientName = bc.Name2
		memHeavierName = bc.Name1
		memImprovement = 1.0 / bc.MemoryRatio
	}

	return fmt.Sprintf("%s is %.2fx faster than %s\n%s uses %.2fx less memory than %s",
		fasterName, speedImprovement, slowerName,
		memEfficientName, memImprovement, memHeavierName)
}

type PerformanceProfiler struct {
	profiles map[string]PerformanceProfile
	mu       sync.RWMutex
}

type PerformanceProfile struct {
	Name            string                 `json:"name"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	Duration        time.Duration          `json:"duration"`
	Operations      []OperationProfile     `json:"operations"`
	MemoryProfile   MemoryProfile          `json:"memory_profile"`
	SystemResources SystemResourcesProfile `json:"system_resources"`
}

type OperationProfile struct {
	Name      string        `json:"name"`
	Count     int           `json:"count"`
	TotalTime time.Duration `json:"total_time"`
	AvgTime   time.Duration `json:"avg_time"`
	MinTime   time.Duration `json:"min_time"`
	MaxTime   time.Duration `json:"max_time"`
}

type MemoryProfile struct {
	StartAllocs    uint64 `json:"start_allocs"`
	EndAllocs      uint64 `json:"end_allocs"`
	AllocsDelta    uint64 `json:"allocs_delta"`
	StartHeapAlloc uint64 `json:"start_heap_alloc"`
	EndHeapAlloc   uint64 `json:"end_heap_alloc"`
	HeapAllocDelta int64  `json:"heap_alloc_delta"`
	StartSys       uint64 `json:"start_sys"`
	EndSys         uint64 `json:"end_sys"`
	SysDelta       int64  `json:"sys_delta"`
}

type SystemResourcesProfile struct {
	NumGoroutines int    `json:"num_goroutines"`
	NumCPU        int    `json:"num_cpu"`
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
}

func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		profiles: make(map[string]PerformanceProfile),
	}
}

func (pp *PerformanceProfiler) StartProfile(name string) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	profile := PerformanceProfile{
		Name:       name,
		StartTime:  time.Now(),
		Operations: make([]OperationProfile, 0),
		MemoryProfile: MemoryProfile{
			StartAllocs:    memStats.Mallocs,
			StartHeapAlloc: memStats.HeapAlloc,
			StartSys:       memStats.Sys,
		},
		SystemResources: SystemResourcesProfile{
			NumGoroutines: runtime.NumGoroutine(),
			NumCPU:        runtime.NumCPU(),
			GOOS:          runtime.GOOS,
			GOARCH:        runtime.GOARCH,
		},
	}

	pp.profiles[name] = profile
}

func (pp *PerformanceProfiler) EndProfile(name string) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	profile, exists := pp.profiles[name]
	if !exists {
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	profile.EndTime = time.Now()
	profile.Duration = profile.EndTime.Sub(profile.StartTime)
	profile.MemoryProfile.EndAllocs = memStats.Mallocs
	profile.MemoryProfile.EndHeapAlloc = memStats.HeapAlloc
	profile.MemoryProfile.EndSys = memStats.Sys

	profile.MemoryProfile.AllocsDelta = profile.MemoryProfile.EndAllocs - profile.MemoryProfile.StartAllocs
	profile.MemoryProfile.HeapAllocDelta = int64(profile.MemoryProfile.EndHeapAlloc) - int64(profile.MemoryProfile.StartHeapAlloc)
	profile.MemoryProfile.SysDelta = int64(profile.MemoryProfile.EndSys) - int64(profile.MemoryProfile.StartSys)

	pp.profiles[name] = profile
}

func (pp *PerformanceProfiler) GetProfile(name string) (PerformanceProfile, bool) {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	profile, exists := pp.profiles[name]
	return profile, exists
}

func (pp *PerformanceProfiler) GetAllProfiles() map[string]PerformanceProfile {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	profiles := make(map[string]PerformanceProfile)
	maps.Copy(profiles, pp.profiles)
	return profiles
}

func (pp *PerformanceProfiler) Clear() {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	pp.profiles = make(map[string]PerformanceProfile)
}

type BenchmarkReport struct {
	GeneratedAt time.Time         `json:"generated_at"`
	Results     []BenchmarkResult `json:"results"`
	Summary     BenchmarkSummary  `json:"summary"`
	SystemInfo  SystemInfo        `json:"system_info"`
}

type BenchmarkSummary struct {
	TotalBenchmarks int           `json:"total_benchmarks"`
	SuccessfulRuns  int           `json:"successful_runs"`
	FailedRuns      int           `json:"failed_runs"`
	TotalDuration   time.Duration `json:"total_duration"`
	AvgDuration     time.Duration `json:"avg_duration"`
	FastestBench    string        `json:"fastest_bench"`
	SlowestBench    string        `json:"slowest_bench"`
}

type SystemInfo struct {
	NumCPU       int              `json:"num_cpu"`
	NumGoroutine int              `json:"num_goroutine"`
	GOOS         string           `json:"goos"`
	GOARCH       string           `json:"goarch"`
	GoVersion    string           `json:"go_version"`
	MemStats     runtime.MemStats `json:"mem_stats"`
}

func (br *BenchmarkRunner) GenerateReport() BenchmarkReport {
	br.mu.RLock()
	defer br.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	report := BenchmarkReport{
		GeneratedAt: time.Now(),
		Results:     make([]BenchmarkResult, len(br.results)),
		SystemInfo: SystemInfo{
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			GOOS:         runtime.GOOS,
			GOARCH:       runtime.GOARCH,
			GoVersion:    runtime.Version(),
			MemStats:     memStats,
		},
	}

	copy(report.Results, br.results)

	summary := BenchmarkSummary{
		TotalBenchmarks: len(br.results),
	}

	if len(br.results) > 0 {
		var totalDuration time.Duration
		var fastestDuration time.Duration = time.Hour
		var slowestDuration time.Duration = 0
		var fastestName, slowestName string

		for _, result := range br.results {
			if result.Success {
				summary.SuccessfulRuns++
			} else {
				summary.FailedRuns++
			}

			totalDuration += result.TotalDuration

			if result.AvgDuration < fastestDuration && result.Success {
				fastestDuration = result.AvgDuration
				fastestName = result.Name
			}
			if result.AvgDuration > slowestDuration && result.Success {
				slowestDuration = result.AvgDuration
				slowestName = result.Name
			}
		}

		summary.TotalDuration = totalDuration
		summary.AvgDuration = totalDuration / time.Duration(len(br.results))
		summary.FastestBench = fastestName
		summary.SlowestBench = slowestName
	}

	report.Summary = summary
	return report
}

func (br *BenchmarkRunner) ExportReport(filename string) error {
	report := br.GenerateReport()

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	return os.WriteFile(filename, data, 0o644)
}

func (br *BenchmarkRunner) PrintResults() {
	results := br.GetResults()
	if len(results) == 0 {
		fmt.Println("No benchmark results to display")
		return
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].AvgDuration < results[j].AvgDuration
	})

	fmt.Println("Benchmark Results:")
	fmt.Println("==================")

	for _, result := range results {
		status := "PASS"
		if !result.Success {
			status = "FAIL"
		}

		fmt.Printf("%s %s\n", status, result.Name)
		if result.Success {
			fmt.Printf("  Iterations: %d\n", result.Iterations)
			fmt.Printf("  Avg time:   %v\n", result.AvgDuration)
			fmt.Printf("  Min time:   %v\n", result.MinDuration)
			fmt.Printf("  Max time:   %v\n", result.MaxDuration)
			fmt.Printf("  Memory:     %d bytes/op\n", result.BytesPerOp)
			fmt.Printf("  Allocs:     %d allocs/op\n", result.AllocsPerOp)
		} else {
			fmt.Printf("  Error:      %s\n", result.Error)
		}
		fmt.Println()
	}
}

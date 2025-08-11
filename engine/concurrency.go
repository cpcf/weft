package engine

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Result struct {
	Success  bool
	Error    error
	Duration time.Duration
}

type Plan struct {
	Operations []Operation
}

type Operation struct {
	Type         string
	TemplatePath string
	OutputPath   string
	Data         any
}

type WorkerPool struct {
	size       int
	queue      chan Task
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	started    int32
	completed  int64
	failed     int64
	processing int64
}

type Task interface {
	Execute(ctx context.Context) error
	ID() string
	Priority() int
}

type TaskResult struct {
	TaskID    string        `json:"task_id"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

type RenderTask struct {
	id           string
	priority     int
	templatePath string
	outputPath   string
	data         any
	renderer     Renderer
	result       chan TaskResult
	tmplFS       fs.FS
}

func (rt *RenderTask) Execute(ctx context.Context) error {
	startTime := time.Now()
	result := TaskResult{
		TaskID:    rt.id,
		StartTime: startTime,
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		select {
		case rt.result <- result:
		case <-ctx.Done():
		}
	}()

	// Create context for rendering
	renderCtx := Context{
		TmplFS:     rt.tmplFS,
		OutputRoot: filepath.Dir(rt.outputPath),
	}

	err := rt.renderer.renderFile(renderCtx, rt.templatePath, rt.data)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return err
	}

	result.Success = true
	return nil
}

func (rt *RenderTask) ID() string {
	return rt.id
}

func (rt *RenderTask) Priority() int {
	return rt.priority
}

func NewWorkerPool(size int) *WorkerPool {
	if size <= 0 {
		size = runtime.NumCPU()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		size:   size,
		queue:  make(chan Task, size*2),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (wp *WorkerPool) Start() {
	if atomic.CompareAndSwapInt32(&wp.started, 0, 1) {
		for i := 0; i < wp.size; i++ {
			go wp.worker(i)
		}
	}
}

func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
}

func (wp *WorkerPool) Submit(task Task) error {
	select {
	case wp.queue <- task:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		return fmt.Errorf("task queue is full")
	}
}

func (wp *WorkerPool) SubmitWithTimeout(task Task, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case wp.queue <- task:
		return nil
	case <-timer.C:
		return fmt.Errorf("timeout submitting task")
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

func (wp *WorkerPool) worker(id int) {
	wp.wg.Add(1)
	defer wp.wg.Done()

	for {
		select {
		case task := <-wp.queue:
			if task == nil {
				return
			}

			atomic.AddInt64(&wp.processing, 1)

			err := task.Execute(wp.ctx)

			atomic.AddInt64(&wp.processing, -1)

			if err != nil {
				atomic.AddInt64(&wp.failed, 1)
			} else {
				atomic.AddInt64(&wp.completed, 1)
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		WorkerCount:     wp.size,
		QueueLength:     len(wp.queue),
		QueueCapacity:   cap(wp.queue),
		TasksCompleted:  atomic.LoadInt64(&wp.completed),
		TasksFailed:     atomic.LoadInt64(&wp.failed),
		TasksProcessing: atomic.LoadInt64(&wp.processing),
	}
}

type WorkerPoolStats struct {
	WorkerCount     int   `json:"worker_count"`
	QueueLength     int   `json:"queue_length"`
	QueueCapacity   int   `json:"queue_capacity"`
	TasksCompleted  int64 `json:"tasks_completed"`
	TasksFailed     int64 `json:"tasks_failed"`
	TasksProcessing int64 `json:"tasks_processing"`
}

type ConcurrentRenderer struct {
	pool      *WorkerPool
	renderer  Renderer
	results   map[string]chan TaskResult
	resultsMu sync.RWMutex
	taskIDGen int64
}

func NewConcurrentRenderer(poolSize int, renderer Renderer) *ConcurrentRenderer {
	return &ConcurrentRenderer{
		pool:     NewWorkerPool(poolSize),
		renderer: renderer,
		results:  make(map[string]chan TaskResult),
	}
}

func (cr *ConcurrentRenderer) Start() {
	cr.pool.Start()
}

func (cr *ConcurrentRenderer) Stop() {
	cr.pool.Stop()

	cr.resultsMu.Lock()
	for _, ch := range cr.results {
		close(ch)
	}
	cr.results = make(map[string]chan TaskResult)
	cr.resultsMu.Unlock()
}

func (cr *ConcurrentRenderer) RenderAsync(tmplFS fs.FS, templatePath, outputPath string, data any) (string, <-chan TaskResult, error) {
	taskID := fmt.Sprintf("task-%d", atomic.AddInt64(&cr.taskIDGen, 1))

	resultChan := make(chan TaskResult, 1)

	task := &RenderTask{
		id:           taskID,
		priority:     1,
		templatePath: templatePath,
		outputPath:   outputPath,
		data:         data,
		renderer:     cr.renderer,
		result:       resultChan,
		tmplFS:       tmplFS,
	}

	cr.resultsMu.Lock()
	cr.results[taskID] = resultChan
	cr.resultsMu.Unlock()

	if err := cr.pool.Submit(task); err != nil {
		cr.resultsMu.Lock()
		delete(cr.results, taskID)
		cr.resultsMu.Unlock()
		close(resultChan)
		return "", nil, err
	}

	return taskID, resultChan, nil
}

func (cr *ConcurrentRenderer) RenderBatch(requests []RenderRequest) ([]TaskResult, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	var results []TaskResult
	var resultChans []<-chan TaskResult
	var taskIDs []string

	for _, req := range requests {
		taskID, resultChan, err := cr.RenderAsync(req.TmplFS, req.TemplatePath, req.OutputPath, req.Data)
		if err != nil {
			continue
		}

		taskIDs = append(taskIDs, taskID)
		resultChans = append(resultChans, resultChan)
	}

	for _, resultChan := range resultChans {
		result := <-resultChan
		results = append(results, result)
	}

	for _, taskID := range taskIDs {
		cr.resultsMu.Lock()
		if ch, exists := cr.results[taskID]; exists {
			delete(cr.results, taskID)
			close(ch)
		}
		cr.resultsMu.Unlock()
	}

	return results, nil
}

type RenderRequest struct {
	TmplFS       fs.FS  `json:"-"`
	TemplatePath string `json:"template_path"`
	OutputPath   string `json:"output_path"`
	Data         any    `json:"data"`
}

func (cr *ConcurrentRenderer) GetStats() WorkerPoolStats {
	return cr.pool.Stats()
}

func (cr *ConcurrentRenderer) WaitForCompletion(timeout time.Duration) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			stats := cr.GetStats()
			if stats.TasksProcessing == 0 && stats.QueueLength == 0 {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("timeout waiting for completion")
		}
	}
}

type SafeEngine struct {
	engine Engine
	mu     sync.RWMutex
}

func NewSafeEngine(engine Engine) *SafeEngine {
	return &SafeEngine{
		engine: engine,
	}
}

func (se *SafeEngine) RenderDir(ctx Context, templateDir string, data any) error {
	se.mu.RLock()
	defer se.mu.RUnlock()

	return se.engine.RenderDir(ctx, templateDir, data)
}

type ConcurrentSafeMap struct {
	data map[string]any
	mu   sync.RWMutex
}

func NewConcurrentSafeMap() *ConcurrentSafeMap {
	return &ConcurrentSafeMap{
		data: make(map[string]any),
	}
}

func (csm *ConcurrentSafeMap) Get(key string) (any, bool) {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	value, exists := csm.data[key]
	return value, exists
}

func (csm *ConcurrentSafeMap) Set(key string, value any) {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	csm.data[key] = value
}

func (csm *ConcurrentSafeMap) Delete(key string) {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	delete(csm.data, key)
}

func (csm *ConcurrentSafeMap) Keys() []string {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	keys := make([]string, 0, len(csm.data))
	for k := range csm.data {
		keys = append(keys, k)
	}
	return keys
}

func (csm *ConcurrentSafeMap) Len() int {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	return len(csm.data)
}

func (csm *ConcurrentSafeMap) Clear() {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	csm.data = make(map[string]any)
}

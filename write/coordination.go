package write

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileLockManager struct {
	locks    map[string]*FileLock
	mu       sync.RWMutex
	timeout  time.Duration
	cleanupInterval time.Duration
	stopChan chan struct{}
}

type FileLock struct {
	Path      string
	Acquired  time.Time
	Owner     string
	mu        sync.Mutex
	lockFile  *os.File
	refs      int
}

type LockOption func(*FileLockManager)

func WithLockTimeout(timeout time.Duration) LockOption {
	return func(flm *FileLockManager) {
		flm.timeout = timeout
	}
}

func WithCleanupInterval(interval time.Duration) LockOption {
	return func(flm *FileLockManager) {
		flm.cleanupInterval = interval
	}
}

func NewFileLockManager(opts ...LockOption) *FileLockManager {
	flm := &FileLockManager{
		locks:           make(map[string]*FileLock),
		timeout:         30 * time.Second,
		cleanupInterval: 5 * time.Minute,
		stopChan:        make(chan struct{}),
	}

	for _, opt := range opts {
		opt(flm)
	}

	go flm.cleanupLoop()
	return flm
}

func (flm *FileLockManager) AcquireLock(path, owner string) (*FileLock, error) {
	return flm.AcquireLockWithContext(context.Background(), path, owner)
}

func (flm *FileLockManager) AcquireLockWithContext(ctx context.Context, path, owner string) (*FileLock, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	lockPath := absPath + ".lock"

	flm.mu.Lock()
	if lock, exists := flm.locks[absPath]; exists {
		lock.refs++
		flm.mu.Unlock()
		return lock, nil
	}
	flm.mu.Unlock()

	timeout := flm.timeout
	if deadline, ok := ctx.Deadline(); ok {
		if ctxTimeout := time.Until(deadline); ctxTimeout < timeout {
			timeout = ctxTimeout
		}
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, fmt.Errorf("timeout acquiring lock for %s", path)
		case <-ticker.C:
			if lock, err := flm.tryAcquire(absPath, lockPath, owner); err == nil {
				return lock, nil
			}
		}
	}
}

func (flm *FileLockManager) tryAcquire(absPath, lockPath, owner string) (*FileLock, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("lock already exists")
		}
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	if _, err := lockFile.WriteString(fmt.Sprintf("%s\n%d\n", owner, time.Now().Unix())); err != nil {
		lockFile.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("failed to write lock info: %w", err)
	}

	lock := &FileLock{
		Path:     absPath,
		Acquired: time.Now(),
		Owner:    owner,
		lockFile: lockFile,
		refs:     1,
	}

	flm.mu.Lock()
	flm.locks[absPath] = lock
	flm.mu.Unlock()

	return lock, nil
}

func (flm *FileLockManager) ReleaseLock(lock *FileLock) error {
	if lock == nil {
		return nil
	}

	flm.mu.Lock()
	defer flm.mu.Unlock()

	existingLock, exists := flm.locks[lock.Path]
	if !exists || existingLock != lock {
		return fmt.Errorf("lock not owned by this manager")
	}

	lock.refs--
	if lock.refs > 0 {
		return nil
	}

	delete(flm.locks, lock.Path)

	lockPath := lock.Path + ".lock"
	
	lock.lockFile.Close()
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	return nil
}

func (flm *FileLockManager) IsLocked(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	flm.mu.RLock()
	_, locked := flm.locks[absPath]
	flm.mu.RUnlock()

	if locked {
		return true
	}

	lockPath := absPath + ".lock"
	_, err = os.Stat(lockPath)
	return err == nil
}

func (flm *FileLockManager) GetLockInfo(path string) (*FileLock, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	flm.mu.RLock()
	lock, exists := flm.locks[absPath]
	flm.mu.RUnlock()

	if exists {
		return lock, nil
	}

	return nil, fmt.Errorf("no lock found for path: %s", path)
}

func (flm *FileLockManager) cleanupLoop() {
	ticker := time.NewTicker(flm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-flm.stopChan:
			return
		case <-ticker.C:
			flm.cleanupStaleLocks()
		}
	}
}

func (flm *FileLockManager) cleanupStaleLocks() {
	flm.mu.Lock()
	defer flm.mu.Unlock()

	staleThreshold := time.Now().Add(-flm.timeout * 2)

	for path, lock := range flm.locks {
		if lock.Acquired.Before(staleThreshold) {
			lockPath := path + ".lock"
			
			lock.lockFile.Close()
			os.Remove(lockPath)
			delete(flm.locks, path)
		}
	}
}

func (flm *FileLockManager) Stop() {
	close(flm.stopChan)

	flm.mu.Lock()
	defer flm.mu.Unlock()

	for path, lock := range flm.locks {
		lockPath := path + ".lock"
		lock.lockFile.Close()
		os.Remove(lockPath)
	}

	flm.locks = make(map[string]*FileLock)
}

func (flm *FileLockManager) GetActiveLocks() []FileLock {
	flm.mu.RLock()
	defer flm.mu.RUnlock()

	locks := make([]FileLock, 0, len(flm.locks))
	for _, lock := range flm.locks {
		locks = append(locks, *lock)
	}

	return locks
}

type CoordinatedWriter struct {
	lockManager *FileLockManager
	baseWriter  Writer
	owner       string
}

func NewCoordinatedWriter(baseWriter Writer, lockManager *FileLockManager, owner string) *CoordinatedWriter {
	return &CoordinatedWriter{
		baseWriter:  baseWriter,
		lockManager: lockManager,
		owner:       owner,
	}
}

func (cw *CoordinatedWriter) Write(path string, content []byte, options WriteOptions) error {
	lock, err := cw.lockManager.AcquireLock(path, cw.owner)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer cw.lockManager.ReleaseLock(lock)

	return cw.baseWriter.Write(path, content, options)
}

func (cw *CoordinatedWriter) CanWrite(path string) bool {
	if cw.lockManager.IsLocked(path) {
		if lock, err := cw.lockManager.GetLockInfo(path); err == nil {
			return lock.Owner == cw.owner
		}
		return false
	}
	return cw.baseWriter.CanWrite(path)
}

func (cw *CoordinatedWriter) NeedsWrite(path string, content []byte) (bool, error) {
	return cw.baseWriter.NeedsWrite(path, content)
}

type ConcurrentWriteManager struct {
	writers     map[string]*CoordinatedWriter
	lockManager *FileLockManager
	mu          sync.RWMutex
}

func NewConcurrentWriteManager() *ConcurrentWriteManager {
	return &ConcurrentWriteManager{
		writers:     make(map[string]*CoordinatedWriter),
		lockManager: NewFileLockManager(),
	}
}

func (cwm *ConcurrentWriteManager) AddWriter(name string, writer Writer, owner string) {
	cwm.mu.Lock()
	defer cwm.mu.Unlock()

	cwm.writers[name] = NewCoordinatedWriter(writer, cwm.lockManager, owner)
}

func (cwm *ConcurrentWriteManager) GetWriter(name string) (*CoordinatedWriter, bool) {
	cwm.mu.RLock()
	defer cwm.mu.RUnlock()

	writer, exists := cwm.writers[name]
	return writer, exists
}

func (cwm *ConcurrentWriteManager) RemoveWriter(name string) {
	cwm.mu.Lock()
	defer cwm.mu.Unlock()

	delete(cwm.writers, name)
}

func (cwm *ConcurrentWriteManager) WriteWithWriter(writerName, path string, content []byte, options WriteOptions) error {
	writer, exists := cwm.GetWriter(writerName)
	if !exists {
		return fmt.Errorf("writer not found: %s", writerName)
	}

	return writer.Write(path, content, options)
}

func (cwm *ConcurrentWriteManager) GetLockManager() *FileLockManager {
	return cwm.lockManager
}

func (cwm *ConcurrentWriteManager) Stop() {
	cwm.lockManager.Stop()
}

func (cwm *ConcurrentWriteManager) GetStats() ConcurrentWriteStats {
	activeLocks := cwm.lockManager.GetActiveLocks()
	
	cwm.mu.RLock()
	writerCount := len(cwm.writers)
	cwm.mu.RUnlock()

	return ConcurrentWriteStats{
		ActiveLocks:  len(activeLocks),
		WriterCount:  writerCount,
		LockDetails:  activeLocks,
	}
}

type ConcurrentWriteStats struct {
	ActiveLocks int        `json:"active_locks"`
	WriterCount int        `json:"writer_count"`
	LockDetails []FileLock `json:"lock_details"`
}
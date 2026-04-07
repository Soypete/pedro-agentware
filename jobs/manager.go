package jobs

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// JobManager creates and tracks async agent jobs.
type JobManager interface {
	// Create registers a new job and returns its ID.
	Create(description string) (jobID string, err error)
	// Start marks a job as running. Should be called immediately before execution.
	Start(jobID string) error
	// Complete marks a job as done with a result.
	Complete(jobID string, result string) error
	// Fail marks a job as failed with an error message.
	Fail(jobID string, errMsg string) error
	// Cancel marks a job as canceled.
	Cancel(jobID string) error
	// Get retrieves the current state of a job.
	Get(jobID string) (*Job, error)
	// List returns all jobs, optionally filtered by status.
	List(status *Status) ([]*Job, error)
	// Watch returns a channel that receives job state updates.
	// Close the context to stop watching.
	Watch(ctx context.Context, jobID string) (<-chan *Job, error)
}

// inMemoryJobManager is an in-memory implementation of JobManager.
type inMemoryJobManager struct {
	mu    sync.RWMutex
	jobs  map[string]*Job
	chans map[string]chan *Job
}

// NewInMemoryManager creates a new in-memory JobManager.
func NewInMemoryManager() JobManager {
	return &inMemoryJobManager{
		jobs:  make(map[string]*Job),
		chans: make(map[string]chan *Job),
	}
}

// Create implements JobManager.
func (m *inMemoryJobManager) Create(description string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := generateJobID()
	now := time.Now()
	m.jobs[id] = &Job{
		ID:          id,
		Status:      StatusPending,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return id, nil
}

// Start implements JobManager.
func (m *inMemoryJobManager) Start(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	job.Status = StatusRunning
	job.UpdatedAt = time.Now()
	m.notify(job)
	return nil
}

// Complete implements JobManager.
func (m *inMemoryJobManager) Complete(jobID string, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	job.Status = StatusComplete
	job.Result = result
	job.UpdatedAt = time.Now()
	m.notify(job)
	return nil
}

// Fail implements JobManager.
func (m *inMemoryJobManager) Fail(jobID string, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	job.Status = StatusFailed
	job.Error = errMsg
	job.UpdatedAt = time.Now()
	m.notify(job)
	return nil
}

// Cancel implements JobManager.
func (m *inMemoryJobManager) Cancel(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	job.Status = StatusCanceled
	job.UpdatedAt = time.Now()
	m.notify(job)
	return nil
}

// Get implements JobManager.
func (m *inMemoryJobManager) Get(jobID string) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}
	return job, nil
}

// List implements JobManager.
func (m *inMemoryJobManager) List(status *Status) ([]*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Job, 0)
	for _, job := range m.jobs {
		if status != nil && job.Status != *status {
			continue
		}
		result = append(result, job)
	}
	return result, nil
}

// Watch implements JobManager.
func (m *inMemoryJobManager) Watch(ctx context.Context, jobID string) (<-chan *Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}

	ch := make(chan *Job, 1)
	ch <- job
	m.chans[jobID] = ch
	return ch, nil
}

func (m *inMemoryJobManager) notify(job *Job) {
	if ch, ok := m.chans[job.ID]; ok {
		select {
		case ch <- job:
		default:
		}
	}
}

var jobIDCounter int64

func generateJobID() string {
	id := atomic.AddInt64(&jobIDCounter, 1)
	return time.Now().Format("20060102150405") + fmt.Sprintf("-%04d", id%10000)
}

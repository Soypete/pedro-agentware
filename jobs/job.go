package jobs

import "time"

// Status represents the current state of an async agent job.
type Status string

const (
	// StatusPending means the job has been created but not started.
	StatusPending Status = "pending"
	// StatusRunning means the job is currently executing.
	StatusRunning Status = "running"
	// StatusComplete means the job finished successfully.
	StatusComplete Status = "complete"
	// StatusFailed means the job finished with an error.
	StatusFailed Status = "failed"
	// StatusCanceled means the job was cancelled before completion.
	StatusCanceled Status = "canceled"
)

// Job represents a single async agent task.
type Job struct {
	ID          string
	Status      Status
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Result      string
	Error       string
}

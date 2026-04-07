package jobs

import (
	"testing"
	"time"
)

func TestStatusConstants(t *testing.T) {
	if StatusPending != "pending" {
		t.Errorf("expected 'pending', got '%s'", StatusPending)
	}
	if StatusRunning != "running" {
		t.Errorf("expected 'running', got '%s'", StatusRunning)
	}
	if StatusComplete != "complete" {
		t.Errorf("expected 'complete', got '%s'", StatusComplete)
	}
	if StatusFailed != "failed" {
		t.Errorf("expected 'failed', got '%s'", StatusFailed)
	}
	if StatusCanceled != "canceled" {
		t.Errorf("expected 'canceled', got '%s'", StatusCanceled)
	}
}

func TestJob(t *testing.T) {
	now := time.Now()
	job := Job{
		ID:          "job_123",
		Status:      StatusRunning,
		Description: "Process data",
		CreatedAt:   now,
		UpdatedAt:   now,
		Result:      "",
		Error:       "",
	}

	if job.ID != "job_123" {
		t.Errorf("expected ID 'job_123', got '%s'", job.ID)
	}
	if job.Status != StatusRunning {
		t.Errorf("expected StatusRunning, got '%s'", job.Status)
	}
	if job.Description != "Process data" {
		t.Errorf("expected 'Process data', got '%s'", job.Description)
	}
}

func TestJobResultAndError(t *testing.T) {
	job := Job{
		ID:     "job_456",
		Status: StatusComplete,
		Result: "processed 100 items",
	}

	if job.Result != "processed 100 items" {
		t.Errorf("expected result 'processed 100 items', got '%s'", job.Result)
	}

	job2 := Job{
		ID:     "job_789",
		Status: StatusFailed,
		Error:  "connection timeout",
	}

	if job2.Error != "connection timeout" {
		t.Errorf("expected error 'connection timeout', got '%s'", job2.Error)
	}
}

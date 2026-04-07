package jobs

import (
	"context"
	"testing"
	"time"
)

func TestNewInMemoryManager(t *testing.T) {
	mgr := NewInMemoryManager()
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestJobManagerCreate(t *testing.T) {
	mgr := NewInMemoryManager()

	jobID, err := mgr.Create("test job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jobID == "" {
		t.Error("expected non-empty job ID")
	}

	job, err := mgr.Get(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Status != StatusPending {
		t.Errorf("expected StatusPending, got '%s'", job.Status)
	}
	if job.Description != "test job" {
		t.Errorf("expected 'test job', got '%s'", job.Description)
	}
}

func TestJobManagerStart(t *testing.T) {
	mgr := NewInMemoryManager()

	jobID, _ := mgr.Create("test job")
	err := mgr.Start(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job, _ := mgr.Get(jobID)
	if job.Status != StatusRunning {
		t.Errorf("expected StatusRunning, got '%s'", job.Status)
	}
}

func TestJobManagerComplete(t *testing.T) {
	mgr := NewInMemoryManager()

	jobID, err := mgr.Create("test job")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	err = mgr.Start(jobID)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	err = mgr.Complete(jobID, "result data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job, _ := mgr.Get(jobID)
	if job.Status != StatusComplete {
		t.Errorf("expected StatusComplete, got '%s'", job.Status)
	}
	if job.Result != "result data" {
		t.Errorf("expected 'result data', got '%s'", job.Result)
	}
}

func TestJobManagerFail(t *testing.T) {
	mgr := NewInMemoryManager()

	jobID, err := mgr.Create("test job")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	err = mgr.Start(jobID)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	err = mgr.Fail(jobID, "something went wrong")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job, _ := mgr.Get(jobID)
	if job.Status != StatusFailed {
		t.Errorf("expected StatusFailed, got '%s'", job.Status)
	}
	if job.Error != "something went wrong" {
		t.Errorf("expected error message, got '%s'", job.Error)
	}
}

func TestJobManagerCancel(t *testing.T) {
	mgr := NewInMemoryManager()

	jobID, _ := mgr.Create("test job")
	err := mgr.Cancel(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job, _ := mgr.Get(jobID)
	if job.Status != StatusCanceled {
		t.Errorf("expected StatusCanceled, got '%s'", job.Status)
	}
}

func TestJobManagerGet_NotFound(t *testing.T) {
	mgr := NewInMemoryManager()

	_, err := mgr.Get("nonexistent")
	if err != ErrJobNotFound {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

func TestJobManagerList(t *testing.T) {
	mgr := NewInMemoryManager()

	_, err := mgr.Create("job1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = mgr.Create("job2")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_, err = mgr.Create("job3")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	jobs, err := mgr.List(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}

	pending := StatusPending
	jobs, err = mgr.List(&pending)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 3 {
		t.Errorf("expected 3 pending jobs, got %d", len(jobs))
	}
}

func TestJobManagerList_FilterByStatus(t *testing.T) {
	mgr := NewInMemoryManager()

	id1, err := mgr.Create("job1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	id2, err := mgr.Create("job2")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = mgr.Start(id1)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	err = mgr.Start(id2)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	running := StatusRunning
	jobs, err := mgr.List(&running)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 running jobs, got %d", len(jobs))
	}

	pending := StatusPending
	jobs, err = mgr.List(&pending)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 pending jobs, got %d", len(jobs))
	}
}

func TestJobManagerWatch(t *testing.T) {
	mgr := NewInMemoryManager()

	jobID, _ := mgr.Create("test job")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch, err := mgr.Watch(ctx, jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := <-ch
	if job.ID != jobID {
		t.Errorf("expected job ID '%s', got '%s'", jobID, job.ID)
	}
}

func TestJobManagerWatch_NotFound(t *testing.T) {
	mgr := NewInMemoryManager()

	ctx := context.Background()
	_, err := mgr.Watch(ctx, "nonexistent")
	if err != ErrJobNotFound {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

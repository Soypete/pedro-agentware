package kitaru

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "api-key", "my-project")

	if client.baseURL != "http://localhost:8080" {
		t.Errorf("expected baseURL, got %s", client.baseURL)
	}
	if client.apiKey != "api-key" {
		t.Errorf("expected apiKey, got %s", client.apiKey)
	}
	if client.project != "my-project" {
		t.Errorf("expected project, got %s", client.project)
	}
	if client.httpClient == nil {
		t.Error("expected httpClient to be set")
	}
}

func TestNewHTTPClient_WithOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 10 * time.Second}
	client := NewHTTPClient("http://localhost:8080", "api-key", "my-project", WithHTTPClient(customClient))

	if client.httpClient != customClient {
		t.Error("expected custom httpClient to be set")
	}
}

func TestHTTPClient_Flow(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "", "")
	handle := client.Flow("test-flow")

	httpHandle, ok := handle.(*httpFlowHandle)
	if !ok {
		t.Fatal("expected httpFlowHandle")
	}
	if httpHandle.flowName != "test-flow" {
		t.Errorf("expected flowName 'test-flow', got %s", httpHandle.flowName)
	}
}

func TestHTTPClient_doRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "api-key", "project")
	resp, err := client.doRequest(context.Background(), "GET", "/test", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp) == "" {
		t.Error("expected non-empty response")
	}
}

func TestHTTPClient_doRequest_MarshalError(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "", "")
	_, err := client.doRequest(context.Background(), "GET", "/test", make(chan int))

	if err == nil {
		t.Fatal("expected error for unserializable body")
	}
}

func TestHTTPClient_doRequest_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error message"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	_, err := client.doRequest(context.Background(), "GET", "/test", nil)

	if err == nil {
		t.Fatal("expected error for 400 status")
	}
}

func TestHTTPClient_doRequest_RequestError(t *testing.T) {
	client := NewHTTPClient("http://invalid-url-that-fails", "", "")
	_, err := client.doRequest(context.Background(), "GET", "/test", nil)

	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestHTTPClient_doRequest_Headers(t *testing.T) {
	var receivedAuth, receivedProject string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedProject = r.Header.Get("X-Project")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "test-api-key", "test-project")
	client.doRequest(context.Background(), "GET", "/test", nil)

	if receivedAuth != "Bearer test-api-key" {
		t.Errorf("expected Authorization header, got %s", receivedAuth)
	}
	if receivedProject != "test-project" {
		t.Errorf("expected X-Project header, got %s", receivedProject)
	}
}

func TestHTTPClient_doRequest_EmptyAPIKey(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "project")
	client.doRequest(context.Background(), "GET", "/test", nil)

	if receivedAuth != "" {
		t.Errorf("expected no Authorization header, got %s", receivedAuth)
	}
}

func TestHTTPClient_doRequest_EmptyProject(t *testing.T) {
	var receivedProject string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedProject = r.Header.Get("X-Project")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "api-key", "")
	client.doRequest(context.Background(), "GET", "/test", nil)

	if receivedProject != "" {
		t.Errorf("expected no X-Project header, got %s", receivedProject)
	}
}

func TestHttpFlowHandle_Run(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RunFlowResponse{ExecutionID: "exec-123"})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	execID, err := handle.Run(context.Background(), map[string]any{"key": "value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if execID != "exec-123" {
		t.Errorf("expected execID 'exec-123', got %s", execID)
	}
}

func TestHttpFlowHandle_Run_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	execID, err := handle.Run(context.Background(), nil)
	if execID == "" && err != nil {
		return
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHttpFlowHandle_Run_MarshalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	execID, err := handle.Run(context.Background(), nil)
	if execID == "" && err != nil {
		return
	}
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHttpFlowHandle_RunWithWait_Completed(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "POST" {
			json.NewEncoder(w).Encode(RunFlowResponse{ExecutionID: "exec-123"})
		} else {
			status := "running"
			if callCount > 1 {
				status = "completed"
			}
			json.NewEncoder(w).Encode(ExecutionResponse{
				ID:     "exec-123",
				Status: status,
			})
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	exec, err := handle.RunWithWait(context.Background(), nil, 10*time.Millisecond)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", exec.Status)
	}
}

func TestHttpFlowHandle_RunWithWait_Failed(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "POST" {
			json.NewEncoder(w).Encode(RunFlowResponse{ExecutionID: "exec-123"})
		} else {
			status := "running"
			if callCount > 1 {
				status = "failed"
			}
			json.NewEncoder(w).Encode(ExecutionResponse{
				ID:     "exec-123",
				Status: status,
			})
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	exec, err := handle.RunWithWait(context.Background(), nil, 10*time.Millisecond)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != "failed" {
		t.Errorf("expected status 'failed', got %s", exec.Status)
	}
}

func TestHttpFlowHandle_RunWithWait_Waiting(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "POST" {
			json.NewEncoder(w).Encode(RunFlowResponse{ExecutionID: "exec-123"})
		} else {
			status := "running"
			if callCount > 1 {
				status = "waiting"
			}
			json.NewEncoder(w).Encode(ExecutionResponse{
				ID:     "exec-123",
				Status: status,
			})
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	exec, err := handle.RunWithWait(context.Background(), nil, 10*time.Millisecond)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != "waiting" {
		t.Errorf("expected status 'waiting', got %s", exec.Status)
	}
}

func TestHttpFlowHandle_RunWithWait_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "POST" {
			json.NewEncoder(w).Encode(RunFlowResponse{ExecutionID: "exec-123"})
		} else {
			json.NewEncoder(w).Encode(ExecutionResponse{
				ID:     "exec-123",
				Status: "running",
			})
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := handle.RunWithWait(ctx, nil, 10*time.Millisecond)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestHttpFlowHandle_GetExecution_NoExecution(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "", "")
	handle := client.Flow("test-flow")

	_, err := handle.GetExecution()

	if err == nil {
		t.Fatal("expected error when no execution started")
	}
}

func TestHttpFlowHandle_GetExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			json.NewEncoder(w).Encode(RunFlowResponse{ExecutionID: "exec-123"})
		} else {
			json.NewEncoder(w).Encode(ExecutionResponse{
				ID:        "exec-123",
				Status:    "completed",
				StartedAt: time.Now().Format(time.RFC3339),
				UpdatedAt: time.Now().Format(time.RFC3339),
				Output:    map[string]any{"result": "success"},
			})
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	execID, err := handle.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	t.Logf("execID from Run: %s", execID)

	httpHandle := handle.(*httpFlowHandle)
	httpHandle.mu.Lock()
	httpHandle.execID = execID
	httpHandle.mu.Unlock()

	exec, err := handle.GetExecution()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.ID != "exec-123" {
		t.Errorf("expected ID 'exec-123', got %s", exec.ID)
	}
	if exec.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", exec.Status)
	}
}

func TestHttpFlowHandle_Checkpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && r.URL.Path == "/api/v1/flows/test-flow/run" {
			json.NewEncoder(w).Encode(RunFlowResponse{ExecutionID: "exec-123"})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	httpHandle := handle.(*httpFlowHandle)
	httpHandle.mu.Lock()
	httpHandle.execID = "exec-123"
	httpHandle.mu.Unlock()

	err := handle.Checkpoint("checkpoint1", map[string]any{"data": "value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHttpFlowHandle_Checkpoint_NoExecution(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "", "")
	handle := client.Flow("test-flow")

	err := handle.Checkpoint("checkpoint1", nil)

	if err == nil {
		t.Fatal("expected error when no execution started")
	}
}

func TestHttpFlowHandle_RestoreCheckpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": "value"})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	httpHandle := handle.(*httpFlowHandle)
	httpHandle.mu.Lock()
	httpHandle.execID = "exec-123"
	httpHandle.mu.Unlock()

	data, err := handle.RestoreCheckpoint("checkpoint1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["data"] != "value" {
		t.Errorf("expected data 'value', got %v", data["data"])
	}
}

func TestHttpFlowHandle_RestoreCheckpoint_NoExecution(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "", "")
	handle := client.Flow("test-flow")

	_, err := handle.RestoreCheckpoint("checkpoint1")

	if err == nil {
		t.Fatal("expected error when no execution started")
	}
}

func TestHttpFlowHandle_SaveArtifact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	httpHandle := handle.(*httpFlowHandle)
	httpHandle.mu.Lock()
	httpHandle.execID = "exec-123"
	httpHandle.mu.Unlock()

	err := handle.SaveArtifact("key1", "data", "text")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHttpFlowHandle_SaveArtifact_NoExecution(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "", "")
	handle := client.Flow("test-flow")

	err := handle.SaveArtifact("key1", "data", "text")

	if err == nil {
		t.Fatal("expected error when no execution started")
	}
}

func TestHttpFlowHandle_LoadArtifact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode("artifact data")
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	httpHandle := handle.(*httpFlowHandle)
	httpHandle.mu.Lock()
	httpHandle.execID = "exec-123"
	httpHandle.mu.Unlock()

	var dest string
	err := handle.LoadArtifact("key1", &dest)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHttpFlowHandle_LoadArtifact_NoExecution(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "", "")
	handle := client.Flow("test-flow")

	err := handle.LoadArtifact("key1", nil)

	if err == nil {
		t.Fatal("expected error when no execution started")
	}
}

func TestHttpFlowHandle_Log(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	httpHandle := handle.(*httpFlowHandle)
	httpHandle.mu.Lock()
	httpHandle.execID = "exec-123"
	httpHandle.mu.Unlock()

	err := handle.Log(LogLevelInfo, "test message", map[string]any{"key": "value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHttpFlowHandle_Log_NoExecution(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "", "")
	handle := client.Flow("test-flow")

	err := handle.Log(LogLevelInfo, "test", nil)

	if err == nil {
		t.Fatal("expected error when no execution started")
	}
}

func TestHttpFlowHandle_Log_AllLevels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	httpHandle := handle.(*httpFlowHandle)
	httpHandle.mu.Lock()
	httpHandle.execID = "exec-123"
	httpHandle.mu.Unlock()

	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarning, LogLevelError}
	for _, level := range levels {
		err := handle.Log(level, "test", nil)
		if err != nil {
			t.Fatalf("unexpected error for level %d: %v", level, err)
		}
	}
}

func TestHttpFlowHandle_Replay(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RunFlowResponse{ExecutionID: "exec-456"})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	httpHandle := handle.(*httpFlowHandle)
	httpHandle.mu.Lock()
	httpHandle.execID = "exec-123"
	httpHandle.mu.Unlock()

	newExecID, err := handle.Replay("checkpoint1", map[string]any{"key": "value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newExecID != "exec-456" {
		t.Errorf("expected newExecID 'exec-456', got %s", newExecID)
	}
}

func TestHttpFlowHandle_Replay_NoExecution(t *testing.T) {
	client := NewHTTPClient("http://localhost:8080", "", "")
	handle := client.Flow("test-flow")

	_, err := handle.Replay("checkpoint1", nil)

	if err == nil {
		t.Fatal("expected error when no execution started")
	}
}

func TestHttpFlowHandle_GetExecID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RunFlowResponse{ExecutionID: "exec-123"})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	handle := client.Flow("test-flow")

	if handle.GetExecID() != "" {
		t.Error("expected empty execID before run")
	}

	handle.Run(context.Background(), nil)

	if handle.GetExecID() != "exec-123" {
		t.Errorf("expected execID 'exec-123', got %s", handle.GetExecID())
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"invalid", false},
		{time.Now().Format(time.RFC3339), true},
		{"2024-01-01T00:00:00Z", true},
	}

	for _, tt := range tests {
		result := parseTime(tt.input)
		if tt.expected && result.IsZero() {
			t.Errorf("expected non-zero time for input %s", tt.input)
		}
		if !tt.expected && !result.IsZero() {
			t.Errorf("expected zero time for input %s", tt.input)
		}
	}
}

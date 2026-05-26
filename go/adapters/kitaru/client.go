package kitaru

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type HTTPClient struct {
	baseURL    string
	apiKey     string
	project    string
	httpClient *http.Client
	mu         sync.RWMutex
}

type HTTPClientOption func(*HTTPClient)

func WithHTTPClient(client *http.Client) HTTPClientOption {
	return func(c *HTTPClient) {
		c.httpClient = client
	}
}

func NewHTTPClient(baseURL, apiKey, project string, opts ...HTTPClientOption) *HTTPClient {
	client := &HTTPClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		project: project,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (c *HTTPClient) Flow(name string) FlowHandle {
	return &httpFlowHandle{
		client:   c,
		flowName: name,
	}
}

func (c *HTTPClient) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if c.project != "" {
		req.Header.Set("X-Project", c.project)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type httpFlowHandle struct {
	client   *HTTPClient
	flowName string
	execID   string
	mu       sync.Mutex
}

type RunFlowRequest struct {
	Inputs map[string]any `json:"inputs"`
}

type RunFlowResponse struct {
	ExecutionID string `json:"execution_id"`
}

type ExecutionResponse struct {
	ID        string         `json:"id"`
	Status    string         `json:"status"`
	StartedAt string         `json:"started_at"`
	UpdatedAt string         `json:"updated_at"`
	Output    map[string]any `json:"output"`
}

func (f *httpFlowHandle) Run(ctx context.Context, inputs map[string]any) (string, error) {
	req := RunFlowRequest{Inputs: inputs}
	respBody, err := f.client.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/flows/%s/run", f.flowName), req)
	if err != nil {
		return "", err
	}

	var resp RunFlowResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	f.mu.Lock()
	f.execID = resp.ExecutionID
	f.mu.Unlock()

	return resp.ExecutionID, nil
}

func (f *httpFlowHandle) RunWithWait(ctx context.Context, inputs map[string]any, pollInterval time.Duration) (*Execution, error) {
	_, err := f.Run(ctx, inputs)
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			exec, err := f.GetExecution()
			if err != nil {
				return nil, err
			}
			if exec.Status == "completed" || exec.Status == "failed" || exec.Status == "waiting" {
				return exec, nil
			}
		}
	}
}

func (f *httpFlowHandle) GetExecution() (*Execution, error) {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return nil, fmt.Errorf("no execution started")
	}

	respBody, err := f.client.doRequest(context.Background(), "GET", fmt.Sprintf("/api/v1/executions/%s", execID), nil)
	if err != nil {
		return nil, err
	}

	var resp ExecutionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &Execution{
		ID:        resp.ID,
		Status:    resp.Status,
		StartedAt: parseTime(resp.StartedAt),
		UpdatedAt: parseTime(resp.UpdatedAt),
		Output:    resp.Output,
	}, nil
}

func (f *httpFlowHandle) Checkpoint(name string, data map[string]any) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	_, err := f.client.doRequest(context.Background(), "POST", fmt.Sprintf("/api/v1/executions/%s/checkpoints", execID), map[string]any{
		"name": name,
		"data": data,
	})
	return err
}

func (f *httpFlowHandle) RestoreCheckpoint(name string) (map[string]any, error) {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return nil, fmt.Errorf("no execution started")
	}

	respBody, err := f.client.doRequest(context.Background(), "GET", fmt.Sprintf("/api/v1/executions/%s/checkpoints/%s", execID, name), nil)
	if err != nil {
		return nil, err
	}

	var checkpoint map[string]any
	if err := json.Unmarshal(respBody, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint: %w", err)
	}

	return checkpoint, nil
}

func (f *httpFlowHandle) SaveArtifact(key string, data interface{}, artifactType string) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	_, err := f.client.doRequest(context.Background(), "POST", fmt.Sprintf("/api/v1/executions/%s/artifacts", execID), map[string]any{
		"key":          key,
		"data":         data,
		"artifactType": artifactType,
	})
	return err
}

func (f *httpFlowHandle) LoadArtifact(key string, dest interface{}) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	respBody, err := f.client.doRequest(context.Background(), "GET", fmt.Sprintf("/api/v1/executions/%s/artifacts/%s", execID, key), nil)
	if err != nil {
		return err
	}

	return json.Unmarshal(respBody, dest)
}

func (f *httpFlowHandle) Log(level LogLevel, message string, metadata map[string]any) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	levelStr := "info"
	switch level {
	case LogLevelDebug:
		levelStr = "debug"
	case LogLevelInfo:
		levelStr = "info"
	case LogLevelWarning:
		levelStr = "warning"
	case LogLevelError:
		levelStr = "error"
	}

	_, err := f.client.doRequest(context.Background(), "POST", fmt.Sprintf("/api/v1/executions/%s/logs", execID), map[string]any{
		"level":    levelStr,
		"message":  message,
		"metadata": metadata,
	})
	return err
}

func (f *httpFlowHandle) Replay(fromCheckpoint string, inputs map[string]any) (string, error) {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return "", fmt.Errorf("no execution to replay")
	}

	req := map[string]any{
		"from_checkpoint": fromCheckpoint,
	}
	if inputs != nil {
		req["inputs"] = inputs
	}

	respBody, err := f.client.doRequest(context.Background(), "POST", fmt.Sprintf("/api/v1/executions/%s/replay", execID), req)
	if err != nil {
		return "", err
	}

	var resp RunFlowResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse replay response: %w", err)
	}

	f.mu.Lock()
	f.execID = resp.ExecutionID
	f.mu.Unlock()

	return resp.ExecutionID, nil
}

func (f *httpFlowHandle) GetExecID() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.execID
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

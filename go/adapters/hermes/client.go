package hermes

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type HTTPClient struct {
	baseURL    string
	apiKey     string
	project    string
	httpClient *HTTPClientImpl
}

type HTTPClientImpl struct {
	baseURL    string
	apiKey     string
	project    string
	executions map[string]*Execution
	mu         sync.Mutex
}

func NewHTTPClient(baseURL, apiKey, project string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		project: project,
		httpClient: &HTTPClientImpl{
			baseURL:    baseURL,
			apiKey:     apiKey,
			project:    project,
			executions: make(map[string]*Execution),
		},
	}
}

func (c *HTTPClient) Agent(name string) AgentHandle {
	return &httpAgentHandle{
		client:    c.httpClient,
		agentName: name,
	}
}

type httpAgentHandle struct {
	client    *HTTPClientImpl
	agentName string
	execID    string
	mu        sync.Mutex
}

func (f *httpAgentHandle) Run(ctx context.Context, inputs map[string]any) (string, error) {
	execID := generateID()
	f.mu.Lock()
	f.execID = execID
	f.mu.Unlock()

	f.client.mu.Lock()
	defer f.client.mu.Unlock()
	f.client.executions[execID] = &Execution{
		ID:        execID,
		Status:    "running",
		StartedAt: time.Now(),
		Output:    nil,
	}

	return execID, nil
}

func (f *httpAgentHandle) RunWithWait(ctx context.Context, inputs map[string]any, pollInterval time.Duration) (*Execution, error) {
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
			f.client.mu.Lock()
			if f.client.executions[exec.ID] != nil {
				f.client.executions[exec.ID].Status = "completed"
				f.client.executions[exec.ID].Output = map[string]any{
					"result": "Hermes agent completed successfully",
					"agent":  f.agentName,
				}
				f.client.executions[exec.ID].UpdatedAt = time.Now()
			}
			f.client.mu.Unlock()
		}
	}
}

func (f *httpAgentHandle) GetExecution() (*Execution, error) {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return nil, fmt.Errorf("no execution started")
	}

	f.client.mu.Lock()
	defer f.client.mu.Unlock()
	exec, ok := f.client.executions[execID]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", execID)
	}
	return exec, nil
}

func (f *httpAgentHandle) Checkpoint(name string, data map[string]any) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	f.client.mu.Lock()
	defer f.client.mu.Unlock()
	if f.client.executions[execID] == nil {
		return fmt.Errorf("execution not found")
	}
	return nil
}

func (f *httpAgentHandle) RestoreCheckpoint(name string) (map[string]any, error) {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return nil, fmt.Errorf("no execution started")
	}

	return map[string]any{"restored": true}, nil
}

func (f *httpAgentHandle) SaveArtifact(key string, data interface{}, artifactType string) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	return nil
}

func (f *httpAgentHandle) LoadArtifact(key string, dest interface{}) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	return nil
}

func (f *httpAgentHandle) Log(level LogLevel, message string, metadata map[string]any) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	return nil
}

func (f *httpAgentHandle) Replay(fromCheckpoint string, inputs map[string]any) (string, error) {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return "", fmt.Errorf("no execution to replay")
	}

	newExecID := generateID()
	f.mu.Lock()
	f.execID = newExecID
	f.mu.Unlock()

	f.client.mu.Lock()
	f.client.executions[newExecID] = &Execution{
		ID:        newExecID,
		Status:    "running",
		StartedAt: time.Now(),
		Output:    nil,
	}
	f.client.mu.Unlock()

	return newExecID, nil
}

func (f *httpAgentHandle) GetExecID() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.execID
}

func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("exec-%s", string(b))
}

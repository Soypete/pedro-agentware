package kitaru

import (
	"context"
	"fmt"
	"sync"
	"time"

	sdk "github.com/zenml-io/kitaru-sdk-go"
)

type ClientOption = sdk.ClientOption

func WithUsernamePassword(username, password string) ClientOption {
	return sdk.WithUsernamePassword(username, password)
}

type KitaruClient struct {
	*sdk.Client
	mu sync.Mutex
}

func NewClient(serverURL, apiKey, project string, opts ...ClientOption) *KitaruClient {
	return &KitaruClient{
		Client: sdk.NewClient(serverURL, apiKey, project, opts...),
	}
}

func (c *KitaruClient) Flow(name string) FlowHandle {
	return &flowHandle{
		client:   c,
		flowName: name,
	}
}

type flowHandle struct {
	client   *KitaruClient
	flowName string
	execID   string
	mu       sync.Mutex
}

func (f *flowHandle) Run(ctx context.Context, inputs map[string]any) (string, error) {
	execID, err := f.client.RunFlow(f.flowName, inputs)
	if err != nil {
		return "", err
	}
	f.mu.Lock()
	f.execID = execID
	f.mu.Unlock()
	return execID, nil
}

func (f *flowHandle) RunWithWait(ctx context.Context, inputs map[string]any, pollInterval time.Duration) (*Execution, error) {
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
			if exec.Status == "completed" || exec.Status == "failed" || exec.Status == "cancelled" {
				return exec, nil
			}
		}
	}
}

func (f *flowHandle) GetExecution() (*Execution, error) {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return nil, fmt.Errorf("no execution started")
	}

	exec, err := f.client.GetExecution(execID)
	if err != nil {
		return nil, err
	}

	return &Execution{
		ID:        exec.ID,
		Status:    string(exec.Status),
		StartedAt: exec.CreatedAt,
		UpdatedAt: exec.UpdatedAt,
		Output:    exec.Outputs,
	}, nil
}

func (f *flowHandle) Checkpoint(name string, data map[string]any) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	_, err := f.client.Client.SaveCheckpoint(execID, name, data)
	return err
}

func (f *flowHandle) RestoreCheckpoint(name string) (map[string]any, error) {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return nil, fmt.Errorf("no execution started")
	}

	checkpoint, err := f.client.GetCheckpoint(execID, name)
	if err != nil {
		return nil, err
	}

	return checkpoint.Data, nil
}

func (f *flowHandle) SaveArtifact(key string, data interface{}, artifactType string) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	_, err := f.client.SaveArtifact(execID, key, data, artifactType)
	return err
}

func (f *flowHandle) LoadArtifact(key string, dest interface{}) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	return f.client.LoadArtifact(execID, key, dest)
}

func (f *flowHandle) Log(level LogLevel, message string, metadata map[string]any) error {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return fmt.Errorf("no execution started")
	}

	// Convert LogLevel to SDK LogLevel
	sdkLevel := sdk.LogLevelInfo
	switch level {
	case LogLevelDebug:
		sdkLevel = sdk.LogLevelDebug
	case LogLevelInfo:
		sdkLevel = sdk.LogLevelInfo
	case LogLevelWarning:
		sdkLevel = sdk.LogLevelWarn
	case LogLevelError:
		sdkLevel = sdk.LogLevelError
	}

	return f.client.Log(execID, sdkLevel, message, metadata)
}

func (f *flowHandle) Replay(fromCheckpoint string, inputs map[string]any) (string, error) {
	f.mu.Lock()
	execID := f.execID
	f.mu.Unlock()

	if execID == "" {
		return "", fmt.Errorf("no execution started")
	}

	newExecID, err := f.client.Replay(execID, fromCheckpoint, inputs)
	if err != nil {
		return "", err
	}

	// Update the flow handle with new execution ID
	f.mu.Lock()
	f.execID = newExecID
	f.mu.Unlock()

	return newExecID, nil
}

func (f *flowHandle) GetExecID() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.execID
}

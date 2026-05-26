package kitaru

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/soypete/pedro-agentware/go/middleware"
	"github.com/soypete/pedro-agentware/go/tools"
)

type Client interface {
	Flow(name string) FlowHandle
}

type FlowHandle interface {
	Run(ctx context.Context, inputs map[string]any) (string, error)
	RunWithWait(ctx context.Context, inputs map[string]any, pollInterval time.Duration) (*Execution, error)
	GetExecution() (*Execution, error)
	Checkpoint(name string, data map[string]any) error
	RestoreCheckpoint(name string) (map[string]any, error)
	SaveArtifact(key string, data interface{}, artifactType string) error
	LoadArtifact(key string, dest interface{}) error
	Log(level LogLevel, message string, metadata map[string]any) error
	Replay(fromCheckpoint string, inputs map[string]any) (string, error)
	GetExecID() string
}

type Execution struct {
	ID        string
	Status    string
	StartedAt time.Time
	UpdatedAt time.Time
	Output    map[string]any
}

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

type FlowToolExecutor struct {
	client        Client
	flowMapping   map[string]string
	defaultConfig *Config
	mu            sync.RWMutex
}

type Config struct {
	WaitInterval   time.Duration
	RequestTimeout time.Duration
	RetryCount     int
}

type flowTool struct {
	name        string
	description string
	executor    *FlowToolExecutor
}

func (t *flowTool) Name() string        { return t.name }
func (t *flowTool) Description() string { return t.description }
func (t *flowTool) Execute(ctx context.Context, args map[string]any) (*tools.Result, error) {
	return t.executor.Execute(ctx, t.name, args)
}
func (t *flowTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"inputs": map[string]any{
				"type":        "object",
				"description": "Flow input parameters",
			},
		},
	}
}
func (t *flowTool) Examples() []tools.ToolExample { return nil }

func NewFlowToolExecutor(client Client, flowMapping map[string]string) *FlowToolExecutor {
	return &FlowToolExecutor{
		client:      client,
		flowMapping: flowMapping,
		defaultConfig: &Config{
			WaitInterval:   2 * time.Second,
			RequestTimeout: 60 * time.Second,
			RetryCount:     3,
		},
	}
}

func NewFlowToolExecutorWithConfig(client Client, flowMapping map[string]string, config *Config) *FlowToolExecutor {
	if config == nil {
		config = &Config{
			WaitInterval:   2 * time.Second,
			RequestTimeout: 60 * time.Second,
			RetryCount:     3,
		}
	}
	return &FlowToolExecutor{
		client:        client,
		flowMapping:   flowMapping,
		defaultConfig: config,
	}
}

func (e *FlowToolExecutor) Execute(ctx context.Context, toolName string, args map[string]any) (*tools.Result, error) {
	flowName, ok := e.flowMapping[toolName]
	if !ok {
		return &tools.Result{
			Success: false,
			Error:   fmt.Sprintf("unknown tool: %s", toolName),
		}, nil
	}

	flow := e.client.Flow(flowName)
	config := e.getConfig()

	ctx, cancel := context.WithTimeout(ctx, config.RequestTimeout)
	defer cancel()

	exec, err := flow.RunWithWait(ctx, args, config.WaitInterval)
	if err != nil {
		return &tools.Result{
			Success: false,
			Error:   fmt.Sprintf("flow execution failed: %v", err),
			Metadata: map[string]any{
				"tool_name": toolName,
				"flow_name": flowName,
			},
		}, err
	}

	return &tools.Result{
		Success: exec.Status == "completed",
		Output:  fmt.Sprintf("%v", exec.Output),
		Metadata: map[string]any{
			"execution_id": exec.ID,
			"status":       exec.Status,
			"tool_name":    toolName,
			"flow_name":    flowName,
		},
	}, nil
}

func (e *FlowToolExecutor) ListTools() []tools.Tool {
	toolsList := make([]tools.Tool, 0, len(e.flowMapping))
	for toolName, flowName := range e.flowMapping {
		toolsList = append(toolsList, &flowTool{
			name:        toolName,
			description: fmt.Sprintf("Execute Kitaru flow: %s", flowName),
			executor:    e,
		})
	}
	return toolsList
}

func (e *FlowToolExecutor) GetFlowMapping() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]string, len(e.flowMapping))
	for k, v := range e.flowMapping {
		result[k] = v
	}
	return result
}

func (e *FlowToolExecutor) UpdateFlowMapping(mapping map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.flowMapping = mapping
}

func (e *FlowToolExecutor) AddFlowMapping(toolName, flowName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.flowMapping[toolName] = flowName
}

func (e *FlowToolExecutor) getConfig() *Config {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.defaultConfig
}

func (e *FlowToolExecutor) SetConfig(config *Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.defaultConfig = config
}

type KitaruMiddlewareAdapter struct {
	executor *FlowToolExecutor
	policy   middleware.PolicyEvaluator
	auditor  middleware.Auditor
}

func NewKitaruMiddlewareAdapter(
	client Client,
	flowMapping map[string]string,
	policy middleware.PolicyEvaluator,
	auditor middleware.Auditor,
) *KitaruMiddlewareAdapter {
	executor := NewFlowToolExecutor(client, flowMapping)
	return &KitaruMiddlewareAdapter{
		executor: executor,
		policy:   policy,
		auditor:  auditor,
	}
}

func (a *KitaruMiddlewareAdapter) Execute(ctx context.Context, toolName string, args map[string]any) (*tools.Result, error) {
	caller := getCallerContext(ctx)

	var decision middleware.Decision
	if a.policy != nil {
		decision = a.policy.Evaluate(toolName, args, caller)
	} else {
		decision = middleware.Decision{Action: middleware.ActionAllow, Reason: "no policy configured"}
	}

	if a.auditor != nil {
		a.auditor.Record(middleware.AuditRecord{
			SessionID: caller.SessionID,
			ToolName:  toolName,
			Args:      args,
			Decision:  decision,
		})
	}

	if decision.Action == middleware.ActionDeny {
		return &tools.Result{
			Success: false,
			Error:   fmt.Sprintf("denied by policy: %s", decision.Reason),
			Metadata: map[string]any{
				"rule": decision.Rule,
			},
		}, nil
	}

	return a.executor.Execute(ctx, toolName, args)
}

func (a *KitaruMiddlewareAdapter) ListTools() []tools.Tool {
	return a.executor.ListTools()
}

func (a *KitaruMiddlewareAdapter) WithPolicy(policy middleware.PolicyEvaluator) *KitaruMiddlewareAdapter {
	a.policy = policy
	return a
}

func (a *KitaruMiddlewareAdapter) WithAuditor(auditor middleware.Auditor) *KitaruMiddlewareAdapter {
	a.auditor = auditor
	return a
}

func (a *KitaruMiddlewareAdapter) GetAuditor() middleware.Auditor {
	return a.auditor
}

func getCallerContext(ctx context.Context) middleware.CallerContext {
	if c, ok := ctx.Value(kitaruCallerKey).(middleware.CallerContext); ok {
		return c
	}
	return middleware.CallerContext{
		Trusted: true,
	}
}

const kitaruCallerKey string = "kitaru_caller_context"

func WithCallerContext(ctx context.Context, caller middleware.CallerContext) context.Context {
	return context.WithValue(ctx, kitaruCallerKey, caller)
}

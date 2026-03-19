package middleware

import (
	"context"
	"time"
)

type Action string

const (
	ActionAllow  Action = "allow"
	ActionDeny   Action = "deny"
	ActionFilter Action = "filter"
)

type ToolResult struct {
	Content    interface{}
	Error      error
	IsStream   bool
	StreamChan <-chan interface{}
	Metadata   map[string]interface{}
}

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

type CallerContext struct {
	Trusted   bool
	Role      string
	UserID    string
	SessionID string
	Source    string
	Metadata  map[string]interface{}
}

type Decision struct {
	Timestamp time.Time
	Tool      string
	Args      map[string]interface{}
	Action    Action
	Rule      string
	Reason    string
	CallerCtx CallerContext
	Success   *bool
}

func (d Decision) IsAllowed() bool {
	return d.Action == ActionAllow
}

func (d Decision) IsDenied() bool {
	return d.Action == ActionDeny
}

func (d Decision) IsFiltered() bool {
	return d.Action == ActionFilter
}

type ToolExecutor interface {
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error)
	ListTools() []ToolDefinition
}

type RateLimit struct {
	Count  int           `yaml:"count"`
	Window time.Duration `yaml:"window"`
}

type Schema struct {
	Type       string      `yaml:"type"`
	Required   bool        `yaml:"required"`
	Pattern    string      `yaml:"pattern"`
	MinLength  int         `yaml:"min_length"`
	MaxLength  int         `yaml:"max_length"`
	Enum       []string    `yaml:"enum"`
	Properties []Schema    `yaml:"properties"`
	Items      *Schema     `yaml:"items"`
	Default    interface{} `yaml:"default"`
}

type Condition struct {
	Field    string `yaml:"field"`
	Operator string `yaml:"operator"`
	Value    string `yaml:"value"`
}

type QuickFilter struct {
	SkipWhen []Condition `yaml:"skip_when"`
}

type Rule struct {
	Name           string            `yaml:"name"`
	Tools          []string          `yaml:"tools"`
	Action         Action            `yaml:"action"`
	Conditions     []Condition       `yaml:"conditions"`
	MaxRate        *RateLimit        `yaml:"max_rate"`
	MaxTurns       *int              `yaml:"max_turns"`
	IterationLimit *int              `yaml:"iteration_limit"`
	QuickFilter    *QuickFilter      `yaml:"quick_filter"`
	ArgSchema      map[string]Schema `yaml:"arg_schema"`
	RedactFields   []string          `yaml:"redact_fields"`
}

type Policy struct {
	Rules       []Rule `yaml:"rules"`
	DefaultDeny bool   `yaml:"default_deny"`
}

package llm

import (
	"context"
)

// Backend is the abstraction over any OpenAI-compatible LLM API.
type Backend interface {
	// Complete sends a conversation to the LLM and returns its response.
	Complete(ctx context.Context, req *Request) (*Response, error)
	// SupportsNativeToolCalling returns true if the backend can handle
	// ToolDefinitions in the request and return structured ToolCalls.
	SupportsNativeToolCalling() bool
	// ModelName returns the model identifier being used.
	ModelName() string
	// ContextWindowSize returns the maximum number of tokens this backend
	// can process per request. Returns 0 if unknown.
	ContextWindowSize() int
}

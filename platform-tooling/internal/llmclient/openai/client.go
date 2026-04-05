package openai

import "context"

// Client is the OpenAI API client interface (Responses API / Chat Completions).
// See docs/analyze-preparation/executive-openai-gap-list.md for real HTTP implementation.
type Client interface {
	// CreateResponse runs a request against the Responses API (or Chat Completions with tools).
	CreateResponse(ctx context.Context, req *ResponseRequest) (*ResponseResponse, error)
	// CreateResponseStream returns a stream of response chunks.
	CreateResponseStream(ctx context.Context, req *ResponseRequest) (ResponseStream, error)
}

// ResponseStream is the interface for streaming response chunks.
type ResponseStream interface {
	Recv() (*ResponseChunk, error)
	Close() error
}

// ResponseChunk is a single streamed chunk (e.g. content delta or tool call).
type ResponseChunk struct {
	ContentDelta string
	ToolCallID   string
	ToolName     string
	ToolArgs     string
	Done         bool
}

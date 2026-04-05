package openai

import "encoding/json"

// ResponseRequest is the request payload for Responses API (or Chat Completions with tools).
type ResponseRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	Tools       []ToolSpec      `json:"tools,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	Seed        *int64          `json:"seed,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	MaxSteps    int             `json:"max_steps,omitempty"`
	Extra       json.RawMessage `json:"-"`
}

// Message represents a single message in the conversation.
type Message struct {
	Role       string            `json:"role"` // system, user, assistant, tool
	Content    string            `json:"content,omitempty"`
	ToolCalls  []MessageToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Name       string            `json:"name,omitempty"`
}

type MessageToolCall struct {
	ID       string              `json:"id,omitempty"`
	Type     string              `json:"type,omitempty"`
	Function *ToolFunctionCaller `json:"function,omitempty"`
}

type ToolFunctionCaller struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

// ToolSpec is the OpenAI function/tool declaration (name, description, parameters schema).
type ToolSpec struct {
	Type     string        `json:"type"` // "function"
	Function *FunctionSpec `json:"function,omitempty"`
}

// FunctionSpec describes a callable function for the model.
type FunctionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ResponseResponse is the non-streaming response from the API.
type ResponseResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents one completion choice (message or tool calls).
type Choice struct {
	Message      *Message `json:"message,omitempty"`
	Delta        *Message `json:"delta,omitempty"`
	FinishReason string   `json:"finish_reason,omitempty"`
	Index        int      `json:"index,omitempty"`
}

// Usage is token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

package compatible

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	openai "github.com/NikolayNam/collabsphere/platform-tooling/internal/llmclient/openai"
)

// Client implements openai.Client for OpenAI-compatible backends (Ollama, vLLM, LM Studio).
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// CompatibleRequest is the request format for /v1/chat/completions.
type CompatibleRequest struct {
	Model       string            `json:"model"`
	Messages    []openai.Message  `json:"messages"`
	Tools       []openai.ToolSpec `json:"tools,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
	Seed        *int64            `json:"seed,omitempty"`
	TopP        *float64          `json:"top_p,omitempty"`
}

// CompatibleResponse is the response format from /v1/chat/completions.
type CompatibleResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      *openai.Message `json:"message,omitempty"`
		FinishReason string          `json:"finish_reason,omitempty"`
		Index        int             `json:"index,omitempty"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// NewClient creates a client for OpenAI-compatible endpoints.
// baseURL: e.g. http://localhost:11434 (Ollama), http://localhost:1234/v1 (LM Studio).
// apiKey: optional; Ollama ignores it, some backends require it.
func NewClient(baseURL, apiKey, model string, timeout time.Duration) (*Client, error) {
	base := normalizeBaseURL(baseURL)
	if base == "" {
		return nil, fmt.Errorf("llm compatible: base URL is required")
	}
	if model == "" {
		model = "llama3.2"
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		baseURL:    base,
		apiKey:     strings.TrimSpace(apiKey),
		model:      model,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

var _ openai.Client = (*Client)(nil)

// CreateResponse calls /v1/chat/completions and returns the response in openai format.
func (c *Client) CreateResponse(ctx context.Context, req *openai.ResponseRequest) (*openai.ResponseResponse, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}
	body := CompatibleRequest{
		Model:       model,
		Messages:    req.Messages,
		Tools:       req.Tools,
		Stream:      false,
		Temperature: req.Temperature,
		Seed:        req.Seed,
		TopP:        req.TopP,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("llm compatible: marshal request: %w", err)
	}

	url := strings.TrimSuffix(c.baseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("llm compatible: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm compatible: request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("llm compatible: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var compat CompatibleResponse
	if err := json.NewDecoder(resp.Body).Decode(&compat); err != nil {
		return nil, fmt.Errorf("llm compatible: decode response: %w", err)
	}

	out := &openai.ResponseResponse{ID: compat.ID}
	if len(compat.Choices) > 0 {
		ch := compat.Choices[0]
		out.Choices = []openai.Choice{{
			Message:      ch.Message,
			FinishReason: ch.FinishReason,
			Index:        ch.Index,
		}}
	}
	if compat.Usage != nil {
		out.Usage = &openai.Usage{
			PromptTokens:     compat.Usage.PromptTokens,
			CompletionTokens: compat.Usage.CompletionTokens,
			TotalTokens:      compat.Usage.TotalTokens,
		}
	}
	return out, nil
}

// CreateResponseStream returns a non-streaming fallback (reads full response then yields).
// Full streaming support can be added later for SSE.
func (c *Client) CreateResponseStream(ctx context.Context, req *openai.ResponseRequest) (openai.ResponseStream, error) {
	resp, err := c.CreateResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	content := ""
	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		content = resp.Choices[0].Message.Content
	}
	return &compatibleStream{content: content, sent: false}, nil
}

type compatibleStream struct {
	content string
	sent    bool
}

func (s *compatibleStream) Recv() (*openai.ResponseChunk, error) {
	if s.sent {
		return &openai.ResponseChunk{Done: true}, nil
	}
	s.sent = true
	return &openai.ResponseChunk{ContentDelta: s.content, Done: false}, nil
}

func (s *compatibleStream) Close() error { return nil }

func normalizeBaseURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	switch {
	case strings.HasSuffix(base, "/chat/completions"):
		base = strings.TrimSuffix(base, "/chat/completions")
	case strings.HasSuffix(base, "/responses"):
		base = strings.TrimSuffix(base, "/responses")
	}
	base = strings.TrimRight(base, "/")
	if strings.HasSuffix(base, "/v1") {
		base = strings.TrimSuffix(base, "/v1")
	}
	return strings.TrimRight(base, "/")
}

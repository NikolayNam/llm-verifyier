package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOpenAIBaseURL = "https://api.openai.com"

// HTTPClient implements direct OpenAI API transport for chat completions with tool calling.
// The internal request/response model remains normalized so callers do not depend on vendor payloads.
type HTTPClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

type chatCompletionsRequest struct {
	Model       string     `json:"model"`
	Messages    []Message  `json:"messages"`
	Tools       []ToolSpec `json:"tools,omitempty"`
	Stream      bool       `json:"stream,omitempty"`
	Temperature *float64   `json:"temperature,omitempty"`
	Seed        *int64     `json:"seed,omitempty"`
	TopP        *float64   `json:"top_p,omitempty"`
}

type chatCompletionsResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

func NewHTTPClient(baseURL, apiKey, model string, timeout time.Duration) (*HTTPClient, error) {
	normalized, err := normalizeOpenAIBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-4.1-mini"
	}
	return &HTTPClient{
		baseURL:    normalized,
		apiKey:     strings.TrimSpace(apiKey),
		model:      strings.TrimSpace(model),
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

var _ Client = (*HTTPClient)(nil)

func (c *HTTPClient) CreateResponse(ctx context.Context, req *ResponseRequest) (*ResponseResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("openai http client is nil")
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, fmt.Errorf("openai api key is required")
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	body := chatCompletionsRequest{
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
		return nil, fmt.Errorf("openai http client: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("openai http client: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai http client: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("openai http client: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var out chatCompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("openai http client: decode response: %w", err)
	}
	return &ResponseResponse{
		ID:      out.ID,
		Choices: out.Choices,
		Usage:   out.Usage,
	}, nil
}

func (c *HTTPClient) CreateResponseStream(ctx context.Context, req *ResponseRequest) (ResponseStream, error) {
	resp, err := c.CreateResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	content := ""
	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		content = resp.Choices[0].Message.Content
	}
	return &httpFallbackStream{content: content}, nil
}

type httpFallbackStream struct {
	content string
	sent    bool
}

func (s *httpFallbackStream) Recv() (*ResponseChunk, error) {
	if s.sent {
		return &ResponseChunk{Done: true}, nil
	}
	s.sent = true
	return &ResponseChunk{ContentDelta: s.content, Done: false}, nil
}

func (s *httpFallbackStream) Close() error { return nil }

func normalizeOpenAIBaseURL(raw string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		base = defaultOpenAIBaseURL
	}
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
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return "", fmt.Errorf("openai http client: base URL must be absolute")
	}
	return base, nil
}

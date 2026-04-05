package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	researchsampling "github.com/NikolayNam/collabsphere/platform-tooling/internal/research/sampling"
)

const (
	benchmarkGoogleBaseURL         = "https://generativelanguage.googleapis.com/v1beta"
	benchmarkMistralBaseURL        = "https://api.mistral.ai"
	googleBenchmarkMax429Retries   = 2
	googleBenchmarkDefaultBackoff  = 5 * time.Second
	mistralBenchmarkMax429Retries  = 2
	mistralBenchmarkDefaultBackoff = time.Second
)

func normalizeBenchmarkProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return "compatible"
	}
	return provider
}

func normalizeBenchmarkProviderLabel(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}

func benchmarkDisplayProvider(provider, providerLabel string) string {
	if label := normalizeBenchmarkProviderLabel(providerLabel); label != "" {
		return label
	}
	return normalizeBenchmarkProvider(provider)
}

func benchmarkDisplayModelLabel(defaultLabel, providerLabel, llmModel string) string {
	if label := normalizeBenchmarkProviderLabel(providerLabel); label != "" && strings.TrimSpace(llmModel) != "" {
		return label + ":" + strings.TrimSpace(llmModel)
	}
	return strings.TrimSpace(defaultLabel)
}

func deriveBenchmarkProvider(modelRef string) string {
	modelRef = strings.TrimSpace(modelRef)
	if modelRef == "" {
		return ""
	}
	parts := strings.SplitN(modelRef, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return normalizeBenchmarkProvider(parts[0])
}

func benchmarkPromptHash(systemPrompt, userPrompt string) string {
	sum := sha256.Sum256([]byte(systemPrompt + "\n\n" + userPrompt))
	return hex.EncodeToString(sum[:])
}

type googleBenchmarkModel struct {
	baseURL        string
	apiKey         string
	model          string
	temperature    *float64
	thinkingConfig *googleThinkingConfig
	minRequestGap  time.Duration
	requestMu      sync.Mutex
	lastRequestAt  time.Time
	httpClient     *http.Client
}

type googleGenerateContentRequest struct {
	SystemInstruction *googleContent          `json:"systemInstruction,omitempty"`
	Contents          []googleContent         `json:"contents"`
	GenerationConfig  *googleGenerationConfig `json:"generationConfig,omitempty"`
}

type googleGenerationConfig struct {
	Temperature    *float64              `json:"temperature,omitempty"`
	ThinkingConfig *googleThinkingConfig `json:"thinkingConfig,omitempty"`
}

type googleThinkingConfig struct {
	ThinkingBudget *int   `json:"thinkingBudget,omitempty"`
	ThinkingLevel  string `json:"thinkingLevel,omitempty"`
}

type googleContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []googlePart `json:"parts"`
}

type googlePart struct {
	Text string `json:"text,omitempty"`
}

type googleGenerateContentResponse struct {
	Candidates []struct {
		Content googleContent `json:"content"`
	} `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback,omitempty"`
}

type googleErrorResponse struct {
	Error *struct {
		Code    int              `json:"code"`
		Message string           `json:"message"`
		Status  string           `json:"status"`
		Details []map[string]any `json:"details"`
	} `json:"error,omitempty"`
}

func normalizeGoogleThinkingMode(mode string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "", "default":
		return "default", nil
	case "minimal", "low", "high", "budget0":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported google thinking mode %q", mode)
	}
}

func googleThinkingConfigForModel(model, mode string) (*googleThinkingConfig, error) {
	normalizedMode, err := normalizeGoogleThinkingMode(mode)
	if err != nil {
		return nil, err
	}
	if normalizedMode == "default" {
		return nil, nil
	}

	resolvedModel := strings.ToLower(strings.TrimSpace(model))
	if strings.HasPrefix(resolvedModel, "gemini-3") {
		switch normalizedMode {
		case "minimal", "low", "high":
			return &googleThinkingConfig{ThinkingLevel: normalizedMode}, nil
		case "budget0":
			budget := 0
			return &googleThinkingConfig{ThinkingBudget: &budget}, nil
		}
	}

	switch normalizedMode {
	case "budget0":
		budget := 0
		return &googleThinkingConfig{ThinkingBudget: &budget}, nil
	case "minimal", "low":
		// Gemini 2.5 family exposes budget-based controls rather than thinkingLevel.
		budget := 1024
		return &googleThinkingConfig{ThinkingBudget: &budget}, nil
	case "high":
		// Current fast-lane default model is gemini-2.5-flash; use a large explicit budget
		// instead of provider-default dynamic thinking so the run metadata stays unambiguous.
		budget := 24576
		return &googleThinkingConfig{ThinkingBudget: &budget}, nil
	default:
		return nil, fmt.Errorf("unsupported google thinking mode %q", mode)
	}
}

func newGoogleBenchmarkModel(baseURL, apiKey, model, thinkingMode string, requestsPerMinute int, timeout time.Duration, effectiveSampling *researchsampling.Params) (benchmarkModel, string, string, error) {
	resolvedModel := strings.TrimSpace(model)
	if resolvedModel == "" {
		return nil, "", "", fmt.Errorf("google benchmark model requires a non-empty model")
	}
	resolvedAPIKey := strings.TrimSpace(apiKey)
	if resolvedAPIKey == "" {
		return nil, "", "", fmt.Errorf("google benchmark model requires an API key")
	}
	resolvedBaseURL := strings.TrimSpace(baseURL)
	if resolvedBaseURL == "" {
		resolvedBaseURL = benchmarkGoogleBaseURL
	}
	resolvedThinkingMode, err := normalizeGoogleThinkingMode(thinkingMode)
	if err != nil {
		return nil, "", "", err
	}
	thinkingConfig, err := googleThinkingConfigForModel(resolvedModel, resolvedThinkingMode)
	if err != nil {
		return nil, "", "", err
	}
	if requestsPerMinute < 0 {
		return nil, "", "", fmt.Errorf("google benchmark requests_per_minute must be >= 0")
	}
	minRequestGap := time.Duration(0)
	if requestsPerMinute > 0 {
		minRequestGap = time.Minute / time.Duration(requestsPerMinute)
	}
	return &googleBenchmarkModel{
		baseURL:        strings.TrimRight(resolvedBaseURL, "/"),
		apiKey:         resolvedAPIKey,
		model:          resolvedModel,
		temperature:    benchmarkSamplingTemperature(effectiveSampling),
		thinkingConfig: thinkingConfig,
		minRequestGap:  minRequestGap,
		httpClient:     &http.Client{Timeout: timeout},
	}, "google:" + resolvedModel, resolvedModel, nil
}

func (m *googleBenchmarkModel) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, time.Duration, error) {
	temperature := benchmarkSamplingTemperatureOrDefault(m.temperature, 0.0)
	payload := googleGenerateContentRequest{
		SystemInstruction: &googleContent{
			Parts: []googlePart{{Text: systemPrompt}},
		},
		Contents: []googleContent{{
			Role:  "user",
			Parts: []googlePart{{Text: userPrompt}},
		}},
		GenerationConfig: &googleGenerationConfig{
			Temperature:    temperature,
			ThinkingConfig: m.thinkingConfig,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("marshal google benchmark request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", m.baseURL, url.PathEscape(m.model), url.QueryEscape(m.apiKey))
	start := time.Now()
	for attempt := 0; ; attempt++ {
		if err := m.waitForRateLimit(ctx); err != nil {
			return "", time.Since(start), err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return "", time.Since(start), fmt.Errorf("build google benchmark request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := m.httpClient.Do(req)
		if err != nil {
			return "", time.Since(start), fmt.Errorf("google benchmark request failed: %w", err)
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return "", time.Since(start), fmt.Errorf("read google benchmark response: %w", readErr)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if retryDelay, ok := googleRetryDelay(resp.StatusCode, respBody); ok && attempt < googleBenchmarkMax429Retries {
				if err := waitForGoogleRetry(ctx, retryDelay); err != nil {
					return "", time.Since(start), err
				}
				continue
			}
			return "", time.Since(start), fmt.Errorf("google benchmark response status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}

		var decoded googleGenerateContentResponse
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			return "", time.Since(start), fmt.Errorf("decode google benchmark response: %w", err)
		}
		if len(decoded.Candidates) == 0 {
			if decoded.PromptFeedback != nil && strings.TrimSpace(decoded.PromptFeedback.BlockReason) != "" {
				return "", time.Since(start), fmt.Errorf("google benchmark returned no candidates: %s", decoded.PromptFeedback.BlockReason)
			}
			return "", time.Since(start), fmt.Errorf("google benchmark returned no candidates")
		}

		text := strings.TrimSpace(joinGoogleTextParts(decoded.Candidates[0].Content.Parts))
		if text == "" {
			return "", time.Since(start), fmt.Errorf("google benchmark returned empty text")
		}
		return text, time.Since(start), nil
	}
}

func (m *googleBenchmarkModel) waitForRateLimit(ctx context.Context) error {
	if m.minRequestGap <= 0 {
		return nil
	}
	m.requestMu.Lock()
	defer m.requestMu.Unlock()
	if !m.lastRequestAt.IsZero() {
		wait := m.lastRequestAt.Add(m.minRequestGap).Sub(time.Now())
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return fmt.Errorf("google benchmark rate-limit wait canceled: %w", ctx.Err())
			case <-timer.C:
			}
		}
	}
	m.lastRequestAt = time.Now()
	return nil
}

func googleRetryDelay(statusCode int, respBody []byte) (time.Duration, bool) {
	if statusCode != http.StatusTooManyRequests {
		return 0, false
	}
	var decoded googleErrorResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return googleBenchmarkDefaultBackoff, true
	}
	if decoded.Error == nil {
		return googleBenchmarkDefaultBackoff, true
	}
	if status := strings.TrimSpace(decoded.Error.Status); status != "" && status != "RESOURCE_EXHAUSTED" {
		return 0, false
	}
	for _, detail := range decoded.Error.Details {
		retryDelayValue, ok := detail["retryDelay"]
		if !ok {
			continue
		}
		retryDelayText, ok := retryDelayValue.(string)
		if !ok {
			continue
		}
		retryDelayText = strings.TrimSpace(retryDelayText)
		if retryDelayText == "" {
			continue
		}
		retryDelay, err := time.ParseDuration(retryDelayText)
		if err == nil && retryDelay >= 0 {
			return retryDelay, true
		}
	}
	return googleBenchmarkDefaultBackoff, true
}

func waitForGoogleRetry(ctx context.Context, retryDelay time.Duration) error {
	if retryDelay <= 0 {
		return nil
	}
	timer := time.NewTimer(retryDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("google benchmark retry wait canceled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func joinGoogleTextParts(parts []googlePart) string {
	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		if text := strings.TrimSpace(part.Text); text != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "\n")
}

type mistralBenchmarkModel struct {
	baseURL       string
	apiKey        string
	model         string
	temperature   *float64
	minRequestGap time.Duration
	requestMu     sync.Mutex
	lastRequestAt time.Time
	httpClient    *http.Client
}

type mistralChatCompletionRequest struct {
	Model          string                     `json:"model"`
	Messages       []mistralChatMessage       `json:"messages"`
	Temperature    *float64                   `json:"temperature,omitempty"`
	ResponseFormat *mistralChatResponseFormat `json:"response_format,omitempty"`
}

type mistralChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralChatResponseFormat struct {
	Type string `json:"type"`
}

type mistralChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type mistralErrorResponse struct {
	Object        string `json:"object"`
	Message       string `json:"message"`
	Type          string `json:"type"`
	Param         any    `json:"param"`
	Code          string `json:"code"`
	RawStatusCode int    `json:"raw_status_code"`
}

func newMistralBenchmarkModel(baseURL, apiKey, model string, requestsPerSecond int, timeout time.Duration, effectiveSampling *researchsampling.Params) (benchmarkModel, string, string, error) {
	resolvedModel := strings.TrimSpace(model)
	if resolvedModel == "" {
		return nil, "", "", fmt.Errorf("mistral benchmark model requires a non-empty model")
	}
	resolvedAPIKey := strings.TrimSpace(apiKey)
	if resolvedAPIKey == "" {
		return nil, "", "", fmt.Errorf("mistral benchmark model requires an API key")
	}
	resolvedBaseURL := strings.TrimSpace(baseURL)
	if resolvedBaseURL == "" {
		resolvedBaseURL = benchmarkMistralBaseURL
	}
	if requestsPerSecond < 0 {
		return nil, "", "", fmt.Errorf("mistral benchmark requests_per_second must be >= 0")
	}
	minRequestGap := time.Duration(0)
	if requestsPerSecond > 0 {
		minRequestGap = time.Second / time.Duration(requestsPerSecond)
	}
	return &mistralBenchmarkModel{
		baseURL:       strings.TrimRight(resolvedBaseURL, "/"),
		apiKey:        resolvedAPIKey,
		model:         resolvedModel,
		temperature:   benchmarkSamplingTemperature(effectiveSampling),
		minRequestGap: minRequestGap,
		httpClient:    &http.Client{Timeout: timeout},
	}, "mistral:" + resolvedModel, resolvedModel, nil
}

func (m *mistralBenchmarkModel) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, time.Duration, error) {
	temperature := benchmarkSamplingTemperatureOrDefault(m.temperature, 0.0)
	payload := mistralChatCompletionRequest{
		Model: m.model,
		Messages: []mistralChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature:    temperature,
		ResponseFormat: &mistralChatResponseFormat{Type: "text"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("marshal mistral benchmark request: %w", err)
	}

	endpoint := m.baseURL + "/v1/chat/completions"
	start := time.Now()
	for attempt := 0; ; attempt++ {
		if err := m.waitForRateLimit(ctx); err != nil {
			return "", time.Since(start), err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return "", time.Since(start), fmt.Errorf("build mistral benchmark request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := m.httpClient.Do(req)
		latency := time.Since(start)
		if err != nil {
			return "", latency, fmt.Errorf("mistral benchmark request failed: %w", err)
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return "", latency, fmt.Errorf("read mistral benchmark response: %w", readErr)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if retryDelay, ok := mistralRetryDelay(resp.StatusCode, respBody); ok && attempt < mistralBenchmarkMax429Retries {
				if err := waitForMistralRetry(ctx, retryDelay); err != nil {
					return "", time.Since(start), err
				}
				continue
			}
			return "", latency, fmt.Errorf("mistral benchmark response status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}

		var decoded mistralChatCompletionResponse
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			return "", latency, fmt.Errorf("decode mistral benchmark response: %w", err)
		}
		if len(decoded.Choices) == 0 {
			return "", latency, fmt.Errorf("mistral benchmark returned no choices")
		}

		text := strings.TrimSpace(stringifyMistralContent(decoded.Choices[0].Message.Content))
		if text == "" {
			return "", latency, fmt.Errorf("mistral benchmark returned empty content")
		}
		return text, latency, nil
	}
}

func (m *mistralBenchmarkModel) waitForRateLimit(ctx context.Context) error {
	if m.minRequestGap <= 0 {
		return nil
	}
	m.requestMu.Lock()
	defer m.requestMu.Unlock()
	if !m.lastRequestAt.IsZero() {
		wait := m.lastRequestAt.Add(m.minRequestGap).Sub(time.Now())
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return fmt.Errorf("mistral benchmark rate-limit wait canceled: %w", ctx.Err())
			case <-timer.C:
			}
		}
	}
	m.lastRequestAt = time.Now()
	return nil
}

func mistralRetryDelay(statusCode int, respBody []byte) (time.Duration, bool) {
	if statusCode != http.StatusTooManyRequests {
		return 0, false
	}
	var decoded mistralErrorResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return mistralBenchmarkDefaultBackoff, true
	}
	if status := strings.TrimSpace(decoded.Type); status != "" && status != "rate_limited" {
		return 0, false
	}
	return mistralBenchmarkDefaultBackoff, true
}

func waitForMistralRetry(ctx context.Context, retryDelay time.Duration) error {
	if retryDelay <= 0 {
		return nil
	}
	timer := time.NewTimer(retryDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("mistral benchmark retry wait canceled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func stringifyMistralContent(content any) string {
	switch typed := content.(type) {
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if part := stringifyMistralContent(item); strings.TrimSpace(part) != "" {
				parts = append(parts, part)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		if text, ok := typed["text"].(string); ok {
			return text
		}
		if nested, ok := typed["content"]; ok {
			return stringifyMistralContent(nested)
		}
	}
	return ""
}

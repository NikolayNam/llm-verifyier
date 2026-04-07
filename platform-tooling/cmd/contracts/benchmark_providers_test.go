package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBenchmarkPromptHashStable(t *testing.T) {
	first := benchmarkPromptHash("system", "user")
	second := benchmarkPromptHash("system", "user")
	third := benchmarkPromptHash("system", "other")

	if first == "" {
		t.Fatalf("benchmarkPromptHash() returned empty hash")
	}
	if first != second {
		t.Fatalf("benchmarkPromptHash() should be stable, got %q and %q", first, second)
	}
	if first == third {
		t.Fatalf("benchmarkPromptHash() should change when prompt changes")
	}
}

func TestBenchmarkDisplayProviderAndModelLabel(t *testing.T) {
	if got := benchmarkDisplayProvider("compatible", "openai"); got != "openai" {
		t.Fatalf("benchmarkDisplayProvider() = %q, want openai", got)
	}
	if got := benchmarkDisplayProvider("compatible", ""); got != "compatible" {
		t.Fatalf("benchmarkDisplayProvider() fallback = %q, want compatible", got)
	}
	if got := benchmarkDisplayModelLabel("compatible:gpt-oss:20b", "openai", "gpt-oss:20b"); got != "openai:gpt-oss:20b" {
		t.Fatalf("benchmarkDisplayModelLabel() = %q, want openai:gpt-oss:20b", got)
	}
	if got := benchmarkDisplayModelLabel("compatible:gpt-oss:20b", "", "gpt-oss:20b"); got != "compatible:gpt-oss:20b" {
		t.Fatalf("benchmarkDisplayModelLabel() fallback = %q, want compatible:gpt-oss:20b", got)
	}
	if got := benchmarkDisplayModelLabel("compatible:gpt-oss:20b", "", "gpt-oss:20b-cloud"); got != "compatible:gpt-oss:20b-cloud" {
		t.Fatalf("benchmarkDisplayModelLabel() canonical fallback = %q, want compatible:gpt-oss:20b-cloud", got)
	}
	if got := resolveCanonicalBenchmarkModel("gpt-oss:20b-cloud", "gpt-oss:20b"); got != "gpt-oss:20b-cloud" {
		t.Fatalf("resolveCanonicalBenchmarkModel() explicit = %q, want gpt-oss:20b-cloud", got)
	}
	if got := resolveCanonicalBenchmarkModel("", "gpt-oss:20b"); got != "gpt-oss:20b" {
		t.Fatalf("resolveCanonicalBenchmarkModel() runtime fallback = %q, want gpt-oss:20b", got)
	}
}

func TestGoogleBenchmarkModelGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/models/gemini-test:generateContent" {
			t.Fatalf("path = %s, want /models/gemini-test:generateContent", got)
		}
		if got := r.URL.Query().Get("key"); got != "google-test-key" {
			t.Fatalf("query key = %q, want google-test-key", got)
		}

		var payload googleGenerateContentRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.SystemInstruction == nil || len(payload.SystemInstruction.Parts) != 1 || payload.SystemInstruction.Parts[0].Text != "system prompt" {
			t.Fatalf("unexpected systemInstruction payload: %#v", payload.SystemInstruction)
		}
		if len(payload.Contents) != 1 || len(payload.Contents[0].Parts) != 1 || payload.Contents[0].Parts[0].Text != "user prompt" {
			t.Fatalf("unexpected contents payload: %#v", payload.Contents)
		}
		if payload.GenerationConfig == nil || payload.GenerationConfig.Temperature == nil || *payload.GenerationConfig.Temperature != 0 {
			t.Fatalf("unexpected generationConfig temperature: %#v", payload.GenerationConfig)
		}
		if payload.GenerationConfig.ThinkingConfig == nil || payload.GenerationConfig.ThinkingConfig.ThinkingBudget == nil || *payload.GenerationConfig.ThinkingConfig.ThinkingBudget != 0 {
			t.Fatalf("unexpected thinkingConfig for budget0: %#v", payload.GenerationConfig.ThinkingConfig)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"NOT_DERIVABLE"}]}}]}`))
	}))
	defer server.Close()

	model, _, _, err := newGoogleBenchmarkModel(server.URL, "google-test-key", "gemini-test", "budget0", 0, 5*time.Second, nil)
	if err != nil {
		t.Fatalf("newGoogleBenchmarkModel() error = %v", err)
	}

	output, _, err := model.Generate(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if output != "NOT_DERIVABLE" {
		t.Fatalf("output = %q, want NOT_DERIVABLE", output)
	}
}

func TestGoogleBenchmarkModelGenerateRetriesOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{
  "error": {
    "code": 429,
    "message": "quota exceeded",
    "status": "RESOURCE_EXHAUSTED",
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.RetryInfo",
        "retryDelay": "0s"
      }
    ]
  }
}`))
			return
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"DERIVABLE"}]}}]}`))
	}))
	defer server.Close()

	model, _, _, err := newGoogleBenchmarkModel(server.URL, "google-test-key", "gemini-test", "default", 0, 5*time.Second, nil)
	if err != nil {
		t.Fatalf("newGoogleBenchmarkModel() error = %v", err)
	}

	output, _, err := model.Generate(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if output != "DERIVABLE" {
		t.Fatalf("output = %q, want DERIVABLE", output)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestGoogleThinkingConfigForModel(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		mode          string
		wantNil       bool
		wantLevel     string
		wantBudget    int
		wantHasBudget bool
	}{
		{name: "default", model: "gemini-2.5-flash", mode: "default", wantNil: true},
		{name: "gemini3 minimal", model: "gemini-3-flash-preview", mode: "minimal", wantLevel: "minimal"},
		{name: "gemini25 low", model: "gemini-2.5-flash", mode: "low", wantBudget: 1024, wantHasBudget: true},
		{name: "gemini25 high", model: "gemini-2.5-flash", mode: "high", wantBudget: 24576, wantHasBudget: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := googleThinkingConfigForModel(tc.model, tc.mode)
			if err != nil {
				t.Fatalf("googleThinkingConfigForModel() error = %v", err)
			}
			if tc.wantNil {
				if cfg != nil {
					t.Fatalf("googleThinkingConfigForModel() = %#v, want nil", cfg)
				}
				return
			}
			if cfg == nil {
				t.Fatalf("googleThinkingConfigForModel() returned nil")
			}
			if got := cfg.ThinkingLevel; got != tc.wantLevel {
				t.Fatalf("thinkingLevel = %q, want %q", got, tc.wantLevel)
			}
			if tc.wantHasBudget {
				if cfg.ThinkingBudget == nil {
					t.Fatalf("thinkingBudget = nil, want %d", tc.wantBudget)
				}
				if *cfg.ThinkingBudget != tc.wantBudget {
					t.Fatalf("thinkingBudget = %d, want %d", *cfg.ThinkingBudget, tc.wantBudget)
				}
			}
		})
	}
}

func TestMistralBenchmarkModelGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/v1/chat/completions" {
			t.Fatalf("path = %s, want /v1/chat/completions", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer mistral-test-key" {
			t.Fatalf("authorization = %q, want Bearer mistral-test-key", got)
		}

		var payload mistralChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != "mistral-test" {
			t.Fatalf("model = %q, want mistral-test", payload.Model)
		}
		if len(payload.Messages) != 2 {
			t.Fatalf("messages len = %d, want 2", len(payload.Messages))
		}
		if payload.Messages[0].Role != "system" || payload.Messages[0].Content != "system prompt" {
			t.Fatalf("unexpected system message: %#v", payload.Messages[0])
		}
		if payload.Messages[1].Role != "user" || payload.Messages[1].Content != "user prompt" {
			t.Fatalf("unexpected user message: %#v", payload.Messages[1])
		}
		if payload.ResponseFormat == nil || payload.ResponseFormat.Type != "text" {
			t.Fatalf("unexpected response_format: %#v", payload.ResponseFormat)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer server.Close()

	model, _, _, err := newMistralBenchmarkModel(server.URL, "mistral-test-key", "mistral-test", 0, 5*time.Second, nil)
	if err != nil {
		t.Fatalf("newMistralBenchmarkModel() error = %v", err)
	}

	output, _, err := model.Generate(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if strings.TrimSpace(output) != `{"ok":true}` {
		t.Fatalf("output = %q, want JSON text payload", output)
	}
}

func TestMistralBenchmarkModelGenerateRetriesOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"object":"error","message":"Rate limit exceeded","type":"rate_limited","code":"1300","raw_status_code":429}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer server.Close()

	model, _, _, err := newMistralBenchmarkModel(server.URL, "mistral-test-key", "mistral-test", 0, 5*time.Second, nil)
	if err != nil {
		t.Fatalf("newMistralBenchmarkModel() error = %v", err)
	}

	output, _, err := model.Generate(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if strings.TrimSpace(output) != `{"ok":true}` {
		t.Fatalf("output = %q, want JSON text payload", output)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

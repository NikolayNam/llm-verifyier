package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
)

func TestVerifyCompatibleModelPreflightRejectsMissingModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-oss:20b"},{"id":"glm-5:cloud"}]}`))
	}))
	defer server.Close()

	plan := &planner.BenchmarkPlan{
		Jobs: []planner.BenchmarkJob{
			{Key: "gpt", Provider: "compatible", BaseURL: server.URL, Model: "gpt-oss:120b-cloud", RuntimeModel: "gpt-oss:120b-cloud"},
			{Key: "deepseek", Provider: "compatible", BaseURL: server.URL, Model: "deepseek-v3.1:671b-cloud", RuntimeModel: "deepseek-v3.1:671b-cloud"},
			{Key: "alias", Provider: "compatible", BaseURL: server.URL, Model: "gpt-oss:20b-cloud", RuntimeModel: "gpt-oss:20b"},
		},
	}

	err := verifyCompatibleModelPreflight(context.Background(), plan)
	if err == nil {
		t.Fatalf("verifyCompatibleModelPreflight() error = nil, want missing-model error")
	}
	text := err.Error()
	for _, want := range []string{
		"compatible model preflight failed",
		"gpt-oss:120b-cloud",
		"deepseek-v3.1:671b-cloud",
		"runtime mappings=[deepseek-v3.1:671b-cloud -> deepseek-v3.1:671b-cloud gpt-oss:120b-cloud -> gpt-oss:120b-cloud gpt-oss:20b-cloud -> gpt-oss:20b]",
		"missing runtime models=[deepseek-v3.1:671b-cloud gpt-oss:120b-cloud]",
		"gpt-oss:20b-cloud -> gpt-oss:20b",
		"glm-5:cloud",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("verifyCompatibleModelPreflight() error = %q, want substring %q", text, want)
		}
	}
}

func TestVerifyCompatibleModelPreflightAllowsConfiguredModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-oss:20b"},{"id":"deepseek-v3.1:671b-cloud"}]}`))
	}))
	defer server.Close()

	plan := &planner.BenchmarkPlan{
		Jobs: []planner.BenchmarkJob{
			{Key: "gpt", Provider: "compatible", BaseURL: server.URL, Model: "gpt-oss:20b-cloud", RuntimeModel: "gpt-oss:20b"},
			{Key: "deepseek", Provider: "compatible", BaseURL: server.URL, Model: "deepseek-v3.1:671b-cloud", RuntimeModel: "deepseek-v3.1:671b-cloud"},
		},
	}

	if err := verifyCompatibleModelPreflight(context.Background(), plan); err != nil {
		t.Fatalf("verifyCompatibleModelPreflight() error = %v", err)
	}
}

func TestVerifyCompatibleModelPreflightIgnoresNonCompatibleProviders(t *testing.T) {
	plan := &planner.BenchmarkPlan{
		Jobs: []planner.BenchmarkJob{
			{Key: "openai", Provider: "openai", BaseURL: "https://api.openai.com/v1", Model: "gpt-4.1"},
		},
	}

	if err := verifyCompatibleModelPreflight(context.Background(), plan); err != nil {
		t.Fatalf("verifyCompatibleModelPreflight() error = %v", err)
	}
}

func TestVerifyCompatibleModelPreflightRejectsConflictingRuntimeMappings(t *testing.T) {
	plan := &planner.BenchmarkPlan{
		Jobs: []planner.BenchmarkJob{
			{Key: "one", Provider: "compatible", BaseURL: "http://localhost:11434", Model: "gpt-oss:20b-cloud", RuntimeModel: "gpt-oss:20b"},
			{Key: "two", Provider: "compatible", BaseURL: "http://localhost:11434", Model: "gpt-oss:20b-cloud", RuntimeModel: "gpt-oss:120b-cloud"},
		},
	}

	err := verifyCompatibleModelPreflight(context.Background(), plan)
	if err == nil {
		t.Fatalf("verifyCompatibleModelPreflight() error = nil, want conflicting runtime mapping error")
	}
	if got := err.Error(); !strings.Contains(got, "conflicting runtime mappings") {
		t.Fatalf("verifyCompatibleModelPreflight() error = %q, want conflicting runtime mappings", got)
	}
}

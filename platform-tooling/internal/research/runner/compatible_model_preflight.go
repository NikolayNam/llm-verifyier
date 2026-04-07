package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/research/planner"
)

type compatibleModelCatalogResponse struct {
	Data []compatibleModelCatalogEntry `json:"data"`
}

type compatibleModelCatalogEntry struct {
	ID string `json:"id"`
}

func verifyCompatibleModelPreflight(ctx context.Context, plan *planner.BenchmarkPlan) error {
	if plan == nil {
		return fmt.Errorf("benchmark plan is required for compatible model preflight")
	}

	type requestedModel struct {
		canonical string
		runtime   string
	}
	modelsByBaseURL := make(map[string]map[string]requestedModel)
	for _, job := range plan.Jobs {
		if strings.ToLower(strings.TrimSpace(job.Provider)) != "compatible" {
			continue
		}
		baseURL := normalizeCompatibleBaseURL(job.BaseURL)
		if baseURL == "" {
			return fmt.Errorf("compatible model preflight requires base_url for job %q", job.Key)
		}
		canonicalModel := strings.TrimSpace(job.Model)
		if canonicalModel == "" {
			return fmt.Errorf("compatible model preflight requires model for job %q", job.Key)
		}
		runtimeModel := strings.TrimSpace(job.RuntimeModel)
		if runtimeModel == "" {
			runtimeModel = canonicalModel
		}
		if modelsByBaseURL[baseURL] == nil {
			modelsByBaseURL[baseURL] = make(map[string]requestedModel)
		}
		if existing, ok := modelsByBaseURL[baseURL][canonicalModel]; ok && existing.runtime != runtimeModel {
			return fmt.Errorf("compatible model preflight found conflicting runtime mappings for %s on %s: %s -> %s and %s -> %s", canonicalModel, baseURL, existing.canonical, existing.runtime, canonicalModel, runtimeModel)
		}
		modelsByBaseURL[baseURL][canonicalModel] = requestedModel{canonical: canonicalModel, runtime: runtimeModel}
	}
	if len(modelsByBaseURL) == 0 {
		return nil
	}

	for baseURL, requestedSet := range modelsByBaseURL {
		available, err := compatibleModelCatalogFetcher(ctx, baseURL)
		if err != nil {
			return err
		}
		available = sortUniqueStrings(available)
		availableSet := make(map[string]struct{}, len(available))
		for _, model := range available {
			availableSet[model] = struct{}{}
		}

		requested := make([]string, 0, len(requestedSet))
		runtimeMappings := make([]string, 0, len(requestedSet))
		missingRuntime := make([]string, 0, len(requestedSet))
		aliasHints := make([]string, 0)
		for canonical, requestedModel := range requestedSet {
			requested = append(requested, canonical)
			runtimeMappings = append(runtimeMappings, fmt.Sprintf("%s -> %s", requestedModel.canonical, requestedModel.runtime))
			if _, ok := availableSet[requestedModel.runtime]; ok {
				continue
			}
			missingRuntime = append(missingRuntime, requestedModel.runtime)
			if alias := findCompatibleAliasHint(canonical, availableSet); alias != "" {
				aliasHints = append(aliasHints, fmt.Sprintf("%s -> %s", canonical, alias))
			}
		}
		if len(missingRuntime) == 0 {
			continue
		}

		sort.Strings(requested)
		runtimeMappings = sortUniqueStrings(runtimeMappings)
		missingRuntime = sortUniqueStrings(missingRuntime)
		sort.Strings(aliasHints)
		errText := fmt.Sprintf(
			"compatible model preflight failed for %s: configured models=%v; runtime mappings=%v; missing runtime models=%v; available models=%v",
			baseURL,
			requested,
			runtimeMappings,
			missingRuntime,
			available,
		)
		if len(aliasHints) > 0 {
			errText += fmt.Sprintf("; alias candidates=%v", aliasHints)
		}
		return fmt.Errorf("%s", errText)
	}
	return nil
}

func fetchCompatibleModelCatalog(ctx context.Context, baseURL string) ([]string, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("compatible model preflight request %q: %w", endpoint, err)
	}
	resp, err := compatibleModelCatalogHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("compatible model preflight GET %q: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		bodyText := strings.TrimSpace(string(body))
		if bodyText == "" {
			return nil, fmt.Errorf("compatible model preflight GET %q returned HTTP %d", endpoint, resp.StatusCode)
		}
		return nil, fmt.Errorf("compatible model preflight GET %q returned HTTP %d: %s", endpoint, resp.StatusCode, bodyText)
	}

	var payload compatibleModelCatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode compatible model catalog from %q: %w", endpoint, err)
	}

	models := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		model := strings.TrimSpace(item.ID)
		if model == "" {
			continue
		}
		models = append(models, model)
	}
	return sortUniqueStrings(models), nil
}

func normalizeCompatibleBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func findCompatibleAliasHint(requested string, available map[string]struct{}) string {
	if !strings.HasSuffix(requested, "-cloud") {
		return ""
	}
	candidate := strings.TrimSuffix(requested, "-cloud")
	if _, ok := available[candidate]; ok {
		return candidate
	}
	return ""
}

func sortUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	sort.Strings(deduped)
	return deduped
}

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/ai"
	"github.com/felixgeelhaar/roady/pkg/domain/drift"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

type AIPlanningService struct {
	repo     domain.WorkspaceRepository
	provider ai.Provider
	audit    domain.AuditLogger
	planSvc  *PlanService
}

const taskSchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "array",
  "items": {
    "type": "object",
    "required": ["id", "feature_id"],
    "properties": {
      "id": { "type": "string" },
      "feature_id": { "type": "string" },
      "title": { "type": "string" },
      "description": { "type": "string" }
    },
    "anyOf": [
      { "required": ["title"] },
      { "required": ["description"] }
    ]
  }
}`

var (
	taskSchemaLoader = gojsonschema.NewStringLoader(taskSchemaJSON)
)

func NewAIPlanningService(repo domain.WorkspaceRepository, provider ai.Provider, audit domain.AuditLogger, planSvc *PlanService) *AIPlanningService {
	return &AIPlanningService{repo: repo, provider: provider, audit: audit, planSvc: planSvc}
}

// GetAuditLogger returns the audit logger used by this service.
func (s *AIPlanningService) GetAuditLogger() domain.AuditLogger {
	return s.audit
}

func (s *AIPlanningService) DecomposeSpec(ctx context.Context) (*planning.Plan, error) {
	// 1. Check Policy & Budget
	cfg, err := s.repo.LoadPolicy()
	if err != nil {
		return nil, err
	}
	if !cfg.AllowAI {
		return nil, fmt.Errorf("AI usage is disabled by project policy")
	}

	if cfg.TokenLimit > 0 {
		stats, _ := s.repo.LoadUsage()
		if stats != nil {
			totalTokens := 0
			for _, count := range stats.ProviderStats {
				totalTokens += count
			}
			if totalTokens >= cfg.TokenLimit {
				return nil, fmt.Errorf("AI token limit reached (%d/%d). Please increase limit in policy.yaml", totalTokens, cfg.TokenLimit)
			}
		}
	}

	// 2. Load Spec
	productSpec, err := s.repo.LoadSpec()
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}
	if productSpec == nil {
		return nil, fmt.Errorf("spec is nil")
	}

	// 3. Prompt AI
	prompt := `Task: Decompose the following features into atomic engineering tasks.
Requirement: Every Feature and every Requirement listed below MUST be implemented.
If a Feature has no Requirements, create at least one task for that Feature based on its description.

MAPPING RULES:
1. For each Requirement, create a task with ID: "task-[requirement-id]".
2. For each Feature, ensure at least one task references its Feature ID.
3. Return ONLY a JSON array of tasks with no surrounding text, no markdown, and no code fences.

Format:
Return ONLY a JSON array of task objects with no surrounding text, no markdown, and no code fences.
Do NOT return placeholder values or the schema itself.

Features to decompose:
`

	for _, f := range productSpec.Features {
		prompt += fmt.Sprintf("- Feature: %s (ID: %s)\n", f.Title, f.ID)
		for _, r := range f.Requirements {
			prompt += fmt.Sprintf("  * Requirement: %s (ID: %s)\n", r.Title, r.ID)
		}
	}

	resp, err := s.completeDecomposition(ctx, prompt, 1)
	if err != nil {
		return nil, fmt.Errorf("AI planning failed: %w", err)
	}

	cleanTasks, err := s.parseTasksFromResponse(resp.Text)
	if err != nil {
		_ = s.audit.Log("plan.ai_decomposition_retry", "ai", map[string]interface{}{
			"reason":  err.Error(),
			"attempt": 2,
		})
		retryPrompt := prompt + "\n\nIMPORTANT: Your previous response was invalid. Return ONLY a JSON array of tasks with valid fields. Do not include any extra text."
		respRetry, retryErr := s.completeDecomposition(ctx, retryPrompt, 2)
		if retryErr != nil {
			return nil, fmt.Errorf("AI planning failed after retry: %w", retryErr)
		}
		cleanTasks, err = s.parseTasksFromResponse(respRetry.Text)
		if err != nil {
			return nil, fmt.Errorf("AI returned invalid JSON after retry: %w", err)
		}
	}

	// Coverage Check: Ensure every feature ID has at least one task
	// (Check both task.FeatureID matching feature.ID OR task.ID matching a requirement ID)
	featureCoverage := make(map[string]bool)
	requirementCoverage := make(map[string]bool)
	for _, t := range cleanTasks {
		featureCoverage[t.FeatureID] = true
		requirementCoverage[t.ID] = true
	}

	var missingFeatures []string
	missingFeatureIDs := make(map[string]spec.Feature)
	for _, f := range productSpec.Features {
		// Feature is covered if:
		// 1. A task explicitly references its FeatureID
		// 2. OR at least one of its Requirements has a task matching its ID
		covered := featureCoverage[f.ID]
		if !covered {
			for _, r := range f.Requirements {
				if requirementCoverage[r.ID] || requirementCoverage["task-"+r.ID] {
					covered = true
					break
				}
			}
		}

		if !covered {
			missingFeatures = append(missingFeatures, f.Title)
			missingFeatureIDs[f.ID] = f
		}
	}

	if len(missingFeatures) > 0 {
		fmt.Printf("WARNING: AI missed coverage for features: %s. Proceeding anyway, use 'roady drift detect' to see gaps.\n", strings.Join(missingFeatures, ", "))
		existingIDs := make(map[string]bool)
		for _, t := range cleanTasks {
			existingIDs[t.ID] = true
		}
		for _, f := range productSpec.Features {
			if _, ok := missingFeatureIDs[f.ID]; !ok {
				continue
			}
			fallbackID := "task-" + f.ID
			if existingIDs[fallbackID] {
				continue
			}
			cleanTasks = append(cleanTasks, planning.Task{
				ID:          fallbackID,
				Title:       fmt.Sprintf("Implement %s", f.Title),
				Description: "Fallback task generated because AI response missed feature coverage.",
				FeatureID:   f.ID,
			})
		}
	}

	// Lock the spec content to this new plan to resolve Intent Drift
	if err := s.repo.SaveSpecLock(productSpec); err != nil {
		return nil, fmt.Errorf("save spec lock: %w", err)
	}

	return s.planSvc.UpdatePlan(cleanTasks)
}

func (s *AIPlanningService) completeDecomposition(ctx context.Context, prompt string, attempt int) (*ai.CompletionResponse, error) {
	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		Prompt:      prompt,
		System:      "You are an expert technical lead. You return a JSON array of technical tasks. You ensure that every feature ID provided is represented in the result.",
		Temperature: 0.2,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, err
	}

	if err := s.audit.Log("plan.ai_decomposition", "ai", map[string]interface{}{
		"model":         resp.Model,
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
		"attempt":       attempt,
	}); err != nil {
		return nil, fmt.Errorf("write audit log: %w", err)
	}

	return resp, nil
}

func (s *AIPlanningService) parseTasksFromResponse(text string) ([]planning.Task, error) {
	var tasks []planning.Task
	cleanJSON := extractJSONPayload(text)
	if os.Getenv("ROADY_AI_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "AI raw response: %s\n", text)
		fmt.Fprintf(os.Stderr, "AI extracted JSON: %s\n", cleanJSON)
	}

	// Validate JSON against schema first
	documentLoader := gojsonschema.NewStringLoader(cleanJSON)
	result, err := gojsonschema.Validate(taskSchemaLoader, documentLoader)
	if err == nil && result.Valid() {
		// Schema validation passed - fast path
		if os.Getenv("ROADY_AI_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "AI JSON schema validation passed\n")
		}
	} else if os.Getenv("ROADY_AI_DEBUG") != "" {
		// Log validation issues for debugging
		if err != nil {
			fmt.Fprintf(os.Stderr, "AI JSON schema validation error: %v\n", err)
		} else {
			for _, desc := range result.Errors() {
				fmt.Fprintf(os.Stderr, "AI JSON schema issue: %s\n", desc)
			}
		}
	}
	// Note: We continue even if schema validation fails because we have robust
	// fallback parsing that can handle many non-conforming responses

	// Helper to validate a task has minimal required fields
	isValid := func(t planning.Task) bool {
		return t.ID != "" && (t.Title != "" || t.Description != "")
	}

	// Try direct list unmarshal
	if err := json.Unmarshal([]byte(cleanJSON), &tasks); err == nil && len(tasks) > 0 && isValid(tasks[0]) {
		// Success
	} else {
		// Reset and try map/wrapped objects
		tasks = nil
		var generic map[string]interface{}
		if err := json.Unmarshal([]byte(cleanJSON), &generic); err == nil {
			if errValue, ok := generic["error"]; ok {
				if msg, ok := errValue.(string); ok && msg != "" {
					return nil, fmt.Errorf("AI response error: %s", msg)
				}
				return nil, fmt.Errorf("AI response error: %v", errValue)
			}

			// 0. If the object already matches the task shape, parse it directly.
			if _, hasID := generic["id"]; hasID {
				if _, hasTitle := generic["title"]; hasTitle {
					var single planning.Task
					if err := json.Unmarshal([]byte(cleanJSON), &single); err == nil && isValid(single) {
						tasks = []planning.Task{single}
					}
				}
			}
			if len(tasks) == 0 && (hasAnyKey(generic, "task-id", "task_id", "taskId") || hasAnyKey(generic, "feature-id", "feature_id", "featureId")) {
				tasks = append(tasks, normalizeTaskMap(generic, 0))
			}

			// 1. Check for common wrapper keys (tasks, task, data)
			for _, key := range []string{"tasks", "task", "data"} {
				if sub, ok := generic[key]; ok {
					if list, ok := sub.([]interface{}); ok {
						for i, item := range list {
							itemMap, ok := item.(map[string]interface{})
							if !ok {
								continue
							}
							t := normalizeTaskMap(itemMap, i)
							if isValid(t) {
								tasks = append(tasks, t)
							}
						}
						if len(tasks) > 0 {
							break
						}
					} else {
						subData, _ := json.Marshal(sub)
						var subTasks []planning.Task
						if err := json.Unmarshal(subData, &subTasks); err == nil && len(subTasks) > 0 && isValid(subTasks[0]) {
							tasks = subTasks
							break
						}
					}
				}
			}

			// 2. If still empty, try parsing as a map of tasks
			if len(tasks) == 0 {
				for k, v := range generic {
					itemMap, ok := v.(map[string]interface{})
					if ok {
						t := normalizeTaskMap(itemMap, 0)
						if t.ID == "" {
							t.ID = k
						}
						if isValid(t) {
							tasks = append(tasks, t)
						}
						continue
					}

					itemData, _ := json.Marshal(v)
					var t planning.Task
					if err := json.Unmarshal(itemData, &t); err == nil {
						if t.ID == "" {
							t.ID = k
						}
						if isValid(t) {
							tasks = append(tasks, t)
						}
					}
				}
			}
		} else {
			// Try single object
			var single planning.Task
			if err := json.Unmarshal([]byte(cleanJSON), &single); err == nil && isValid(single) {
				tasks = []planning.Task{single}
			}
		}
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("AI returned no valid tasks in JSON (Response: %s)", text)
	}

	// Final Sanity Check: Filter out hallucinations (empty or malformed tasks)
	var cleanTasks []planning.Task
	for _, t := range tasks {
		if isValid(t) {
			cleanTasks = append(cleanTasks, t)
		}
	}

	if len(cleanTasks) == 0 {
		return nil, fmt.Errorf("AI returned tasks, but none passed the structural sanity check")
	}

	return cleanTasks, nil
}

var slugCleaner = regexp.MustCompile(`[^a-z0-9-]+`)

func normalizeTaskMap(raw map[string]interface{}, index int) planning.Task {
	id := getString(raw, "id", "task-id", "task_id", "taskId")
	title := getString(raw, "title", "name")
	description := getString(raw, "description", "details")
	featureID := getString(raw, "feature_id", "feature-id", "featureId", "feature")
	priority := getString(raw, "priority")
	estimate := getString(raw, "estimate")

	if title == "" && description != "" {
		title = summarizeText(description)
	}
	if title == "" && id != "" {
		title = humanizeID(id)
	}
	if id == "" && title != "" {
		id = "task-" + slugify(title)
	}
	if id == "" {
		id = fmt.Sprintf("task-%d", index+1)
	}

	return planning.Task{
		ID:          id,
		Title:       title,
		Description: description,
		Priority:    planning.TaskPriority(priority),
		Estimate:    estimate,
		FeatureID:   featureID,
	}
}

func getString(raw map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			if str, ok := value.(string); ok {
				return strings.TrimSpace(str)
			}
		}
	}
	return ""
}

func summarizeText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	if idx := strings.Index(trimmed, "."); idx > 0 && idx < 80 {
		return strings.TrimSpace(trimmed[:idx])
	}
	if len(trimmed) > 80 {
		return strings.TrimSpace(trimmed[:80]) + "…"
	}
	return trimmed
}

func humanizeID(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "task-")
	trimmed = strings.ReplaceAll(trimmed, "_", " ")
	trimmed = strings.ReplaceAll(trimmed, "-", " ")
	words := strings.Fields(trimmed)
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func slugify(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = slugCleaner.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	return normalized
}

func hasAnyKey(raw map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if _, ok := raw[key]; ok {
			return true
		}
	}
	return false
}

func extractJSONPayload(text string) string {
	clean := strings.TrimSpace(text)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	if clean == "" {
		return clean
	}

	// If the response includes extra text, attempt to extract the first JSON array/object.
	startArray := strings.Index(clean, "[")
	startObject := strings.Index(clean, "{")
	start := -1
	if startArray == -1 {
		start = startObject
	} else if startObject == -1 || startArray < startObject {
		start = startArray
	} else {
		start = startObject
	}
	if start == -1 {
		return clean
	}

	endArray := strings.LastIndex(clean, "]")
	endObject := strings.LastIndex(clean, "}")
	end := -1
	if endArray == -1 {
		end = endObject
	} else if endObject == -1 || endArray > endObject {
		end = endArray
	} else {
		end = endObject
	}
	if end == -1 || end <= start {
		return clean
	}

	return strings.TrimSpace(clean[start : end+1])
}

func (s *AIPlanningService) ReconcileSpec(ctx context.Context, rawSpec *spec.ProductSpec) (*spec.ProductSpec, error) {
	// 1. Check Policy & Budget
	cfg, err := s.repo.LoadPolicy()
	if err != nil {
		return nil, err
	}
	if !cfg.AllowAI {
		return nil, fmt.Errorf("AI usage is disabled by project policy")
	}

	if cfg.TokenLimit > 0 {
		stats, _ := s.repo.LoadUsage()
		if stats != nil {
			totalTokens := 0
			for _, count := range stats.ProviderStats {
				totalTokens += count
			}
			if totalTokens >= cfg.TokenLimit {
				return nil, fmt.Errorf("AI token limit reached (%d/%d) during reconciliation", totalTokens, cfg.TokenLimit)
			}
		}
	}

	// 2. Prompt AI for semantic merge
	prompt := fmt.Sprintf(`Analyze the following software specification which has been merged from multiple documents.
It contains redundant features, overlapping descriptions, and inconsistent requirements.

TASK:
1. Deduplicate features that refer to the same functional area.
2. Merge descriptions from different "angles" into a single, comprehensive explanation string.
3. Normalize Requirement IDs and titles.
4. Return a single, valid JSON ProductSpec matching the schema below.

SCHEMA:
{
  "id": "project-id",
  "title": "Project Title",
  "description": "General project description",
  "version": "0.1.0",
  "features": [
    {
      "id": "feature-id",
      "title": "Feature Title",
      "description": "DETAILED MERGED DESCRIPTION (MUST BE A STRING, NOT AN OBJECT)",
      "requirements": [
        {
          "id": "req-id",
          "title": "Requirement Title",
          "description": "Requirement detail",
          "priority": "low|medium|high",
          "estimate": "e.g. 4h"
        }
      ]
    }
  ]
}

INPUT SPEC:
Title: %s
Features:
`, rawSpec.Title)

	for _, f := range rawSpec.Features {
		prompt += fmt.Sprintf("- Feature: %s (ID: %s)\n  Description: %s\n", f.Title, f.ID, f.Description)
	}

	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		Prompt: prompt,
		System: "You are a Technical Architect. You take messy, multi-document specifications and reconcile them into a clean, high-integrity ProductSpec JSON. You respond ONLY with the reconciled JSON.",
	})
	if err != nil {
		return nil, fmt.Errorf("AI reconciliation failed: %w", err)
	}

	// 3. Parse Result
	var reconciled spec.ProductSpec
	cleanJSON := strings.TrimSpace(resp.Text)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	if err := json.Unmarshal([]byte(cleanJSON), &reconciled); err != nil {
		return nil, fmt.Errorf("parse reconciled spec: %w", err)
	}

	// 4. Save and Lock
	if err := s.repo.SaveSpec(&reconciled); err != nil {
		return nil, err
	}
	if err := s.repo.SaveSpecLock(&reconciled); err != nil {
		return nil, err
	}

	if err := s.audit.Log("spec.reconcile", "ai", map[string]interface{}{
		"model": resp.Model,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to log audit event: %v\n", err)
	}

	return &reconciled, nil
}

func (s *AIPlanningService) ExplainSpec(ctx context.Context) (string, error) {
	// 1. Check Policy
	cfg, err := s.repo.LoadPolicy()
	if err != nil {
		return "", err
	}
	if !cfg.AllowAI {
		return "", fmt.Errorf("AI usage is disabled by project policy")
	}

	// 2. Load Spec
	spec, err := s.repo.LoadSpec()
	if err != nil {
		return "", err
	}

	// 3. Prompt AI
	prompt := fmt.Sprintf("Provide a high-level architectural walkthrough and explanation of this software specification. "+
		"Explain 'What' we are building and 'Why' based on the features and requirements.\n\nSpec: %s\n\nFeatures:\n", spec.Title)

	for _, f := range spec.Features {
		prompt += fmt.Sprintf("- %s: %s\n", f.Title, f.Description)
		for _, r := range f.Requirements {
			prompt += fmt.Sprintf("  * %s: %s\n", r.Title, r.Description)
		}
	}

	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		Prompt: prompt,
		System: "You are an expert technical lead. Provide a clear, concise, and professional explanation.",
	})
	if err != nil {
		return "", fmt.Errorf("AI explanation failed: %w", err)
	}

	// 4. Log Usage
	if err := s.audit.Log("spec.ai_explanation", "ai", map[string]interface{}{
		"model":         resp.Model,
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to log audit event: %v\n", err)
	}

	return resp.Text, nil
}

func (s *AIPlanningService) ReviewSpec(ctx context.Context) (*spec.SpecReview, error) {
	// 1. Check Policy
	cfg, err := s.repo.LoadPolicy()
	if err != nil {
		return nil, err
	}
	if !cfg.AllowAI {
		return nil, fmt.Errorf("AI usage is disabled by project policy")
	}

	// 2. Load Spec
	productSpec, err := s.repo.LoadSpec()
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}

	// 3. Build Prompt
	prompt := fmt.Sprintf(`Review the following product specification for quality.
Evaluate: completeness, clarity, ambiguity, dependencies, priority balance, and testability.

Return ONLY a JSON object with no surrounding text, no markdown, and no code fences.

Format:
{
  "score": <0-100>,
  "summary": "<brief overall assessment>",
  "findings": [
    {
      "category": "<completeness|clarity|ambiguity|dependency|priority|testability>",
      "severity": "<info|warning|critical>",
      "feature_id": "<feature ID or empty string if spec-level>",
      "title": "<short finding title>",
      "suggestion": "<actionable suggestion>"
    }
  ]
}

Specification: %s
Description: %s
Features:
`, productSpec.Title, productSpec.Description)

	for _, f := range productSpec.Features {
		prompt += fmt.Sprintf("- Feature: %s (ID: %s)\n  Description: %s\n", f.Title, f.ID, f.Description)
		for _, r := range f.Requirements {
			prompt += fmt.Sprintf("  * Requirement: %s (ID: %s, Priority: %s)\n    %s\n", r.Title, r.ID, r.Priority, r.Description)
		}
	}

	// 4. Call AI
	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		Prompt:      prompt,
		System:      "You are an expert technical lead reviewing a product specification for quality. Return structured JSON only.",
		Temperature: 0.2,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("AI review failed: %w", err)
	}

	// 5. Parse Response
	cleanJSON := extractJSONPayload(resp.Text)
	var review spec.SpecReview
	if err := json.Unmarshal([]byte(cleanJSON), &review); err != nil {
		return nil, fmt.Errorf("parse review response: %w", err)
	}

	// 6. Log Usage
	if err := s.audit.Log("spec.ai_review", "ai", map[string]interface{}{
		"model":         resp.Model,
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
		"score":         review.Score,
		"findings":      len(review.Findings),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to log audit event: %v\n", err)
	}

	return &review, nil
}

func (s *AIPlanningService) SuggestPriorities(ctx context.Context) (*planning.PrioritySuggestions, error) {
	// 1. Check Policy
	cfg, err := s.repo.LoadPolicy()
	if err != nil {
		return nil, err
	}
	if !cfg.AllowAI {
		return nil, fmt.Errorf("AI usage is disabled by project policy")
	}

	// 2. Load Spec and Plan
	productSpec, err := s.repo.LoadSpec()
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}

	plan, err := s.repo.LoadPlan()
	if err != nil {
		return nil, fmt.Errorf("load plan: %w", err)
	}

	// 3. Build Prompt
	prompt := `Analyze the following tasks and their dependencies to suggest priority adjustments.
Consider: dependency chains (blockers should be high priority), feature importance,
and balanced workload distribution.

Return ONLY a JSON object with no surrounding text, no markdown, and no code fences.

Format:
{
  "suggestions": [
    {
      "task_id": "<task ID>",
      "current_priority": "<current priority>",
      "suggested_priority": "<low|medium|high>",
      "reason": "<brief explanation>"
    }
  ],
  "summary": "<overall assessment>"
}

Only include tasks whose priority should change. If all priorities are appropriate, return an empty suggestions array.

`
	prompt += fmt.Sprintf("Specification: %s\n\nTasks:\n", productSpec.Title)

	for _, t := range plan.Tasks {
		deps := "none"
		if len(t.DependsOn) > 0 {
			deps = strings.Join(t.DependsOn, ", ")
		}
		prompt += fmt.Sprintf("- %s (ID: %s, Priority: %s, Feature: %s, DependsOn: %s)\n  %s\n",
			t.Title, t.ID, t.Priority, t.FeatureID, deps, t.Description)
	}

	// 4. Call AI
	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		Prompt:      prompt,
		System:      "You are an expert technical lead analyzing task priorities. Return structured JSON only.",
		Temperature: 0.2,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("AI prioritization failed: %w", err)
	}

	// 5. Parse Response
	cleanJSON := extractJSONPayload(resp.Text)
	var suggestions planning.PrioritySuggestions
	if err := json.Unmarshal([]byte(cleanJSON), &suggestions); err != nil {
		return nil, fmt.Errorf("parse priority suggestions: %w", err)
	}

	// 6. Log Usage
	if err := s.audit.Log("plan.ai_prioritize", "ai", map[string]interface{}{
		"model":         resp.Model,
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
		"suggestions":   len(suggestions.Suggestions),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to log audit event: %v\n", err)
	}

	return &suggestions, nil
}

func (s *AIPlanningService) QueryProject(ctx context.Context, question string) (string, error) {
	// 1. Check Policy
	cfg, err := s.repo.LoadPolicy()
	if err != nil {
		return "", err
	}
	if !cfg.AllowAI {
		return "", fmt.Errorf("AI usage is disabled by project policy")
	}

	// 2. Load context: spec, plan, state
	productSpec, err := s.repo.LoadSpec()
	if err != nil {
		return "", fmt.Errorf("load spec: %w", err)
	}

	plan, _ := s.repo.LoadPlan()
	state, _ := s.repo.LoadState()

	// 3. Build context prompt
	context := fmt.Sprintf("Project: %s\nDescription: %s\n\n", productSpec.Title, productSpec.Description)

	context += "Features:\n"
	for _, f := range productSpec.Features {
		context += fmt.Sprintf("- %s (ID: %s): %s\n", f.Title, f.ID, f.Description)
	}

	if plan != nil && len(plan.Tasks) > 0 {
		context += fmt.Sprintf("\nPlan (%d tasks, approval: %s):\n", len(plan.Tasks), plan.ApprovalStatus)
		for _, t := range plan.Tasks {
			status := "pending"
			owner := ""
			if state != nil {
				if ts, ok := state.TaskStates[t.ID]; ok {
					status = string(ts.Status)
					owner = ts.Owner
				}
			}
			ownerStr := ""
			if owner != "" {
				ownerStr = fmt.Sprintf(", owner: %s", owner)
			}
			context += fmt.Sprintf("- %s (ID: %s, status: %s, priority: %s%s)\n", t.Title, t.ID, status, t.Priority, ownerStr)
		}
	}

	prompt := fmt.Sprintf("%s\nUser question: %s", context, question)

	// 4. Call AI
	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		Prompt:      prompt,
		System:      "You are a project management assistant. Answer questions about the project based on the provided context. Be concise and specific. If the answer cannot be determined from the context, say so.",
		Temperature: 0.3,
		MaxTokens:   1000,
	})
	if err != nil {
		return "", fmt.Errorf("AI query failed: %w", err)
	}

	// 5. Log Usage
	if err := s.audit.Log("project.ai_query", "ai", map[string]interface{}{
		"model":         resp.Model,
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to log audit event: %v\n", err)
	}

	return resp.Text, nil
}

func (s *AIPlanningService) ExplainDrift(ctx context.Context, report *drift.Report) (string, error) {
	if len(report.Issues) == 0 {
		return "No drift detected. Project is healthy.", nil
	}

	// 1. Check Policy
	cfg, err := s.repo.LoadPolicy()
	if err != nil {
		return "", err
	}
	if !cfg.AllowAI {
		return "", fmt.Errorf("AI usage is disabled by project policy")
	}

	// 2. Load Spec for context
	spec, err := s.repo.LoadSpec()
	if err != nil {
		return "", fmt.Errorf("failed to load spec: %w", err)
	}

	// 3. Prompt AI
	prompt := fmt.Sprintf("Analyze these detected drift issues in a software project. "+
		"Explain the potential impact of each issue and suggest specific resolution steps.\n\n"+
		"Project: %s\nIssues:\n", spec.Title)

	for _, issue := range report.Issues {
		prompt += fmt.Sprintf("- [%s] %s: %s (Component: %s)\n", issue.Severity, issue.Type, issue.Message, issue.ComponentID)
	}

	prompt += "\nProvide a concise analysis and actionable next steps."

	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		Prompt: prompt,
		System: "You are an expert technical lead. You help developers align reality with their plans and specifications.",
	})
	if err != nil {
		return "", fmt.Errorf("AI drift explanation failed: %w", err)
	}

	// 4. Log Usage
	if err := s.audit.Log("drift.ai_explanation", "ai", map[string]interface{}{
		"model":         resp.Model,
		"issue_count":   len(report.Issues),
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to log audit event: %v\n", err)
	}

	return resp.Text, nil
}

// SmartDecompose performs context-aware task decomposition using codebase structure.
func (s *AIPlanningService) SmartDecompose(ctx context.Context, codebaseRoot string) (*planning.SmartPlan, error) {
	cfg, err := s.repo.LoadPolicy()
	if err != nil {
		return nil, err
	}
	if !cfg.AllowAI {
		return nil, fmt.Errorf("AI usage is disabled by project policy")
	}

	productSpec, err := s.repo.LoadSpec()
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}
	if productSpec == nil {
		return nil, fmt.Errorf("spec is nil")
	}

	// Scan codebase for structure context
	fileTree := ScanCodebaseTree(codebaseRoot, 200)

	// Build prompt with codebase context
	prompt := `Task: Decompose the following features into atomic engineering tasks, using the existing codebase structure to inform your task breakdown.

For each task, suggest which existing files would need modification and estimate complexity (low, medium, high).

Return ONLY a JSON object with no surrounding text, no markdown, and no code fences.
The JSON must have this shape:
{
  "tasks": [
    {
      "id": "task-<id>",
      "feature_id": "<feature-id>",
      "title": "<title>",
      "description": "<description>",
      "files": ["path/to/file.go", ...],
      "complexity": "low|medium|high"
    }
  ],
  "summary": "<brief summary of the decomposition strategy>"
}

Features to decompose:
`

	for _, f := range productSpec.Features {
		prompt += fmt.Sprintf("- Feature: %s (ID: %s)\n", f.Title, f.ID)
		if f.Description != "" {
			prompt += fmt.Sprintf("  Description: %s\n", f.Description)
		}
		for _, r := range f.Requirements {
			prompt += fmt.Sprintf("  * Requirement: %s (ID: %s)\n", r.Title, r.ID)
		}
	}

	prompt += "\nExisting codebase structure:\n" + fileTree + "\n"

	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		Prompt: prompt,
		System: "You are an expert software architect. Analyze the codebase structure and decompose features into concrete, file-level engineering tasks.",
	})
	if err != nil {
		return nil, fmt.Errorf("AI smart decomposition failed: %w", err)
	}

	clean := extractJSONPayload(resp.Text)

	var result planning.SmartPlan
	if err := json.Unmarshal([]byte(clean), &result); err != nil {
		return nil, fmt.Errorf("failed to parse smart decomposition: %w", err)
	}

	_ = s.audit.Log("plan.ai_smart_decompose", "ai", map[string]interface{}{
		"task_count":        len(result.Tasks),
		"files_in_codebase": len(strings.Split(fileTree, "\n")),
	})

	return &result, nil
}

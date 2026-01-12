package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/roady/internal/domain"
	"github.com/felixgeelhaar/roady/internal/domain/ai"
	"github.com/felixgeelhaar/roady/internal/domain/drift"
	"github.com/felixgeelhaar/roady/internal/domain/planning"
	"github.com/felixgeelhaar/roady/internal/domain/spec"
)

type AIPlanningService struct {
	repo     domain.WorkspaceRepository
	provider ai.Provider
	audit    *AuditService
}

func NewAIPlanningService(repo domain.WorkspaceRepository, provider ai.Provider, audit *AuditService) *AIPlanningService {
	return &AIPlanningService{repo: repo, provider: provider, audit: audit}
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
	spec, err := s.repo.LoadSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to load spec: %w", err)
	}
	if spec == nil {
		return nil, fmt.Errorf("spec is nil")
	}

	// 3. Prompt AI
	prompt := fmt.Sprintf(`Task: Decompose the following features into atomic engineering tasks.
Requirement: Every Feature and every Requirement listed below MUST be implemented.

MAPPING RULES:
1. For each Requirement, create a task with ID: "task-[requirement-id]".
2. For each Feature, ensure at least one task references its Feature ID.
3. Return ONLY a JSON list of tasks.

Format:
[
  {"id": "task-slug", "title": "...", "description": "...", "priority": "medium", "estimate": "4h", "feature_id": "matching-feature-id"}
]

Features to decompose:
`, spec.Title)

	for _, f := range spec.Features {
		prompt += fmt.Sprintf("- Feature: %s (ID: %s)\n", f.Title, f.ID)
		for _, r := range f.Requirements {
			prompt += fmt.Sprintf("  * Requirement: %s (ID: %s)\n", r.Title, r.ID)
		}
	}

	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		Prompt: prompt,
		System: "You are an expert technical lead. You return a JSON array of technical tasks. You ensure that every feature ID provided is represented in the result.",
	})
	if err != nil {
		return nil, fmt.Errorf("AI planning failed: %w", err)
	}

	// 4. Log Usage
	if err := s.audit.Log("plan.ai_decomposition", "ai", map[string]interface{}{
		"model":         resp.Model,
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
	}); err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	// 5. Parse and Reconcile
	var tasks []planning.Task
	cleanJSON := strings.TrimSpace(resp.Text)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	// Helper to validate a task has minimal required fields
	isValid := func(t planning.Task) bool {
		return t.ID != "" && t.Title != ""
	}

	// Try direct list unmarshal
	if err := json.Unmarshal([]byte(cleanJSON), &tasks); err == nil && len(tasks) > 0 && isValid(tasks[0]) {
		// Success
	} else {
		// Reset and try map/wrapped objects
		tasks = nil
		var generic map[string]interface{}
		if err := json.Unmarshal([]byte(cleanJSON), &generic); err == nil {
			// 1. Check for common wrapper keys (tasks, task, data)
			for _, key := range []string{"tasks", "task", "data"} {
				if sub, ok := generic[key]; ok {
					subData, _ := json.Marshal(sub)
					var subTasks []planning.Task
					if err := json.Unmarshal(subData, &subTasks); err == nil && len(subTasks) > 0 && isValid(subTasks[0]) {
						tasks = subTasks
						break
					}
				}
			}

			// 2. If still empty, try parsing as a map of tasks
			if len(tasks) == 0 {
				for k, v := range generic {
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
		return nil, fmt.Errorf("AI returned no valid tasks in JSON (Response: %s)", resp.Text)
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

	// Coverage Check: Ensure every feature ID has at least one task
	// (Check both task.FeatureID matching feature.ID OR task.ID matching a requirement ID)
	featureCoverage := make(map[string]bool)
	requirementCoverage := make(map[string]bool)
	for _, t := range cleanTasks {
		featureCoverage[t.FeatureID] = true
		requirementCoverage[t.ID] = true
	}

	var missingFeatures []string
	for _, f := range spec.Features {
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
		}
	}

	if len(missingFeatures) > 0 {
		fmt.Printf("WARNING: AI missed coverage for features: %s. Proceeding anyway, use 'roady drift detect' to see gaps.\n", strings.Join(missingFeatures, ", "))
	}

	// Lock the spec content to this new plan to resolve Intent Drift
	if err := s.repo.SaveSpecLock(spec); err != nil {
		return nil, fmt.Errorf("failed to save spec lock: %w", err)
	}

	planSvc := NewPlanService(s.repo, s.audit)
	return planSvc.UpdatePlan(cleanTasks)
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
		return nil, fmt.Errorf("failed to parse reconciled spec: %w", err)
	}

	// 4. Save and Lock
	if err := s.repo.SaveSpec(&reconciled); err != nil {
		return nil, err
	}
	if err := s.repo.SaveSpecLock(&reconciled); err != nil {
		return nil, err
	}

	_ = s.audit.Log("spec.reconcile", "ai", map[string]interface{}{
		"model": resp.Model,
	})

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
	prompt := fmt.Sprintf("Provide a high-level architectural walkthrough and explanation of this software specification. " +
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
	_ = s.audit.Log("spec.ai_explanation", "ai", map[string]interface{}{
		"model":         resp.Model,
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
	})

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

	// 2. Load Spec and Plan for context
	spec, _ := s.repo.LoadSpec()

	// 3. Prompt AI
	prompt := fmt.Sprintf("Analyze these detected drift issues in a software project. " +
		"Explain the potential impact of each issue and suggest specific resolution steps.\n\n" +
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
	_ = s.audit.Log("drift.ai_explanation", "ai", map[string]interface{}{
		"model":         resp.Model,
		"issue_count":   len(report.Issues),
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
	})

	return resp.Text, nil
}
package application

import (
	"context"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/roady/pkg/domain/org"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/domain/policy"
	"github.com/felixgeelhaar/roady/pkg/storage"
	"gopkg.in/yaml.v3"
)

// OrgService provides organizational multi-project operations.
type OrgService struct {
	root string
}

// NewOrgService creates a new OrgService rooted at the given directory.
func NewOrgService(root string) *OrgService {
	return &OrgService{root: root}
}

// DiscoverProjects walks the root directory tree and returns paths containing .roady directories.
func (s *OrgService) DiscoverProjects() ([]string, error) {
	var projects []string
	err := filepath.Walk(s.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && info.Name() == ".roady" {
			projects = append(projects, filepath.Dir(path))
			return filepath.SkipDir
		}
		return nil
	})
	return projects, err
}

// AggregateMetrics collects metrics from all discovered projects.
func (s *OrgService) AggregateMetrics() (*org.OrgMetrics, error) {
	projects, err := s.DiscoverProjects()
	if err != nil {
		return nil, err
	}

	config, _ := s.LoadOrgConfig()
	orgName := ""
	if config != nil {
		orgName = config.Name
	}

	metrics := &org.OrgMetrics{
		OrgName: orgName,
	}

	for _, p := range projects {
		pm := s.projectMetrics(p)
		metrics.Projects = append(metrics.Projects, pm)
		metrics.TotalTasks += pm.Total
		metrics.TotalVerified += pm.Verified
		metrics.TotalWIP += pm.WIP
	}

	metrics.TotalProjects = len(metrics.Projects)
	if metrics.TotalProjects > 0 {
		var sum float64
		for _, pm := range metrics.Projects {
			sum += pm.Progress
		}
		metrics.AvgProgress = sum / float64(metrics.TotalProjects)
	}

	return metrics, nil
}

func (s *OrgService) projectMetrics(path string) org.ProjectMetrics {
	repo := storage.NewFilesystemRepository(path)
	spec, _ := repo.LoadSpec()
	plan, _ := repo.LoadPlan()
	state, _ := repo.LoadState()

	pm := org.ProjectMetrics{
		Name: filepath.Base(path),
		Path: path,
	}

	if absPath, err := filepath.Abs(path); err == nil {
		pm.Path = absPath
	}

	if spec != nil {
		pm.Name = spec.Title
	}

	if plan != nil {
		pm.Total = len(plan.Tasks)
		for _, t := range plan.Tasks {
			if state != nil {
				if res, ok := state.TaskStates[t.ID]; ok {
					switch res.Status {
					case planning.StatusVerified:
						pm.Verified++
					case planning.StatusDone:
						pm.Done++
					case planning.StatusInProgress:
						pm.WIP++
					case planning.StatusBlocked:
						pm.Blocked++
					default:
						pm.Pending++
					}
					continue
				}
			}
			pm.Pending++
		}
		if pm.Total > 0 {
			pm.Progress = float64(pm.Verified) / float64(pm.Total) * 100
		}
	}

	// Check drift
	auditSvc := NewAuditService(repo)
	policySvc := NewPolicyService(repo)
	inspector := storage.NewCodebaseInspector()
	driftSvc := NewDriftService(repo, auditSvc, inspector, policySvc)
	if driftReport, err := driftSvc.DetectDrift(context.Background()); err == nil && driftReport != nil {
		if len(driftReport.Issues) > 0 {
			pm.HasDrift = true
			pm.DriftCount = len(driftReport.Issues)
		}
	}

	return pm
}

// LoadOrgConfig loads the org config from .roady/org.yaml in the root directory.
func (s *OrgService) LoadOrgConfig() (*org.OrgConfig, error) {
	path := filepath.Join(s.root, ".roady", "org.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config org.OrgConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveOrgConfig saves the org config to .roady/org.yaml in the root directory.
func (s *OrgService) SaveOrgConfig(config *org.OrgConfig) error {
	dir := filepath.Join(s.root, ".roady")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "org.yaml"), data, 0600)
}

// LoadMergedPolicy loads org-level SharedPolicy and overlays project-level policy.yaml values.
func (s *OrgService) LoadMergedPolicy(projectPath string) (*policy.PolicyConfig, error) {
	merged := &policy.PolicyConfig{}

	// Load org-level shared policy
	config, err := s.LoadOrgConfig()
	if err == nil && config != nil && config.SharedPolicy != nil {
		merged.MaxWIP = config.SharedPolicy.MaxWIP
		merged.AllowAI = config.SharedPolicy.AllowAI
		merged.TokenLimit = config.SharedPolicy.TokenLimit
	}

	// Overlay project-level policy
	repo := storage.NewFilesystemRepository(projectPath)
	projectPolicy, err := repo.LoadPolicy()
	if err == nil && projectPolicy != nil {
		if projectPolicy.MaxWIP > 0 {
			merged.MaxWIP = projectPolicy.MaxWIP
		}
		if projectPolicy.AllowAI {
			merged.AllowAI = true
		}
		if projectPolicy.TokenLimit > 0 {
			merged.TokenLimit = projectPolicy.TokenLimit
		}
	}

	return merged, nil
}

// DetectCrossDrift discovers projects and aggregates drift reports.
func (s *OrgService) DetectCrossDrift() (*org.CrossDriftReport, error) {
	projects, err := s.DiscoverProjects()
	if err != nil {
		return nil, err
	}

	report := &org.CrossDriftReport{}

	for _, p := range projects {
		repo := storage.NewFilesystemRepository(p)
		auditSvc := NewAuditService(repo)
		policySvc := NewPolicyService(repo)
		inspector := storage.NewCodebaseInspector()
		driftSvc := NewDriftService(repo, auditSvc, inspector, policySvc)

		driftReport, err := driftSvc.DetectDrift(context.Background())
		summary := org.ProjectDriftSummary{
			Name: filepath.Base(p),
			Path: p,
		}

		if absPath, absErr := filepath.Abs(p); absErr == nil {
			summary.Path = absPath
		}

		// Try to get project name from spec
		if spec, specErr := repo.LoadSpec(); specErr == nil && spec != nil {
			summary.Name = spec.Title
		}

		if err == nil && driftReport != nil && len(driftReport.Issues) > 0 {
			summary.IssueCount = len(driftReport.Issues)
			summary.HasDrift = true
			report.TotalIssues += summary.IssueCount
		}

		report.Projects = append(report.Projects, summary)
	}

	return report, nil
}

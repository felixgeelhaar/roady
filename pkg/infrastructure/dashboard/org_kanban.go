package dashboard

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
	"time"

	"github.com/felixgeelhaar/roady/pkg/application"
	"github.com/felixgeelhaar/roady/pkg/domain/planning"
	"github.com/felixgeelhaar/roady/pkg/storage"
)

// OrgKanbanBoard is a cross-project Kanban board. Tasks from every discovered
// project (root + sub-projects) are merged into the standard five columns,
// each card tagged with its origin project label.
type OrgKanbanBoard struct {
	Columns    []KanbanColumn `json:"columns"`
	Projects   []OrgKanbanRef `json:"projects"`
	TotalTasks int            `json:"total_tasks"`
	TotalDone  int            `json:"total_done"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Err        string         `json:"error,omitempty"`
}

// OrgKanbanRef is a per-project rollup used in the header strip.
type OrgKanbanRef struct {
	Label      string `json:"label"`
	Path       string `json:"path"`
	SubProject string `json:"sub_project,omitempty"`
	Total      int    `json:"total"`
	Done       int    `json:"done"`
}

// OrgKanbanProvider supplies the discovery walker. In tests this is a fake;
// in production the dashboard wires application.OrgService.
type OrgKanbanProvider interface {
	DiscoverProjectsWithSub() ([]application.DiscoveredProject, error)
}

// repoOpener constructs a storage repository for a discovered project. Split
// out so tests can inject in-memory data.
type repoOpener func(p application.DiscoveredProject) (*storage.FilesystemRepository, error)

func defaultRepoOpener(p application.DiscoveredProject) (*storage.FilesystemRepository, error) {
	return storage.NewFilesystemRepositoryForProject(p.Path, p.SubProject)
}

// buildOrgKanbanBoard discovers every project under root and aggregates their
// plans+states into one cross-project board. Cards in each column are tagged
// with a project label so the viewer can tell which task belongs where.
func buildOrgKanbanBoard(prov OrgKanbanProvider, open repoOpener) OrgKanbanBoard {
	if open == nil {
		open = defaultRepoOpener
	}
	board := OrgKanbanBoard{UpdatedAt: time.Now()}

	projects, err := prov.DiscoverProjectsWithSub()
	if err != nil {
		board.Err = err.Error()
		for _, c := range kanbanColumnOrder {
			board.Columns = append(board.Columns, KanbanColumn{Name: c.Name, Status: c.Status})
		}
		return board
	}

	cols := map[string]*KanbanColumn{}
	for _, c := range kanbanColumnOrder {
		cols[c.Status] = &KanbanColumn{Name: c.Name, Status: c.Status}
	}

	for _, p := range projects {
		label := projectLabel(p)
		repo, err := open(p)
		if err != nil {
			continue
		}
		plan, err := repo.LoadPlan()
		if err != nil || plan == nil {
			board.Projects = append(board.Projects, OrgKanbanRef{
				Label: label, Path: repo.ProjectBase(), SubProject: p.SubProject,
			})
			continue
		}
		state, _ := repo.LoadState()

		sub := buildKanbanBoard(plan, state)
		// Merge sub-board tasks into the org columns, tagging each card with
		// its project so consumers can disambiguate and so action endpoints
		// can route the mutation back to the right sub-project.
		for _, sc := range sub.Columns {
			for _, tv := range sc.Tasks {
				tv.ProjectLabel = label
				tv.ProjectPath = p.Path
				tv.ProjectName = p.SubProject
				cols[sc.Status].Tasks = append(cols[sc.Status].Tasks, tv)
			}
		}
		ref := OrgKanbanRef{
			Label:      label,
			Path:       repo.ProjectBase(),
			SubProject: p.SubProject,
			Total:      sub.TotalTasks,
		}
		for _, sc := range sub.Columns {
			if sc.Status == string(planning.StatusDone) {
				ref.Done = sc.Count
			}
		}
		board.Projects = append(board.Projects, ref)
	}

	// Stable order within columns: project label, then task ID.
	for _, c := range kanbanColumnOrder {
		col := cols[c.Status]
		sort.SliceStable(col.Tasks, func(i, j int) bool {
			if col.Tasks[i].ProjectLabel != col.Tasks[j].ProjectLabel {
				return col.Tasks[i].ProjectLabel < col.Tasks[j].ProjectLabel
			}
			return col.Tasks[i].Task.ID < col.Tasks[j].Task.ID
		})
		col.Count = len(col.Tasks)
		board.Columns = append(board.Columns, *col)
		board.TotalTasks += col.Count
		if c.Status == string(planning.StatusDone) {
			board.TotalDone = col.Count
		}
	}

	sort.SliceStable(board.Projects, func(i, j int) bool {
		return board.Projects[i].Label < board.Projects[j].Label
	})
	return board
}

func projectLabel(p application.DiscoveredProject) string {
	base := filepath.Base(p.Path)
	if p.SubProject == "" {
		return base
	}
	return base + "/" + p.SubProject
}

// orgKanbanData is the template input.
type orgKanbanData struct {
	Title string
	Board OrgKanbanBoard
}

// orgKanbanHandler returns an http.HandlerFunc bound to a provider+opener.
// The dashboard server registers this when an OrgKanbanProvider is wired.
func (s *Server) orgKanbanHandler(prov OrgKanbanProvider, open repoOpener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		board := buildOrgKanbanBoard(prov, open)
		s.render(w, "org_kanban.html", orgKanbanData{Title: "Org Kanban", Board: board})
	}
}

func (s *Server) orgKanbanAPIHandler(prov OrgKanbanProvider, open repoOpener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		board := buildOrgKanbanBoard(prov, open)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(board)
	}
}

// EnableOrgKanban attaches the cross-project Kanban routes. Call before Start.
// Pass nil for open to use the default filesystem repository constructor.
func (s *Server) EnableOrgKanban(prov OrgKanbanProvider, open repoOpener) {
	s.orgProvider = prov
	s.orgRepoOpener = open
}

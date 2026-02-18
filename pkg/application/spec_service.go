package application

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/roady/pkg/domain"
	"github.com/felixgeelhaar/roady/pkg/domain/spec"
)

type SpecService struct {
	repo domain.WorkspaceRepository
}

func NewSpecService(repo domain.WorkspaceRepository) *SpecService {
	return &SpecService{repo: repo}
}

// ImportFromMarkdown reads a markdown file and converts it into a ProductSpec.
func (s *SpecService) ImportFromMarkdown(path string) (*spec.ProductSpec, error) {
	productSpec, err := s.parseMarkdownFile(path)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveSpec(productSpec); err != nil {
		return nil, fmt.Errorf("failed to save spec: %w", err)
	}

	return productSpec, nil
}

// AnalyzeDirectory crawls a directory for markdown files and merges them into a single Spec.
func (s *SpecService) AnalyzeDirectory(root string) (*spec.ProductSpec, error) {
	mergedSpec := &spec.ProductSpec{
		ID:          "analyzed-spec",
		Version:     "0.1.0",
		Constraints: []spec.Constraint{},
		Features:    []spec.Feature{},
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".md") || strings.HasSuffix(info.Name(), ".markdown")) {
			// Skip roady internal docs or hidden files
			if strings.Contains(path, ".roady") || strings.HasPrefix(info.Name(), ".") {
				return nil
			}

			fileSpec, err := s.parseMarkdownFile(path)
			if err != nil {
				return nil // Skip files that fail to parse
			}

			// 1. Merge Title/Description
			if mergedSpec.Title == "" {
				mergedSpec.Title = fileSpec.Title
			}
			if mergedSpec.Description == "" {
				mergedSpec.Description = fileSpec.Description
			}

			// 2. Intelligent Feature Merge
			for _, newFeat := range fileSpec.Features {
				found := false
				for i, existingFeat := range mergedSpec.Features {
					if existingFeat.ID == newFeat.ID {
						// Merge Angle: Append descriptions with a separator
						if !strings.Contains(existingFeat.Description, newFeat.Description) {
							mergedSpec.Features[i].Description += "\n\n---\n\n" + newFeat.Description
						}
						// Merge Requirements
						mergedSpec.Features[i].Requirements = append(mergedSpec.Features[i].Requirements, newFeat.Requirements...)
						found = true
						break
					}
				}
				if !found {
					mergedSpec.Features = append(mergedSpec.Features, newFeat)
				}
			}

			// 3. Merge Constraints
			mergedSpec.Constraints = append(mergedSpec.Constraints, fileSpec.Constraints...)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(mergedSpec.Features) == 0 {
		return nil, fmt.Errorf("no features found in directory: %s", root)
	}

	if err := s.repo.SaveSpec(mergedSpec); err != nil {
		return nil, fmt.Errorf("failed to save merged spec: %w", err)
	}

	return mergedSpec, nil
}

func (s *SpecService) parseMarkdownFile(path string) (*spec.ProductSpec, error) {
	cleanPath := filepath.Clean(path)
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close() //nolint:errcheck // best-effort close on read path

	scanner := bufio.NewScanner(file)

	productSpec := &spec.ProductSpec{
		ID:          "imported-spec",
		Version:     "0.1.0",
		Constraints: []spec.Constraint{},
		Features:    []spec.Feature{},
	}

	var currentFeature *spec.Feature
	var descriptionBuilder strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "# ") {
			productSpec.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		} else if strings.HasPrefix(line, "## ") {
			if currentFeature != nil {
				currentFeature.Description = strings.TrimSpace(descriptionBuilder.String())
				productSpec.Features = append(productSpec.Features, *currentFeature)
				descriptionBuilder.Reset()
			}

			title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			id := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
			currentFeature = &spec.Feature{
				ID:    id,
				Title: title,
			}
		} else {
			if currentFeature != nil {
				descriptionBuilder.WriteString(line + "\n")
			} else if productSpec.Description == "" && line != "" {
				productSpec.Description = line
			}
		}
	}

	if currentFeature != nil {
		currentFeature.Description = strings.TrimSpace(descriptionBuilder.String())
		productSpec.Features = append(productSpec.Features, *currentFeature)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return productSpec, nil
}

func (s *SpecService) GetSpec() (*spec.ProductSpec, error) {

	return s.repo.LoadSpec()

}

// AddFeature programmatically adds a new functional unit and syncs it back to documentation.

func (s *SpecService) AddFeature(title, description string) (*spec.ProductSpec, error) {

	current, err := s.repo.LoadSpec()

	if err != nil {

		return nil, fmt.Errorf("failed to load spec: %w", err)

	}

	id := strings.ToLower(strings.ReplaceAll(title, " ", "-"))

	newFeat := spec.Feature{

		ID: id,

		Title: title,

		Description: description,
	}

	current.Features = append(current.Features, newFeat)

	if err := s.repo.SaveSpec(current); err != nil {

		return nil, err

	}

	if err := s.repo.SaveSpecLock(current); err != nil {

		return nil, err

	}

	// Step 4: Sync back to documentation
	if err := s.syncToMarkdown(newFeat); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to sync feature to markdown: %v\n", err)
	}

	return current, nil

}

func (s *SpecService) syncToMarkdown(f spec.Feature) (err error) {

	path := "docs/backlog.md"

	// Ensure directory exists
	if err := os.MkdirAll("docs", 0700); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	content := fmt.Sprintf("\n## %s\n\n%s\n\n---\n", f.Title, f.Description)

	fWriter, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {

		return err

	}

	defer func() {
		if cerr := fWriter.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close file: %w", cerr)
		}
	}()

	_, err = fWriter.WriteString(content)

	return err

}

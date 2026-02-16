package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/roady/internal/infrastructure/wiring"
)

func loadServices(root string) (*wiring.AppServices, error) {
	services, loadErr := wiring.BuildAppServices(root)
	if services == nil {
		return nil, fmt.Errorf("failed to build services: %w", loadErr)
	}
	if loadErr != nil {
		fmt.Printf("Warning: %v\n", loadErr)
	}
	return services, nil
}

func getProjectRoot() (string, error) {
	if projectPath != "" {
		abs, err := filepath.Abs(projectPath)
		if err != nil {
			return "", fmt.Errorf("invalid project path %q: %w", projectPath, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", fmt.Errorf("project path %q: %w", abs, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("project path %q is not a directory", abs)
		}
		return abs, nil
	}
	return os.Getwd()
}

func loadServicesForCurrentDir() (*wiring.AppServices, error) {
	root, err := getProjectRoot()
	if err != nil {
		return nil, err
	}
	return loadServices(root)
}

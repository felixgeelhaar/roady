package cli

import (
	"fmt"
	"os"

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

func loadServicesForCurrentDir() (*wiring.AppServices, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return loadServices(cwd)
}

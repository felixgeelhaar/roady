package wiring

import (
	"github.com/felixgeelhaar/roady/internal/infrastructure/config"
	infraai "github.com/felixgeelhaar/roady/pkg/ai"
	domainai "github.com/felixgeelhaar/roady/pkg/domain/ai"
)

func LoadAIProvider(root string) (domainai.Provider, error) {
	cfg, err := config.LoadAIConfig(root)
	if err != nil {
		return nil, err
	}

	providerName := "ollama"
	modelName := "llama3"
	if cfg != nil {
		if cfg.Provider != "" {
			providerName = cfg.Provider
		}
		if cfg.Model != "" {
			modelName = cfg.Model
		}
	}

	baseProvider, err := infraai.GetDefaultProvider(providerName, modelName)
	if err != nil {
		return nil, err
	}

	return infraai.NewResilientProvider(baseProvider), nil
}

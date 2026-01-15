package wiring

import (
	"time"

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
	resilienceConfig := infraai.DefaultResilienceConfig()

	if cfg != nil {
		if cfg.Provider != "" {
			providerName = cfg.Provider
		}
		if cfg.Model != "" {
			modelName = cfg.Model
		}
		if cfg.MaxRetries > 0 {
			resilienceConfig.MaxRetries = cfg.MaxRetries
		}
		if cfg.RetryDelayMs > 0 {
			resilienceConfig.RetryDelay = time.Duration(cfg.RetryDelayMs) * time.Millisecond
		}
		if cfg.TimeoutSec > 0 {
			resilienceConfig.Timeout = time.Duration(cfg.TimeoutSec) * time.Second
		}
	}

	baseProvider, err := infraai.GetDefaultProvider(providerName, modelName)
	if err != nil {
		return nil, err
	}

	return infraai.NewResilientProviderWithConfig(baseProvider, resilienceConfig), nil
}

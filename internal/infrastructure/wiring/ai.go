package wiring

import (
	"fmt"
	"os"
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

	envProvider := os.Getenv("ROADY_AI_PROVIDER")
	envModel := os.Getenv("ROADY_AI_MODEL")

	var providerName, modelName string
	switch {
	case envProvider != "":
		providerName = envProvider
		modelName = envModel
	case cfg != nil && cfg.Provider != "":
		providerName = cfg.Provider
		modelName = cfg.Model
	default:
		return nil, fmt.Errorf("no AI provider configured. Please run 'roady ai configure' or set ROADY_AI_PROVIDER environment variable")
	}

	resilienceConfig := infraai.DefaultResilienceConfig()
	if cfg != nil {
		if modelName == "" {
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

package ai

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/roady/internal/domain/ai"
)

func NewProvider(providerName string, modelName string) (ai.Provider, error) {
	switch providerName {
	case "ollama", "":
		if modelName == "" {
			modelName = "llama3"
		}
		return NewOllamaProvider(modelName), nil
	case "mock":
		return &MockProvider{Model: modelName}, nil
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		return NewOpenAIProvider(modelName, apiKey), nil
	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		return NewAnthropicProvider(modelName, apiKey), nil
	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		return NewGeminiProvider(modelName, apiKey), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", providerName)
	}
}

// GetDefaultProvider returns a provider based on environment variables or policy defaults.
func GetDefaultProvider(providerName, modelName string) (ai.Provider, error) {
	// 1. Check environment variables
	envProvider := os.Getenv("ROADY_AI_PROVIDER")
	envModel := os.Getenv("ROADY_AI_MODEL")

	if envProvider != "" {
		providerName = envProvider
	}
	if envModel != "" {
		modelName = envModel
	}

	return NewProvider(providerName, modelName)
}
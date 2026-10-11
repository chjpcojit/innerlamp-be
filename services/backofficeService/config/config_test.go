package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
)

func TestTempleSuggestionConfig_Validate(t *testing.T) {
	assert.NoError(t, TempleSuggestionConfig{PageSize: 1}.Validate())
	assert.Error(t, TempleSuggestionConfig{}.Validate(), "a config that forgot the block must fail at startup")
	assert.Error(t, TempleSuggestionConfig{PageSize: -5}.Validate())
}

func TestDevYAML_TempleSuggestionConfig(t *testing.T) {
	b, err := os.ReadFile("config.dev.yaml")
	require.NoError(t, err)
	var cfg AppConfig
	require.NoError(t, yaml.Unmarshal(b, &cfg))
	assert.Equal(t, 20, cfg.TempleSuggestion.PageSize)
	assert.NoError(t, cfg.TempleSuggestion.Validate())
}

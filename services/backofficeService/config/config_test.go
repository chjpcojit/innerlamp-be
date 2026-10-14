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

func validAnnouncementConfig() TempleAnnouncementConfig {
	return TempleAnnouncementConfig{
		TitleMaxChars: 150, BodyMaxChars: 5000, MaxValidityDays: 365,
		IssuingBodyNameMaxChars: 100, PageSize: 20, MaxPageSize: 100,
	}
}

func TestTempleAnnouncementConfig_Validate(t *testing.T) {
	assert.NoError(t, validAnnouncementConfig().Validate())
	assert.Error(t, TempleAnnouncementConfig{}.Validate(), "a config that forgot the block must fail at startup")

	for name, mutate := range map[string]func(*TempleAnnouncementConfig){
		"title zero":         func(c *TempleAnnouncementConfig) { c.TitleMaxChars = 0 },
		"body negative":      func(c *TempleAnnouncementConfig) { c.BodyMaxChars = -1 },
		"validity zero":      func(c *TempleAnnouncementConfig) { c.MaxValidityDays = 0 },
		"body name zero":     func(c *TempleAnnouncementConfig) { c.IssuingBodyNameMaxChars = 0 },
		"title over column":  func(c *TempleAnnouncementConfig) { c.TitleMaxChars = 151 },
		"name over column":   func(c *TempleAnnouncementConfig) { c.IssuingBodyNameMaxChars = 101 },
		"page above max":     func(c *TempleAnnouncementConfig) { c.PageSize = 101 },
		"max page size zero": func(c *TempleAnnouncementConfig) { c.MaxPageSize = 0 },
	} {
		c := validAnnouncementConfig()
		mutate(&c)
		assert.Error(t, c.Validate(), name)
	}
}

func TestDevYAML_TempleAnnouncementConfig(t *testing.T) {
	b, err := os.ReadFile("config.dev.yaml")
	require.NoError(t, err)
	var cfg AppConfig
	require.NoError(t, yaml.Unmarshal(b, &cfg))
	assert.Equal(t, validAnnouncementConfig(), cfg.TempleAnnouncement)
	assert.NoError(t, cfg.TempleAnnouncement.Validate())
}

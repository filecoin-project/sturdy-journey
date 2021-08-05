package config

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoading(t *testing.T) {
	type config struct {
		BaseURL URL
	}

	loadConfig := func(input io.Reader) *config {
		cfg, err := FromReader(input, &config{})
		require.Nil(t, err)

		return cfg.(*config)
	}

	cfg := loadConfig(strings.NewReader(`
	BaseURL = "https://website.example/path/"
	`))

	assert.Equal(t, cfg.BaseURL.Path, "/path/")
	assert.Equal(t, cfg.BaseURL.Host, "website.example")
	assert.Equal(t, cfg.BaseURL.Scheme, "https")
}

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigureWithoutEnv(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	cfg, _ := Configure()
	assert.NotNil(t, cfg)
}

func TestConfigure(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret")

	cfg, _ := Configure()
	assert.NotNil(t, cfg)

	assert.Equal(t, "secret", cfg.GitHubToken)
}

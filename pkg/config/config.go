package config

import (
	"context"

	"github.com/kelseyhightower/envconfig"
)

// Config is the main configuration for the github action watcher
type Config struct {
	GitHubToken     string `required:"true" envconfig:"GITHUB_TOKEN"`
	GitHubAPIHost   string `default:"api.github.com" envconfig:"GITHUB_API_HOST"`
	GitHubAPIScheme string `default:"https" envconfig:"GITHUB_API_SCHEME"`
	Debug           bool   `default:"false" envconfig:"DEBUG"`
	Owner           string `default:"fr123k" envconfig:"OWNER"`
	Project         string `default:"flink-core-shared" envconfig:"PROJECT"`
}

func Configure() (Config, context.Context) {
	var cfg Config
	err := envconfig.Process("GITHUB_ACTION_WATCHER", &cfg)
	if err != nil {
		panic(err)
	}

	return cfg, context.Background()
}

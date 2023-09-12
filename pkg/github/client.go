package github

import (
	"context"
	"net/http"

	"github.com/google/go-github/v54/github"
	"golang.org/x/oauth2"

	"github.com/fr123k/github-operator/pkg/config"
)

type Secret = github.Secret
type Secrets = github.Secrets
type PublicKey = github.PublicKey

type GithubClient struct {
	client *github.Client
	ctx    context.Context
	cfg    config.Config
}

type Option func(*GithubClient)

func WithContext(ctx context.Context) Option {
	return func(g *GithubClient) {
		g.ctx = ctx
	}
}

func WithClient(cli *http.Client) Option {
	return func(g *GithubClient) {
		g.client = github.NewClient(cli)
	}
}

func NewClient(cfg config.Config, opts ...Option) GithubClient {
	gc := GithubClient{cfg: cfg}

	for _, opt := range opts {
		opt(&gc)
	}

	if gc.client == nil {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: cfg.GitHubToken},
		)
		tc := oauth2.NewClient(gc.ctx, ts)

		gc.client = github.NewClient(tc)
		gc.client.BaseURL.Host = cfg.GitHubAPIHost
		gc.client.BaseURL.Scheme = cfg.GitHubAPIScheme
	}
	return gc
}

func (gh GithubClient) RemoveDependaBotSecrets(repository string, secretName string) error {
	_, err := gh.client.Dependabot.DeleteRepoSecret(gh.ctx, gh.cfg.Owner, repository, secretName)
	if err != nil {
		return err
	}

	return nil
}

func (gh GithubClient) ListDependaBotSecrets(repository string) (*github.Secrets, error) {
	opts := &github.ListOptions{Page: 0, PerPage: 100}
	secrets, _, err := gh.client.Dependabot.ListRepoSecrets(gh.ctx, gh.cfg.Owner, repository, opts)
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

func (gh GithubClient) AddDependaBotSecrets(owner, repository string, name string, value string) (*github.DependabotEncryptedSecret, error) {

	pk, _, err := gh.client.Dependabot.GetRepoPublicKey(gh.ctx, gh.cfg.Owner, repository)
	if err != nil {
		return nil, err
	}

	secret := &github.DependabotEncryptedSecret{
		Name:  name,
		KeyID: *pk.KeyID,
	}

	secret.EncryptedValue, err = Encrypt(*pk.Key, value)

	if err != nil {
		return nil, err
	}

	_, err = gh.client.Dependabot.CreateOrUpdateRepoSecret(gh.ctx, owner, repository, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// gh.client.Actions.CreateOrUpdateOrgSecret()

// gh.client.Actions.ListOrgSecrets()

// gh.client.Actions.GetOrgPublicKey()

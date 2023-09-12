package github

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/fr123k/github-operator/pkg/config"

	"github.com/google/go-github/v54/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	_ = NewClient(config.Config{}, WithContext(context.Background()))
}

// MemorySink implements zap.Sink by writing all messages to a buffer.
type MemorySink struct {
	*bytes.Buffer
}

// Implement Close and Sync as no-ops to satisfy the interface. The Write
// method is provided by the embedded buffer.

func (s *MemorySink) Close() error { return nil }
func (s *MemorySink) Sync() error  { return nil }

func NewSecret(name string) *github.Secret {
	return &github.Secret{Name: name}
}

func DependaBotSecrets(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	secrets := []*github.Secret{
		NewSecret("Secret 1"),
		NewSecret("Secret 2"),
	}
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(mock.MustMarshal(github.Secrets{
			TotalCount: len(secrets),
			Secrets:    secrets,
		}))
		assert.NoError(t, err)
	}
}

func AddDependaBotSecrets(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	secrets := []*github.Secret{
		NewSecret("Secret 1"),
		NewSecret("Secret 2"),
	}
	return func(w http.ResponseWriter, r *http.Request) {

		body, _ := r.GetBody()
		fmt.Println(body)
		_, err := w.Write(mock.MustMarshal(github.Secrets{
			TotalCount: len(secrets),
			Secrets:    secrets,
		}))
		assert.NoError(t, err)
	}
}

func DependaBotPublicKey(t *testing.T, key string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keyId := "test_key_id"
		_, err := w.Write(mock.MustMarshal(github.PublicKey{
			Key:   &key,
			KeyID: &keyId,
		}))
		assert.NoError(t, err)
	}
}

func TestListDependaBotSecrets(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetReposDependabotSecretsByOwnerByRepo,
			http.HandlerFunc(DependaBotSecrets(t)),
		),
	)
	client := NewClient(config.Config{Owner: "fr123k"}, WithContext(context.Background()), WithClient(mockedHTTPClient))
	secret, err := client.ListDependaBotSecrets("test_repo")

	assert.NoError(t, err)

	assert.Equal(t, 2, len(secret.Secrets))
	assert.Equal(t, "Secret 1", secret.Secrets[0].Name)
	assert.Equal(t, "Secret 2", secret.Secrets[1].Name)
}

func TestAddDependaBotSecrets(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.PutReposDependabotSecretsByOwnerByRepoBySecretName,
			http.HandlerFunc(DependaBotSecrets(t)),
		),
		mock.WithRequestMatchHandler(
			mock.GetReposDependabotSecretsPublicKeyByOwnerByRepo,
			http.HandlerFunc(DependaBotPublicKey(t, "aWk5RWlwaDlwdTVvaHNvaGZhM2FheTRDaGk1Ym9oeQo=")),
		),
	)
	client := NewClient(config.Config{Owner: "fr123k"}, WithContext(context.Background()), WithClient(mockedHTTPClient))
	secret, err := client.AddDependaBotSecrets("test_owner", "test_repo", "test_secret", "test_value")

	assert.Equal(t, "test_secret", secret.Name)
	assert.True(t, len(secret.EncryptedValue) > 0)

	assert.NoError(t, err)
}

func TestAddDependaBotSecretsError(t *testing.T) {
	for _, test := range []struct {
		name             string
		mockedHTTPClient *http.Client
		expectedError    string
	}{
		{
			name: "no public key found",
			mockedHTTPClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.PutReposDependabotSecretsByOwnerByRepoBySecretName,
					http.HandlerFunc(DependaBotSecrets(t)),
				),
			),
			expectedError: "repos/fr123k/test_repo/dependabot/secrets/public-key: 405  []",
		},
		{
			name:             "to short public key",
			mockedHTTPClient: MockedGithubClient(t, "dUs2ZWVnb28zaG9vCg=="),
			expectedError:    "recipient public key has invalid length (13 bytes)",
		},
		{
			name:             "empty public key",
			mockedHTTPClient: MockedGithubClient(t, ""),
			expectedError:    "recipient public key has invalid length (0 bytes)",
		},
		{
			name:             "non base64 encoded public key",
			mockedHTTPClient: MockedGithubClient(t, "test_key"),

			expectedError: "illegal base64 data at input byte 4",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(config.Config{Owner: "fr123k"}, WithContext(context.Background()), WithClient(test.mockedHTTPClient))
			_, err := client.AddDependaBotSecrets("test_owner", "test_repo", "test_secret", "test_value")
			assert.Contains(t, err.Error(), test.expectedError)
		})
	}
}

func MockedGithubClient(t *testing.T, key string) *http.Client {
	return mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.PutReposDependabotSecretsByOwnerByRepoBySecretName,
			http.HandlerFunc(DependaBotSecrets(t)),
		),
		mock.WithRequestMatchHandler(
			mock.GetReposDependabotSecretsPublicKeyByOwnerByRepo,
			http.HandlerFunc(DependaBotPublicKey(t, key)),
		),
	)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func ErrorStatus(t *testing.T, status int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		mock.WriteError(
			w,
			status,
			"github went belly up or something",
		)
	}
}

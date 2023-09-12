/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	// v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	secretv1alpha1 "github.com/fr123k/github-operator/api/v1alpha1"
	"github.com/fr123k/github-operator/pkg/config"
	"github.com/fr123k/github-operator/pkg/gcloud"

	//+kubebuilder:scaffold:imports

	"github.com/fr123k/github-operator/pkg/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"

	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var (
	ctx    context.Context
	cancel context.CancelFunc
)

var _ = BeforeSuite(func() {
	os.Setenv("GITHUB_TOKEN", "GITHUB_TOKEN")

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:        []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = secretv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	cfg, ctx := config.Configure()

	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetReposDependabotSecretsByOwnerByRepo,
			http.HandlerFunc(DependaBotSecrets()),
		),
		mock.WithRequestMatchHandler(
			mock.PutReposDependabotSecretsByOwnerByRepoBySecretName,
			http.HandlerFunc(DependaBotSecrets()),
		),
		mock.WithRequestMatchHandler(
			mock.GetReposDependabotSecretsPublicKeyByOwnerByRepo,
			http.HandlerFunc(DependaBotPublicKey("aWk5RWlwaDlwdTVvaHNvaGZhM2FheTRDaGk1Ym9oeQo=")),
		),
	)
	client := github.NewClient(config.Config{Owner: "fr123k"}, github.WithContext(context.Background()), github.WithClient(mockedHTTPClient))

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	gsrv := grpc.NewServer()
	fakeSecretManagerServer := &fakeSecretManagerServer{}
	secretmanagerpb.RegisterSecretManagerServiceServer(gsrv, fakeSecretManagerServer)
	fakeServerAddr := l.Addr().String()
	go func() {
		if err := gsrv.Serve(l); err != nil {
			panic(err)
		}
	}()

	gc := gcloud.NewClient(cfg,
		gcloud.WithContext(ctx),
		gcloud.WithOptions(option.WithEndpoint(fakeServerAddr),
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithInsecure())),
	)

	err = (&GithubSecretReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		Github: client,
		GCloud: gc,
		Config: cfg,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := (func() (err error) {
		// Need to sleep if the first stop fails due to a bug:
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1571
		sleepTime := 1 * time.Millisecond
		for i := 0; i < 12; i++ { // Exponentially sleep up to ~4s
			if err = testEnv.Stop(); err == nil {
				return
			}
			sleepTime *= 2
			time.Sleep(sleepTime)
		}
		return
	})()
	Expect(err).NotTo(HaveOccurred())
})

func NewSecret(name string) *github.Secret {
	return &github.Secret{Name: name}
}

func DependaBotSecrets() func(w http.ResponseWriter, r *http.Request) {
	secrets := []*github.Secret{
		NewSecret("Secret 1"),
		NewSecret("Secret 2"),
	}
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(mock.MustMarshal(github.Secrets{
			TotalCount: len(secrets),
			Secrets:    secrets,
		}))
		if err != nil {
			panic(err)
		}
	}
}

func DependaBotPublicKey(key string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keyId := "test_key_id"
		_, err := w.Write(mock.MustMarshal(github.PublicKey{
			Key:   &key,
			KeyID: &keyId,
		}))
		if err != nil {
			panic(err)
		}
	}
}

type fakeSecretManagerServer struct {
	secretmanagerpb.UnimplementedSecretManagerServiceServer
}

func (f *fakeSecretManagerServer) AccessSecretVersion(context.Context, *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	resp := &secretmanagerpb.AccessSecretVersionResponse{
		Name: "projects/fr123k/secrets/secret/versions/latest",
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte("secret"),
		},
	}
	return resp, nil
}

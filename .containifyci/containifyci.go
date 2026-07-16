//go:generate bash -c "if [ ! -f go.mod ]; then echo 'Initializing go.mod...'; go mod init .containifyci; else echo 'go.mod already exists. Skipping initialization.'; fi"
//go:generate go get github.com/containifyci/engine-ci/protos2
//go:generate go get github.com/containifyci/engine-ci/client
//go:generate go mod tidy

package main

import (
	"os"

	"github.com/containifyci/engine-ci/client/pkg/build"
	"github.com/containifyci/engine-ci/protos2"
)

func main() {
	os.Chdir("../")
	opts := build.NewGoServiceBuild("github-operator")
	opts.Image = ""
	opts.ContainerFiles = map[string]*protos2.ContainerFile{
		"build": DockerFile(),
	}
	build.BuildAsync(opts)
}

func DockerFile() *protos2.ContainerFile {
	return &protos2.ContainerFile{
		Name: "golang-1.26.4-alpine-custom",
		Content: `FROM golang:1.26-alpine

RUN apk --no-cache add git openssh-client curl bash && \
  rm -rf /var/cache/apk/*

# Install kubebuilder envtest binaries (etcd, kube-apiserver, kubectl)
# Required by controller tests that use envtest
RUN go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest && \
  setup-envtest use 1.26.0 --bin-dir /usr/local/kubebuilder/bin -p path || true && \
  export KUBEBUILDER_ASSETS=$(setup-envtest use 1.26.0 --bin-dir /usr/local/kubebuilder/bin -p path 2>/dev/null) && \
  mkdir -p /usr/local/kubebuilder/bin && \
  if [ -n "$KUBEBUILDER_ASSETS" ]; then \
    cp -r $KUBEBUILDER_ASSETS/* /usr/local/kubebuilder/bin/ 2>/dev/null || true; \
  fi && \
  go clean -cache && \
  go clean -modcache

# Also try downloading envtest binaries directly as fallback
RUN curl -fsSL https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-4.15.0-linux-amd64.tar.gz -o /tmp/kubebuilder-tools.tar.gz && \
  mkdir -p /usr/local/kubebuilder && \
  tar -xzf /tmp/kubebuilder-tools.tar.gz -C /usr/local/kubebuilder --strip-components=1 && \
  rm -f /tmp/kubebuilder-tools.tar.gz && \
  ls -la /usr/local/kubebuilder/bin/ || true

WORKDIR /app`,
	}
}

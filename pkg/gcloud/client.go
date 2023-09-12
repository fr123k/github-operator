package gcloud

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/option"

	"github.com/fr123k/github-operator/pkg/config"
)

type GCloudClient struct {
	client *secretmanager.Client
	ctx    context.Context
	cfg    config.Config
	opts   []option.ClientOption
}

type Option func(*GCloudClient)

func WithContext(ctx context.Context) Option {
	return func(g *GCloudClient) {
		g.ctx = ctx
	}
}

func WithOptions(opts ...option.ClientOption) Option {
	return func(g *GCloudClient) {
		g.opts = opts
	}
}

func NewClient(cfg config.Config, opts ...Option) GCloudClient {

	gc := GCloudClient{cfg: cfg}

	for _, opt := range opts {
		opt(&gc)
	}

	c, err := secretmanager.NewClient(gc.ctx, gc.opts...)

	if err != nil {
		//TODO add logging and error management
		panic(err)
	}
	gc.client = c
	return gc
}

func (gc GCloudClient) GetSecretValue(key string) (*string, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", gc.cfg.Project, key),
	}
	resp, err := gc.client.AccessSecretVersion(gc.ctx, req)
	if err != nil {
		// logger.Errorw("failed to access secret version", "error", err, "request", req)
		return nil, err
	}
	str := string(resp.Payload.Data)
	return &str, nil
}

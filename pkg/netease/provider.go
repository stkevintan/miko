package netease

import (
	"context"
	"fmt"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
	nmlog "github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/registry"
)

// Initialize default logger for netease package
func init() {
	nmlog.Default = nmlog.New(&nmlog.Config{
		Level:  "info",
		Format: "json",
		Stdout: true,
	})

}

func NewNetEaseProvider(i do.Injector) (registry.Provider, error) {
	jar := do.MustInvoke[cookiecloud.CookieJar](i)
	return NewProvider(jar)
}

type NMProvider struct {
	cli     *api.Client
	request *weapi.Api
	jar     cookiecloud.CookieJar
}

var _ registry.Provider = (*NMProvider)(nil)

// NewProvider creates a new NMProvider for multiple songs (returns concrete type)
func NewProvider(jar cookiecloud.CookieJar) (*NMProvider, error) {
	cli, err := NewClient(jar)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	request := weapi.New(cli)

	return &NMProvider{
		cli:     cli,
		request: request,
		jar:     jar,
	}, nil
}

// GetCookieJar returns the CookieJar used by the provider
func (d *NMProvider) GetCookieJar() cookiecloud.CookieJar {
	return d.jar
}

// Close closes the provider and cleans up resources
func (d *NMProvider) Close(ctx context.Context) error {

	return d.cli.Close(ctx)
}

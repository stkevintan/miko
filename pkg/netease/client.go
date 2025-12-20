package netease

import (
	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/pkg/cookie"
	"github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/stkevintan/miko/pkg/cookiecloud"
)

func NewClient(cfg *cookiecloud.Config) (*api.Client, error) {
	client, err := api.NewClient(&api.Config{
		Timeout: cfg.Timeout,
		Retry:   cfg.Retry,
		Debug:   false,
		Cookie: cookie.Config{
			Interval: 0,
		},
	}, log.Default)

	client.GetClient().Jar, err = cookiecloud.NewCookieCloudJar(cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}

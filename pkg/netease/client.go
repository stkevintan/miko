package netease

import (
	"net/http"
	"os"
	"path"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/pkg/cookie"
	"github.com/chaunsin/netease-cloud-music/pkg/log"
)

func NewClient(cookieJar http.CookieJar) (*api.Client, error) {
	client, err := api.NewClient(&api.Config{
		Timeout: 0,
		Retry:   0,
		Debug:   false,
		Cookie: cookie.Config{
			Interval: 0,
			// we don't need local cookie file, but make the lib happy
			Filepath: path.Join(os.TempDir(), "cookie.json"),
		},
	}, log.Default)
	if err != nil {
		return nil, err
	}

	client.GetClient().Jar = cookieJar
	return client, nil
}

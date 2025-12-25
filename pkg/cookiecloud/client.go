package cookiecloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chaunsin/cookiecloud-go-sdk"
	"github.com/stkevintan/miko/pkg/log"
)

type Config struct {
	Url     string        `json:"url" mapstructure:"url"`
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
	Retry   int           `json:"retry" mapstructure:"retry"`
	Debug   bool          `json:"debug" mapstructure:"debug"`
}

type CookieJar interface {
	http.CookieJar
	PullAll() error
}

func NewCookieCloudJar(ctx context.Context, config *Config, identity *Identity) (CookieJar, error) {
	cli, err := cookiecloud.NewClient(&cookiecloud.Config{
		Url:     config.Url,
		Timeout: config.Timeout,
		Retry:   config.Retry,
		Debug:   config.Debug,
	})

	if err != nil {
		return nil, err
	}

	localJar, _ := cookiejar.New(nil)
	jar := &CookieCloudJar{
		client:   cli,
		ctx:      ctx,
		identity: identity,
		localJar: localJar,
	}

	if identity != nil {
		_ = jar.PullAll()
	}

	return jar, nil
}

type CookieCloudJar struct {
	client   *cookiecloud.Client
	ctx      context.Context
	identity *Identity
	localJar http.CookieJar
	mu       sync.RWMutex
}

func (c *CookieCloudJar) PullAll() error {
	cookie, err := c.identity.download(c.ctx, c.client)
	if err != nil {
		return err
	}

	newJar, _ := cookiejar.New(nil)
	for domain, cookies := range cookie.CookieData {
		u := &url.URL{
			Scheme: "https",
			Host:   strings.TrimPrefix(domain, "."),
		}
		var httpCookies []*http.Cookie
		for _, v := range cookies {
			httpCookies = append(httpCookies, cookieDataToHttpCookies(domain, &v))
		}
		newJar.SetCookies(u, httpCookies)
	}

	c.mu.Lock()
	c.localJar = newJar
	c.mu.Unlock()

	log.Info("CookieCloudJar: pulled cookies for %d domains", len(cookie.CookieData))
	return nil
}

func (c *CookieCloudJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.mu.RLock()
	jar := c.localJar
	c.mu.RUnlock()

	jar.SetCookies(u, cookies)

	if c.identity == nil {
		log.Warn("CookieCloudJar: identity not set, cannot set cookies")
		return
	}
	log.Debug("CookieCloudJar: set %d cookies for %s", len(cookies), u.Hostname())

	// Push may be nonsense because the cookie will shortly be overridden by the browser

	// err := c.identity.push(c.ctx, c.client, u.Hostname(), cookies)
	// if err != nil {
	// 	log.Warn("CookieCloudJar: failed to push cookies: %v", err)
	// 	return
	// }

	// // Re-pull to ensure localJar is in sync with the server's merged state
	// if err := c.PullAll(); err != nil {
	// 	log.Warn("CookieCloudJar: failed to refresh cookies after push: %v", err)
	// }
}

func (c *CookieCloudJar) Cookies(u *url.URL) []*http.Cookie {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.localJar.Cookies(u)
}

// TODO: partial push
// update cookies of u to cookiecloud server
// func (c *Identity) push(ctx context.Context, client *cookiecloud.Client, originHost string, cookies []*http.Cookie) error {
// 	cookie, err := c.download(ctx, client)
// 	if err != nil {
// 		return fmt.Errorf("Failed to download cookies before push: %v", err)
// 	}

// 	cookieDataDict := make(map[string][]cookiecloud.CookieData)
// 	for _, ck := range cookies {
// 		domainKey, cookieData := httpCookieToCookieData(originHost, ck)
// 		if cookieData == nil || domainKey == "" {
// 			continue
// 		}
// 		cookieDataDict[domainKey] = append(cookieDataDict[domainKey], *cookieData)
// 	}
// 	maps.Copy(cookie.CookieData, cookieDataDict)

// 	log.Info("Pushing cookies of %d domains to cookiecloud", len(cookieDataDict))
// 	_, err = client.Update(ctx, &cookiecloud.UpdateReq{
// 		Uuid:     c.UUID,
// 		Password: c.Password,
// 		Cookie:   cookie,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("Failed to push cookies to cookiecloud: %v", err)
// 	}
// 	log.Info("Successfully pushed cookies to cookiecloud")
// 	return nil
// }

func (c *Identity) download(ctx context.Context, client *cookiecloud.Client) (cookiecloud.Cookie, error) {
	res, err := client.Get(ctx, &cookiecloud.GetReq{
		Uuid:            c.UUID,
		Password:        c.Password,
		CloudDecryption: false,
	})
	if err != nil {
		return cookiecloud.Cookie{}, fmt.Errorf("failed to pull cookies: %w", err)
	}
	return res.Cookie, nil
}

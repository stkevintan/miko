package cookiecloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/chaunsin/cookiecloud-go-sdk"
	"github.com/stkevintan/miko/pkg/log"
)

type Config struct {
	Url          string        `json:"url" mapstructure:"url"`
	Timeout      time.Duration `json:"timeout" mapstructure:"timeout"`
	Retry        int           `json:"retry" mapstructure:"retry"`
	Debug        bool          `json:"debug" mapstructure:"debug"`
	SyncInterval time.Duration `json:"sync_interval" mapstructure:"sync_interval"`
}

type CookieJar interface {
	http.CookieJar
	UpdateCredential(uuid, password string) error
}

func NewCookieCloudJar(config *Config) (CookieJar, error) {
	cli, err := cookiecloud.NewClient(&cookiecloud.Config{
		Url:     config.Url,
		Timeout: config.Timeout,
		Retry:   config.Retry,
		Debug:   config.Debug,
	})

	if err != nil {
		return nil, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	cc := &CookieCloudJar{
		jar:          jar,
		client:       cli,
		domains:      make(map[string]struct{}),
		syncInterval: config.SyncInterval,
	}

	return cc, nil
}

type CookieCloudJar struct {
	jar          http.CookieJar
	client       *cookiecloud.Client
	syncInterval time.Duration
	domains      map[string]struct{}
	credential   *ccloudCredential
}

func (c *CookieCloudJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.jar.SetCookies(u, cookies)
	c.domains[u.Host] = struct{}{}
}

func (c *CookieCloudJar) Cookies(u *url.URL) []*http.Cookie {
	return c.jar.Cookies(u)
}

func (c *CookieCloudJar) UpdateCredential(uuid, password string) error {
	if uuid == "" || password == "" {
		return fmt.Errorf("uuid and password are required")
	}
	if c.client == nil {
		return fmt.Errorf("cookiecloud client not initialized")
	}

	cred := &ccloudCredential{
		Uuid:     uuid,
		Password: password,
		client:   c.client,
		jar:      c,
	}

	if err := cred.Pull(context.Background()); err != nil {
		return fmt.Errorf("failed to pull cookies after setting credentials: %w", err)
	}
	cred.StartSync()
	c.credential = cred
	return nil
}

type ccloudCredential struct {
	Uuid       string
	Password   string
	client     *cookiecloud.Client
	syncCancel context.CancelFunc
	jar        *CookieCloudJar
}

func (c *ccloudCredential) StopSync() {
	if c.syncCancel != nil {
		c.syncCancel()
		c.syncCancel = nil
	}
}

func (c *ccloudCredential) StartSync() {
	c.StopSync()
	if c.jar.syncInterval <= 0 {
		log.Info("CookieCloud sync disabled (sync_interval <= 0)")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	go c.sync(ctx)
	c.syncCancel = cancel
}

func (c *ccloudCredential) sync(ctx context.Context) {
	interval := c.jar.syncInterval
	log.Info("Starting CookieCloud sync every %s", interval.String())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Skip the first tick so we don't sync immediately on startup.
	<-ticker.C
	for {
		select {
		case <-ctx.Done():
			log.Info("CookieCloud sync stopped")
			return
		case <-ticker.C:
			log.Info("Syncing cookies with cookiecloud server...")
			c.Push(ctx)
		}
	}
}

// update cookies of u to cookiecloud server
func (c *ccloudCredential) Push(ctx context.Context) {
	cookie, err := c.download(ctx)
	if err != nil {
		log.Error("Failed to download cookies before push: %v", err)
		return
	}

	for domain := range c.jar.domains {
		cookie.CookieData[domain] = []cookiecloud.CookieData{}
		cookies := c.jar.Cookies(&url.URL{Scheme: "http", Host: domain})
		for _, ck := range cookies {
			_, cookieData := httpCookieToCookieData(ck)
			cookie.CookieData[domain] = append(cookie.CookieData[domain], *cookieData)
		}
	}
	log.Info("Pushing %d domains' cookies to cookiecloud", len(c.jar.domains))
	_, err = c.client.Update(ctx, &cookiecloud.UpdateReq{
		Uuid:     c.Uuid,
		Password: c.Password,
		Cookie:   *cookie,
	})
	if err != nil {
		log.Error("Failed to push cookies to cookiecloud: %v", err)
		return
	}
	log.Info("Successfully pushed cookies to cookiecloud")
}

func (c *ccloudCredential) download(ctx context.Context) (*cookiecloud.Cookie, error) {
	res, err := c.client.Get(ctx, &cookiecloud.GetReq{
		Uuid:            c.Uuid,
		Password:        c.Password,
		CloudDecryption: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull cookies: %w", err)
	}
	return &res.Cookie, nil
}

func (c *ccloudCredential) Pull(ctx context.Context) error {

	cookie, err := c.download(ctx)
	if err != nil {
		return fmt.Errorf("failed to download cookies: %w", err)
	}
	for domain, cookies := range cookie.CookieData {
		c.jar.domains[domain] = struct{}{}
		log.Info("Pulled %d cookies for domain %s from cookiecloud", len(cookies), domain)
		var httpCookies []*http.Cookie
		for _, v := range cookies {
			httpCookies = append(httpCookies, cookieDataToHttpCookies(domain, &v))
		}
		c.jar.SetCookies(&url.URL{Scheme: "http", Host: domain}, httpCookies)
	}
	return nil
}

package cookiecloud

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
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
	UpdateCredential(uuid, password string) error
}

func NewCookieCloudJar(ctx context.Context, config *Config) (CookieJar, error) {
	cli, err := cookiecloud.NewClient(&cookiecloud.Config{
		Url:     config.Url,
		Timeout: config.Timeout,
		Retry:   config.Retry,
		Debug:   config.Debug,
	})

	if err != nil {
		return nil, err
	}
	return &CookieCloudJar{
		client: cli,
		ctx:    ctx,
	}, nil
}

type CookieCloudJar struct {
	client     *cookiecloud.Client
	ctx        context.Context
	mu         sync.RWMutex
	credential *ccloudCredential
}

func (c *CookieCloudJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.mu.RLock()
	cred := c.credential
	c.mu.RUnlock()

	if cred == nil {
		log.Warn("CookieCloudJar: credential not set, cannot set cookies")
		return
	}
	err := cred.push(c.ctx, u.Hostname(), cookies)
	if err != nil {
		log.Warn("CookieCloudJar: failed to push cookies: %v", err)
	}
}

func (c *CookieCloudJar) Cookies(u *url.URL) []*http.Cookie {
	c.mu.RLock()
	cred := c.credential
	c.mu.RUnlock()

	if cred == nil {
		log.Warn("CookieCloudJar: credential not set, cannot get cookies")
		return nil
	}
	ret, err := cred.pull(c.ctx, u.Hostname())
	if err != nil {
		log.Warn("CookieCloudJar: failed to pull cookies: %v", err)
		return nil
	}
	return ret
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
	}

	c.mu.Lock()
	c.credential = cred
	c.mu.Unlock()
	return nil
}

type ccloudCredential struct {
	Uuid     string
	Password string
	client   *cookiecloud.Client
}

// update cookies of u to cookiecloud server
func (c *ccloudCredential) push(ctx context.Context, originHost string, cookies []*http.Cookie) error {
	cookie, err := c.download(ctx)
	if err != nil {
		return fmt.Errorf("Failed to download cookies before push: %v", err)
	}

	cookieDataDict := make(map[string][]cookiecloud.CookieData)
	for _, ck := range cookies {
		domainKey, cookieData := httpCookieToCookieData(originHost, ck)
		if cookieData == nil || domainKey == "" {
			continue
		}
		cookieDataDict[domainKey] = append(cookieDataDict[domainKey], *cookieData)
	}
	maps.Copy(cookie.CookieData, cookieDataDict)

	log.Info("Pushing cookies of %d domains to cookiecloud", len(cookieDataDict))
	_, err = c.client.Update(ctx, &cookiecloud.UpdateReq{
		Uuid:     c.Uuid,
		Password: c.Password,
		Cookie:   cookie,
	})
	if err != nil {
		return fmt.Errorf("Failed to push cookies to cookiecloud: %v", err)
	}
	log.Info("Successfully pushed cookies to cookiecloud")
	return nil
}

func (c *ccloudCredential) download(ctx context.Context) (cookiecloud.Cookie, error) {
	res, err := c.client.Get(ctx, &cookiecloud.GetReq{
		Uuid:            c.Uuid,
		Password:        c.Password,
		CloudDecryption: false,
	})
	if err != nil {
		return cookiecloud.Cookie{}, fmt.Errorf("failed to pull cookies: %w", err)
	}
	return res.Cookie, nil
}

func (c *ccloudCredential) pull(ctx context.Context, domainWanted string) ([]*http.Cookie, error) {

	cookie, err := c.download(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to download cookies: %w", err)
	}
	var httpCookies []*http.Cookie
	needle := normalizeDomain(domainWanted)
	for domain, cookies := range cookie.CookieData {
		if needle != "" && needle != normalizeDomain(domain) {
			continue
		}
		log.Info("Pulled %d cookies for domain %s from cookiecloud", len(cookies), domain)
		for _, v := range cookies {
			httpCookies = append(httpCookies, cookieDataToHttpCookies(domain, &v))
		}
	}
	return httpCookies, nil
}

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
	"gorm.io/gorm"
)

type Config struct {
	Url     string        `json:"url" mapstructure:"url"`
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
	Retry   int           `json:"retry" mapstructure:"retry"`
	Debug   bool          `json:"debug" mapstructure:"debug"`
}

type CookieJar interface {
	http.CookieJar
	UpdateIdentity(uuid, password string) error
	GetUrl() string
}

func NewCookieCloudJar(ctx context.Context, config *Config, db *gorm.DB, identity *Identity) (CookieJar, error) {
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
		client:   cli,
		ctx:      ctx,
		url:      config.Url,
		db:       db,
		identity: identity,
	}, nil
}

type CookieCloudJar struct {
	url      string
	client   *cookiecloud.Client
	ctx      context.Context
	mu       sync.RWMutex
	identity *Identity
	db       *gorm.DB
}

func (c *CookieCloudJar) GetUrl() string {
	return c.url
}

func (c *CookieCloudJar) getIdentity() (*Identity, error) {
	c.mu.RLock()
	if c.identity != nil {
		defer c.mu.RUnlock()
		return c.identity, nil
	}
	c.mu.RUnlock()

	if c.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var identity Identity
	if err := c.db.First(&identity).Error; err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.identity = &identity
	c.mu.Unlock()
	return &identity, nil
}

func (c *CookieCloudJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	identity, err := c.getIdentity()
	if err != nil {
		log.Warn("CookieCloudJar: failed to get identity: %v", err)
		return
	}

	err = identity.push(c.ctx, c.client, u.Hostname(), cookies)
	if err != nil {
		log.Warn("CookieCloudJar: failed to push cookies: %v", err)
	}
}

func (c *CookieCloudJar) Cookies(u *url.URL) []*http.Cookie {
	identity, err := c.getIdentity()
	if err != nil {
		log.Warn("CookieCloudJar: failed to get identity: %v", err)
		return nil
	}

	ret, err := identity.pull(c.ctx, c.client, u.Hostname())
	if err != nil {
		log.Warn("CookieCloudJar: failed to pull cookies: %v", err)
		return nil
	}
	return ret
}

func (c *CookieCloudJar) UpdateIdentity(uuid, password string) error {
	if uuid == "" || password == "" {
		return fmt.Errorf("uuid and password are required")
	}
	if c.client == nil {
		return fmt.Errorf("cookiecloud client not initialized")
	}

	identity := &Identity{
		UUID:     uuid,
		Password: password,
	}

	c.mu.Lock()
	c.identity = identity
	c.mu.Unlock()
	return nil
}

// update cookies of u to cookiecloud server
func (c *Identity) push(ctx context.Context, client *cookiecloud.Client, originHost string, cookies []*http.Cookie) error {
	cookie, err := c.download(ctx, client)
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
	_, err = client.Update(ctx, &cookiecloud.UpdateReq{
		Uuid:     c.UUID,
		Password: c.Password,
		Cookie:   cookie,
	})
	if err != nil {
		return fmt.Errorf("Failed to push cookies to cookiecloud: %v", err)
	}
	log.Info("Successfully pushed cookies to cookiecloud")
	return nil
}

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

func (c *Identity) pull(ctx context.Context, client *cookiecloud.Client, domainWanted string) ([]*http.Cookie, error) {

	cookie, err := c.download(ctx, client)
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

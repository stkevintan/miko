package cookiecloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/stkevintan/miko/pkg/log"
)

func NewCookieCloudJar(config *Config) (*CookieCloudJar, error) {
	cli, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	cc := &CookieCloudJar{
		jar:      jar,
		client:   cli,
		uuid:     config.Uuid,
		password: config.Password,
		domains:  make(map[string]struct{}),
	}
	if err := cc.Pull(); err != nil {
		return nil, err
	}
	log.Info("Initialized CookieCloudJar with UUID: %s", config.Uuid)
	return cc, nil
}

type CookieCloudJar struct {
	jar      http.CookieJar
	client   *Client
	uuid     string
	password string
	domains  map[string]struct{}
}

// update cookies of u to cookiecloud server
func (c *CookieCloudJar) Push() {
	cookieDataMap := make(map[string][]CookieData)
	for domain := range c.domains {
		cookies := c.jar.Cookies(&url.URL{Scheme: "http", Host: domain})
		for _, cookie := range cookies {
			_, cookieData := httpCookieToCookieData(cookie)
			cookieDataMap[domain] = append(cookieDataMap[domain], *cookieData)
		}
	}
	log.Info("Pushing %d domains' cookies to cookiecloud", len(cookieDataMap))
	for domain, cookies := range cookieDataMap {
		log.Info("Domain: %s, Cookies: %d", domain, len(cookies))
		// for _, cookie := range cookies {
		// 	log.Info("  - Name: %s, Value: %s, Path: %s, Expires: %v, HttpOnly: %v, Secure: %v, SameSite: %s",
		// 		cookie.Name, cookie.Value, cookie.Path, cookie.GetExpired(), cookie.HttpOnly, cookie.Secure, cookie.SameSite)
		// }
	}
	// how to partially push?
	// ctx, cancel := context.WithTimeout(context.Background(), c.client.cfg.Timeout)
	// defer cancel()
	// _, _ = c.client.Push(ctx, &PushReq{
	// 	Uuid:     c.uuid,
	// 	Password: c.password,
	// 	Cookie: Cookie{
	// 		CookieData: cookieDataMap,
	// 	},
	// })
}
func (c *CookieCloudJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.jar.SetCookies(u, cookies)
	c.domains[u.Host] = struct{}{}
	c.Push()
}

func (c *CookieCloudJar) Cookies(u *url.URL) []*http.Cookie {
	return c.jar.Cookies(u)

}

func (c *CookieCloudJar) Pull() error {
	res, err := c.client.Get(context.Background(), &GetReq{
		Uuid:     c.uuid,
		Password: c.password,
	})
	if err != nil {
		return fmt.Errorf("failed to pull cookies: %w", err)
	}
	for domain, cookies := range res.CookieData {
		c.domains[domain] = struct{}{}
		log.Info("Pulled %d cookies for domain %s from cookiecloud", len(cookies), domain)
		var httpCookies []*http.Cookie
		for _, v := range cookies {
			httpCookies = append(httpCookies, cookieDataToHttpCookies(domain, &v))
		}
		c.jar.SetCookies(&url.URL{Scheme: "http", Host: domain}, httpCookies)
	}
	return nil
}

// IsPrint returns whether s is ASCII and printable according to
// https://tools.ietf.org/html/rfc20#section-4.2.
func isPrint(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			return false
		}
	}
	return true
}

// ToLower returns the lowercase version of s if s is ASCII and printable.
func toLower(s string) (lower string, ok bool) {
	if !isPrint(s) {
		return "", false
	}
	return strings.ToLower(s), true
}

func sameSite(val string) http.SameSite {
	lowerVal, ascii := toLower(val)
	if !ascii {
		return http.SameSiteDefaultMode
	}
	switch lowerVal {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	case "unspecified": // is means http.SameSiteDefaultMode or http.SameSiteNoneMode ?
		return http.SameSiteDefaultMode
	default:
		return http.SameSiteDefaultMode
	}
}
func sameSiteR(val http.SameSite) string {
	switch val {
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteNoneMode:
		return "None"
	case http.SameSiteDefaultMode:
		return "Unspecified"
	default:
		return "Unspecified"
	}
}

func cookieDataToHttpCookies(domain string, v *CookieData) *http.Cookie {
	return &http.Cookie{
		Domain:   domain, // Use original domain value
		Expires:  v.GetExpired(),
		HttpOnly: v.HttpOnly,
		Name:     v.Name,
		Path:     v.Path,
		Secure:   v.Secure,
		Value:    v.Value,
		SameSite: sameSite(v.SameSite),
	}
}

func httpCookieToCookieData(cookie *http.Cookie) (string, *CookieData) {
	domain := cookie.Domain
	return domain, &CookieData{
		Domain:         cookie.Domain,
		ExpirationDate: float64(cookie.Expires.Unix()) + float64(cookie.Expires.Nanosecond())/1e9,
		HttpOnly:       cookie.HttpOnly,
		// HostOnly
		Name:     cookie.Name,
		Path:     cookie.Path,
		SameSite: sameSiteR(cookie.SameSite),
		Secure:   cookie.Secure,
		// Session
		// StoreId: cookie.StoreId
		Value: cookie.Value,
	}
}

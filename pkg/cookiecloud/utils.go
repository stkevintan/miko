package cookiecloud

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/chaunsin/cookiecloud-go-sdk"
)

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
	case "no_restriction":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	case "unspecified":
		return http.SameSiteDefaultMode
	default:
		return http.SameSiteDefaultMode
	}
}
func sameSiteR(val http.SameSite) string {
	switch val {
	case http.SameSiteStrictMode:
		return "strict"
	case http.SameSiteLaxMode:
		return "lax"
	case http.SameSiteNoneMode:
		return "no_restriction"
	case http.SameSiteDefaultMode:
		return "unspecified"
	default:
		return "unspecified"
	}
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, ".")
	host = strings.ToLower(host)

	// Strip port when present (e.g. example.com:443).
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.Trim(h, "[]")
	}
	return strings.Trim(host, "[]")
}

func normalizeDomain(domain string) string {
	return normalizeHost(domain)
}

func cookieDataToHttpCookies(domain string, v *cookiecloud.CookieData) *http.Cookie {
	if v == nil {
		return nil
	}

	path := v.Path
	if path == "" {
		path = "/"
	}

	// Chrome CookieData.Domain can start with '.', while Go's Cookie.Domain should not.
	normalizedDomain := normalizeDomain(v.Domain)
	if normalizedDomain == "" {
		normalizedDomain = normalizeDomain(domain)
	}

	var expires time.Time
	if !v.Session && v.ExpirationDate != 0 {
		expires = v.GetExpired()
	}

	cookie := &http.Cookie{
		Expires:  expires,
		HttpOnly: v.HttpOnly,
		Name:     v.Name,
		Path:     path,
		Secure:   v.Secure,
		Value:    v.Value,
		SameSite: sameSite(v.SameSite),
	}

	// HostOnly cookies should not have Domain set in Go.
	if v.HostOnly || strings.HasPrefix(v.Name, "__Host-") {
		cookie.Domain = ""
	} else {
		cookie.Domain = normalizedDomain
	}

	return cookie
}

func httpCookieToCookieData(originHost string, cookie *http.Cookie) (string, *cookiecloud.CookieData) {
	if cookie == nil {
		return "", nil
	}

	host := normalizeHost(originHost)
	domainAttr := normalizeDomain(cookie.Domain)
	hostOnly := domainAttr == ""
	if hostOnly {
		domainAttr = host
	}

	path := cookie.Path
	if path == "" {
		path = "/"
	}

	session := cookie.Expires.IsZero()
	expirationDate := float64(0)
	if !session {
		expirationDate = float64(cookie.Expires.Unix()) + float64(cookie.Expires.Nanosecond())/1e9
	}

	secure := cookie.Secure
	// Enforce Chrome prefix rules best-effort.
	if strings.HasPrefix(cookie.Name, "__Secure-") {
		secure = true
	}
	if strings.HasPrefix(cookie.Name, "__Host-") {
		secure = true
		hostOnly = true
		domainAttr = host
		path = "/"
	}

	// Use the normalized domain as the map key (CookieCloud uses domain-keyed storage).
	domainKey := domainAttr

	return domainKey, &cookiecloud.CookieData{
		Domain:         domainAttr,
		ExpirationDate: expirationDate,
		HostOnly:       hostOnly,
		HttpOnly:       cookie.HttpOnly,
		Name:           cookie.Name,
		Path:           path,
		SameSite:       sameSiteR(cookie.SameSite),
		Secure:         secure,
		Session:        session,
		Value:          cookie.Value,
	}
}

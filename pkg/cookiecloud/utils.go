package cookiecloud

import (
	"net/http"
	"strings"

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

func cookieDataToHttpCookies(domain string, v *cookiecloud.CookieData) *http.Cookie {
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

func httpCookieToCookieData(cookie *http.Cookie) (string, *cookiecloud.CookieData) {
	domain := cookie.Domain
	return domain, &cookiecloud.CookieData{
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

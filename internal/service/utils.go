package service

import (
	"net/http"
	"strings"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/pkg/cookie"
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

func (s *Service) getNMApiConfig() *api.Config {
	return &api.Config{
		Debug:   false,
		Timeout: 0,
		Retry:   0,
		Cookie: cookie.Config{
			Options:  nil,
			Filepath: "../testdata/cookie.json",
			Interval: 0,
		},
	}
}

package netease

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/chaunsin/netease-cloud-music/api/types"
	nmTypes "github.com/chaunsin/netease-cloud-music/api/types"
)

// ValidateQualityLevel validates and converts quality level string to types.Level
func ValidateQualityLevel(level string) (nmTypes.Level, error) {
	if level == "" {
		return types.LevelHires, nil // default to hires
	}

	// Handle numeric levels
	if lv, err := strconv.ParseInt(string(level), 10, 64); err == nil {
		switch lv {
		case 128:
			return types.LevelStandard, nil
		case 192:
			return types.LevelHigher, nil
		case 320:
			return types.LevelExhigh, nil
		default:
			return "", fmt.Errorf("%v level is not supported", lv)
		}
	}

	// Handle string levels
	switch types.Level(level) {
	case types.LevelStandard,
		types.LevelHigher,
		types.LevelExhigh,
		types.LevelLossless,
		types.LevelHires:
		return types.Level(level), nil
	default:
		// Handle uppercase aliases
		switch strings.ToUpper(level) {
		case "HQ":
			return types.LevelExhigh, nil
		case "SQ":
			return types.LevelLossless, nil
		case "HR":
			return types.LevelHires, nil
		default:
			return "", fmt.Errorf("[%s] quality is not supported", level)
		}
	}
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

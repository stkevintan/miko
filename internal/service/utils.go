package service

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chaunsin/netease-cloud-music/pkg/utils"
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

func writeFile(out string, data []byte) error {
	if out == "" {
		return fmt.Errorf("output path is empty")
	}

	// 写入文件
	var file string
	if !filepath.IsAbs(out) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		file = filepath.Join(wd, out)
		if !utils.DirExists(file) {
			if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
				return fmt.Errorf("MkdirAll: %w", err)
			}
		}
	}
	if err := os.WriteFile(file, data, os.ModePerm); err != nil {
		return fmt.Errorf("WriteFile: %w", err)
	}
	return nil
}

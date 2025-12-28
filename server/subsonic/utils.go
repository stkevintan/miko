package subsonic

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
)

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func getQueryInt[T Integer](r *http.Request, key string) (T, error) {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		var zero T
		return zero, fmt.Errorf("missing required parameter: %s", key)
	}

	var zero T
	var res T
	var err error

	switch any(zero).(type) {
	case uint, uint8, uint16, uint32, uint64:
		var val uint64
		val, err = strconv.ParseUint(valStr, 10, 64)
		res = T(val)
	default:
		var val int64
		val, err = strconv.ParseInt(valStr, 10, 64)
		res = T(val)
	}

	if err != nil {
		return zero, fmt.Errorf("invalid value for parameter '%s': %s", key, valStr)
	}
	return res, nil
}

func getQueryIntOrDefault[T Integer](r *http.Request, key string, defaultValue T) T {
	val, err := getQueryInt[T](r, key)
	if err != nil {
		return defaultValue
	}
	return val
}

func getAuthUsername(r *http.Request) (string, error) {
	username, ok := r.Context().Value(usernameKey).(string)
	if !ok {
		// Fallback to query parameter for cases where middleware might not have run
		username = r.URL.Query().Get("u")
	}

	if username == "" {
		return "", fmt.Errorf("username not found")
	}
	return username, nil
}

func safeServeFile(w http.ResponseWriter, r *http.Request, path string) {
	writer := w
	// if arch is darwin or windows, use simpleWriter to avoid sendfile(2)
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		// Disable the sendfile(2) optimization for http.ServeFile, by stripping/hiding the ReadFrom method
		// since sendfile(2) is problematic on macOS (Racing headers) and Windows (2-concurrent-stream)
		type IOCopyWriter struct {
			http.ResponseWriter
		}
		writer = &IOCopyWriter{ResponseWriter: w}
	}
	http.ServeFile(writer, r, path)
}

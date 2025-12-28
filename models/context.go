package models

import (
	"fmt"
	"net/http"
)

type ContextKey string

const (
	UsernameKey ContextKey = "username"
)

func GetUsername(r *http.Request) (string, error) {
	username, ok := r.Context().Value(UsernameKey).(string)
	if ok {
		return username, nil
	}
	return "", fmt.Errorf("username not found in context, request may not be authenticated")
}

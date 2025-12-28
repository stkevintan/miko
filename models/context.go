package models

import (
	"net/http"
)

type ContextKey string

const (
	UsernameKey ContextKey = "username"
)

func GetUsername(r *http.Request) string {
	username, ok := r.Context().Value(UsernameKey).(string)
	if !ok {
		panic("username not found in context")
	}
	return username
}

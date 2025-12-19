package types

import "time"

type LoginArgs struct {
	Timeout  time.Duration // 超时时间
	Server   string
	UUID     string
	Password string
}

// LoginResult represents internal login result from the service
type LoginResult struct {
	Username string
	UserID   int64
}

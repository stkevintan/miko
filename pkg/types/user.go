package types

type User struct {
	Username string `json:"username" description:"The username of the user"`
	UserID   int64  `json:"userId" description:"The unique identifier of the user"`
}

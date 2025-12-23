package subsonic

import "encoding/xml"

type ResponseStatus string

const (
	ResponseStatusOK     ResponseStatus = "ok"
	ResponseStatusFailed ResponseStatus = "failed"
)

type SubsonicResponse struct {
	XMLName       xml.Name       `xml:"subsonic-response" json:"-"`
	Xmlns         string         `xml:"xmlns,attr" json:"-"`
	Status        ResponseStatus `xml:"status,attr" json:"status"`
	Version       string         `xml:"version,attr" json:"version"`
	ServerVersion string         `xml:"serverVersion,attr,omitempty" json:"serverVersion,omitempty"`
	Error         *Error         `xml:"error,omitempty" json:"error,omitempty"`
	Ping          *Ping          `xml:"ping,omitempty" json:"ping,omitempty"`
}

type Error struct {
	Code    int    `xml:"code,attr" json:"code"`
	Message string `xml:"message,attr" json:"message"`
}

type Ping struct {
}

func NewResponse(status ResponseStatus) *SubsonicResponse {
	return &SubsonicResponse{
		Xmlns:         "http://subsonic.org/restapi",
		Status:        status,
		Version:       "1.16.1",
		ServerVersion: "1.0.0", // Or get from config
	}
}

func NewErrorResponse(code int, message string) *SubsonicResponse {
	resp := NewResponse(ResponseStatusFailed)
	resp.Error = &Error{
		Code:    code,
		Message: message,
	}
	return resp
}

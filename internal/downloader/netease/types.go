package netease

import (
	"github.com/chaunsin/netease-cloud-music/api/types"
)

// SongPlayerInfoReq represents the request for song player info
type SongPlayerInfoReq struct {
	Ids        string      `json:"ids"` // song id (separated by comma)
	Level      types.Level `json:"level"`
	EncodeType string      `json:"encodeType,omitempty"`
}

// SongPlayerInfoRes represents the response for song player info
type SongPlayerInfoRes struct {
	Code int                `json:"code"`
	Data []SongDownloadInfo `json:"data"`
}

// SongDownloadInfo contains download information for a song
type SongDownloadInfo struct {
	Id         int64  `json:"id"`
	Url        string `json:"url"`
	Md5        string `json:"md5"`
	Level      string `json:"level"`
	Type       string `json:"type"`
	Size       int64  `json:"size"`
	Br         int64  `json:"br"`
	EncodeType string `json:"encodeType"`
	Fee        int64  `json:"fee"`
}

package models

import "github.com/stkevintan/miko/pkg/types"

// DownloadRequest represents the download request
// @Description Music download request
type DownloadRequest struct {
	SongID  string `json:"song_id" binding:"required" example:"2161154646" description:"Song ID to download"`
	Level   string `json:"level,omitempty" example:"lossless" description:"Audio quality: standard/128, higher/192, exhigh/320, lossless/SQ, hires/HR"`
	Output  string `json:"output,omitempty" example:"./downloads" description:"Output directory path"`
	Timeout int    `json:"timeout,omitempty" example:"30000" description:"Timeout in milliseconds"`
}

type DownloadSummary struct {
	Summary string                  `json:"summary" example:"Downloaded 8 out of 10 songs." description:"Summary of the download operation"`
	Details []*types.DownloadResult `json:"details" description:"Detailed batch download response"`
}

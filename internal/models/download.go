package models

// DownloadRequest represents the download request
// @Description Music download request
type DownloadRequest struct {
	SongID  string `json:"song_id" binding:"required" example:"2161154646" description:"Song ID to download"`
	Level   string `json:"level,omitempty" example:"lossless" description:"Audio quality: standard/128, higher/192, exhigh/320, lossless/SQ, hires/HR"`
	Output  string `json:"output,omitempty" example:"./downloads" description:"Output directory path"`
	Timeout int    `json:"timeout,omitempty" example:"30000" description:"Timeout in milliseconds"`
}

// DownloadResponse represents the download response
// @Description Music download response
type DownloadResponse struct {
	SongID         string `json:"song_id" example:"2161154646" description:"Song ID"`
	SongName       string `json:"song_name" example:"Song Title" description:"Song name"`
	Artist         string `json:"artist" example:"Artist Name" description:"Artist name"`
	Album          string `json:"album" example:"Album Name" description:"Album name"`
	AlPicUrl       string `json:"album_pic_url" example:"https://..." description:"Album picture URL"`
	DownloadURL    string `json:"download_url" example:"https://..." description:"Direct download URL"`
	DownloadedPath string `json:"downloaded_path" example:"./downloads/Song Title.flac" description:"Local path where the song is downloaded"`
	Quality        string `json:"quality" example:"lossless" description:"Actual quality level"`
	FileType       string `json:"file_type" example:"flac" description:"Audio file type"`
	FileSize       int64  `json:"file_size" example:"15728640" description:"File size in bytes"`
	Duration       int64  `json:"duration" example:"240000" description:"Duration in milliseconds"`
	Success        bool   `json:"success" example:"true" description:"Download success status"`
	Message        string `json:"message,omitempty" example:"Download URL generated successfully" description:"Additional message"`
}

// BatchDownloadResponse represents the batch download response
// @Description Batch music download response
type BatchDownloadResponse struct {
	Total   int64              `json:"total" example:"10" description:"Total number of songs"`
	Success int64              `json:"success" example:"8" description:"Number of successful downloads"`
	Failed  int64              `json:"failed" example:"2" description:"Number of failed downloads"`
	Songs   []DownloadResponse `json:"songs" description:"Individual song download results"`
	Message string             `json:"message" example:"Batch download completed" description:"Overall status message"`
}

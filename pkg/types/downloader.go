package types

import (
	"context"

	"github.com/stkevintan/miko/config"
)

type DownloaderConfig struct {
	Level          string
	Output         string
	ConflictPolicy string
	Root           *config.Config
}

// @Description Music download response
// type DownloadResponse struct {
// 	SongID         string   `json:"song_id" example:"2161154646" description:"Song ID"`
// 	SongName       string   `json:"song_name" example:"Song Title" description:"Song name"`
// 	Artist         []string `json:"artist" description:"Artist name"`
// 	Album          string   `json:"album" example:"Album Name" description:"Album name"`
// 	AlPicUrl       string   `json:"album_pic_url" example:"https://..." description:"Album picture URL"`
// 	DownloadURL    string   `json:"download_url" example:"https://..." description:"Direct download URL"`
// 	DownloadedPath string   `json:"downloaded_path" example:"./downloads/Song Title.flac" description:"Local path where the song is downloaded"`
// 	Quality        string   `json:"quality" example:"lossless" description:"Actual quality level"`
// 	FileType       string   `json:"file_type" example:"flac" description:"Audio file type"`
// 	FileSize       int64    `json:"file_size" example:"15728640" description:"File size in bytes"`
// 	Duration       int64    `json:"duration" example:"240000" description:"Duration in milliseconds"`
// 	Success        bool     `json:"success" example:"true" description:"Download success status"`
// 	Lyrics         string   `json:"lyrics,omitempty" example:"[00:10.00] Lyrics line..." description:"Lyrics of the song"`
// }

// type BatchDownloadResponse struct {
// 	Total   int64               `json:"total" example:"10" description:"Total number of songs"`
// 	Success int64               `json:"success" example:"8" description:"Number of successful downloads"`
// 	Failed  int64               `json:"failed" example:"2" description:"Number of failed downloads"`
// 	Songs   []*DownloadResponse `json:"songs" description:"Individual song download results"`
// 	Errors  []string            `json:"errors,omitempty" description:"Error messages for failed downloads"`
// }

type DownloadResult struct {
	Err  error            `json:"error,omitempty" description:"Error encountered during download, if any"`
	Data *DownloadedMusic `json:"data,omitempty" description:"Downloaded music information"`
}

type MusicDownloadResults struct {
	Results []*DownloadResult
}

func (d *MusicDownloadResults) Add(result *DownloadResult) {
	d.Results = append(d.Results, result)
}

func (d *MusicDownloadResults) Total() int64 {
	return int64(len(d.Results))
}

func (d *MusicDownloadResults) SuccessCount() int64 {
	var count int64
	for _, r := range d.Results {
		if r.Err == nil {
			count++
		}
	}
	return count
}

func (d *MusicDownloadResults) FailedCount() int64 {
	return d.Total() - d.SuccessCount()
}

// DownloadResult represents the result of a download operation
// Downloader interface defines the contract for different music downloaders
type Downloader interface {
	// DownloadBatch downloads multiple songs and returns the batch result
	Download(ctx context.Context, music []*Music) (*MusicDownloadResults, error)

	// GetMusic returns the music information array
	GetMusic(ctx context.Context, uris []string) ([]*Music, error)

	// GetLevel returns the quality level
	GetLevel() string

	// GetOutput returns the output directory
	GetOutput() string

	// GetConflictPolicy returns the conflict handling policy
	GetConflictPolicy() ConflictPolicy

	Close(ctx context.Context) error
}

// DownloaderConfig represents the configuration for creating downloaders

// DownloaderFactory creates downloaders for different music platforms
type DownloaderFactory interface {
	// CreateDownloader creates a new downloader instance
	CreateDownloader(ctx context.Context, config *DownloaderConfig) (Downloader, error)

	// SupportedPlatforms returns the list of supported platforms
	SupportedPlatforms() []string
}

package netease

import (
	"context"
	"fmt"
	"strings"

	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/models"
	"go.senan.xyz/taglib"
)

// setMusicTags sets ID3 tags for the downloaded music file
func (d *NMDownloader) setMusicTags(ctx context.Context, music *models.Music, filePath string) error {
	artistNames := make([]string, 0, len(music.Artist))
	for _, ar := range music.Artist {
		artistNames = append(artistNames, ar.Name)
	}

	err := taglib.WriteTags(filePath, map[string][]string{
		// Multi-valued tags allowed
		taglib.Artist:      artistNames,
		taglib.Album:       {music.Album.Name},
		taglib.Title:       {music.Name},
		taglib.Length:      {fmt.Sprintf("%d", music.Time/1000)}, // convert milliseconds to seconds
		taglib.Lyrics:      {music.Lyrics},
		taglib.TrackNumber: {music.TrackNumber},
	}, 0)
	if err != nil {
		return fmt.Errorf("WriteTags: %w", err)
	}
	// picture tag
	data, err := d.downloadCover(ctx, music.Album.PicUrl)
	if err != nil {
		log.Warn("download cover err: %v", err)
		return nil // ignore picture download error
	}
	err = taglib.WriteImage(filePath, data)
	if err != nil {
		log.Warn("write image err: %v", err)
		return nil
	}
	return nil
}

// downloadCover downloads album cover art
func (d *NMDownloader) downloadCover(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("image url is empty")
	}

	// Create a buffer to store the downloaded data
	var buf strings.Builder

	// Download image data to buffer
	resp, err := d.cli.Download(ctx, url, nil, nil, &buf, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// Check response status
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return nil, fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}

	// Convert string buffer to bytes
	data := []byte(buf.String())

	// Basic validation - check if we got some data
	if len(data) == 0 {
		return nil, fmt.Errorf("downloaded image data is empty")
	}

	log.Debug("Downloaded image: %s, size: %d bytes", url, len(data))

	return data, nil
}

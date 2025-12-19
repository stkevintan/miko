package netease

import (
	"context"
	"fmt"

	nmTypes "github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
	"github.com/stkevintan/miko/internal/models"
)

// getSongDetail retrieves detailed information for a song
func (d *NMDownloader) getSongDetail(ctx context.Context, music *models.Music) (*weapi.SongDetailRespSongs, error) {
	resp, err := d.request.SongDetail(ctx, &weapi.SongDetailReq{
		C: []weapi.SongDetailReqList{{Id: music.SongId(), V: 0}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("SongDetail API error: %+v", resp)
	}
	if len(resp.Songs) == 0 {
		return nil, fmt.Errorf("song not found")
	}
	return &resp.Songs[0], nil
}

// getBestQuality gets the best available quality for a song
func (d *NMDownloader) getBestQuality(ctx context.Context, music *models.Music) (*nmTypes.Quality, nmTypes.Level, error) {
	qualityResp, err := d.request.SongMusicQuality(ctx, &weapi.SongMusicQualityReq{SongId: music.SongId()})
	if err != nil {
		return nil, "", fmt.Errorf("SongMusicQuality: %w", err)
	}
	if qualityResp.Code != 200 {
		return nil, "", fmt.Errorf("SongMusicQuality API error: %+v", qualityResp)
	}

	quality, level, ok := qualityResp.Data.Qualities.FindBetter(d.Level)
	if !ok {
		// log.Warn would be imported in downloader.go with the main logic
	}

	return quality, level, nil
}

// downloadLyrics downloads lyrics for a song
func (d *NMDownloader) downloadLyrics(ctx context.Context, id int64) (string, error) {
	lyricResp, err := d.request.Lyric(ctx, &weapi.LyricReq{Id: id})
	if err != nil {
		return "", fmt.Errorf("download lyric: %w", err)
	}
	if lyricResp.Code != 200 {
		return "", fmt.Errorf("download lyric API error: %+v", lyricResp)
	}
	return lyricResp.Lrc.Lyric, nil
}

package netease

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chaunsin/netease-cloud-music/api"
	nmTypes "github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
)

// getSongDetail retrieves detailed information for a song
// func (d *NMDownloader) getSongDetail(ctx context.Context, id string) (*weapi.SongDetailRespSongs, error) {
// 	resp, err := d.request.SongDetail(ctx, &weapi.SongDetailReq{
// 		C: []weapi.SongDetailReqList{{Id: id, V: 0}},
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	if resp.Code != 200 {
// 		return nil, fmt.Errorf("SongDetail API error: %+v", resp)
// 	}
// 	if len(resp.Songs) == 0 {
// 		return nil, fmt.Errorf("song not found")
// 	}
// 	return &resp.Songs[0], nil
// }

// getBestQuality gets the best available quality for a song
func (d *NMProvider) getBestQuality(ctx context.Context, id string, level nmTypes.Level) (*nmTypes.Quality, nmTypes.Level, error) {
	qualityResp, err := d.request.SongMusicQuality(ctx, &weapi.SongMusicQualityReq{SongId: id})
	if err != nil {
		return nil, "", fmt.Errorf("SongMusicQuality: %w", err)
	}
	if qualityResp.Code != 200 {
		return nil, "", fmt.Errorf("SongMusicQuality API error: %+v", qualityResp)
	}

	quality, level, ok := qualityResp.Data.Qualities.FindBetter(level)
	if !ok {
		// log.Warn would be imported in downloader.go with the main logic
	}

	return quality, level, nil
}

// getLyrics downloads lyrics for a song
func (d *NMProvider) getLyrics(ctx context.Context, id int64) (string, error) {
	lyricResp, err := d.request.Lyric(ctx, &weapi.LyricReq{Id: id})
	if err != nil {
		return "", fmt.Errorf("download lyric: %w", err)
	}
	if lyricResp.Code != 200 {
		return "", fmt.Errorf("download lyric API error: %+v", lyricResp)
	}
	return lyricResp.Lrc.Lyric, nil
}

// fetchDownloadInfo retrieves download information for a song
func (d *NMProvider) fetchDownloadInfo(ctx context.Context, musicId int64, bitrate int64, level nmTypes.Level) (*SongDownloadInfo, error) {
	downResp, err := d.request.SongDownloadUrl(ctx, &weapi.SongDownloadUrlReq{
		Id: fmt.Sprintf("%d", musicId),
		Br: fmt.Sprintf("%d", bitrate),
	})
	if err != nil {
		return nil, fmt.Errorf("SongDownloadUrl: %w", err)
	}
	if downResp.Code != 200 {
		return nil, fmt.Errorf("SongDownloadUrl API error: %+v", downResp)
	}

	data := downResp.Data

	if data.Code != 200 || data.Url == "" {
		switch data.Code {
		case -110:
			return nil, fmt.Errorf("no audio source available")
		case -105:
			return nil, fmt.Errorf("insufficient permissions or no membership")
		case -103:
			alInfo, err := d.getDownloadAlternativeData(ctx, musicId, level)
			if err != nil {
				return nil, fmt.Errorf("getDownloadAlternativeData: %w", err)
			}
			if alInfo.Url == "" {
				return nil, fmt.Errorf("no audio source available in alternative data")
			}
			return alInfo, nil
		default:
			return nil, fmt.Errorf("resource unavailable or no copyright (code: %v)", data.Code)
		}
	}

	return &SongDownloadInfo{
		Id:         data.Id,
		Url:        data.Url,
		Md5:        data.Md5,
		Level:      data.Level,
		Type:       data.Type,
		Size:       data.Size,
		Br:         data.Br,
		EncodeType: data.EncodeType,
		Fee:        data.Fee,
	}, nil
}

// getDownloadAlternativeData fetches alternative download data when primary source fails
func (d *NMProvider) getDownloadAlternativeData(ctx context.Context, musicId int64, level nmTypes.Level) (*SongDownloadInfo, error) {
	var (
		url   = "https://music.163.com/weapi/song/enhance/player/url/v1"
		reply SongPlayerInfoRes
		opts  = api.NewOptions()
	)
	Ids := []int64{musicId}
	// json
	IdsBytes, _ := json.Marshal(Ids)
	resp, err := d.cli.Request(ctx, url, &SongPlayerInfoReq{
		Ids:        string(IdsBytes),
		Level:      level,
		EncodeType: "flac",
	}, &reply, opts)

	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	_ = resp
	return &reply.Data[0], nil
}

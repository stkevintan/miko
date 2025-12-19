package netease

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
	"github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/chaunsin/netease-cloud-music/pkg/utils"
	"github.com/stkevintan/miko/internal/models"
)

// GetMusic returns the music information array
func (d *NMDownloader) GetMusic(ctx context.Context, uris []string) ([]*models.Music, error) {
	var (
		source = make(map[string][]int64)
		set    = make(map[int64]struct{})
		musics []*models.Music
	)

	for _, uri := range uris {
		kind, id, err := ParseURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse uri: %w", err)
		}
		if v, ok := source[kind]; ok {
			source[kind] = append(v, id)
		} else {
			source[kind] = []int64{id}
		}
	}

	for k, ids := range source {
		switch k {
		case "song":
			{
				var tmp = make([]int64, 0, len(ids))
				for _, id := range ids {
					if _, ok := set[id]; ok {
						continue
					}
					set[id] = struct{}{}
					tmp = append(tmp, id)
				}

				// Process in batches of 500
				pages, _ := utils.SplitSlice(tmp, 500)
				for _, p := range pages {
					var c = make([]weapi.SongDetailReqList, 0, len(p))
					for _, v := range p {
						c = append(c, weapi.SongDetailReqList{Id: fmt.Sprintf("%v", v), V: 0})
					}
					resp, err := d.request.SongDetail(ctx, &weapi.SongDetailReq{C: c})
					if err != nil {
						return nil, fmt.Errorf("SongDetail: %w", err)
					}
					if resp.Code != 200 {
						return nil, fmt.Errorf("SongDetail err: %+v", resp)
					}
					if len(resp.Songs) <= 0 {
						log.Warn("SongDetail() Songs is empty")
						continue
					}
					for _, v := range resp.Songs {
						musics = append(musics, &models.Music{
							Id:          v.Id,
							Name:        v.Name,
							Artist:      v.Ar,
							Album:       v.Al,
							Time:        v.Dt,
							TrackNumber: v.Cd,
						})
					}
				}
			}
		case "album":
			for _, id := range ids {
				album, err := d.request.Album(ctx, &weapi.AlbumReq{Id: fmt.Sprintf("%d", id)})
				if err != nil {
					return nil, fmt.Errorf("album(%v): %w", id, err)
				}
				if album.Code != 200 {
					return nil, fmt.Errorf("album(%v) err: %+v", id, album)
				}
				if len(album.Songs) <= 0 {
					log.Warn("Album(%v) Songs is empty", id)
					continue
				}
				for _, v := range album.Songs {
					if _, ok := set[v.Id]; ok {
						continue
					}
					set[v.Id] = struct{}{}
					musics = append(musics, &models.Music{
						Id:          v.Id,
						Name:        v.Name,
						Artist:      v.Ar,
						Album:       v.Al,
						Time:        v.Dt,
						TrackNumber: v.Cd,
					})
				}
			}
		case "playlist":
			for _, id := range ids {
				playlist, err := d.request.PlaylistDetail(ctx, &weapi.PlaylistDetailReq{Id: fmt.Sprintf("%d", id)})
				if err != nil {
					return nil, fmt.Errorf("PlaylistDetail(%v): %w", id, err)
				}
				if playlist.Code != 200 {
					return nil, fmt.Errorf("PlaylistDetail(%v) err: %+v", id, playlist)
				}
				if playlist.Playlist.TrackIds == nil {
					log.Warn("PlaylistDetail(%v) Tracks is nil", id)
					continue
				}
				var tmp = make([]int64, 0, len(playlist.Playlist.TrackIds))
				for _, v := range playlist.Playlist.TrackIds {
					if _, ok := set[v.Id]; ok {
						continue
					}
					set[v.Id] = struct{}{}
					tmp = append(tmp, v.Id)
				}

				// Process in batches
				pages, _ := utils.SplitSlice(tmp, 500)
				for _, p := range pages {
					var c = make([]weapi.SongDetailReqList, 0, len(p))
					for _, v := range p {
						c = append(c, weapi.SongDetailReqList{Id: fmt.Sprintf("%v", v), V: 0})
					}
					resp, err := d.request.SongDetail(ctx, &weapi.SongDetailReq{C: c})
					if err != nil {
						return nil, fmt.Errorf("SongDetail: %w", err)
					}
					if resp.Code != 200 {
						return nil, fmt.Errorf("SongDetail err: %+v", resp)
					}
					if len(resp.Songs) <= 0 {
						log.Warn("SongDetail Songs is empty")
						continue
					}
					for _, v := range resp.Songs {
						musics = append(musics, &models.Music{
							Id:          v.Id,
							Name:        v.Name,
							Artist:      v.Ar,
							Album:       v.Al,
							Time:        v.Dt,
							TrackNumber: v.Cd,
						})
					}
				}
			}
		default:
			return nil, fmt.Errorf("[%s] is not supported", k)
		}
	}

	if len(musics) <= 0 {
		return nil, fmt.Errorf("input uri is empty or the song is copyrighted")
	}

	return musics, nil
}

// fetchDownloadInfo retrieves download information for a song
func (d *NMDownloader) fetchDownloadInfo(ctx context.Context, music *models.Music, bitrate int64) (*SongDownloadInfo, error) {
	downResp, err := d.request.SongDownloadUrl(ctx, &weapi.SongDownloadUrlReq{
		Id: music.SongId(),
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
			alInfo, err := d.getDownloadAlternativeData(ctx, music)
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
func (d *NMDownloader) getDownloadAlternativeData(ctx context.Context, music *models.Music) (*SongDownloadInfo, error) {
	var (
		url   = "https://music.163.com/weapi/song/enhance/player/url/v1"
		reply SongPlayerInfoRes
		opts  = api.NewOptions()
	)
	Ids := []int64{music.Id}
	// json
	IdsBytes, _ := json.Marshal(Ids)
	resp, err := d.cli.Request(ctx, url, &SongPlayerInfoReq{
		Ids:        string(IdsBytes),
		Level:      d.Level,
		EncodeType: "aac",
	}, &reply, opts)

	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	_ = resp
	return &reply.Data[0], nil
}

package netease

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/chaunsin/netease-cloud-music/api/weapi"
	"github.com/chaunsin/netease-cloud-music/pkg/utils"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/types"
)

var (
	urlPattern = "/(song|artist|album|playlist)\\?id=(\\d+)"
	reg        = regexp.MustCompile(urlPattern)
)

// parseURI parses a NetEase music URI and returns the type and ID
func parseURI(source string) (string, int64, error) {
	// 歌曲id
	id, err := strconv.ParseInt(source, 10, 64)
	if err == nil {
		return "song", id, nil
	}

	if !strings.Contains(source, "music.163.com") {
		return "", 0, fmt.Errorf("could not parse the url: %s", source)
	}

	matched, ok := reg.FindStringSubmatch(source), reg.MatchString(source)
	if !ok || len(matched) < 3 {
		return "", 0, fmt.Errorf("could not parse the url: %s", source)
	}

	id, err = strconv.ParseInt(matched[2], 10, 64)
	if err != nil {
		return "", 0, err
	}
	return matched[1], id, nil
}

// GetMusic returns the music information array
func (d *NMProvider) GetMusic(ctx context.Context, uris []string) ([]*types.Music, error) {
	var (
		source = make(map[string][]int64)
		set    = make(map[int64]struct{})
		musics []*types.Music
	)

	for _, uri := range uris {
		kind, id, err := parseURI(uri)
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
						artist := make([]types.Artist, 0, len(v.Ar))
						for _, a := range v.Ar {
							artist = append(artist, types.Artist{
								Id:   a.Id,
								Name: a.Name,
							})
						}
						album := types.Album{
							Id:     v.Al.Id,
							Name:   v.Al.Name,
							PicUrl: v.Al.PicUrl,
						}
						musics = append(musics, &types.Music{
							Id:          v.Id,
							Name:        v.Name,
							Artist:      artist,
							Album:       album,
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
					artist := make([]types.Artist, 0, len(v.Ar))
					for _, a := range v.Ar {
						artist = append(artist, types.Artist{
							Id:   a.Id,
							Name: a.Name,
						})
					}
					album := types.Album{
						Id:     v.Al.Id,
						Name:   v.Al.Name,
						PicUrl: v.Al.PicUrl,
					}
					musics = append(musics, &types.Music{
						Id:          v.Id,
						Name:        v.Name,
						Artist:      artist,
						Album:       album,
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
						artist := make([]types.Artist, 0, len(v.Ar))
						for _, a := range v.Ar {
							artist = append(artist, types.Artist{
								Id:   a.Id,
								Name: a.Name,
							})
						}
						album := types.Album{
							Id:     v.Al.Id,
							Name:   v.Al.Name,
							PicUrl: v.Al.PicUrl,
						}
						musics = append(musics, &types.Music{
							Id:          v.Id,
							Name:        v.Name,
							Artist:      artist,
							Album:       album,
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

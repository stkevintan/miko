package downloader

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httputil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
	"github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/chaunsin/netease-cloud-music/pkg/utils"
	"github.com/stkevintan/miko/internal/models"
	"go.senan.xyz/taglib"
	"golang.org/x/sync/semaphore"
)

type NMDownloader struct {
	cli            *api.Client
	request        *weapi.Api
	Level          types.Level
	Output         string
	ConflictPolicy ConflictPolicy
}

// Ensure NMDownloader implements the Downloader interface
var _ Downloader = (*NMDownloader)(nil)

var (
	urlPattern = "/(song|artist|album|playlist)\\?id=(\\d+)"
	reg        = regexp.MustCompile(urlPattern)
)

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
func (d *NMDownloader) GetMusic(ctx context.Context, uris []string) ([]*models.Music, error) {
	var (
		source = make(map[string][]int64)
		set    = make(map[int64]struct{})
		musics []*models.Music
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

// GetLevel returns the quality level
func (d *NMDownloader) GetLevel() types.Level {
	return d.Level
}

// GetOutput returns the output directory
func (d *NMDownloader) GetOutput() string {
	return d.Output
}

// GetConflictPolicy returns the conflict handling policy
func (d *NMDownloader) GetConflictPolicy() ConflictPolicy {
	return d.ConflictPolicy
}

func (d *NMDownloader) Close(ctx context.Context) error {
	refresh, err := d.request.TokenRefresh(ctx, &weapi.TokenRefreshReq{})
	if err != nil || refresh.Code != 200 {
		log.Warn("TokenRefresh resp:%+v err: %s", refresh, err)
	}
	return d.cli.Close(ctx)
}

// NewBatchDownloader creates a new NMDownloader for multiple songs
func NewDownloader(config *DownloaderConfig) (*NMDownloader, error) {
	// Validate basic config
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate and parse conflict policy
	policy, err := ParseConflictPolicy(config.ConflictPolicy)
	if err != nil {
		return nil, fmt.Errorf("invalid conflict policy: %w", err)
	}

	cli := api.New(config.Root.NmApi)
	request := weapi.New(cli)

	// Validate and parse level
	dlevel, err := validateQualityLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid quality level: %w", err)
	}

	return &NMDownloader{
		cli:            cli,
		request:        request,
		Level:          dlevel,
		Output:         config.Output,
		ConflictPolicy: policy,
	}, nil
}

// NewNetEaseDownloader creates a new NetEase downloader that implements the Downloader interface
func NewNetEaseDownloader(config *DownloaderConfig) (Downloader, error) {
	return NewDownloader(config)
}

// Download downloads multiple songs concurrently
func (d *NMDownloader) Download(ctx context.Context, musics []*models.Music) (*models.BatchDownloadResponse, error) {
	var (
		total   = int64(len(musics))
		success = &atomic.Int64{}
		failed  = &atomic.Int64{}
		sema    = semaphore.NewWeighted(5) // parallel download count
		results = make([]*models.DownloadResponse, 0, total)
		errors  = make([]string, 0)
		mutex   sync.Mutex
	)

	// Process songs concurrently
	for _, music := range musics {
		var music = music // capture loop variable
		if err := sema.Acquire(ctx, 1); err != nil {
			return nil, fmt.Errorf("acquire: %w", err)
		}
		go func() {
			defer sema.Release(1)

			result, err := d.downloadSingle(ctx, music)
			if err != nil {
				failed.Add(1)
				mutex.Lock()
				errors = append(errors, fmt.Sprintf("download %s err: %v", music.String(), err))
				mutex.Unlock()
				log.Error("download %s err: %v", music.String(), err)
				return
			}
			success.Add(1)
			mutex.Lock()
			results = append(results, result)
			mutex.Unlock()
		}()
	}

	// Wait for all downloads to complete
	if err := sema.Acquire(ctx, 5); err != nil {
		return nil, fmt.Errorf("wait: %w", err)
	}

	return &models.BatchDownloadResponse{
		Total:   total,
		Success: success.Load(),
		Failed:  failed.Load(),
		Songs:   results,
		Errors:  errors,
	}, nil
}

// downloadSingle downloads a single music item
func (d *NMDownloader) downloadSingle(ctx context.Context, music *models.Music) (*models.DownloadResponse, error) {
	if music == nil {
		return nil, fmt.Errorf("music is required")
	}

	// Get song details first
	songDetail, err := d.getSongDetail(ctx, music)
	if err != nil {
		return nil, fmt.Errorf("failed to get song details: %w", err)
	}

	// Get available qualities for the song
	quality, actualLevel, err := d.getBestQuality(ctx, music)
	if err != nil {
		return nil, fmt.Errorf("failed to get song quality: %w", err)
	}

	// Get download URL
	downloadInfo, err := d.fetchDownloadInfo(ctx, music, quality.Br)
	if err != nil {
		return nil, fmt.Errorf("failed to get download URL: %w", err)
	}
	if downloadInfo.Type == "" {
		// check if the ext type is on the song name
		ext := path.Ext(music.Name)
		if ext != "" {
			downloadInfo.Type = ext[1:] // remove the dot
			music.Name = music.Name[:len(music.Name)-len(ext)]
		} else {
			// TODO: probe the file type
			downloadInfo.Type = "mp3" // default to mp3
		}
	}
	lyric, err := d.downloadLyrics(ctx, music.Id)
	if err != nil {
		log.Warn("download lyric err: %v", err)
	}
	music.Lyrics = lyric

	var dest string
	if d.Output != "" {
		// Download to local file
		dest, proceed, err := d.downloadToLocal(ctx, music, downloadInfo, quality, actualLevel)
		if err != nil {
			return nil, fmt.Errorf("failed to download to local: %w", err)
		}
		if proceed {
			// set music tags
			err = d.setMusicTags(ctx, music, dest)
			if err != nil {
				log.Warn("setMusicTags %s err: %v", dest, err)
			}
		}
	}

	// Return download result
	result := &models.DownloadResponse{
		SongID:         fmt.Sprintf("%d", music.Id),
		SongName:       songDetail.Name,
		Artist:         formatArtists(songDetail.Ar),
		Album:          songDetail.Al.Name,
		AlPicUrl:       songDetail.Al.PicUrl,
		DownloadURL:    downloadInfo.Url,
		DownloadedPath: dest,
		Quality:        types.LevelString[actualLevel],
		FileType:       downloadInfo.Type,
		FileSize:       downloadInfo.Size,
		Duration:       songDetail.Dt,
		Lyrics:         music.Lyrics,
	}

	return result, nil
}

func validateQualityLevel(level string) (types.Level, error) {
	if level == "" {
		return types.LevelHires, nil // default to hires
	}

	// Handle numeric levels
	if lv, err := strconv.ParseInt(string(level), 10, 64); err == nil {
		switch lv {
		case 128:
			return types.LevelStandard, nil
		case 192:
			return types.LevelHigher, nil
		case 320:
			return types.LevelExhigh, nil
		default:
			return "", fmt.Errorf("%v level is not supported", lv)
		}
	}

	// Handle string levels
	switch types.Level(level) {
	case types.LevelStandard,
		types.LevelHigher,
		types.LevelExhigh,
		types.LevelLossless,
		types.LevelHires:
		return types.Level(level), nil
	default:
		// Handle uppercase aliases
		switch strings.ToUpper(level) {
		case "HQ":
			return types.LevelExhigh, nil
		case "SQ":
			return types.LevelLossless, nil
		case "HR":
			return types.LevelHires, nil
		default:
			return "", fmt.Errorf("[%s] quality is not supported", level)
		}
	}
}

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

func (d *NMDownloader) getBestQuality(ctx context.Context, music *models.Music) (*types.Quality, types.Level, error) {
	qualityResp, err := d.request.SongMusicQuality(ctx, &weapi.SongMusicQualityReq{SongId: music.SongId()})
	if err != nil {
		return nil, "", fmt.Errorf("SongMusicQuality: %w", err)
	}
	if qualityResp.Code != 200 {
		return nil, "", fmt.Errorf("SongMusicQuality API error: %+v", qualityResp)
	}

	quality, level, ok := qualityResp.Data.Qualities.FindBetter(d.Level)
	if !ok {
		log.Warn("requested quality level not available for songID %d", music.Id)
	}

	return quality, level, nil
}

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

type SongPlayerInfoReq struct {
	Ids        string      `json:"ids"` // song id (separated by comma)
	Level      types.Level `json:"level"`
	EncodeType string      `json:"encodeType,omitempty"`
}

type SongPlayerInfoRes struct {
	Code int                `json:"code"`
	Data []SongDownloadInfo `json:"data"`
}

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

/**

csrf_token: "791a8931ec96e749752e0e0cca3f138e"
encodeType: "aac"
ids: "[2619158763]"
level: "exhigh"
*/

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

// downloadToLocal downloads the song to a local file
// returns the file path, whether the file need proceed to tag, and an error if any
func (d *NMDownloader) downloadToLocal(ctx context.Context, music *models.Music, info *SongDownloadInfo, quality *types.Quality, level types.Level) (string, bool, error) {
	var (
		// drd = downResp.Data[0]
		dest     = filepath.Join(d.Output, music.Filename(info.Type, 0))
		tempName = fmt.Sprintf("download-*-%s.tmp", music.NameString())
	)

	conflicted := utils.FileExists(dest)

	// if dest exists, skip download
	if conflicted && d.ConflictPolicy == ConflictPolicySkip {
		log.Info("file %s already exists, skip download", dest)
		return dest, false, nil
	}

	if conflicted && d.ConflictPolicy == ConflictPolicyUpdateTags {
		log.Info("file %s already exists, update tags only", dest)
		return dest, true, nil
	}
	// 创建临时文件
	file, err := os.CreateTemp(d.Output, tempName)
	if err != nil {
		return "", false, fmt.Errorf("CreateTemp: %w", err)
	}
	defer file.Close()

	// 下载
	resp, err := d.cli.Download(ctx, info.Url, nil, nil, file, nil)
	if err != nil {
		_ = os.Remove(file.Name())
		return "", false, fmt.Errorf("download: %w", err)
	}
	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		log.Debug("DumpResponse err: %s", err)
	} else {
		log.Debug("Download DumpResponse: %s", dump)
	}

	size, _ := strconv.ParseFloat(resp.Header.Get("Content-Length"), 64)
	log.Debug("id=%v downloadUrl=%v wantLevel=%v-%v realLevel=%v-%v encodeType=%v type=%v size=%0.2fM,%vKB free=%v tempFile=%s outDir=%s",
		info.Id, info.Url, level, quality.Br, info.Level, info.Br, info.EncodeType, info.Type, size/float64(utils.MB), int64(size), types.Free(info.Fee), file.Name(), dest)

	// 校验md5文件完整性
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = os.Remove(file.Name())
		return "", false, fmt.Errorf("seek: %w", err)
	}
	var m = md5.New()
	if _, err := io.Copy(m, file); err != nil {
		_ = os.Remove(file.Name())
		return "", false, err
	}
	if m := hex.EncodeToString(m.Sum(nil)); m != info.Md5 {
		_ = os.Remove(file.Name())
		return "", false, fmt.Errorf("file %v md5 not match, want=%s, got=%s", file.Name(), info.Md5, m)
	}

	// 避免文件重名
	for i := 1; utils.FileExists(dest); i++ {
		if d.ConflictPolicy == ConflictPolicyOverwrite {
			_ = os.Remove(dest)
			break
		}
		if d.ConflictPolicy == ConflictPolicyRename {
			dest = filepath.Join(d.Output, music.Filename(info.Type, i))
		}
	}

	// 显示关闭文件避免Windows系统无法重命名错误: The process cannot access the file because it is being used by another process
	if err := file.Close(); err != nil {
		log.Error("close %s file err: %s", file.Name(), err)
		_ = os.Remove(file.Name())
	}
	if err := os.Rename(file.Name(), dest); err != nil {
		_ = os.Remove(file.Name())
		return "", false, fmt.Errorf("rename: %w", err)
	}
	if err := os.Chmod(dest, 0644); err != nil {
		return "", false, fmt.Errorf("chmod: %w", err)
	}

	return dest, true, nil
}

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

// https://p2.music.126.net/bfr3FRWXzPBIJSo2HCK2PA==/109951171924374807.jpg
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

	// Log response for debugging
	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		log.Debug("DumpResponse err: %s", err)
	} else {
		log.Debug("Picture download DumpResponse: %s", dump)
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

func formatArtists(artists []types.Artist) string {
	if len(artists) == 0 {
		return "Unknown Artist"
	}

	var names []string
	for _, artist := range artists {
		names = append(names, artist.Name)
	}
	return strings.Join(names, ", ")
}

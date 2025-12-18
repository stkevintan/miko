package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
	"github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/chaunsin/netease-cloud-music/pkg/utils"
	"github.com/stkevintan/miko/internal/models"
	"go.senan.xyz/taglib"
	"golang.org/x/sync/semaphore"
)

var (
	urlPattern = "/(song|artist|album|playlist)\\?id=(\\d+)"
	reg        = regexp.MustCompile(urlPattern)
)

// DownloadArgs represents internal download arguments for the service
type DownloadArgs struct {
	Music   *models.Music
	Level   string // quality level
	Output  string // output directory
	Timeout time.Duration
}

// DownloadResourceArgs represents download arguments for any resource type
type DownloadResourceArgs struct {
	Resource string // can be song ID, URL, etc.
	Level    string // quality level
	Output   string // output directory
	Timeout  time.Duration
}

// DownloadResult represents internal download result from the service
type DownloadResult struct {
	SongID         string
	SongName       string
	Artist         string
	Album          string
	AlPicUrl       string
	DownloadURL    string
	DownloadedPath string
	Quality        string
	FileType       string
	FileSize       int64
	Duration       int64
}

// BatchDownloadResult represents internal batch download result from the service
type BatchDownloadResult struct {
	Total   int64
	Success int64
	Failed  int64
	Songs   []DownloadResult
	Errors  []string
}

func (s *Service) Download(ctx context.Context, c *DownloadArgs) (*DownloadResult, error) {
	if c.Music == nil {
		return nil, fmt.Errorf("music is required")
	}

	// Set default values
	if c.Level == "" {
		c.Level = string(types.LevelLossless) // default to lossless
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	// Validate quality level
	level, err := s.validateQualityLevel(c.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid quality level: %w", err)
	}

	nctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	// Create API client
	cli := api.New(s.config.NmApi)
	defer cli.Close(nctx)

	request := weapi.New(cli)

	songId := fmt.Sprintf("%d", c.Music.Id)
	// Get song details first
	songDetail, err := s.getSongDetail(nctx, request, songId)
	if err != nil {
		return nil, fmt.Errorf("failed to get song details: %w", err)
	}

	// Get available qualities for the song
	quality, actualLevel, err := s.getBestQuality(nctx, request, songId, level)
	if err != nil {
		return nil, fmt.Errorf("failed to get song quality: %w", err)
	}

	// Get download URL
	downloadData, err := s.getDownloadURL(nctx, request, songId, quality.Br)
	if err != nil {
		return nil, fmt.Errorf("failed to get download URL: %w", err)
	}
	dest := ""
	if c.Output != "" {
		// Here you would normally handle the actual downloading and saving to c.Output
		// download it!
		dest, err = s.downloadToLocal(nctx, cli, downloadData, c.Output, c.Music, quality, actualLevel)
		if err != nil {
			log.Warn("failed to download to local: %v", err)
		}
	}

	result := &DownloadResult{
		SongID:         fmt.Sprintf("%d", c.Music.Id),
		SongName:       songDetail.Name,
		Artist:         s.formatArtists(songDetail.Ar),
		Album:          songDetail.Al.Name,
		AlPicUrl:       songDetail.Al.PicUrl,
		DownloadURL:    downloadData.Url,
		DownloadedPath: dest,
		Quality:        types.LevelString[actualLevel],
		FileType:       downloadData.Type,
		FileSize:       downloadData.Size,
		Duration:       songDetail.Dt,
	}

	return result, nil
}

// downloadWithClient is a helper function that uses a shared API client to avoid cookie file conflicts
func (s *Service) downloadWithClient(ctx context.Context, c *DownloadArgs, cli *api.Client) (*DownloadResult, error) {
	if c.Music == nil {
		return nil, fmt.Errorf("music is required")
	}

	// Set default values
	if c.Level == "" {
		c.Level = string(types.LevelLossless) // default to lossless
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	// Validate quality level
	level, err := s.validateQualityLevel(c.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid quality level: %w", err)
	}

	nctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	request := weapi.New(cli)

	songId := fmt.Sprintf("%d", c.Music.Id)
	// Get song details first
	songDetail, err := s.getSongDetail(nctx, request, songId)
	if err != nil {
		return nil, fmt.Errorf("failed to get song details: %w", err)
	}

	// Get available qualities for the song
	quality, actualLevel, err := s.getBestQuality(nctx, request, songId, level)
	if err != nil {
		return nil, fmt.Errorf("failed to get song quality: %w", err)
	}

	// Get download URL
	downloadData, err := s.getDownloadURL(nctx, request, songId, quality.Br)
	if err != nil {
		return nil, fmt.Errorf("failed to get download URL: %w", err)
	}

	var dest string
	if c.Output != "" {
		// Download to local file
		dest, err = s.downloadToLocal(nctx, cli, downloadData, c.Output, c.Music, quality, actualLevel)
		if err != nil {
			return nil, fmt.Errorf("failed to download to local: %w", err)
		}
	}

	// Return download result
	result := &DownloadResult{
		SongID:         fmt.Sprintf("%d", c.Music.Id),
		SongName:       songDetail.Name,
		Artist:         s.formatArtists(songDetail.Ar),
		Album:          songDetail.Al.Name,
		AlPicUrl:       songDetail.Al.PicUrl,
		DownloadURL:    downloadData.Url,
		DownloadedPath: dest,
		Quality:        types.LevelString[actualLevel],
		FileType:       downloadData.Type,
		FileSize:       downloadData.Size,
		Duration:       songDetail.Dt,
	}

	return result, nil
}

func (s *Service) validateQualityLevel(level string) (types.Level, error) {
	// Handle numeric levels
	if lv, err := strconv.ParseInt(level, 10, 64); err == nil {
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

func (s *Service) getSongDetail(ctx context.Context, request *weapi.Api, songID string) (*weapi.SongDetailRespSongs, error) {
	resp, err := request.SongDetail(ctx, &weapi.SongDetailReq{
		C: []weapi.SongDetailReqList{{Id: songID, V: 0}},
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

func (s *Service) getBestQuality(ctx context.Context, request *weapi.Api, songID string, requestedLevel types.Level) (*types.Quality, types.Level, error) {
	qualityResp, err := request.SongMusicQuality(ctx, &weapi.SongMusicQualityReq{SongId: songID})
	if err != nil {
		return nil, "", fmt.Errorf("SongMusicQuality: %w", err)
	}
	if qualityResp.Code != 200 {
		return nil, "", fmt.Errorf("SongMusicQuality API error: %+v", qualityResp)
	}

	quality, level, ok := qualityResp.Data.Qualities.FindBetter(requestedLevel)
	if !ok {
		log.Warn("requested quality level not available for songID %s", songID)
	}

	return quality, level, nil
}

func (s *Service) getDownloadURL(ctx context.Context, request *weapi.Api, songID string, bitrate int64) (*weapi.SongDownloadUrlRespData, error) {
	downResp, err := request.SongDownloadUrl(ctx, &weapi.SongDownloadUrlReq{
		Id: songID,
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
		default:
			return nil, fmt.Errorf("resource unavailable or no copyright (code: %v)", data.Code)
		}
	}

	return &data, nil
}

func (s *Service) formatArtists(artists []types.Artist) string {
	if len(artists) == 0 {
		return "Unknown Artist"
	}

	var names []string
	for _, artist := range artists {
		names = append(names, artist.Name)
	}
	return strings.Join(names, ", ")
}

// DownloadFromResource handles downloading from various resource types (ID, URL, etc.)
func (s *Service) DownloadFromResource(ctx context.Context, c *DownloadResourceArgs) (*BatchDownloadResult, error) {
	if c.Resource == "" {
		return nil, fmt.Errorf("resource is required")
	}

	// Set default values
	if c.Level == "" {
		c.Level = string(types.LevelLossless)
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	nctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cli := api.New(s.config.NmApi)
	defer cli.Close(nctx)

	request := weapi.New(cli)

	// Check if login is required
	if request.NeedLogin(nctx) {
		return nil, fmt.Errorf("login required for download")
	}

	defer func() {
		refresh, err := request.TokenRefresh(ctx, &weapi.TokenRefreshReq{})
		if err != nil || refresh.Code != 200 {
			log.Warn("TokenRefresh resp:%+v err: %s", refresh, err)
		}
	}()

	// Parse input to get songs
	songs, err := s.inputParse(nctx, strings.Split(c.Resource, ","), request)
	if err != nil {
		return nil, fmt.Errorf("inputParse: %w", err)
	}

	if len(songs) == 0 {
		return nil, fmt.Errorf("no songs found in resource")
	}

	// Handle batch downloads with concurrency
	return s.downloadSongsBatch(nctx, songs, c.Level, c.Output, c.Timeout)
}

// downloadSongsBatch downloads multiple songs concurrently following the reference pattern
func (s *Service) downloadSongsBatch(ctx context.Context, songs []models.Music, level string, output string, timeout time.Duration) (*BatchDownloadResult, error) {
	// Create a shared API client for all downloads to avoid cookie file conflicts
	cli := api.New(s.config.NmApi)
	defer cli.Close(ctx)

	var (
		total   = int64(len(songs))
		failed  atomic.Int64
		success atomic.Int64
		sema    = semaphore.NewWeighted(5) // parallel download count
		results = make([]DownloadResult, 0, total)
		errors  = make([]string, 0)
		mutex   sync.Mutex
	)

	// Process songs concurrently
	for _, song := range songs {
		var song = song // capture loop variable
		if err := sema.Acquire(ctx, 1); err != nil {
			return nil, fmt.Errorf("acquire: %w", err)
		}
		go func() {
			defer sema.Release(1)
			args := &DownloadArgs{
				Music:   &song,
				Level:   level,
				Timeout: timeout,
				Output:  output,
			}

			result, err := s.downloadWithClient(ctx, args, cli)
			if err != nil {
				failed.Add(1)
				mutex.Lock()
				errors = append(errors, fmt.Sprintf("download %s err: %v", song.String(), err))
				mutex.Unlock()
				log.Error("download %s err: %v", song.String(), err)
				return
			}
			success.Add(1)
			mutex.Lock()
			results = append(results, *result)
			mutex.Unlock()
		}()
	}

	// Wait for all downloads to complete
	if err := sema.Acquire(ctx, 5); err != nil {
		return nil, fmt.Errorf("wait: %w", err)
	}

	return &BatchDownloadResult{
		Total:   total,
		Success: success.Load(),
		Failed:  failed.Load(),
		Songs:   results,
		Errors:  errors,
	}, nil
}

// inputParse parses input resources and returns list of songs
func (s *Service) inputParse(ctx context.Context, args []string, request *weapi.Api) ([]models.Music, error) {
	var (
		source = make(map[string][]int64)
		set    = make(map[int64]struct{})
		list   []models.Music
	)

	for _, arg := range args {
		kind, id, err := s.Parse(arg)
		if err != nil {
			return nil, fmt.Errorf("Parse: %w", err)
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
					resp, err := request.SongDetail(ctx, &weapi.SongDetailReq{C: c})
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
						list = append(list, models.Music{
							Id:     v.Id,
							Name:   v.Name,
							Artist: v.Ar,
							Album:  v.Al,
							Time:   v.Dt,
						})
					}
				}
			}
		case "album":
			for _, id := range ids {
				album, err := request.Album(ctx, &weapi.AlbumReq{Id: fmt.Sprintf("%d", id)})
				if err != nil {
					return nil, fmt.Errorf("Album(%v): %w", id, err)
				}
				if album.Code != 200 {
					return nil, fmt.Errorf("Album(%v) err: %+v", id, album)
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
					list = append(list, models.Music{
						Id:     v.Id,
						Name:   v.Name,
						Artist: v.Ar,
						Album:  v.Al,
						Time:   v.Dt,
					})
				}
			}
		case "playlist":
			for _, id := range ids {
				playlist, err := request.PlaylistDetail(ctx, &weapi.PlaylistDetailReq{Id: fmt.Sprintf("%d", id)})
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
					resp, err := request.SongDetail(ctx, &weapi.SongDetailReq{C: c})
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
						list = append(list, models.Music{
							Id:     v.Id,
							Name:   v.Name,
							Artist: v.Ar,
							Album:  v.Al,
							Time:   v.Dt,
						})
					}
				}
			}
		default:
			return nil, fmt.Errorf("[%s] is not supported", k)
		}
	}

	if len(list) <= 0 {
		return nil, fmt.Errorf("input resource is empty or the song is copyrighted")
	}
	return list, nil
}

// Parse parses input and returns resource kind and ID
func (s *Service) Parse(source string) (string, int64, error) {
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

func (s *Service) downloadToLocal(ctx context.Context, cli *api.Client, drd *weapi.SongDownloadUrlRespData, output string, music *models.Music, quality *types.Quality, level types.Level) (string, error) {
	var (
		// drd = downResp.Data[0]
		dest     = filepath.Join(output, fmt.Sprintf("%s - %s.%s", music.ArtistString(), music.NameString(), strings.ToLower(drd.Type)))
		tempName = fmt.Sprintf("download-*-%s.tmp", music.NameString())
	)

	// if dest exists, skip download
	if utils.FileExists(dest) {
		log.Info("file %s already exists, skip download", dest)
		return dest, nil
	}

	// 创建临时文件
	file, err := os.CreateTemp(output, tempName)
	if err != nil {
		return "", fmt.Errorf("CreateTemp: %w", err)
	}
	defer file.Close()

	// 下载
	resp, err := cli.Download(ctx, drd.Url, nil, nil, file, nil)
	if err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("download: %w", err)
	}
	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		log.Debug("DumpResponse err: %s", err)
	} else {
		log.Debug("Download DumpResponse: %s", dump)
	}

	size, _ := strconv.ParseFloat(resp.Header.Get("Content-Length"), 64)
	log.Debug("id=%v downloadUrl=%v wantLevel=%v-%v realLevel=%v-%v encodeType=%v type=%v size=%0.2fM,%vKB free=%v tempFile=%s outDir=%s",
		drd.Id, drd.Url, level, quality.Br, drd.Level, drd.Br, drd.EncodeType, drd.Type, size/float64(utils.MB), int64(size), types.Free(drd.Fee), file.Name(), dest)

	// 校验md5文件完整性
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("seek: %w", err)
	}
	var m = md5.New()
	if _, err := io.Copy(m, file); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	if m := hex.EncodeToString(m.Sum(nil)); m != drd.Md5 {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("file %v md5 not match, want=%s, got=%s", file.Name(), drd.Md5, m)
	}

	// 设置歌曲tag值
	if err := s.setMusicTag(file.Name(), music); err != nil {
		log.Warn("setMusicTag %s err: %v", file.Name(), err)
	}

	// 避免文件重名
	for i := 1; utils.FileExists(dest); i++ {
		dest = filepath.Join(output, fmt.Sprintf("%s - %s(%d).%s", music.ArtistString(), music.NameString(), i, strings.ToLower(drd.Type)))
	}
	// 显示关闭文件避免Windows系统无法重命名错误: The process cannot access the file because it is being used by another process
	if err := file.Close(); err != nil {
		log.Error("close %s file err: %s", file.Name(), err)
		_ = os.Remove(file.Name())
	}
	if err := os.Rename(file.Name(), dest); err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("rename: %w", err)
	}
	if err := os.Chmod(dest, 0644); err != nil {
		return "", fmt.Errorf("chmod: %w", err)
	}
	return dest, nil
}

func (s *Service) setMusicTag(filePath string, music *models.Music) error {
	artistNames := make([]string, 0, len(music.Artist))
	for _, ar := range music.Artist {
		artistNames = append(artistNames, ar.Name)
	}
	err := taglib.WriteTags(filePath, map[string][]string{
		// Multi-valued tags allowed
		taglib.AlbumArtist: artistNames,
		taglib.Album:       {music.Album.Name},
		taglib.Title:       {music.Name},
		taglib.Length:      {fmt.Sprintf("%d", music.Time/1000)}, // convert milliseconds to seconds
	}, 0)
	if err != nil {
		return fmt.Errorf("WriteTags: %w", err)
	}
	// picture tag
	data, err := s.downloadPicture(context.Background(), api.New(s.config.NmApi), music.Album.PicUrl)
	if err != nil {
		log.Warn("downloadPicture err: %v", err)
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
func (s *Service) downloadPicture(ctx context.Context, cli *api.Client, url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("image url is empty")
	}

	// Create a buffer to store the downloaded data
	var buf strings.Builder

	// Download image data to buffer
	resp, err := cli.Download(ctx, url, nil, nil, &buf, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// Log response for debugging
	if s.config.NmApi.Debug {
		dump, err := httputil.DumpResponse(resp, false)
		if err != nil {
			log.Debug("DumpResponse err: %s", err)
		} else {
			log.Debug("Picture download DumpResponse: %s", dump)
		}
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

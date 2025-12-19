package netease

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http/httputil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"

	nmTypes "github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/pkg/utils"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/types"
	"golang.org/x/sync/semaphore"
)

// Download downloads multiple songs concurrently
func (d *NMDownloader) Download(ctx context.Context, musics []*types.Music) (*types.MusicDownloadResults, error) {
	var (
		total = int64(len(musics))
		// success = &atomic.Int64{}
		// failed  = &atomic.Int64{}
		sema    = semaphore.NewWeighted(5) // parallel download count
		results = &types.MusicDownloadResults{
			Results: make([]*types.DownloadResult, 0, total),
		}
		mutex sync.Mutex
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
				mutex.Lock()
				results.Add(&types.DownloadResult{
					Err: fmt.Errorf("download %s: %w", music.String(), err),
				})
				mutex.Unlock()
				log.Error("download %s err: %v", music.String(), err)
				return
			}
			mutex.Lock()
			results.Add(&types.DownloadResult{
				Data: result,
			})
			mutex.Unlock()
		}()
	}

	// Wait for all downloads to complete
	if err := sema.Acquire(ctx, 5); err != nil {
		return nil, fmt.Errorf("wait: %w", err)
	}
	return results, nil
}

// downloadSingle downloads a single music item
func (d *NMDownloader) downloadSingle(ctx context.Context, music *types.Music) (*types.DownloadedMusic, error) {
	if music == nil {
		return nil, fmt.Errorf("music is required")
	}
	musicId := music.SongId()

	// Get available qualities for the song
	quality, actualLevel, err := d.getBestQuality(ctx, musicId)
	if err != nil {
		return nil, fmt.Errorf("failed to get song quality: %w", err)
	}

	// Get download URL
	downloadInfo, err := d.fetchDownloadInfo(ctx, music.Id, quality.Br)
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
	lyric, err := d.getLyrics(ctx, music.Id)
	if err != nil {
		log.Warn("download lyric err: %v", err)
	}
	music.Lyrics = lyric

	var dest string
	if d.Output != "" {
		// Download to local file
		var proceed bool
		dest, proceed, err = d.downloadToLocal(ctx, music, downloadInfo, quality, actualLevel)
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
	result := &types.DownloadedMusic{
		Music: *music,
		DownloadInfo: types.DownloadInfo{
			URL:      downloadInfo.Url,
			FilePath: dest,
			Type:     downloadInfo.Type,
			Size:     downloadInfo.Size,
			Quality:  nmTypes.LevelString[actualLevel],
		},
	}

	return result, nil
}

// downloadToLocal downloads the song to a local file
// returns the file path, whether the file need proceed to tag, and an error if any
func (d *NMDownloader) downloadToLocal(ctx context.Context, music *types.Music, info *SongDownloadInfo, quality *nmTypes.Quality, level nmTypes.Level) (string, bool, error) {
	var (
		// drd = downResp.Data[0]
		dest     = filepath.Join(d.Output, music.Filename(info.Type, 0))
		tempName = fmt.Sprintf("download-*-%s.tmp", music.NameString())
	)

	conflicted := utils.FileExists(dest)

	// if dest exists, skip download
	if conflicted && d.ConflictPolicy == types.ConflictPolicySkip {
		log.Info("file %s already exists, skip download", dest)
		return dest, false, nil
	}

	if conflicted && d.ConflictPolicy == types.ConflictPolicyUpdateTags {
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
		info.Id, info.Url, level, quality.Br, info.Level, info.Br, info.EncodeType, info.Type, size/float64(utils.MB), int64(size), nmTypes.Free(info.Fee), file.Name(), dest)

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
		if d.ConflictPolicy == types.ConflictPolicyOverwrite {
			_ = os.Remove(dest)
			break
		}
		if d.ConflictPolicy == types.ConflictPolicyRename {
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

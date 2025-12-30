package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/tags"
	"gorm.io/gorm"
)

type Scanner struct {
	db           *gorm.DB
	cfg          *config.Config
	isScanning   atomic.Bool
	scanCount    atomic.Int64
	lastScanTime atomic.Int64
}

func New(db *gorm.DB, cfg *config.Config) *Scanner {
	return &Scanner{
		db:  db,
		cfg: cfg,
	}
}

type scanTask struct {
	path     string
	rootPath string
	folder   models.MusicFolder
	d        fs.DirEntry
}

type scanResult struct {
	child *models.Child
	tags  *tags.Tags
}

func (s *Scanner) IsScanning() bool {
	return s.isScanning.Load()
}

func (s *Scanner) ScanCount() int64 {
	return s.scanCount.Load()
}

func (s *Scanner) LastScanTime() int64 {
	return s.lastScanTime.Load()
}

func (s *Scanner) Scan(ctx context.Context, incremental bool) {
	if !s.isScanning.CompareAndSwap(false, true) {
		return
	}
	s.scanCount.Store(0)
	defer s.isScanning.Store(false)

	existingFiles := make(map[string]time.Time)
	if incremental {
		var files []struct {
			ID      string
			Created *time.Time
		}
		s.db.Model(&models.Child{}).Select("id, created").Where("is_dir = ?", false).Find(&files)
		for _, f := range files {
			if f.Created != nil {
				existingFiles[f.ID] = *f.Created
			}
		}
		log.Info("Incremental scan: loaded %d existing files", len(existingFiles))
	} else {
		log.Info("Full scan started")
	}

	cacheDir := GetCoverCacheDir(s.cfg)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Error("Failed to create cache directory %q: %v", cacheDir, err)
		return
	}

	seenIDs := &sync.Map{}

	numWorkers := max(runtime.NumCPU(), 4)
	taskChan := make(chan scanTask, numWorkers*10)
	resultChan := make(chan scanResult, numWorkers*10)
	var wg sync.WaitGroup

	// Workers
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				select {
				case <-ctx.Done():
					return
				default:
				}
				relPath, err := filepath.Rel(task.rootPath, task.path)
				if err != nil {
					log.Warn("Failed to get relative path for %q: %v", task.path, err)
					continue
				}
				id := GenerateID(task.rootPath, relPath)
				parentID := GetParentID(task.rootPath, relPath)

				if task.d.IsDir() {
					seenIDs.Store(id, true)
					child := &models.Child{
						ID:            id,
						Parent:        parentID,
						IsDir:         true,
						Title:         task.d.Name(),
						Path:          task.path,
						MusicFolderID: task.folder.ID,
					}
					resultChan <- scanResult{child: child}
					continue
				}

				// File processing
				if !IsAudioFile(task.path) {
					continue
				}

				info, err := task.d.Info()
				if err != nil {
					log.Warn("Failed to get file info for %q: %v", task.path, err)
					continue
				}
				modTime := info.ModTime()

				if incremental {
					if lastMod, ok := existingFiles[id]; ok {
						if !modTime.After(lastMod) {
							seenIDs.Store(id, true)
							continue
						}
					}
				}

				seenIDs.Store(id, true)
				contentType := GetContentType(task.path)
				child := &models.Child{
					ID:            id,
					Parent:        parentID,
					IsDir:         false,
					Title:         task.d.Name(),
					Path:          task.path,
					Size:          info.Size(),
					Suffix:        strings.TrimPrefix(filepath.Ext(task.path), "."),
					ContentType:   contentType,
					MusicFolderID: task.folder.ID,
					Created:       &modTime, // Corresponds to file modification time for incremental scans.
					// TODO: Add audiobook support
					Type: "music",
				}

				t, err := tags.Read(task.path)
				if err != nil {
					// Still add the child even if tags fail
					resultChan <- scanResult{child: child}
					continue
				}
				resultChan <- scanResult{child: child, tags: t}
			}
		}()
	}

	// Saver
	doneSaver := make(chan struct{})
	go func() {
		defer close(doneSaver)
		s.saveResults(resultChan, cacheDir)
	}()

	// Producer
	for _, rootPath := range s.cfg.Subsonic.Folders {
		var folder models.MusicFolder
		s.db.Where(models.MusicFolder{Path: rootPath}).Attrs(models.MusicFolder{Name: filepath.Base(rootPath)}).FirstOrCreate(&folder)

		filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Warn("Error accessing path %q: %v", path, err)
				return nil
			}
			select {
			case <-ctx.Done():
				return filepath.SkipAll
			default:
			}
			taskChan <- scanTask{path: path, rootPath: rootPath, folder: folder, d: d}
			return nil
		})
	}

	close(taskChan)
	wg.Wait()
	close(resultChan)
	<-doneSaver

	s.Prune(seenIDs)

	s.lastScanTime.Store(time.Now().Unix())
	log.Info("Scan completed. Total files: %d", s.scanCount.Load())
}

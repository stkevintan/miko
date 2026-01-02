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
	numWorkers   int
}

func New(db *gorm.DB, cfg *config.Config) *Scanner {
	return &Scanner{
		db:         db,
		cfg:        cfg,
		numWorkers: max(runtime.NumCPU(), 4),
	}
}

type ScanTask struct {
	Path     string
	RootPath string
	Folder   models.MusicFolder
	D        fs.DirEntry
}

type scanResult struct {
	path  string
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

func (s *Scanner) ScanAll(ctx context.Context, incremental bool) {
	if s.IsScanning() {
		return
	}

	taskChan := make(chan ScanTask, s.numWorkers*10)
	done := make(chan struct{})
	go func() {
		defer close(taskChan)
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
				case <-done:
					return filepath.SkipAll
				default:
				}
				taskChan <- ScanTask{Path: path, RootPath: rootPath, Folder: folder, D: d}
				return nil
			})
		}
	}()

	seenIDs := s.scan(ctx, incremental, taskChan)
	close(done)

	if seenIDs != nil {
		s.Prune(seenIDs)
	}

	s.lastScanTime.Store(time.Now().Unix())
	log.Info("Scan completed. Total files: %d", s.scanCount.Load())
}

func (s *Scanner) ScanPath(ctx context.Context, path string) {
	var rootPath string
	for _, folder := range s.cfg.Subsonic.Folders {
		if strings.HasPrefix(path, folder) {
			rootPath = folder
			break
		}
	}

	if rootPath == "" {
		log.Warn("Path %q is not in any music folder", path)
		return
	}

	var folder models.MusicFolder
	s.db.Where(models.MusicFolder{Path: rootPath}).First(&folder)

	taskChan := make(chan ScanTask, s.numWorkers)
	done := make(chan struct{})
	go func() {
		defer close(taskChan)
		filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return filepath.SkipAll
			case <-done:
				return filepath.SkipAll
			default:
			}
			taskChan <- ScanTask{Path: p, RootPath: rootPath, Folder: folder, D: d}
			return nil
		})
	}()

	s.scan(ctx, false, taskChan)
	close(done)
}

func (s *Scanner) scan(ctx context.Context, incremental bool, taskChan <-chan ScanTask) *sync.Map {
	if !s.isScanning.CompareAndSwap(false, true) {
		return nil
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
	}

	cacheDir := GetCoverCacheDir(s.cfg)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Error("Failed to create cache directory %q: %v", cacheDir, err)
		return nil
	}

	seenIDs := &sync.Map{}

	resultChan := make(chan scanResult, s.numWorkers*10)
	var wg sync.WaitGroup

	// Workers
	for range s.numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				select {
				case <-ctx.Done():
					return
				default:
				}
				relPath, err := filepath.Rel(task.RootPath, task.Path)
				if err != nil {
					log.Warn("Failed to get relative path for %q: %v", task.Path, err)
					continue
				}
				id := GenerateID(task.RootPath, relPath)
				parentID := GetParentID(task.RootPath, relPath)

				if task.D.IsDir() {
					seenIDs.Store(id, true)
					child := &models.Child{
						ID:            id,
						Parent:        parentID,
						IsDir:         true,
						Title:         task.D.Name(),
						Path:          task.Path,
						MusicFolderID: task.Folder.ID,
					}
					resultChan <- scanResult{path: task.Path, child: child}
					continue
				}

				// File processing
				if !IsAudioFile(task.Path) {
					continue
				}

				info, err := task.D.Info()
				if err != nil {
					log.Warn("Failed to get file info for %q: %v", task.Path, err)
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
				contentType := GetContentType(task.Path)
				child := &models.Child{
					ID:            id,
					Parent:        parentID,
					IsDir:         false,
					Title:         task.D.Name(),
					Path:          task.Path,
					Size:          info.Size(),
					Suffix:        strings.TrimPrefix(filepath.Ext(task.Path), "."),
					ContentType:   contentType,
					MusicFolderID: task.Folder.ID,
					Created:       &modTime, // Corresponds to file modification time for incremental scans.
					// TODO: Add audiobook support
					Type: "music",
				}

				t, err := tags.Read(task.Path)
				if err != nil {
					// Still add the child even if tags fail
					resultChan <- scanResult{path: task.Path, child: child}
					continue
				}
				resultChan <- scanResult{path: task.Path, child: child, tags: t}
			}
		}()
	}

	// Saver
	doneSaver := make(chan struct{})
	go func() {
		defer close(doneSaver)
		s.saveResults(resultChan, cacheDir)
	}()

	wg.Wait()
	close(resultChan)
	<-doneSaver

	return seenIDs
}

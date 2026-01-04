package scanner

import (
	"context"
	"fmt"
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
	"github.com/stkevintan/miko/pkg/shared"
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
	walker       *shared.Walker
}

func New(db *gorm.DB, cfg *config.Config) *Scanner {
	return &Scanner{
		db:         db,
		cfg:        cfg,
		numWorkers: max(runtime.NumCPU(), 4),
		walker:     shared.NewWalker(db, cfg),
	}
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

	taskChan, err := s.walker.WalkAllRoots(ctx)
	if err != nil {
		log.Warn("ScanAll failed: %v", err)
		return
	}

	seenIDs, err := s.Scan(ctx, incremental, taskChan)
	if err != nil {
		log.Warn("ScanAll failed: %v", err)
	}

	if seenIDs != nil {
		s.Prune(seenIDs)
	}

	s.lastScanTime.Store(time.Now().Unix())
	log.Info("Scan completed. Total files: %d", s.scanCount.Load())
}

func (s *Scanner) ScanPath(ctx context.Context, id string) (*sync.Map, error) {
	if s.IsScanning() {
		return nil, fmt.Errorf("scan already in progress")
	}

	taskChan, err := s.walker.WalkByID(ctx, id)
	if err != nil {
		log.Warn("ScanPath failed: %v", err)
		return nil, err
	}

	return s.Scan(ctx, false, taskChan)
}

func (s *Scanner) Scan(ctx context.Context, incremental bool, taskChan <-chan shared.WalkTask) (*sync.Map, error) {
	if !s.isScanning.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("scan already in progress")
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
		return nil, fmt.Errorf("failed to create cover cache dir: %w", err)
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
				id := GenerateID(task.Path, task.Folder)
				parentID := GetParentID(task.Path, task.Folder)
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
				if !shared.IsAudioFile(task.Path) {
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
					Created:       &modTime, // Corresponds to file modification time for incremental scans.
					MusicFolderID: task.Folder.ID,
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

	return seenIDs, nil
}

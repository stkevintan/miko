package shared

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

type WalkTask struct {
	Path   string
	D      fs.DirEntry
	Folder models.MusicFolder
}

type Walker struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewWalker(db *gorm.DB, cfg *config.Config) *Walker {
	return &Walker{db: db, cfg: cfg}
}

func (w *Walker) WalkPath(ctx context.Context, path string, folder models.MusicFolder) (<-chan WalkTask, error) {
	walkChan := make(chan WalkTask, runtime.NumCPU()*10) // Default buffer size
	go func() {
		defer close(walkChan)
		w.walk(ctx, path, folder, walkChan)
	}()

	return walkChan, nil
}

func (w *Walker) walk(ctx context.Context, path string, folder models.MusicFolder, outChan chan<- WalkTask) error {
	return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}
		// Normalize path to use forward slashes and be clean
		p = filepath.ToSlash(filepath.Clean(p))
		outChan <- WalkTask{Path: p, D: d, Folder: folder}
		return nil
	})
}

func (w *Walker) WalkByID(ctx context.Context, id string) (<-chan WalkTask, error) {
	var item models.Child
	if err := w.db.Select("id, path").Where("id = ?", id).First(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to find item with ID %q: %w", id, err)
	}

	var folder models.MusicFolder
	// Find the music folder that contains this path
	// We search for the folder whose path is a prefix of the item's path
	// We order by length descending to get the most specific folder if they are nested
	if err := w.db.Where("? LIKE path || '%'", item.Path).Order("LENGTH(path) DESC").First(&folder).Error; err != nil {
		return nil, fmt.Errorf("failed to find music folder for path %q: %w", item.Path, err)
	}

	return w.WalkPath(ctx, item.Path, folder)
}

func (w *Walker) WalkAllRoots(ctx context.Context) (<-chan WalkTask, error) {
	walkChan := make(chan WalkTask, runtime.NumCPU()*10) // Default buffer size
	go func() {
		defer close(walkChan)
		var folders []models.MusicFolder
		if err := w.db.Find(&folders).Error; err != nil {
			return
		}
		for _, folder := range folders {
			w.walk(ctx, folder.Path, folder, walkChan)
		}
	}()

	return walkChan, nil
}

package scanner

import (
	"crypto/md5"
	"fmt"
	"mime"
	"path/filepath"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/shared"
)

func GetContentType(path string) string {
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" && len(ext) > 1 {
		contentType = "audio/" + ext[1:]
	}
	return contentType
}

func GetCoverCacheDir(cfg *config.Config) string {
	return filepath.Join(cfg.Subsonic.DataDir, "cache", "covers")
}

func GenerateID(path string, folder models.MusicFolder) string {
	// path and folder.Path are already normalized
	rel, err := filepath.Rel(folder.Path, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d:%s", folder.ID, rel))))
}

func GenerateAlbumID(artist, album string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(artist+"|"+album)))
}

func GenerateArtistID(name string) string {
	return shared.GenerateHash(name)
}

func GetParentID(path string, folder models.MusicFolder) string {
	// path and folder.Path are already normalized
	if path == folder.Path {
		return ""
	}

	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == path {
		return ""
	}
	return GenerateID(dir, folder)
}

package scanner

import (
	"crypto/md5"
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/stkevintan/miko/config"
)

func IsAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp3" || ext == ".flac" || ext == ".m4a" || ext == ".wav"
}

func GetContentType(path string) string {
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" && len(ext) > 1 {
		contentType = "audio/" + ext[1:]
	}
	return contentType
}

func GenerateID(rootPath, relPath string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(rootPath+relPath)))
}

func GenerateAlbumID(artist, album string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(artist+"|"+album)))
}

func GenerateArtistID(name string) string {
	return GenerateHash(name)
}

func GenerateHash(data string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

func GetParentID(rootPath, relPath string) string {
	if relPath == "." {
		return ""
	}
	parent := filepath.Dir(relPath)
	return GenerateID(rootPath, parent)
}

func GetCoverCacheDir(cfg *config.Config) string {
	return filepath.Join(cfg.Subsonic.DataDir, "cache", "covers")
}

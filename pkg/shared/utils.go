package shared

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"strings"
)

func IsAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp3" || ext == ".flac" || ext == ".m4a" || ext == ".wav"
}

func GenerateHash(data string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

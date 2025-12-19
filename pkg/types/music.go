package types

import (
	"fmt"
	"regexp"
	"strings"
)

type Artist struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

type Album struct {
	Id     int64  `json:"id"`
	Name   string `json:"name"`
	PicUrl string `json:"picUrl"`
}

// Music represents a music track
type Music struct {
	Id          int64    `json:"id"`
	Name        string   `json:"name"`
	Artist      []Artist `json:"artist"`
	Album       Album    `json:"album"`
	Time        int64    `json:"time"`
	Lyrics      string   `json:"lyrics"`
	TrackNumber string   `json:"trackNumber"`
}

type DownloadInfo struct {
	URL      string `json:"url"`
	FilePath string `json:"filePath"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
	Quality  string `json:"quality"`
}

type DownloadedMusic struct {
	Music
	DownloadInfo
}

func (m Music) ArtistString() string {
	if len(m.Artist) <= 0 {
		return ""
	}
	var artistList = make([]string, 0, len(m.Artist))
	for _, ar := range m.Artist {
		artistList = append(artistList, normalize(ar.Name, "_")) // #11 避免文件名中包含特殊字符
	}
	return strings.Join(artistList, ",")
}

func (m Music) String() string {
	var (
		seconds = m.Time / 1000 // 毫秒换成秒
		hours   = seconds / 3600
		minutes = (seconds % 3600) / 60
		secs    = seconds % 60
		format  = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	)
	return fmt.Sprintf("%s-%s(%v) [%s]", m.ArtistString(), m.Name, m.Id, format)
}

// NameString returns cleaned song name
func (m *Music) NameString() string {
	return normalize(m.Name, "_")
}

func (m *Music) SongId() string {
	return fmt.Sprintf("%d", m.Id)
}

func (m *Music) Filename(extType string, index int) string {
	if index <= 0 {
		return fmt.Sprintf("%s - %s.%s", m.ArtistString(), m.NameString(), strings.ToLower(extType))
	}
	return fmt.Sprintf("%s - %s (%d).%s", m.ArtistString(), m.NameString(), index, strings.ToLower(extType))
}

var filenameRegexp = regexp.MustCompile("[\\\\/:*?\"<>|]")

// normalize 清理文件名中的非法字符
func normalize(path string, new ...string) string {
	path = strings.TrimSpace(path)
	if len(new) > 0 {
		return filenameRegexp.ReplaceAllString(path, new[0])
	}
	return filenameRegexp.ReplaceAllString(path, "")
}

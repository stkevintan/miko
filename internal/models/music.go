package models

import (
	"fmt"
	"strings"

	"github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/pkg/utils"
)

// Music represents a music track
type Music struct {
	Id     int64
	Name   string
	Artist []types.Artist
	Album  types.Album
	Time   int64
}

func (m Music) ArtistString() string {
	if len(m.Artist) <= 0 {
		return ""
	}
	var artistList = make([]string, 0, len(m.Artist))
	for _, ar := range m.Artist {
		artistList = append(artistList, utils.Filename(ar.Name, "_")) // #11 避免文件名中包含特殊字符
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
	return utils.Filename(m.Name, "_")
}

func (m *Music) SongId() string {
	return fmt.Sprintf("%d", m.Id)
}

func (m *Music) Filename(extType string) string {
	return fmt.Sprintf("%s - %s.%s", m.ArtistString(), m.NameString(), strings.ToLower(extType))
}

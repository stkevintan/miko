package tags

import (
	"strconv"
	"strings"

	"github.com/stkevintan/miko/pkg/log"
	"go.senan.xyz/taglib"
)

const (
	Title       = taglib.Title
	Artist      = taglib.Artist
	Album       = taglib.Album
	AlbumArtist = taglib.AlbumArtist
	TrackNumber = taglib.TrackNumber
	DiscNumber  = taglib.DiscNumber
	Date        = taglib.Date
	Genre       = taglib.Genre
	Lyrics      = taglib.Lyrics
	Length      = taglib.Length
)

type Tags struct {
	Title        string
	Artist       string
	Artists      []string
	Album        string
	AlbumArtist  string
	AlbumArtists []string
	Track        int
	Disc         int
	Year         int
	Genre        string
	Genres       []string
	Lyrics       string
	Duration     int
	Bitrate      int
}

func Read(path string) (*Tags, error) {
	t, err := taglib.ReadTags(path)
	if err != nil {
		return nil, err
	}

	res := &Tags{}
	if v, ok := t[taglib.Title]; ok && len(v) > 0 {
		res.Title = v[0]
	}
	if v, ok := t[taglib.Artist]; ok && len(v) > 0 {
		res.Artists = v
		res.Artist = strings.Join(v, "; ")
	}
	if v, ok := t[taglib.Album]; ok && len(v) > 0 {
		res.Album = v[0]
	}

	// Album Artist
	if v, ok := t[taglib.AlbumArtist]; ok && len(v) > 0 {
		res.AlbumArtists = v
		res.AlbumArtist = strings.Join(v, "; ")
	} else if v, ok := t["ALBUM ARTIST"]; ok && len(v) > 0 {
		res.AlbumArtists = v
		res.AlbumArtist = strings.Join(v, "; ")
	}

	if v, ok := t[taglib.TrackNumber]; ok && len(v) > 0 {
		res.Track = parseTagInt("Track", v[0])
	}
	if v, ok := t[taglib.DiscNumber]; ok && len(v) > 0 {
		res.Disc = parseTagInt("Disc", v[0])
	}
	if v, ok := t[taglib.Date]; ok && len(v) > 0 {
		res.Year = parseTagInt("Year", v[0])
	}
	if v, ok := t[taglib.Genre]; ok && len(v) > 0 {
		res.Genres = v
		res.Genre = strings.Join(v, "; ")
	}

	// Lyrics
	if v, ok := t[taglib.Lyrics]; ok && len(v) > 0 {
		res.Lyrics = v[0]
	} else if v, ok := t["UNSYNCED LYRICS"]; ok && len(v) > 0 {
		res.Lyrics = v[0]
	}

	// Extract properties
	if props, err := taglib.ReadProperties(path); err == nil {
		res.Duration = int(props.Length.Seconds())
		res.Bitrate = int(props.Bitrate)
	}

	return res, nil
}

func parseTagInt(name string, v string) int {
	if v == "" {
		return 0
	}
	// Handle cases like "1/12" or "2023-10-12"
	parts := strings.FieldsFunc(v, func(r rune) bool {
		return r == '/' || r == '-' || r == '\\'
	})
	if len(parts) > 0 {
		v = parts[0]
	}

	i, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		log.Warn("Failed to parse %s tag value %q as integer: %v", name, v, err)
		return 0
	}
	return i
}

func ReadImage(path string) ([]byte, error) {
	return taglib.ReadImage(path)
}

// Write writes tags to the file
func Write(path string, tags map[string][]string) error {
	return taglib.WriteTags(path, tags, 0)
}

// WriteImage writes an image to the file
func WriteImage(path string, data []byte) error {
	return taglib.WriteImage(path, data)
}

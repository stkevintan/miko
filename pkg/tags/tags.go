package tags

import (
	"strconv"
	"strings"

	"github.com/stkevintan/miko/pkg/log"
	"go.senan.xyz/taglib"
)

// copied from https://taglib.org/api/p_propertymapping.html
const (
	AcoustIDFingerprint       = "ACOUSTID_FINGERPRINT"
	AcoustIDID                = "ACOUSTID_ID"
	Album                     = "ALBUM"
	AlbumArtist               = "ALBUMARTIST"
	AlbumArtistSort           = "ALBUMARTISTSORT"
	AlbumSort                 = "ALBUMSORT"
	Arranger                  = "ARRANGER"
	Artist                    = "ARTIST"
	Artists                   = "ARTISTS"
	ArtistSort                = "ARTISTSORT"
	ArtistWebpage             = "ARTISTWEBPAGE"
	ASIN                      = "ASIN"
	AudioSourceWebpage        = "AUDIOSOURCEWEBPAGE"
	Barcode                   = "BARCODE"
	BPM                       = "BPM"
	CatalogNumber             = "CATALOGNUMBER"
	Comment                   = "COMMENT"
	Compilation               = "COMPILATION"
	Composer                  = "COMPOSER"
	ComposerSort              = "COMPOSERSORT"
	Conductor                 = "CONDUCTOR"
	Copyright                 = "COPYRIGHT"
	CopyrightURL              = "COPYRIGHTURL"
	Date                      = "DATE"
	DiscNumber                = "DISCNUMBER"
	DiscSubtitle              = "DISCSUBTITLE"
	DJMixer                   = "DJMIXER"
	EncodedBy                 = "ENCODEDBY"
	Encoding                  = "ENCODING"
	EncodingTime              = "ENCODINGTIME"
	Engineer                  = "ENGINEER"
	FileType                  = "FILETYPE"
	FileWebpage               = "FILEWEBPAGE"
	GaplessPlayback           = "GAPLESSPLAYBACK"
	Genre                     = "GENRE"
	Grouping                  = "GROUPING"
	InitialKey                = "INITIALKEY"
	InvolvedPeople            = "INVOLVEDPEOPLE"
	ISRC                      = "ISRC"
	Label                     = "LABEL"
	Language                  = "LANGUAGE"
	Length                    = "LENGTH"
	License                   = "LICENSE"
	Lyricist                  = "LYRICIST"
	Lyrics                    = "LYRICS"
	Media                     = "MEDIA"
	Mixer                     = "MIXER"
	Mood                      = "MOOD"
	MovementCount             = "MOVEMENTCOUNT"
	MovementName              = "MOVEMENTNAME"
	MovementNumber            = "MOVEMENTNUMBER"
	MusicBrainzAlbumID        = "MUSICBRAINZ_ALBUMID"
	MusicBrainzAlbumArtistID  = "MUSICBRAINZ_ALBUMARTISTID"
	MusicBrainzArtistID       = "MUSICBRAINZ_ARTISTID"
	MusicBrainzReleaseGroupID = "MUSICBRAINZ_RELEASEGROUPID"
	MusicBrainzReleaseTrackID = "MUSICBRAINZ_RELEASETRACKID"
	MusicBrainzTrackID        = "MUSICBRAINZ_TRACKID"
	MusicBrainzWorkID         = "MUSICBRAINZ_WORKID"
	MusicianCredits           = "MUSICIANCREDITS"
	MusicIPPUID               = "MUSICIP_PUID"
	OriginalAlbum             = "ORIGINALALBUM"
	OriginalArtist            = "ORIGINALARTIST"
	OriginalDate              = "ORIGINALDATE"
	OriginalFilename          = "ORIGINALFILENAME"
	OriginalLyricist          = "ORIGINALLYRICIST"
	Owner                     = "OWNER"
	PaymentWebpage            = "PAYMENTWEBPAGE"
	Performer                 = "PERFORMER"
	PlaylistDelay             = "PLAYLISTDELAY"
	Podcast                   = "PODCAST"
	PodcastCategory           = "PODCASTCATEGORY"
	PodcastDesc               = "PODCASTDESC"
	PodcastID                 = "PODCASTID"
	PodcastURL                = "PODCASTURL"
	ProducedNotice            = "PRODUCEDNOTICE"
	Producer                  = "PRODUCER"
	PublisherWebpage          = "PUBLISHERWEBPAGE"
	RadioStation              = "RADIOSTATION"
	RadioStationOwner         = "RADIOSTATIONOWNER"
	RadioStationWebpage       = "RADIOSTATIONWEBPAGE"
	ReleaseCountry            = "RELEASECOUNTRY"
	ReleaseDate               = "RELEASEDATE"
	ReleaseStatus             = "RELEASESTATUS"
	ReleaseType               = "RELEASETYPE"
	Remixer                   = "REMIXER"
	Script                    = "SCRIPT"
	ShowSort                  = "SHOWSORT"
	ShowWorkMovement          = "SHOWWORKMOVEMENT"
	Subtitle                  = "SUBTITLE"
	TaggingDate               = "TAGGINGDATE"
	Title                     = "TITLE"
	TitleSort                 = "TITLESORT"
	TrackNumber               = "TRACKNUMBER"
	TVEpisode                 = "TVEPISODE"
	TVEpisodeID               = "TVEPISODEID"
	TVNetwork                 = "TVNETWORK"
	TVSeason                  = "TVSEASON"
	TVShow                    = "TVSHOW"
	URL                       = "URL"
	Work                      = "WORK"
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
	// prefer Artists tag for multiple artists
	if v, ok := t[taglib.Artists]; ok && len(v) > 0 {
		res.Artists = v
		if res.Artist == "" {
			res.Artist = strings.Join(v, "; ")
		}
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

func ReadAll(path string) (map[string][]string, error) {
	return taglib.ReadTags(path)
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

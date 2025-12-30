package models

import (
	"encoding/xml"
	"time"

	"gorm.io/gorm"
)

type ResponseStatus string

const (
	ResponseStatusOK     ResponseStatus = "ok"
	ResponseStatusFailed ResponseStatus = "failed"
)

type OpenSubsonicExtension struct {
	Name     string `xml:"name,attr" json:"name"`
	Versions []int  `xml:"versions,attr" json:"versions"`
}

type SubsonicResponse struct {
	XMLName                xml.Name                `xml:"subsonic-response" json:"-"`
	Xmlns                  string                  `xml:"xmlns,attr" json:"-"`
	Status                 ResponseStatus          `xml:"status,attr" json:"status"`
	Version                string                  `xml:"version,attr" json:"version"`
	ServerVersion          string                  `xml:"serverVersion,attr,omitempty" json:"serverVersion,omitempty"`
	OpenSubsonic           bool                    `xml:"openSubsonic,attr,omitempty" json:"openSubsonic,omitempty"`
	Type                   string                  `xml:"type,attr,omitempty" json:"type,omitempty"`
	OpenSubsonicExtensions []OpenSubsonicExtension `xml:"openSubsonicExtensions,omitempty" json:"openSubsonicExtensions,omitempty"`

	// Choice elements
	MusicFolders    *MusicFolders        `xml:"musicFolders,omitempty" json:"musicFolders,omitempty"`
	Indexes         *Indexes             `xml:"indexes,omitempty" json:"indexes,omitempty"`
	Directory       *Directory           `xml:"directory,omitempty" json:"directory,omitempty"`
	Genres          *Genres              `xml:"genres,omitempty" json:"genres,omitempty"`
	Artists         *ArtistsID3          `xml:"artists,omitempty" json:"artists,omitempty"`
	Artist          *ArtistWithAlbumsID3 `xml:"artist,omitempty" json:"artist,omitempty"`
	Album           *AlbumWithSongsID3   `xml:"album,omitempty" json:"album,omitempty"`
	Song            *Child               `xml:"song,omitempty" json:"song,omitempty"`
	Videos          *Videos              `xml:"videos,omitempty" json:"videos,omitempty"`
	VideoInfo       *VideoInfo           `xml:"videoInfo,omitempty" json:"videoInfo,omitempty"`
	NowPlaying      *NowPlaying          `xml:"nowPlaying,omitempty" json:"nowPlaying,omitempty"`
	SearchResult    *SearchResult        `xml:"searchResult,omitempty" json:"searchResult,omitempty"`
	SearchResult2   *SearchResult2       `xml:"searchResult2,omitempty" json:"searchResult2,omitempty"`
	SearchResult3   *SearchResult3       `xml:"searchResult3,omitempty" json:"searchResult3,omitempty"`
	Playlists       *Playlists           `xml:"playlists,omitempty" json:"playlists,omitempty"`
	Playlist        *PlaylistWithSongs   `xml:"playlist,omitempty" json:"playlist,omitempty"`
	JukeboxStatus   *JukeboxStatus       `xml:"jukeboxStatus,omitempty" json:"jukeboxStatus,omitempty"`
	JukeboxPlaylist *JukeboxPlaylist     `xml:"jukeboxPlaylist,omitempty" json:"jukeboxPlaylist,omitempty"`
	License         *License             `xml:"license,omitempty" json:"license,omitempty"`
	Users           *Users               `xml:"users,omitempty" json:"users,omitempty"`
	User            *User                `xml:"user,omitempty" json:"user,omitempty"`
	ChatMessages    *ChatMessages        `xml:"chatMessages,omitempty" json:"chatMessages,omitempty"`
	AlbumList       *AlbumList           `xml:"albumList,omitempty" json:"albumList,omitempty"`
	AlbumList2      *AlbumList2          `xml:"albumList2,omitempty" json:"albumList2,omitempty"`
	RandomSongs     *Songs               `xml:"randomSongs,omitempty" json:"randomSongs,omitempty"`
	SongsByGenre    *Songs               `xml:"songsByGenre,omitempty" json:"songsByGenre,omitempty"`
	Lyrics          *Lyrics              `xml:"lyrics,omitempty" json:"lyrics,omitempty"`
	// opensubsonic extension
	LyricsList            *LyricsList            `xml:"lyricsList,omitempty" json:"lyricsList,omitempty"`
	Podcasts              *Podcasts              `xml:"podcasts,omitempty" json:"podcasts,omitempty"`
	NewestPodcasts        *NewestPodcasts        `xml:"newestPodcasts,omitempty" json:"newestPodcasts,omitempty"`
	InternetRadioStations *InternetRadioStations `xml:"internetRadioStations,omitempty" json:"internetRadioStations,omitempty"`
	Bookmarks             *Bookmarks             `xml:"bookmarks,omitempty" json:"bookmarks,omitempty"`
	PlayQueue             *PlayQueue             `xml:"playQueue,omitempty" json:"playQueue,omitempty"`
	Shares                *Shares                `xml:"shares,omitempty" json:"shares,omitempty"`
	Starred               *Starred               `xml:"starred,omitempty" json:"starred,omitempty"`
	Starred2              *Starred2              `xml:"starred2,omitempty" json:"starred2,omitempty"`
	AlbumInfo             *AlbumInfo             `xml:"albumInfo,omitempty" json:"albumInfo,omitempty"`
	ArtistInfo            *ArtistInfo            `xml:"artistInfo,omitempty" json:"artistInfo,omitempty"`
	ArtistInfo2           *ArtistInfo2           `xml:"artistInfo2,omitempty" json:"artistInfo2,omitempty"`
	SimilarSongs          *SimilarSongs          `xml:"similarSongs,omitempty" json:"similarSongs,omitempty"`
	SimilarSongs2         *SimilarSongs2         `xml:"similarSongs2,omitempty" json:"similarSongs2,omitempty"`
	TopSongs              *TopSongs              `xml:"topSongs,omitempty" json:"topSongs,omitempty"`
	ScanStatus            *ScanStatus            `xml:"scanStatus,omitempty" json:"scanStatus,omitempty"`
	Error                 *Error                 `xml:"error,omitempty" json:"error,omitempty"`
	Ping                  *Ping                  `xml:"ping,omitempty" json:"ping,omitempty"`
}

type Error struct {
	Code    int    `xml:"code,attr" json:"code"`
	Message string `xml:"message,attr" json:"message"`
}

type Ping struct{}

type MusicFolders struct {
	MusicFolder []MusicFolder `xml:"musicFolder" json:"musicFolder"`
}

type MusicFolder struct {
	ID   uint   `gorm:"primaryKey" xml:"id,attr" json:"id"`
	Name string `xml:"name,attr,omitempty" json:"name,omitempty"`
	Path string `gorm:"uniqueIndex" xml:"-" json:"path"`
}

type Indexes struct {
	LastModified    int64    `xml:"lastModified,attr" json:"lastModified"`
	IgnoredArticles string   `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Shortcut        []Artist `xml:"shortcut,omitempty" json:"shortcut,omitempty"`
	Index           []Index  `xml:"index,omitempty" json:"index,omitempty"`
	Child           []Child  `xml:"child,omitempty" json:"child,omitempty"`
}

type Index struct {
	Name   string   `xml:"name,attr" json:"name"`
	Artist []Artist `xml:"artist" json:"artist"`
}

type Artist struct {
	ID             string     `xml:"id,attr" json:"id"`
	Name           string     `xml:"name,attr" json:"name"`
	ArtistImageUrl string     `xml:"artistImageUrl,attr,omitempty" json:"artistImageUrl,omitempty"`
	Starred        *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	UserRating     int        `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	AverageRating  float64    `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
}

type Genres struct {
	Genre []Genre `xml:"genre" json:"genre"`
}

type Genre struct {
	Name       string     `gorm:"primaryKey" xml:",chardata" json:"value"`
	SongCount  int        `xml:"songCount,attr" json:"songCount"`
	AlbumCount int        `xml:"albumCount,attr" json:"albumCount"`
	Songs      []Child    `gorm:"many2many:song_genres;" xml:"-" json:"-"`
	Albums     []AlbumID3 `gorm:"many2many:album_genres;" xml:"-" json:"-"`
}

type ArtistsID3 struct {
	IgnoredArticles string     `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Index           []IndexID3 `xml:"index" json:"index"`
}

type IndexID3 struct {
	Name   string      `xml:"name,attr" json:"name"`
	Artist []ArtistID3 `xml:"artist" json:"artist"`
}

type ArtistID3 struct {
	ID             string     `gorm:"primaryKey" xml:"id,attr" json:"id"`
	Name           string     `gorm:"index" xml:"name,attr" json:"name"`
	CoverArt       string     `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	ArtistImageUrl string     `xml:"artistImageUrl,attr,omitempty" json:"artistImageUrl,omitempty"`
	AlbumCount     int        `gorm:"->;-:migration" xml:"albumCount,attr" json:"albumCount"`
	Starred        *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	UserRating     int        `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	AverageRating  float64    `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
	Albums         []AlbumID3 `gorm:"many2many:album_artists;" xml:"-" json:"-"`
	Songs          []Child    `gorm:"many2many:song_artists;" xml:"-" json:"-"`
}

type ArtistWithAlbumsID3 struct {
	ArtistID3
	Album []AlbumID3 `xml:"album" json:"album"`
}

type AlbumID3 struct {
	ID            string      `gorm:"primaryKey" xml:"id,attr" json:"id"`
	Name          string      `gorm:"index" xml:"name,attr" json:"name"`
	Artist        string      `gorm:"index" xml:"artist,attr,omitempty" json:"artist,omitempty"`
	ArtistID      string      `gorm:"index" xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	CoverArt      string      `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	SongCount     int         `gorm:"->;-:migration" xml:"songCount,attr" json:"songCount"`
	Duration      int         `gorm:"->;-:migration" xml:"duration,attr" json:"duration"`
	PlayCount     int64       `gorm:"->;-:migration" xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	LastPlayed    *time.Time  `gorm:"->;-:migration" xml:"-" json:"lastPlayed,omitempty"`
	Created       time.Time   `xml:"created,attr" json:"created"`
	Starred       *time.Time  `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	UserRating    int         `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	AverageRating float64     `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
	Year          int         `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre         string      `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	Artists       []ArtistID3 `gorm:"many2many:album_artists;" xml:"-" json:"-"`
}

type AlbumWithSongsID3 struct {
	AlbumID3
	Song []Child `xml:"song" json:"song"`
}

type Videos struct {
	Video []Child `xml:"video" json:"video"`
}

type VideoInfo struct {
	ID         string            `xml:"id,attr" json:"id"`
	Captions   []Captions        `xml:"captions,omitempty" json:"captions,omitempty"`
	AudioTrack []AudioTrack      `xml:"audioTrack,omitempty" json:"audioTrack,omitempty"`
	Conversion []VideoConversion `xml:"conversion,omitempty" json:"conversion,omitempty"`
}

type Captions struct {
	ID   string `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr,omitempty" json:"name,omitempty"`
}

type AudioTrack struct {
	ID           string `xml:"id,attr" json:"id"`
	Name         string `xml:"name,attr,omitempty" json:"name,omitempty"`
	LanguageCode string `xml:"languageCode,attr,omitempty" json:"languageCode,omitempty"`
}

type VideoConversion struct {
	ID           string `xml:"id,attr" json:"id"`
	BitRate      int    `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	AudioTrackID int    `xml:"audioTrackId,attr,omitempty" json:"audioTrackId,omitempty"`
}

type Directory struct {
	ID            string     `xml:"id,attr" json:"id"`
	Parent        string     `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	Name          string     `xml:"name,attr" json:"name"`
	Starred       *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	UserRating    int        `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	AverageRating float64    `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
	PlayCount     int64      `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Child         []Child    `xml:"child" json:"child"`
}

type Child struct {
	ID                    string      `gorm:"primaryKey" xml:"id,attr" json:"id"`
	Parent                string      `gorm:"index" xml:"parent,attr,omitempty" json:"parent,omitempty"`
	IsDir                 bool        `xml:"isDir,attr" json:"isDir"`
	Title                 string      `gorm:"index" xml:"title,attr" json:"title"`
	Album                 string      `gorm:"index" xml:"album,attr,omitempty" json:"album,omitempty"`
	Artist                string      `gorm:"index" xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Track                 int         `xml:"track,attr,omitempty" json:"track,omitempty"`
	Year                  int         `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre                 string      `gorm:"index" xml:"genre,attr,omitempty" json:"genre,omitempty"`
	CoverArt              string      `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Size                  int64       `xml:"size,attr,omitempty" json:"size,omitempty"`
	ContentType           string      `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	Suffix                string      `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	TranscodedContentType string      `xml:"transcodedContentType,attr,omitempty" json:"transcodedContentType,omitempty"`
	TranscodedSuffix      string      `xml:"transcodedSuffix,attr,omitempty" json:"transcodedSuffix,omitempty"`
	Duration              int         `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	BitRate               int         `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	Path                  string      `gorm:"uniqueIndex" xml:"path,attr,omitempty" json:"path,omitempty"`
	IsVideo               bool        `xml:"isVideo,attr,omitempty" json:"isVideo,omitempty"`
	UserRating            int         `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	AverageRating         float64     `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
	PlayCount             int64       `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	LastPlayed            *time.Time  `xml:"lastPlayed,attr,omitempty" json:"lastPlayed,omitempty"`
	DiscNumber            int         `xml:"discNumber,attr,omitempty" json:"discNumber,omitempty"`
	Created               *time.Time  `xml:"created,attr,omitempty" json:"created,omitempty"`
	Starred               *time.Time  `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	AlbumID               string      `gorm:"index" xml:"albumId,attr,omitempty" json:"albumId,omitempty"`
	ArtistID              string      `gorm:"index" xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	MusicFolderID         uint        `gorm:"index" xml:"-" json:"musicFolderId,omitempty"`
	Type                  string      `xml:"type,attr,omitempty" json:"type,omitempty"`
	BookmarkPosition      int64       `xml:"bookmarkPosition,attr,omitempty" json:"bookmarkPosition,omitempty"`
	OriginalWidth         int         `xml:"originalWidth,attr,omitempty" json:"originalWidth,omitempty"`
	OriginalHeight        int         `xml:"originalHeight,attr,omitempty" json:"originalHeight,omitempty"`
	Artists               []ArtistID3 `gorm:"many2many:song_artists;" xml:"-" json:"-"`
	Genres                []Genre     `gorm:"many2many:song_genres;" xml:"-" json:"-"`
	Lyrics                string      `xml:"-" json:"-"`
}

type NowPlaying struct {
	Entry []NowPlayingEntry `xml:"entry" json:"entry"`
}

type NowPlayingRecord struct {
	Username   string
	ChildID    string
	PlayerID   int
	PlayerName string
	UpdatedAt  time.Time
}

type NowPlayingEntry struct {
	Child
	Username   string `xml:"username,attr" json:"username"`
	MinutesAgo int    `xml:"minutesAgo,attr" json:"minutesAgo"`
	PlayerID   int    `xml:"playerId,attr" json:"playerId"`
	PlayerName string `xml:"playerName,attr,omitempty" json:"playerName,omitempty"`
}

type SearchResult struct {
	Offset    int     `xml:"offset,attr" json:"offset"`
	TotalHits int     `xml:"totalHits,attr" json:"totalHits"`
	Match     []Child `xml:"match" json:"match"`
}

type SearchResult2 struct {
	Artist []Artist `xml:"artist,omitempty" json:"artist,omitempty"`
	Album  []Child  `xml:"album,omitempty" json:"album,omitempty"`
	Song   []Child  `xml:"song,omitempty" json:"song,omitempty"`
}

type SearchResult3 struct {
	Artist []ArtistID3 `xml:"artist,omitempty" json:"artist,omitempty"`
	Album  []AlbumID3  `xml:"album,omitempty" json:"album,omitempty"`
	Song   []Child     `xml:"song,omitempty" json:"song,omitempty"`
}

type Playlists struct {
	Playlist []Playlist `xml:"playlist" json:"playlist"`
}

type Playlist struct {
	ID          string    `xml:"id,attr" json:"id"`
	Name        string    `xml:"name,attr" json:"name"`
	Comment     string    `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	Owner       string    `xml:"owner,attr,omitempty" json:"owner,omitempty"`
	Public      bool      `xml:"public,attr,omitempty" json:"public,omitempty"`
	SongCount   int       `xml:"songCount,attr" json:"songCount"`
	Duration    int       `xml:"duration,attr" json:"duration"`
	Created     time.Time `xml:"created,attr" json:"created"`
	Changed     time.Time `xml:"changed,attr" json:"changed"`
	CoverArt    string    `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AllowedUser []string  `xml:"allowedUser,omitempty" json:"allowedUser,omitempty"`
}

type PlaylistWithSongs struct {
	Playlist
	Entry []Child `xml:"entry" json:"entry"`
}

type JukeboxStatus struct {
	CurrentIndex int     `xml:"currentIndex,attr" json:"currentIndex"`
	Playing      bool    `xml:"playing,attr" json:"playing"`
	Gain         float32 `xml:"gain,attr" json:"gain"`
	Position     int     `xml:"position,attr,omitempty" json:"position,omitempty"`
}

type JukeboxPlaylist struct {
	JukeboxStatus
	Entry []Child `xml:"entry" json:"entry"`
}

type ChatMessages struct {
	ChatMessage []ChatMessage `xml:"chatMessage" json:"chatMessage"`
}

type ChatMessage struct {
	Username string `xml:"username,attr" json:"username"`
	Time     int64  `xml:"time,attr" json:"time"`
	Message  string `xml:"message,attr" json:"message"`
}

type AlbumList struct {
	Album []Child `xml:"album" json:"album"`
}

type AlbumList2 struct {
	Album []AlbumID3 `xml:"album" json:"album"`
}

type Songs struct {
	Song []Child `xml:"song" json:"song"`
}

type Lyrics struct {
	Value  string `xml:",chardata" json:"value"`
	Artist string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Title  string `xml:"title,attr,omitempty" json:"title,omitempty"`
}

type LyricsList struct {
	StructuredLyrics []StructuredLyrics `xml:"structuredLyrics" json:"structuredLyrics"`
}

type StructuredLyrics struct {
	Lang          string       `xml:"lang,attr,omitempty" json:"lang,omitempty"`
	Synced        bool         `xml:"synced,attr" json:"synced"`
	DisplayArtist string       `xml:"displayArtist,attr,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string       `xml:"displayTitle,attr,omitempty" json:"displayTitle,omitempty"`
	Lines         []LyricsLine `xml:"line,omitempty" json:"line,omitempty"`
	Offset        int          `xml:"offset,attr,omitempty" json:"offset,omitempty"`
}

type LyricsLine struct {
	Start int    `xml:"start,attr" json:"start"`
	Value string `xml:",chardata" json:"value"`
}

type Podcasts struct {
	Channel []PodcastChannel `xml:"channel" json:"channel"`
}

type PodcastChannel struct {
	ID               string           `xml:"id,attr" json:"id"`
	URL              string           `xml:"url,attr" json:"url"`
	Title            string           `xml:"title,attr,omitempty" json:"title,omitempty"`
	Description      string           `xml:"description,attr,omitempty" json:"description,omitempty"`
	CoverArt         string           `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	OriginalImageUrl string           `xml:"originalImageUrl,attr,omitempty" json:"originalImageUrl,omitempty"`
	Status           string           `xml:"status,attr" json:"status"`
	ErrorMessage     string           `xml:"errorMessage,attr,omitempty" json:"errorMessage,omitempty"`
	Episode          []PodcastEpisode `xml:"episode,omitempty" json:"episode,omitempty"`
}

type NewestPodcasts struct {
	Episode []PodcastEpisode `xml:"episode" json:"episode"`
}

type PodcastEpisode struct {
	Child
	StreamID    string     `xml:"streamId,attr,omitempty" json:"streamId,omitempty"`
	ChannelID   string     `xml:"channelId,attr" json:"channelId"`
	Description string     `xml:"description,attr,omitempty" json:"description,omitempty"`
	Status      string     `xml:"status,attr" json:"status"`
	PublishDate *time.Time `xml:"publishDate,attr,omitempty" json:"publishDate,omitempty"`
}

type InternetRadioStations struct {
	InternetRadioStation []InternetRadioStation `xml:"internetRadioStation" json:"internetRadioStation"`
}

type InternetRadioStation struct {
	ID          string `xml:"id,attr" json:"id"`
	Name        string `xml:"name,attr" json:"name"`
	StreamURL   string `xml:"streamUrl,attr" json:"streamUrl"`
	HomePageURL string `xml:"homePageUrl,attr,omitempty" json:"homePageUrl,omitempty"`
}

type Bookmarks struct {
	Bookmark []Bookmark `xml:"bookmark" json:"bookmark"`
}

type Bookmark struct {
	Position int64     `xml:"position,attr" json:"position"`
	Username string    `xml:"username,attr" json:"username"`
	Comment  string    `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	Created  time.Time `xml:"created,attr" json:"created"`
	Changed  time.Time `xml:"changed,attr" json:"changed"`
	Entry    Child     `xml:"entry" json:"entry"`
}

type PlayQueue struct {
	Current   int       `xml:"current,attr,omitempty" json:"current,omitempty"`
	Position  int64     `xml:"position,attr,omitempty" json:"position,omitempty"`
	Username  string    `xml:"username,attr" json:"username"`
	Changed   time.Time `xml:"changed,attr" json:"changed"`
	ChangedBy string    `xml:"changedBy,attr" json:"changedBy"`
	Entry     []Child   `xml:"entry,omitempty" json:"entry,omitempty"`
}

type Shares struct {
	Share []Share `xml:"share" json:"share"`
}

type Share struct {
	ID          string     `xml:"id,attr" json:"id"`
	URL         string     `xml:"url,attr" json:"url"`
	Description string     `xml:"description,attr,omitempty" json:"description,omitempty"`
	Username    string     `xml:"username,attr" json:"username"`
	Created     time.Time  `xml:"created,attr" json:"created"`
	Expires     *time.Time `xml:"expires,attr,omitempty" json:"expires,omitempty"`
	LastVisited *time.Time `xml:"lastVisited,attr,omitempty" json:"lastVisited,omitempty"`
	VisitCount  int        `xml:"visitCount,attr" json:"visitCount"`
	Entry       []Child    `xml:"entry,omitempty" json:"entry,omitempty"`
}

type Starred struct {
	Artist []Artist `xml:"artist,omitempty" json:"artist,omitempty"`
	Album  []Child  `xml:"album,omitempty" json:"album,omitempty"`
	Song   []Child  `xml:"song,omitempty" json:"song,omitempty"`
}

type Starred2 struct {
	Artist []ArtistID3 `xml:"artist,omitempty" json:"artist,omitempty"`
	Album  []AlbumID3  `xml:"album,omitempty" json:"album,omitempty"`
	Song   []Child     `xml:"song,omitempty" json:"song,omitempty"`
}

type AlbumInfo struct {
	Notes          string `xml:"notes,omitempty" json:"notes,omitempty"`
	MusicBrainzID  string `xml:"musicBrainzId,omitempty" json:"musicBrainzId,omitempty"`
	LastFmURL      string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageUrl  string `xml:"smallImageUrl,omitempty" json:"smallImageUrl,omitempty"`
	MediumImageUrl string `xml:"mediumImageUrl,omitempty" json:"mediumImageUrl,omitempty"`
	LargeImageUrl  string `xml:"largeImageUrl,omitempty" json:"largeImageUrl,omitempty"`
}

type ArtistInfoBase struct {
	Biography      string `xml:"biography,omitempty" json:"biography,omitempty"`
	MusicBrainzID  string `xml:"musicBrainzId,omitempty" json:"musicBrainzId,omitempty"`
	LastFmURL      string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageUrl  string `xml:"smallImageUrl,omitempty" json:"smallImageUrl,omitempty"`
	MediumImageUrl string `xml:"mediumImageUrl,omitempty" json:"mediumImageUrl,omitempty"`
	LargeImageUrl  string `xml:"largeImageUrl,omitempty" json:"largeImageUrl,omitempty"`
}

type ArtistInfo struct {
	ArtistInfoBase
	SimilarArtist []Artist `xml:"similarArtist,omitempty" json:"similarArtist,omitempty"`
}

type ArtistInfo2 struct {
	ArtistInfoBase
	SimilarArtist []ArtistID3 `xml:"similarArtist,omitempty" json:"similarArtist,omitempty"`
}

type SimilarSongs struct {
	Song []Child `xml:"song" json:"song"`
}

type SimilarSongs2 struct {
	Song []Child `xml:"song" json:"song"`
}

type TopSongs struct {
	Song []Child `xml:"song" json:"song"`
}

type License struct {
	Valid          bool       `xml:"valid,attr" json:"valid"`
	Email          string     `xml:"email,attr,omitempty" json:"email,omitempty"`
	LicenseExpires *time.Time `xml:"licenseExpires,attr,omitempty" json:"licenseExpires,omitempty"`
	TrialExpires   *time.Time `xml:"trialExpires,attr,omitempty" json:"trialExpires,omitempty"`
}

type ScanStatus struct {
	Scanning bool  `xml:"scanning,attr" json:"scanning"`
	Count    int64 `xml:"count,attr,omitempty" json:"count,omitempty"`
}

type Users struct {
	User []*User `xml:"user" json:"user"`
}

func NewResponse(status ResponseStatus) *SubsonicResponse {
	return &SubsonicResponse{
		Xmlns:         "http://subsonic.org/restapi",
		Status:        status,
		Version:       "1.16.1",
		ServerVersion: "1.0.0",
		Type:          "miko",
		OpenSubsonic:  true,
	}
}

func NewErrorResponse(code int, message string) *SubsonicResponse {
	resp := NewResponse(ResponseStatusFailed)
	resp.Error = &Error{
		Code:    code,
		Message: message,
	}
	return resp
}

func AlbumWithStats(includeLastPlayed bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		selects := "album_id3.*, " +
			"(SELECT COUNT(*) FROM children WHERE album_id = album_id3.id AND is_dir = false) AS song_count, " +
			"(SELECT CAST(IFNULL(SUM(duration), 0) AS INTEGER) FROM children WHERE album_id = album_id3.id AND is_dir = false) AS duration, " +
			"(SELECT CAST(IFNULL(SUM(play_count), 0) AS INTEGER) FROM children WHERE album_id = album_id3.id AND is_dir = false) AS play_count"

		if includeLastPlayed {
			selects += ", (SELECT last_played FROM children WHERE album_id = album_id3.id AND is_dir = false ORDER BY last_played DESC LIMIT 1) AS last_played"
		}

		return db.Select(selects)
	}
}

func ArtistWithStats(db *gorm.DB) *gorm.DB {
	return db.Select("artist_id3.*, " +
		"(SELECT COUNT(*) FROM album_artists WHERE artist_id3_id = artist_id3.id) AS album_count")
}

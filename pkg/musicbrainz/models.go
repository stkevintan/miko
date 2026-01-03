package musicbrainz

type NamedEntity struct {
	Name string `json:"name"`
}

type IDNamedEntity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Media struct {
	Format string `json:"format"`
	Track  []struct {
		Number string `json:"number"`
	} `json:"track"`
	TrackCount int `json:"track-count"`
	Position   int `json:"position"`
}
type ReleaseGroup struct {
	ID   string `json:"id"`
	Type string `json:"primary-type"`
}

type Release struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Date         string          `json:"date"`
	Status       string          `json:"status"`
	Country      string          `json:"country"`
	Barcode      string          `json:"barcode"`
	Media        []Media         `json:"media"`
	ReleaseGroup ReleaseGroup    `json:"release-group"`
	ArtistCredit []IDNamedEntity `json:"artist-credit"`
}

type WorkRelation struct {
	Type   string      `json:"type"`
	Artist NamedEntity `json:"artist"`
}

type Work struct {
	Title     string         `json:"title"`
	Relations []WorkRelation `json:"relations"`
}

type Relation struct {
	Type   string      `json:"type"`
	Artist NamedEntity `json:"artist"`
	Work   Work        `json:"work"`
}

type Recording struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	Length    int             `json:"length"`
	ISRCs     []string        `json:"isrcs"`
	Artists   []IDNamedEntity `json:"artist-credit"`
	Releases  []Release       `json:"releases"`
	Tags      []NamedEntity   `json:"tags"`
	Genres    []NamedEntity   `json:"genres"`
	Relations []Relation      `json:"relations"`
}

type SearchResponse struct {
	Recordings []Recording `json:"recordings"`
}

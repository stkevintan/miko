package musicbrainz

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://musicbrainz.org/ws/2"

type Client struct {
	httpClient *http.Client
	userAgent  string
}

func NewClient(userAgent string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		userAgent: userAgent,
	}
}

type Recording struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Length  int      `json:"length"`
	ISRCs   []string `json:"isrcs"`
	Artists []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artist-credit"`
	Releases []struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Date    string `json:"date"`
		Status  string `json:"status"`
		Country string `json:"country"`
		Barcode string `json:"barcode"`
		Media   []struct {
			Format string `json:"format"`
			Track  []struct {
				Number string `json:"number"`
			} `json:"track"`
			TrackCount int `json:"track-count"`
			Position   int `json:"position"`
		} `json:"media"`
		ReleaseGroup struct {
			ID   string `json:"id"`
			Type string `json:"primary-type"`
		} `json:"release-group"`
		ArtistCredit []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist-credit"`
	} `json:"releases"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
	Genres []struct {
		Name string `json:"name"`
	} `json:"genres"`
	Relations []struct {
		Type   string `json:"type"`
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
		Work struct {
			Title     string `json:"title"`
			Relations []struct {
				Type   string `json:"type"`
				Artist struct {
					Name string `json:"name"`
				} `json:"artist"`
			} `json:"relations"`
		} `json:"work"`
	} `json:"relations"`
}

type SearchResponse struct {
	Recordings []Recording `json:"recordings"`
}

func (c *Client) SearchRecording(artist, album, title string) (*Recording, error) {
	query := ""
	if artist != "" {
		query += fmt.Sprintf("artist:\"%s\" ", artist)
	}
	if album != "" {
		query += fmt.Sprintf("release:\"%s\" ", album)
	}
	if title != "" {
		query += fmt.Sprintf("recording:\"%s\" ", title)
	}

	u, _ := url.Parse(baseURL + "/recording/")
	q := u.Query()
	q.Set("query", query)
	q.Set("fmt", "json")
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz api error: %s", resp.Status)
	}

	var sr SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	if len(sr.Recordings) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	// Return the first result as the best match
	return &sr.Recordings[0], nil
}

func (c *Client) GetRecording(id string) (*Recording, error) {
	u, _ := url.Parse(baseURL + "/recording/" + id)
	q := u.Query()
	q.Set("inc", "artist-credits+releases+isrcs+media+release-groups+tags+genres+artist-rels+work-rels")
	q.Set("fmt", "json")
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz api error: %s", resp.Status)
	}

	var r Recording
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return &r, nil
}

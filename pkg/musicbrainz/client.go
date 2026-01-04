package musicbrainz

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const baseURL = "https://musicbrainz.org/ws/2"

type Client struct {
	restyClient *resty.Client
	limiter     <-chan time.Time
}

func NewClient() *Client {
	client := resty.New().
		SetBaseURL(baseURL).
		SetHeader("User-Agent", "Miko/1.0.0 (https://github.com/stkevintan/miko)").
		SetTimeout(10*time.Second).
		SetHeader("Accept", "application/json").
		// Add retry logic for 503 errors
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(10 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			return r.StatusCode() == 503
		})

	return &Client{
		restyClient: client,
		// MusicBrainz allows 1 request per second for non-authenticated users
		limiter: time.Tick(1100 * time.Millisecond),
	}
}

func (c *Client) SearchRecording(ctx context.Context, artist, album, title string) (*Recording, error) {
	select {
	case <-c.limiter:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var queryBuilder strings.Builder
	writeQuery := func(key, val string) {
		escaped := strings.ReplaceAll(val, "\"", "\\\"")
		queryBuilder.WriteString(fmt.Sprintf("%s:\"%s\" ", key, escaped))
	}

	if artist != "" {
		writeQuery("artist", artist)
	}
	if album != "" {
		writeQuery("release", album)
	}
	if title != "" {
		writeQuery("recording", title)
	}
	query := queryBuilder.String()

	var sr SearchResponse
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetQueryParam("query", query).
		SetQueryParam("fmt", "json").
		SetResult(&sr).
		Get("/recording/")

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("musicbrainz api error: %s", resp.Status())
	}

	if len(sr.Recordings) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	// Return the first result as the best match
	return &sr.Recordings[0], nil
}

func (c *Client) GetRecording(ctx context.Context, id string) (*Recording, error) {
	select {
	case <-c.limiter:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var r Recording
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetQueryParam("inc", "artist-credits+releases+isrcs+media+release-groups+tags+genres+artist-rels+work-rels").
		SetQueryParam("fmt", "json").
		SetResult(&r).
		Get("/recording/" + id)

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("musicbrainz api error: %s", resp.Status())
	}

	return &r, nil
}

func (c *Client) FetchCoverArt(ctx context.Context, releaseID string) ([]byte, error) {
	// Cover Art Archive doesn't have the same rate limit as MusicBrainz API,
	// but it's good practice to be respectful.
	resp, err := c.restyClient.R().
		SetContext(ctx).
		Get("https://coverartarchive.org/release/" + releaseID + "/front")

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("cover art archive error: %s", resp.Status())
	}

	return resp.Body(), nil
}

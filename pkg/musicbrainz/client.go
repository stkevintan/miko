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
}

func NewClient() *Client {
	client := resty.New().
		SetBaseURL(baseURL).
		SetHeader("User-Agent", "Miko/1.0.0 (https://github.com/stkevintan/miko)").
		SetTimeout(10*time.Second).
		SetHeader("Accept", "application/json")

	return &Client{
		restyClient: client,
	}
}

func (c *Client) SearchRecording(ctx context.Context, artist, album, title string) (*Recording, error) {
	newQuery := func(key, val string) string {
		escaped := strings.ReplaceAll(val, "\"", "\\\"")
		return fmt.Sprintf("%s:\"%s\" ", key, escaped)
	}
	query := ""
	if artist != "" {
		query += newQuery("artist", artist)
	}
	if album != "" {
		query += newQuery("release", album)
	}
	if title != "" {
		query += newQuery("recording", title)
	}

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

package dsh

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
)

// readBootDocument is shared by the native consumers of existing index globals.
// The document is never evaluated, exposed to logs or persisted.
func (c *Client) readBootDocument(ctx context.Context) ([]byte, error) {
	origin, err := url.Parse(c.Endpoint.Origin)
	if err != nil || origin.Host == "" || origin.User != nil || (origin.Scheme != "http" && origin.Scheme != "https") {
		return nil, errors.New("invalid Host origin")
	}
	endpoint := *origin
	endpoint.Path, endpoint.RawPath, endpoint.RawQuery, endpoint.Fragment = "/", "", "", ""
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html")
	if c.Endpoint.Cookie != "" {
		req.Header.Set("Cookie", c.Endpoint.Cookie)
	}
	client := *c.http()
	previousRedirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != origin.Scheme || req.URL.Host != origin.Host {
			return errors.New("Host bootstrap cross-origin redirect refused")
		}
		if previousRedirect != nil {
			return previousRedirect(req, via)
		}
		if len(via) >= 10 {
			return errors.New("too many Host bootstrap redirects")
		}
		return nil
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Host bootstrap could not be loaded")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Host bootstrap unavailable")
	}
	const limit = 4 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(body) > limit {
		return nil, errors.New("invalid Host boot document")
	}
	return body, nil
}

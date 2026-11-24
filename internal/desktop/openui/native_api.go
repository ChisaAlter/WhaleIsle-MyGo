package openui

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// Native calls of the public resource adapters for the native Whale UI
// renderer (internal/ui/openui). They run the same adapters, limits and
// public-address-only transports as the helper's /api/* bridge, as plain Go
// calls: no envelope, no instance, no HTTP between the renderer and Host.

// nativeEvent is the request the adapters validate; the instance only
// satisfies the response envelope check of the bridge path.
func nativeEvent(path string, query map[string]string) Event {
	return Event{Channel: "whale-openui", Version: 1, InstanceID: strings.Repeat("0", 64), Type: "api-request", RequestID: "native", Path: path, Query: query}
}

func nativeFailure(r APIResponse) error {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(r.Body, &body) == nil && body.Error.Message != "" {
		return errors.New(body.Error.Message)
	}
	return errors.New("public resource unavailable")
}

// PublicImage fetches a public http(s) image (the /api/image adapter: public
// addresses only, no credentials, redirects re-validated, 16 MiB budget) and
// returns its validated bytes for the media service to decode.
func (a *ResourceAPI) PublicImage(ctx context.Context, src string) ([]byte, error) {
	e := nativeEvent("/api/image", map[string]string{"src": src})
	if err := validateAPIRequest(e); err != nil {
		return nil, err
	}
	resource, failure := readPublicResource(ctx, e)
	if failure != nil {
		return nil, nativeFailure(*failure)
	}
	return resource.data, nil
}

// SearchImages runs the /api/image-search adapter and returns up to four
// public image URLs for query.
func (a *ResourceAPI) SearchImages(ctx context.Context, query string) ([]string, error) {
	e := nativeEvent("/api/image-search", map[string]string{"q": query})
	if err := validateAPIRequest(e); err != nil {
		return nil, err
	}
	r := a.images(ctx, e)
	if r.Status != 200 {
		return nil, nativeFailure(r)
	}
	var body struct {
		Images []string `json:"images"`
	}
	if err := json.Unmarshal(r.Body, &body); err != nil {
		return nil, err
	}
	return body.Images, nil
}

// SearchImage finds a picture for query: the first search result that loads.
func (a *ResourceAPI) SearchImage(ctx context.Context, query string) ([]byte, error) {
	urls, err := a.SearchImages(ctx, query)
	if err != nil {
		return nil, err
	}
	for _, u := range urls {
		data, err := a.PublicImage(ctx, u)
		if err == nil {
			return data, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, errors.New("image not found")
}

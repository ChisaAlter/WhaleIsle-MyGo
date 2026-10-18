package openui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxAPIResponseBytes matches the existing helper large-body transport budget.
// Event and state envelopes retain their smaller existing limits.
const MaxAPIResponseBytes = 25 << 20

// Resource-image URLs come only from the selected public adapter's JSON, never
// from an iframe-supplied URL. The exact Wikimedia media origin is fixed here.
func wikimediaImageURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "upload.wikimedia.org" && u.User == nil && u.Fragment == "" && strings.HasPrefix(u.Path, "/")
}
func (a *ResourceAPI) imageData(ctx context.Context, raw string, budget int) (string, error) {
	if !wikimediaImageURL(raw) {
		return "", errors.New("resource image is not from the supported Wikimedia origin")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "open-intelligent-ui (https://github.com/thesysdev/open-intelligent-ui)")
	res, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", errors.New("Wikimedia image unavailable")
	}
	limit := int64(budget) * 3 / 4
	if limit <= 0 {
		return "", errors.New("resource images exceed the helper response budget")
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > limit {
		return "", errors.New("resource image exceeds the helper response budget")
	}
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
	default:
		return "", errors.New("resource image is not a supported bitmap")
	}
	result := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	if len(result) > budget {
		return "", errors.New("resource image exceeds the helper response budget")
	}
	return result, nil
}

func (a *ResourceAPI) inlineResourceImages(ctx context.Context, e Event, r APIResponse) APIResponse {
	if r.Status != 200 || (e.Path != "/api/wiki" && e.Path != "/api/image-search") {
		return r
	}
	key, sourceKey := "photos", "photoSources"
	if e.Path == "/api/image-search" {
		key, sourceKey = "images", "imageSources"
	}
	var body map[string]json.RawMessage
	if json.Unmarshal(r.Body, &body) != nil {
		return apiError(e, 502, "Invalid public image response")
	}
	var sources []string
	if json.Unmarshal(body[key], &sources) != nil {
		return apiError(e, 502, "Invalid public image list")
	}
	// Keep the original source list unchanged alongside display-only data URLs.
	// The upstream adapter supplies no author/license fields; none are invented.
	body[sourceKey], _ = json.Marshal(sources)
	display := make([]string, 0, len(sources))
	budget := MaxAPIResponseBytes - len(r.Body) - len(body[sourceKey])
	for _, source := range sources {
		data, err := a.imageData(ctx, source, budget)
		if err != nil {
			return apiError(e, 502, err.Error())
		}
		display = append(display, data)
		budget -= len(data)
	}
	body[key], _ = json.Marshal(display)
	r.Body, _ = json.Marshal(body)
	if err := ValidateAPIResponse(r, e.InstanceID); err != nil {
		return apiError(e, 502, "Resource images exceed the helper response budget")
	}
	return r
}

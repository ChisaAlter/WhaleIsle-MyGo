package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/dsh"
)

// OpenInAppSource is the Host's existing directory-opening HTTP capability.
// File association/reveal RPCs are a different original control.
type OpenInAppSource struct {
	store    *Store
	client   *dsh.Client
	endpoint dsh.Endpoint
	http     *http.Client
	epoch    uint64
}

func (s *Store) OpenInAppSource() *OpenInAppSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		return nil
	}
	httpClient := *http.DefaultClient
	if s.client.HTTP != nil {
		httpClient = *s.client.HTTP
	}
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &OpenInAppSource{store: s, client: s.client, endpoint: s.client.Endpoint, http: &httpClient, epoch: s.composerHostGeneration}
}

func (o *OpenInAppSource) Key() string { return fmt.Sprintf("%p:%d", o.client, o.epoch) }
func (o *OpenInAppSource) currentLocked() bool {
	return o.store.client == o.client && o.store.composerHostGeneration == o.epoch && o.client.Endpoint == o.endpoint
}
func (o *OpenInAppSource) Current() bool {
	if o == nil {
		return false
	}
	o.store.mu.Lock()
	defer o.store.mu.Unlock()
	return o.currentLocked()
}

func (o *OpenInAppSource) request(ctx context.Context, method, path string, body []byte, limit int64) ([]byte, error) {
	if !o.Current() {
		return nil, errors.New("open-in-app Host changed")
	}
	endpoint, err := url.JoinPath(o.endpoint.Origin, path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	if o.endpoint.Cookie != "" {
		req.Header.Set("Cookie", o.endpoint.Cookie)
	}
	response, err := o.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("open-in-app: HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = errors.New("open-in-app response exceeds its image/catalog budget")
	}
	return data, err
}

func (o *OpenInAppSource) Applications(ctx context.Context) ([]string, error) {
	data, err := o.request(ctx, http.MethodGet, "/open-in-app/apps", nil, 64<<10)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Apps []string `json:"apps"`
	}
	if err = json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload.Apps, nil
}

func (o *OpenInAppSource) Icon(ctx context.Context, id string) ([]byte, error) {
	return o.request(ctx, http.MethodGet, "/open-in-app/icon/"+url.PathEscape(id), nil, imagepreview.MaxBytes)
}

// CaptureOpen freezes the selected Session/path at the gesture. It never consults the
// next selection to redirect an admitted launch, or retries an uncertain POST.
func (o *OpenInAppSource) CaptureOpen(sessionID, path, app string) (func(context.Context) error, error) {
	o.store.mu.Lock()
	valid := o.currentLocked() && sessionID != "" && o.store.selectedID == sessionID && path != ""
	cwd := false
	for _, row := range o.store.items {
		if row.SessionID == sessionID && row.Cwd == path {
			cwd = true
			break
		}
	}
	o.store.mu.Unlock()
	if !valid || !cwd {
		return nil, errors.New("open-in-app workspace changed")
	}
	body, err := json.Marshal(struct {
		App  string `json:"app"`
		Path string `json:"path"`
	}{app, path})
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error {
		_, err := o.request(ctx, http.MethodPost, "/open-in-app/open", body, 64<<10)
		return err
	}, nil
}

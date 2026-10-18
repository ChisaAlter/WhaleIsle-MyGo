package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const DefaultStopSequenceMS int64 = 500
const MaximumStopSequenceMS int64 = 2147483646

// ReadStopSequenceInterval reads the existing authenticated Host boot document.
// It interprets only the shortcuts plugin's JSON assignment, never JavaScript.
func (c *Client) ReadStopSequenceInterval(ctx context.Context) (int64, error) {
	origin, err := url.Parse(c.Endpoint.Origin)
	if err != nil || origin.Host == "" || origin.User != nil || (origin.Scheme != "http" && origin.Scheme != "https") {
		return 0, errors.New("invalid Host origin")
	}
	endpoint := *origin
	endpoint.Path, endpoint.RawPath, endpoint.RawQuery, endpoint.Fragment = "/", "", "", ""
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, err
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
		return 0, errors.New("Host shortcut configuration could not be loaded")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, errors.New("Host shortcut configuration unavailable")
	}
	const limit = 4 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(body) > limit {
		return 0, errors.New("invalid Host boot document")
	}
	return parseStopSequenceInterval(string(body))
}

func parseStopSequenceInterval(document string) (int64, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(document))
	interval := DefaultStopSequenceMS
	script := false
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if tokenizer.Err() != io.EOF {
				return 0, errors.New("invalid Host boot document")
			}
			return interval, nil
		case html.StartTagToken:
			token := tokenizer.Token()
			script = token.Data == "script"
			for _, attr := range token.Attr {
				if attr.Key == "src" {
					script = false
				}
			}
		case html.EndTagToken:
			script = false
		case html.TextToken:
			if !script {
				continue
			}
			text := strings.TrimSpace(string(tokenizer.Text()))
			const prefix = `globalThis["__DSH_SHORTCUTS_CONFIG__"]`
			if !strings.HasPrefix(text, prefix) {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(text, prefix))
			if !strings.HasPrefix(value, "=") {
				return 0, errors.New("invalid Host shortcut configuration")
			}
			value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value[1:]), ";"))
			var config map[string]json.RawMessage
			if json.Unmarshal([]byte(value), &config) != nil || config == nil {
				return 0, errors.New("invalid Host shortcut configuration")
			}
			raw, ok := config["stopSequenceMs"]
			if !ok {
				interval = DefaultStopSequenceMS
				continue
			}
			var ms int64
			if json.Unmarshal(raw, &ms) != nil || ms < 1 || ms > MaximumStopSequenceMS {
				return 0, errors.New("invalid Host stopSequenceMs")
			}
			interval = ms
		}
	}
}

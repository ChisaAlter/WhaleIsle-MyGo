package pad

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Origin, Cookie string
	HTTP           *http.Client
}
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("Pad HTTP %d: %s", e.Status, e.Message) }

func NewClient(origin, cookie string) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(origin, "/"))
	if err != nil || u.User != nil || u.Scheme != "http" && u.Scheme != "https" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("Pad: invalid Host origin")
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return nil, errors.New("Pad: exact loopback Host required")
	}
	if strings.ContainsAny(cookie, "\r\n") {
		return nil, errors.New("Pad: invalid Host session")
	}
	return &Client{Origin: u.String(), Cookie: cookie}, nil
}
func (c *Client) request(ctx context.Context, method, path string, value any) (*http.Response, error) {
	var body io.Reader
	if value != nil {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	r, err := http.NewRequestWithContext(ctx, method, c.Origin+path, body)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Cookie", c.Cookie)
	r.Header.Set("Accept", "application/json")
	if value != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	h := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if c.HTTP != nil {
		h = *c.HTTP
		h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	resp, err := h.Do(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&e)
		if e.Error == "" {
			e.Error = http.StatusText(resp.StatusCode)
		}
		return nil, &HTTPError{Status: resp.StatusCode, Message: e.Error}
	}
	return resp, nil
}
func (c *Client) json(ctx context.Context, method, path string, value, out any) error {
	resp, err := c.request(ctx, method, path, value)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}
func (c *Client) Act(ctx context.Context, a Action) error {
	var out RunResult
	if err := c.json(ctx, "POST", "/api/dsh-pad.action", a, &out); err != nil {
		return err
	}
	if !out.OK {
		return errors.New(out.Error)
	}
	return nil
}
func (c *Client) Models(ctx context.Context) (Catalog, error) {
	var out Catalog
	err := c.json(ctx, "GET", "/api/dsh-pad.models", nil, &out)
	return out, err
}
func (c *Client) Run(ctx context.Context, id, prompt, system string) (RunResult, error) {
	var out RunResult
	body := map[string]string{"panelId": id, "prompt": prompt}
	if system != "" {
		body["system"] = system
	}
	err := c.json(ctx, "POST", "/api/dsh-pad.run", body, &out)
	return out, err
}

type streamEvent struct {
	Type    string  `json:"type"`
	Panels  []Panel `json:"panels"`
	PanelID string  `json:"panelId"`
	Data    Data    `json:"data"`
}

func (c *Client) stream(ctx context.Context, ids []string, receive func(streamEvent)) error {
	resp, err := c.request(ctx, "GET", "/api/dsh-pad.stream?watch="+url.QueryEscape(strings.Join(ids, ",")), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return errors.New("Pad: expected event stream")
	}
	reader := bufio.NewReader(resp.Body)
	var data strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if line == "" {
			if data.Len() > 0 {
				var e streamEvent
				if err := json.Unmarshal([]byte(data.String()), &e); err != nil {
					return err
				}
				receive(e)
				data.Reset()
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
}
func requestContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 310*time.Second)
}

// Package platformview owns the account-only Platform document transport.
// Credentials remain in memory; ordinary browsing never receives this session.
package platformview

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Session struct {
	Origin           string            `json:"origin"`
	Token            string            `json:"token"`
	UserID           *string           `json:"userId"`
	EmbeddedPageDist string            `json:"embeddedPageDist"`
	RequestHeaders   map[string]string `json:"requestHeaders"`
}

func (s Session) Validate() error {
	u, e := url.Parse(s.Origin)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || s.Token == "" || strings.ContainsAny(s.Token, "\r\n") {
		return errors.New("Platform session unavailable")
	}
	for k, v := range s.RequestHeaders {
		if !validHeader(k) || strings.ContainsAny(v, "\r\n") {
			return errors.New("Platform session headers invalid")
		}
	}
	return nil
}
func validHeader(k string) bool {
	if k == "" {
		return false
	}
	for _, c := range k {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c)) {
			return false
		}
	}
	switch strings.ToLower(k) {
	case "host", "content-length", "connection", "transfer-encoding", "upgrade", "proxy-authorization", "proxy-connection":
		return false
	}
	return true
}

// Fetch only talks to the captured loopback Host, never follows redirects and
// never returns remote diagnostics (which might contain credentials).
func Fetch(ctx context.Context, origin, publisherToken string) (Session, error) {
	u, e := url.Parse(origin)
	if e != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || publisherToken == "" {
		return Session{}, errors.New("Platform Host unavailable")
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return Session{}, errors.New("Platform Host must be loopback")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(origin, "/")+"/dshd-platform/session", nil)
	if e != nil {
		return Session{}, errors.New("Platform session request failed")
	}
	req.Header.Set("Authorization", "Bearer "+publisherToken)
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, e := client.Do(req)
	if e != nil {
		return Session{}, errors.New("Platform session request failed")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return Session{}, errors.New("Platform session unavailable")
	}
	data, e := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if e != nil {
		return Session{}, errors.New("Platform session unavailable")
	}
	var body struct {
		Session *Session `json:"session"`
	}
	if json.Unmarshal(data, &body) != nil || body.Session == nil {
		return Session{}, errors.New("Platform account unavailable")
	}
	if e = body.Session.Validate(); e != nil {
		return Session{}, e
	}
	return *body.Session, nil
}

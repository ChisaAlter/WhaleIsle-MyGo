package harness

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// cookiePrefix names the browser-session cookie dsh web sets.
const cookiePrefix = "dsh-auth-"

// Endpoint is an authenticated dsh web origin.
type Endpoint struct {
	// Origin is scheme://host:port, without a path.
	Origin string
	// Cookie is the `name=value` pair sent with every request.
	Cookie string
}

// Redeem trades the one-shot launch token in readyURL for the session
// cookie. A URL without a token is an open server and needs no cookie.
func Redeem(ctx context.Context, client *http.Client, readyURL string) (Endpoint, error) {
	u, err := url.Parse(readyURL)
	if err != nil {
		return Endpoint{}, err
	}
	ep := Endpoint{Origin: u.Scheme + "://" + u.Host}
	if u.Query().Get("token") == "" {
		return ep, nil
	}
	noRedirect := *client
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
	if err != nil {
		return Endpoint{}, err
	}
	res, err := noRedirect.Do(req)
	if err != nil {
		return Endpoint{}, err
	}
	res.Body.Close()
	for _, c := range res.Cookies() {
		if strings.HasPrefix(c.Name, cookiePrefix) {
			ep.Cookie = c.Name + "=" + c.Value
			return ep, nil
		}
	}
	return Endpoint{}, fmt.Errorf("dsh web answered %d without a session cookie", res.StatusCode)
}

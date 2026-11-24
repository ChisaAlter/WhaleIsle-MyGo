package harness

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/startuptrace"
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
	if startuptrace.Enabled() {
		trace := &httptrace.ClientTrace{
			GetConn:              func(string) { startuptrace.Mark("host.auth.get_connection") },
			ConnectStart:         func(string, string) { startuptrace.Mark("host.auth.connect_start") },
			ConnectDone:          func(string, string, error) { startuptrace.Mark("host.auth.connect_done") },
			GotConn:              func(httptrace.GotConnInfo) { startuptrace.Mark("host.auth.got_connection") },
			WroteRequest:         func(httptrace.WroteRequestInfo) { startuptrace.Mark("host.auth.request_written") },
			GotFirstResponseByte: func() { startuptrace.Mark("host.auth.first_response_byte") },
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
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
	return Endpoint{}, &redeemStatusError{status: res.StatusCode}
}

// redeemStatusError is a completed token exchange that set no session cookie.
type redeemStatusError struct{ status int }

func (e *redeemStatusError) Error() string {
	return fmt.Sprintf("dsh web answered %d without a session cookie", e.status)
}

// redeemBackoff paces the retries of a failed token exchange: the Host's
// socket can reset while its routes settle, which is no reason to kill it.
var redeemBackoff = []time.Duration{250 * time.Millisecond, 500 * time.Millisecond, time.Second}

// redeemWithRetry runs Redeem and repeats it up to three times when the
// failure is transient (a reset, refusal or timeout, or a 429/5xx answer).
// The caller's ctx bounds the whole exchange.
func redeemWithRetry(ctx context.Context, client *http.Client, readyURL string) (Endpoint, error) {
	ep, err := Redeem(ctx, client, readyURL)
	for i := 0; err != nil && i < len(redeemBackoff) && redeemRetryable(err) && ctx.Err() == nil; i++ {
		startuptrace.Mark("host.auth.retry", strconv.Itoa(i+1))
		timer := time.NewTimer(redeemBackoff[i])
		select {
		case <-ctx.Done():
			timer.Stop()
			return ep, err
		case <-timer.C:
		}
		ep, err = Redeem(ctx, client, readyURL)
	}
	return ep, err
}

func redeemRetryable(err error) bool {
	var status *redeemStatusError
	if errors.As(err, &status) {
		return status.status == http.StatusTooManyRequests || status.status >= 500
	}
	return (&dsh.TransportError{Err: err}).Temporary()
}

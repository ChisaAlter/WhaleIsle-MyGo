package dsh

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
)

// DownloadSessionLog streams the published Host archive, including descendants
// and attachments. It is not a text export of the currently loaded chat page.
func (c *Client) DownloadSessionLog(ctx context.Context, sessionID string, destination io.Writer) error {
	if sessionID == "" {
		return fmt.Errorf("select a session before exporting its log")
	}
	endpoint, err := url.Parse(c.Endpoint.Origin)
	if err != nil {
		return err
	}
	endpoint.Path = "/api/session.export"
	endpoint.RawPath = ""
	endpoint.RawQuery = url.Values{"sessionId": {sessionID}, "includeDescendants": {"true"}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", c.Endpoint.Cookie)
	req.Header.Set("Accept", "application/zip")
	carrier := *c.http()
	// The Host owns this stream; its authenticated request must not follow an
	// unrelated redirect or hand its cookie to another origin.
	carrier.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := carrier.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("session log export failed (HTTP %d)", response.StatusCode)
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "application/zip" {
		return fmt.Errorf("the Host did not return a session ZIP archive")
	}
	_, err = io.Copy(destination, response.Body)
	return err
}

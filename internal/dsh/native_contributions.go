package dsh

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NativeContributions is an authenticated declaration for five fixed Go
// adapters. It does not assert browser plugin activation or grant code-loading
// authority. The caller owns the EventsOwner generation and stale-publication
// fence; cancellation withdraws that owner's declarations in the caller.
type NativeContributions struct {
	Known                                                                         bool
	Revision, ReleaseVersion, ReleaseCommit                                       string
	FileCommand, ModelCommand, PermissionCommand, FeedbackCommand, ModelLabel1017 bool
	Error                                                                         string
	fileRevision, modelRevision, permissionRevision, feedbackRevision             string
}

// CommandRevision identifies the fixed command owner's code and dependencies,
// rather than unrelated changes to the Host graph. An unavailable or unknown
// port has no identity. Release and connection ownership remain caller scopes.
func (n NativeContributions) CommandRevision(name string) string {
	if !n.Known {
		return ""
	}
	switch name {
	case "file":
		if n.FileCommand {
			return n.fileRevision
		}
	case "model":
		if n.ModelCommand {
			return n.modelRevision
		}
	case "permission":
		if n.PermissionCommand {
			return n.permissionRevision
		}
	case "feedback":
		if n.FeedbackCommand {
			return n.feedbackRevision
		}
	}
	return ""
}

const nativeContributionLimit = 4 << 20

type nativeContributionWatch struct {
	graph     *nativeContributionGraph
	accepting bool
	last      NativeContributions
	published bool
	publish   func(NativeContributions)
}

func (w *nativeContributionWatch) emit(value NativeContributions) {
	if !w.published || w.last != value {
		w.last, w.published = value, true
		w.publish(value)
	}
}

func (w *nativeContributionWatch) unavailable(message string) {
	w.accepting = false
	w.emit(NativeContributions{Error: message})
}

// WatchNativeContributions consumes the existing authenticated /plugins/events
// graph stream, without fetching boot HTML, bundles or graph URLs. Like the
// original loopback EventSource, a transport disconnect withdraws authority and
// reconnects after three seconds. HTTP refusal, cross-origin redirects, a
// non-SSE response or a different Host release require a new caller-owned watch.
func (c *Client) WatchNativeContributions(ctx context.Context, publish func(NativeContributions)) error {
	if publish == nil {
		return errors.New("native contribution receiver is missing")
	}
	w := nativeContributionWatch{publish: publish}
	endpoint := c.Endpoint // Bind redirects and reconnects to this generation.
	origin, err := url.Parse(endpoint.Origin)
	if err != nil || origin.Hostname() == "" || origin.User != nil || origin.Opaque != "" || (origin.Scheme != "http" && origin.Scheme != "https") || (origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
		err = errors.New("native contributions require a valid Host origin")
		w.unavailable(err.Error())
		return err
	}
	// The original HMR client opens EventSource only for a loopback page when
	// no shell transport exists. Native ports have no shell transport either;
	// an authenticated remote RPC endpoint must not gain an HMR connection.
	hostname := origin.Hostname()
	ip := net.ParseIP(hostname)
	if !strings.EqualFold(hostname, "localhost") && (ip == nil || !ip.IsLoopback()) {
		err = errors.New("native contributions require a loopback Host origin")
		w.unavailable(err.Error())
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		response, reconnect, err := c.openNativeContributions(ctx, origin, endpoint.Cookie)
		if err == nil {
			w.accepting = false // Every connection must first supply a valid graph.
			err = w.read(ctx, response.Body)
			response.Body.Close()
			reconnect = !errors.Is(err, errNativeContributionRelease)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		w.unavailable(err.Error())
		if !reconnect {
			return err
		}
		// This is the existing EventSource reconnect cadence, not RPC retry,
		// backoff, an independent poller or a second concurrent stream.
		timer := time.NewTimer(3 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) openNativeContributions(ctx context.Context, origin *url.URL, cookie string) (*http.Response, bool, error) {
	endpoint := *origin
	endpoint.Path, endpoint.RawPath = "/plugins/events", ""
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, false, errors.New("native contribution request is invalid")
	}
	req.Header.Set("Accept", "text/event-stream")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	client := *c.http()
	previous := client.CheckRedirect
	refusedRedirect := false
	sameOrigin := func(u *url.URL) bool {
		return u != nil && u.User == nil && u.Scheme == origin.Scheme && strings.EqualFold(u.Host, origin.Host)
	}
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if !sameOrigin(next.URL) || len(via) >= 10 {
			refusedRedirect = true
			return errors.New("native contribution redirect refused")
		}
		if previous != nil {
			if err := previous(next, via); err != nil {
				refusedRedirect = true
				return errors.New("native contribution redirect refused")
			}
		}
		if !sameOrigin(next.URL) {
			refusedRedirect = true
			return errors.New("native contribution redirect refused")
		}
		return nil
	}
	response, err := client.Do(req)
	if err != nil {
		// Never return url.Error, transport text, cookies or response bodies.
		if refusedRedirect {
			return nil, false, errors.New("native contribution redirect refused")
		}
		return nil, true, errors.New("native contribution connection failed")
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, false, errors.New("Host refused the native contribution stream")
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "text/event-stream" {
		response.Body.Close()
		return nil, false, errors.New("Host did not return a native contribution event stream")
	}
	return response, true, nil
}

var errNativeContributionRelease = errors.New("Host release changed; native contributions require a new connection")

func (w *nativeContributionWatch) receive(raw []byte) error {
	frame, ok := nativeContributionObject(raw)
	if !ok {
		w.unavailable("invalid native contribution event")
		return nil
	}
	kind, ok := nativeContributionString(frame["type"])
	if !ok {
		w.unavailable("invalid native contribution event")
		return nil
	}
	switch kind {
	case "graph":
		graph, err := parseNativeContributionGraph(frame["graph"])
		if err != nil {
			w.unavailable("invalid native contribution graph")
			return nil
		}
		if w.graph != nil && !sameNativeContributionRelease(w.graph.Release, graph.Release) {
			return errNativeContributionRelease
		}
		w.graph, w.accepting = graph, true
		w.emit(graph.snapshot())
	case "rebuilt":
		id, idOK := nativeContributionString(frame["id"])
		rev, revOK := nativeContributionString(frame["rev"])
		if !idOK || !revOK {
			w.unavailable("invalid native contribution rebuilt event")
			return nil
		}
		if !w.accepting || w.graph == nil {
			return nil
		}
		// Original ClientEntries.reload ignores entries absent from its graph
		// and duplicate artifact revisions. A rebuild never invents an owner.
		if index, exists := w.graph.index[id]; exists && w.graph.Entries[index].Rev != rev {
			w.graph.Entries[index].Rev = rev
			w.emit(w.graph.snapshot())
		}
	default:
		// Forward-compatible unknown types carry no native authority.
	}
	return nil
}

func (w *nativeContributionWatch) read(ctx context.Context, body io.Reader) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 4096), nativeContributionLimit+1)
	scanner.Split(nativeContributionLines)
	var data bytes.Buffer
	hasData, oversized, firstLine := false, false, true
	event := ""
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Text()
		if firstLine {
			line = strings.TrimPrefix(line, "\uFEFF")
			firstLine = false
		}
		if line == "" {
			if hasData && !oversized && (event == "" || event == "message") {
				if err := w.receive(bytes.TrimSuffix(data.Bytes(), []byte{'\n'})); err != nil {
					return err
				}
			}
			data.Reset()
			hasData, oversized, event = false, false, ""
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "data":
			hasData = true
			if oversized {
				continue
			}
			if data.Len()+len(value)+1 > nativeContributionLimit {
				oversized = true
				data.Reset()
				w.unavailable("native contribution event exceeds the wire limit")
				continue
			}
			data.WriteString(value)
			data.WriteByte('\n')
		case "event":
			if value == "" || value == "message" {
				event = value
			} else {
				event = "other"
			}
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// EventSource does not dispatch an unterminated final event on EOF.
	if scanner.Err() != nil {
		return errors.New("native contribution stream could not be read")
	}
	return errors.New("native contribution stream disconnected")
}

func nativeContributionLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if index := bytes.IndexAny(data, "\r\n"); index >= 0 {
		if data[index] == '\r' && index+1 == len(data) && !atEOF {
			return 0, nil, nil
		}
		advance = index + 1
		if data[index] == '\r' && advance < len(data) && data[advance] == '\n' {
			advance++
		}
		return advance, data[:index], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

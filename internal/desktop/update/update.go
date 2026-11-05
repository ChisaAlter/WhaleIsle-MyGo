package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// EnvFeedURL is the optional JSON update feed URL (replaces electron-updater).
const EnvFeedURL = "WHALEISLE_UPDATE_FEED"

// EnvFeedURLGithub / EnvFeedURLCNB are optional route-specific feeds (Electron downloadRoute).
const (
	EnvFeedURLGithub = "WHALEISLE_UPDATE_FEED_GITHUB"
	EnvFeedURLCNB    = "WHALEISLE_UPDATE_FEED_CNB"
)

// Status is the outcome of a single update check.
type Status string

const (
	// StatusNotConfigured means no update feed or channel is configured yet.
	StatusNotConfigured Status = "not_configured"
	// StatusUpToDate means the running build matches the latest published version.
	StatusUpToDate Status = "up_to_date"
	// StatusUpdateAvailable means a newer signed build is available to install.
	StatusUpdateAvailable Status = "update_available"
	// StatusError means the feed was configured but the check failed.
	StatusError Status = "error"
)

// Result is returned by CheckForUpdates.
type Result struct {
	Status     Status
	Message    string
	Version    string
	ReleaseURL string
	SHA256     string
	Size       int64
	// IncrementalPossible is true when CanIncremental(current, feed) — heuristic
	// for ApplyIncremental / ApplyBlockPatch (Range patch when a previous installer exists).
	IncrementalPossible bool
	// Feed is the parsed feed document (for ApplyIncremental); zero when not configured.
	Feed Feed
}

// Options configures a networked update check.
type Options struct {
	CurrentVersion string
	FeedURL        string
	HTTPClient     *http.Client
}

// Feed is the JSON update feed document served at WHALEISLE_UPDATE_FEED.
//
// Optional incremental fields (M11 — ApplyBlockPatch when a previous installer
// exists; otherwise ApplyIncremental full-downloads then VerifyBlockMap):
//
//	blockMapURL — URL of a blockmap JSON document (see BlockMapInfo)
//	blockMap    — inline blockmap with the same schema
type Feed struct {
	Version     string        `json:"version"`
	URL         string        `json:"url"`
	Notes       string        `json:"notes"`
	SHA256      string        `json:"sha256"`
	Size        int64         `json:"size"`
	BlockMapURL string        `json:"blockMapURL,omitempty"`
	BlockMap    *BlockMapInfo `json:"blockMap,omitempty"`
}

// CheckForUpdates uses Options from the environment when feed/version are empty.
func CheckForUpdates(ctx context.Context) (Result, error) {
	return Check(ctx, Options{
		CurrentVersion: os.Getenv("WHALEISLE_VERSION"),
		FeedURL:        strings.TrimSpace(os.Getenv(EnvFeedURL)),
	})
}

// CheckForUpdatesVersion is like CheckForUpdates but pins the running version.
func CheckForUpdatesVersion(ctx context.Context, currentVersion string) (Result, error) {
	return CheckForUpdatesVersionRoute(ctx, currentVersion, "")
}

// CheckForUpdatesVersionRoute pins the running version and selects a feed by
// Electron downloadRoute ("", "github", "cnb"). Explicit WHALEISLE_UPDATE_FEED wins.
func CheckForUpdatesVersionRoute(ctx context.Context, currentVersion, route string) (Result, error) {
	return Check(ctx, Options{
		CurrentVersion: currentVersion,
		FeedURL:        FeedURLForRoute(route),
	})
}

// FeedURLForRoute resolves the update feed for a launcher download route.
func FeedURLForRoute(route string) string {
	if u := strings.TrimSpace(os.Getenv(EnvFeedURL)); u != "" {
		return u
	}
	switch strings.ToLower(strings.TrimSpace(route)) {
	case "cnb":
		if u := strings.TrimSpace(os.Getenv(EnvFeedURLCNB)); u != "" {
			return u
		}
	case "github":
		if u := strings.TrimSpace(os.Getenv(EnvFeedURLGithub)); u != "" {
			return u
		}
	}
	if u := strings.TrimSpace(os.Getenv(EnvFeedURLGithub)); u != "" {
		return u
	}
	return strings.TrimSpace(os.Getenv(EnvFeedURLCNB))
}

// Check performs a feed lookup when FeedURL is set; otherwise returns not_configured.
func Check(ctx context.Context, opt Options) (Result, error) {
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	feed := strings.TrimSpace(opt.FeedURL)
	if feed == "" {
		return Result{
			Status:  StatusNotConfigured,
			Message: "MyGo updater feed not configured (set " + EnvFeedURL + ")",
		}, nil
	}
	if err := validateUpdateURL(feed); err != nil {
		return Result{Status: StatusError, Message: err.Error()}, err
	}
	client := opt.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, Transport: updateTransport()}
	} else {
		clone := *client
		client = &clone
	}
	client.CheckRedirect = func(next *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed, nil)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("update feed HTTP %d", resp.StatusCode)
		return Result{Status: StatusError, Message: err.Error()}, err
	}
	var payload Feed
	if err := json.Unmarshal(body, &payload); err != nil {
		return Result{Status: StatusError, Message: "invalid update feed JSON"}, err
	}
	latest := strings.TrimSpace(payload.Version)
	if latest == "" {
		err := fmt.Errorf("update feed missing version")
		return Result{Status: StatusError, Message: err.Error()}, err
	}
	current := strings.TrimSpace(opt.CurrentVersion)
	if current == "" {
		current = "dev"
	}
	rel := strings.TrimSpace(payload.URL)
	sha := strings.TrimSpace(payload.SHA256)
	inc := CanIncremental(current, payload)
	if CompareVersions(latest, current) > 0 {
		msg := "Update available: " + latest
		if notes := strings.TrimSpace(payload.Notes); notes != "" {
			msg += "\n" + notes
		}
		if inc {
			msg += "\n(blockmap present; block patch when a previous installer is available)"
		}
		return Result{
			Status:              StatusUpdateAvailable,
			Message:             msg,
			Version:             latest,
			ReleaseURL:          rel,
			SHA256:              sha,
			Size:                payload.Size,
			IncrementalPossible: inc,
			Feed:                payload,
		}, nil
	}
	return Result{
		Status:              StatusUpToDate,
		Message:             "Up to date (" + current + ")",
		Version:             latest,
		ReleaseURL:          rel,
		SHA256:              sha,
		Size:                payload.Size,
		IncrementalPossible: inc,
		Feed:                payload,
	}, nil
}

func validateUpdateURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Hostname() == "" {
		return fmt.Errorf("update feed URL must be an HTTPS URL without userinfo")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && (host == "localhost" || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())) {
		return nil
	}
	return fmt.Errorf("update feed URL must use HTTPS (HTTP is allowed only for loopback tests)")
}

// CompareVersions returns -1 if a<b, 0 if equal, 1 if a>b for dotted numeric
// versions (non-numeric suffixes are stripped per segment).
func CompareVersions(a, b string) int {
	as := versionParts(a)
	bs := versionParts(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func versionParts(v string) []int {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.ToLower(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				break
			}
			n = n*10 + int(ch-'0')
		}
		out = append(out, n)
	}
	return out
}

package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeedURLForRoute(t *testing.T) {
	t.Setenv(EnvFeedURL, "")
	t.Setenv(EnvFeedURLGithub, "https://example.com/gh.json")
	t.Setenv(EnvFeedURLCNB, "https://example.com/cnb.json")
	if got := FeedURLForRoute("cnb"); got != "https://example.com/cnb.json" {
		t.Fatalf("cnb = %q", got)
	}
	if got := FeedURLForRoute("github"); got != "https://example.com/gh.json" {
		t.Fatalf("github = %q", got)
	}
	t.Setenv(EnvFeedURL, "https://example.com/primary.json")
	if got := FeedURLForRoute("cnb"); got != "https://example.com/primary.json" {
		t.Fatalf("primary wins = %q", got)
	}
}

func TestCheckForUpdatesNotConfigured(t *testing.T) {
	t.Setenv(EnvFeedURL, "")
	t.Setenv(EnvFeedURLGithub, "")
	t.Setenv(EnvFeedURLCNB, "")
	t.Setenv("WHALEISLE_VERSION", "")
	res, err := CheckForUpdates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusNotConfigured {
		t.Fatalf("Status = %q", res.Status)
	}
	if res.Message == "" {
		t.Fatal("expected message")
	}
	if res.Version != "" || res.ReleaseURL != "" {
		t.Fatalf("unexpected fields: %+v", res)
	}
}

func TestCheckForUpdatesRespectsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := CheckForUpdates(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestCheckUpdateAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Feed{
			Version: "1.2.0",
			URL:     "https://example.com/WhaleIsle-1.2.0.exe",
			Notes:   "bugfix",
		})
	}))
	defer srv.Close()

	res, err := Check(context.Background(), Options{
		CurrentVersion: "1.0.0",
		FeedURL:        srv.URL,
		HTTPClient:     srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusUpdateAvailable {
		t.Fatalf("Status = %q msg=%q", res.Status, res.Message)
	}
	if res.Version != "1.2.0" || res.ReleaseURL == "" {
		t.Fatalf("fields: %+v", res)
	}
}

func TestCheckUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.0.0"})
	}))
	defer srv.Close()

	res, err := Check(context.Background(), Options{
		CurrentVersion: "1.0.0",
		FeedURL:        srv.URL,
		HTTPClient:     srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusUpToDate {
		t.Fatalf("Status = %q", res.Status)
	}
}

func TestCompareVersions(t *testing.T) {
	if CompareVersions("1.2.0", "1.1.9") <= 0 {
		t.Fatal("1.2.0 should be newer")
	}
	if CompareVersions("v1.0.0", "1.0.0") != 0 {
		t.Fatal("v prefix")
	}
	if CompareVersions("1.0.0-beta", "1.0.0") >= 0 {
		t.Fatal("a prerelease must precede the final release")
	}
}

func TestCompareVersionsPrereleasePrecedence(t *testing.T) {
	ordered := []string{
		"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta",
		"1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0",
	}
	for i := 1; i < len(ordered); i++ {
		if CompareVersions(ordered[i-1], ordered[i]) >= 0 || CompareVersions(ordered[i], ordered[i-1]) <= 0 {
			t.Fatalf("incorrect prerelease order: %s < %s", ordered[i-1], ordered[i])
		}
	}
	for _, pair := range [][2]string{{"v1.2", "1.2.0"}, {"1.2.3+build.9", "1.2.3+other"}, {"1.2.3-rc.2+build", "1.2.3-rc.2"}} {
		if CompareVersions(pair[0], pair[1]) != 0 {
			t.Fatalf("equivalent versions differ: %q", pair)
		}
	}
	if CompareVersions("1.0.0-rc.999999999999999999999", "1.0.0-rc.1000000000000000000000") >= 0 {
		t.Fatal("numeric prerelease identifiers overflowed")
	}
}

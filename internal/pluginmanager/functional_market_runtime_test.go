//go:build functionalacceptance

package pluginmanager_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	manager "whaleisle/internal/pluginmanager"
)

// The experimental bundle is explicitly mounted only for this fresh QA Host.
// The application's default profile and its observed 404 remain unchanged.
func TestFunctionalOwnedMarketCatalog(t *testing.T) {
	stage, base, output := os.Getenv("WHALEISLE_FUNCTIONAL_RUNTIME"), os.Getenv("WHALEISLE_FUNCTIONAL_ROOT"), os.Getenv("WHALEISLE_FUNCTIONAL_OUTPUT")
	for _, p := range []string{stage, base, output} {
		if !filepath.IsAbs(p) {
			t.Fatal("explicit absolute runtime, owned parent and evidence output required")
		}
	}
	root, e := os.MkdirTemp(base, "market-functional-")
	if e != nil {
		t.Fatal(e)
	}
	type check struct{ Name, Status, Detail string }
	report := struct {
		Root           string  `json:"ownedRoot"`
		Runtime        string  `json:"runtime"`
		Scope          string  `json:"scope"`
		Checks         []check `json:"checks"`
		HostStopped    bool    `json:"ownedHostStopped"`
		ListenerClosed bool    `json:"ownedListenerClosed"`
		ValidFetches   int64   `json:"ownIndexFetches"`
	}{Root: root, Runtime: stage, Scope: "Explicit original experimental bundle only in fresh QA; original default-profile 404 unchanged. Actual Host+production pluginmanager Client/Controller, own loopback catalog, no installation/auth/provider/model/user GUI."}
	write := func() {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(output, append(raw, '\n'), 0600); err != nil {
			t.Errorf("save evidence: %v", err)
		}
	}
	add := func(name, status, detail string) {
		report.Checks = append(report.Checks, check{name, status, detail})
		write()
		t.Log(name, status, detail)
	}
	failure := func(err error) string {
		if err == nil {
			return ""
		}
		sum := sha256.Sum256([]byte(err.Error()))
		code := "operation-error"
		var rpc *dsh.Error
		if errors.As(err, &rpc) {
			code = rpc.Code
		}
		return code + " sha256:" + hex.EncodeToString(sum[:])
	}
	var fetches atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/bad.json" {
			_, _ = w.Write([]byte(`{"format":2}`))
			return
		}
		if r.URL.Path != "/index.json" {
			http.NotFound(w, r)
			return
		}
		fetches.Add(1)
		_, _ = w.Write([]byte(`{"format":1,"title":{"en":"Owned fixture market","zh":"自有目录"},"plugins":[{"name":"functional-market-alpha","title":{"en":"Alpha needle","zh":"甲"},"versions":[{"version":"1.0.0","publishTimestamp":1,"source":{"type":"npm"}}]},{"name":"functional-market-beta","title":"Beta","versions":[{"version":"1.0.0","publishTimestamp":2,"source":{"type":"npm"},"engines":{"dsh":"0.0.1"}}]}]}`))
	}))
	defer server.Close()
	workspace, ownedHome := filepath.Join(root, "workspace"), filepath.Join(root, "owned-home")
	for _, p := range []string{workspace, ownedHome, filepath.Join(root, "appdata"), filepath.Join(root, "local-appdata")} {
		if e = os.MkdirAll(p, 0700); e != nil {
			t.Fatal(e)
		}
	}
	t.Setenv(paths.EnvUserData, filepath.Join(root, "user-data"))
	t.Setenv(paths.EnvVendor, filepath.Join(stage, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(root, "user-data", "dsh-home"))
	vendor, home := paths.VendorDir(), paths.DshHome()
	hroot := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	patch := filepath.Join(hroot, "packages", "experimental", "plugin-market-bundle", "cordis.patch.yml")
	if _, e = os.Stat(patch); e != nil {
		add("explicit original bundle", "blocked", "bundled original patch absent")
		return
	}
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: hroot, VendorDir: vendor, Host: "127.0.0.1", Port: 36800, Home: home, ExtraEnv: []string{"HOME=" + ownedHome, "USERPROFILE=" + ownedHome, "APPDATA=" + filepath.Join(root, "appdata"), "LOCALAPPDATA=" + filepath.Join(root, "local-appdata"), "XDG_CONFIG_HOME=" + filepath.Join(ownedHome, ".config"), "XDG_DATA_HOME=" + filepath.Join(ownedHome, ".local", "share"), "COREPACK_ENABLE_NETWORK=0", "npm_config_offline=true"}}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: hroot, DshHome: home, ProfileDir: paths.WebProfileDir(), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var endpoint harness.Endpoint
	stop := func() {
		sup.Stop()
		report.HostStopped = sup.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			u, _ := url.Parse(endpoint.Origin)
			conn, err := net.DialTimeout("tcp", u.Host, time.Second)
			report.ListenerClosed = err != nil
			if conn != nil {
				conn.Close()
			}
		}
	}
	defer func() { stop(); report.ValidFetches = fetches.Load(); write() }()
	start := func() *dsh.Client {
		sup.Start()
		until := time.Now().Add(195 * time.Second)
		for time.Now().Before(until) {
			st := sup.Status()
			if st.State == harness.Ready {
				endpoint = st.Endpoint
				return &dsh.Client{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}, HTTP: &http.Client{Timeout: 45 * time.Second}}
			}
			if st.State == harness.Failed {
				add("owned Host", "blocked", failure(st.Err))
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
		add("owned Host", "blocked", "readiness timeout")
		return nil
	}
	client := start()
	if client == nil {
		return
	}
	add("owned Host", "pass", "fresh HOME/USERPROFILE/AppData; explicit original experimental bundle, no dependencies installed")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pc := manager.Client{Caller: client}
	bundleName := "@deepseek-ai/dsh-experimental-plugin-market-bundle"
	change, e := pc.SetBundleEnabled(ctx, bundleName, true)
	if e != nil || change.Error != nil {
		detail := failure(e)
		if change.Error != nil {
			detail += " " + change.Error.Code
		}
		add("PLUG-02 explicit bundle", "blocked", detail)
		return
	}
	if change.Application == "restart-required" {
		stop()
		client = start()
		if client == nil {
			return
		}
		pc = manager.Client{Caller: client}
	}
	add("PLUG-02 explicit bundle", "pass", "original manager selected already-bundled package in owned profile; no package installation")
	indexes, e := pc.Indexes(ctx)
	if e != nil {
		add("PLUG-02 explicit bundle", "blocked", failure(e))
		return
	}
	if len(indexes) != 0 {
		t.Fatal("fresh experimental bundle unexpectedly has external sources")
	}
	add("PLUG-02 served indexes", "pass", "empty original default indexes, no remote fetch")
	valid, bad := server.URL+"/index.json", server.URL+"/bad.json"
	if e = pc.SetIndexes(ctx, []string{valid, bad}); e != nil {
		add("PLUG-02 own sources", "fail", failure(e))
		t.Fatal("owned sources rejected")
	}
	first, e := pc.Search(ctx, valid, manager.Query{Sort: manager.MarketSort{Kind: "index"}, Limit: 1})
	if e != nil {
		add("PLUG-02 search", "fail", failure(e))
		t.Fatal("local index search failed")
	}
	if first.Total != 2 || len(first.Entries) != 1 || first.Entries[0].Name != "functional-market-alpha" || first.NextOffset == nil || *first.NextOffset != 1 {
		t.Fatal("first page contract mismatch")
	}
	second, e := pc.Search(ctx, valid, manager.Query{Sort: manager.MarketSort{Kind: "index"}, Offset: 1, Limit: 1, FetchedAt: first.FetchedAt})
	if e != nil {
		t.Fatal(failure(e))
	}
	if len(second.Entries) != 1 || second.Entries[0].Name != "functional-market-beta" || second.NextOffset != nil || second.FetchedAt != first.FetchedAt || fetches.Load() != 1 {
		t.Fatal("cached paging contract mismatch")
	}
	if second.Entries[0].Versions[0].Compatibility.Compatible || second.Entries[0].DefaultVersion != "" {
		t.Fatal("incompatible fixture gained an install default")
	}
	add("PLUG-02 catalog paging", "pass", "two real parser entries, localized text, cached fetchedAt, incompatible version has no default")
	filtered, e := pc.Search(ctx, valid, manager.Query{Text: "needle", Sort: manager.MarketSort{Kind: "relevance"}, Limit: 20})
	if e != nil || filtered.Total != 1 || len(filtered.Entries) != 1 || fetches.Load() != 1 {
		t.Fatal("filtered/cache read mismatch: " + failure(e))
	}
	add("PLUG-02 search/cache", "pass", "original search worker filtered local catalog; cached reads did not refetch")
	_, e = pc.Search(ctx, valid, manager.Query{Sort: manager.MarketSort{Kind: "index"}, Limit: 20, Refresh: true})
	if e != nil || fetches.Load() != 2 {
		t.Fatal("explicit refresh mismatch: " + failure(e))
	}
	add("PLUG-02 refresh", "pass", "explicit refresh fetched own index exactly once more")
	_, e = pc.Search(ctx, bad, manager.Query{Sort: manager.MarketSort{Kind: "index"}, Limit: 20})
	var rpc *dsh.Error
	if !errors.As(e, &rpc) || rpc.Code != "plugin-market/invalid-index" {
		t.Fatal("malformed index did not retain typed refusal: " + failure(e))
	}
	add("PLUG-02 invalid source", "pass", "original parser returned typed plugin-market/invalid-index; no install attempted")
	c := manager.New()
	c.Attach(client, nil)
	defer c.Detach()
	if e = c.LoadIndexes(ctx, c.Snapshot().Owner); e != nil {
		t.Fatal(failure(e))
	}
	s := c.Snapshot()
	if s.Discover.Status != "ready" || s.Discover.Listing == nil || len(s.Discover.Listing.Entries) != 2 {
		t.Fatal("production Discover did not accept real catalog")
	}
	add("PLUG-02 production Discover", "pass", "Controller decoded real index sources and populated local listing")
	stop()
	client = start()
	if client == nil {
		return
	}
	pc = manager.Client{Caller: client}
	indexes, e = pc.Indexes(ctx)
	if e != nil || len(indexes) != 2 || indexes[0].URL != valid || indexes[1].URL != bad {
		t.Fatal("own source list not durable: " + failure(e))
	}
	add("PLUG-02 source persistence", "pass", "own configured URLs retained after actual Host restart")
	if e = pc.SetIndexes(ctx, []string{}); e != nil {
		t.Fatal("restore empty source list: " + failure(e))
	}
	indexes, e = pc.Indexes(ctx)
	if e != nil || len(indexes) != 0 {
		t.Fatal("empty source list restore failed")
	}
	add("PLUG-02 source restore", "pass", "own experimental profile restored to empty source list")
	change, e = pc.SetBundleEnabled(ctx, bundleName, false)
	if e != nil || change.Error != nil {
		t.Fatal("own experimental bundle restore failed: " + failure(e))
	}
	add("PLUG-02 bundle restore", "pass", "experimental bundle disabled again in owned QA profile")
}

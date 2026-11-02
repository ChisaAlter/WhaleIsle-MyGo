package sidebar

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSessionSearchInputPreservesMultiwordSpacesAndBounds(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(previous)
	query, menu := "", false
	state := sessionSearchState{Expanded: true, Focus: true}
	m := Model{SearchQuery: &query, searchState: &state}
	colors := theme.Resolve("deepseek.light").Light
	tt := ui.NewTester(func(c *ui.Context) {
		workspaceHeading(c, m, &menu, chrome.Color(colors.FG), chrome.Color(colors.Muted))
	}, 340, 80)
	tt.Type("foo ")
	if query != "foo " {
		t.Fatalf("typing a separating space changed input: %q", query)
	}
	tt.Type("bar")
	if query != "foo bar" || session.NormalizeSessionSearchQuery(query) != "foo bar" {
		t.Fatalf("multiword input lost: %q", query)
	}
	if err := tt.SetValue(i18n.T(i18n.KeySidebarSearch), strings.Repeat("中", 499)+"\x00😀文"); err != nil {
		t.Fatal(err)
	}
	if query != strings.Repeat("中", 499) {
		t.Fatal("bound search input retained NUL or split the 500th UTF16 pair")
	}
}

func TestSessionSearchProjectionUsesSameCanonicalQueryAsScopeAndRPC(t *testing.T) {
	prefix := strings.Repeat("中", 498) + "😀"
	first, second := prefix+strings.Repeat("甲", 1000), prefix+strings.Repeat("乙", 1000)
	snap := session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch", SearchRPCQuery: prefix,
		Items:         []session.Item{{SessionID: "local", Title: prefix, Cwd: "/scratch"}, {SessionID: "remote", Title: "body match", Cwd: "/scratch"}},
		SearchResults: []api.SessionSearchValueItemsItem{{SessionId: "remote", Snippet: "found by canonical wire query"}},
	}
	for _, raw := range []string{first, second, "\x00" + second} {
		rows, _ := deriveSessionSearchResults(snap, raw)
		if len(rows) != 2 || rows[0].Item.SessionID != "local" || rows[1].Item.SessionID != "remote" || rows[1].Snippet == "" {
			t.Fatalf("local/content search disagrees for a suffix beyond the limit: %+v", rows)
		}
		if got := BatchSelectionScope(Model{Snapshot: snap, SearchQuery: &raw}).Query; got != snap.SearchRPCQuery {
			t.Fatalf("scope %q differs from RPC %q", got, snap.SearchRPCQuery)
		}
	}
}

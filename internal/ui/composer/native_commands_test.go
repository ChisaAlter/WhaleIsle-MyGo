package composer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// Read the real authenticated graph parser over a local SSE stream. In
// particular CommandRevision's private dependency hashes are never fabricated.
func nativeCommandContributionFixture(t *testing.T, revisions map[string]string, omit ...string) dsh.NativeContributions {
	t.Helper()
	names := []string{"ui-conversation", "connection", "api-session-controller", "file-upload", "locale", "ui-layout", "ui-renderer", "ui-session", "ui-settings", "ui-workspace", "ui-commands", "api-remotes", "ui-input-trigger", "ui-model-selection", "ui-permission-presets", "ui-message-feedback", "ui-deepseek-model-label-1017", "unrelated"}
	idOf := func(name string) string {
		if strings.HasPrefix(name, "api-") {
			return "@deepseek-ai/dsh-" + name
		}
		return "@deepseek-ai/dsh-client-" + name
	}
	entries := []map[string]any{}
	ids := []string{}
	for _, name := range names {
		excluded := false
		for _, o := range omit {
			if name == o {
				excluded = true
			}
		}
		if excluded {
			continue
		}
		rev := "r1"
		if r, ok := revisions[name]; ok {
			rev = r
		}
		id := idOf(name)
		ids = append(ids, id)
		entries = append(entries, map[string]any{"id": id, "url": "/unused/" + name + ".js", "rev": rev, "inject": []string{}, "external": []string{}})
	}
	graph := map[string]any{"rev": "graph", "release": map[string]any{"version": "fixture", "commit": "fixed"}, "entries": entries, "batches": []map[string]any{{"phase": "application", "url": "/unused/batch.js", "rev": "batch", "entries": ids}}}
	raw, err := json.Marshal(map[string]any{"type": "graph", "graph": graph})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/plugins/events" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", raw)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	values := make(chan dsh.NativeContributions, 2)
	go func() {
		defer close(done)
		_ = (&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}).WatchNativeContributions(ctx, func(v dsh.NativeContributions) {
			select {
			case values <- v:
			case <-ctx.Done():
			}
		})
	}()
	var got dsh.NativeContributions
	select {
	case got = <-values:
	case <-time.After(3 * time.Second):
		cancel()
		srv.Close()
		t.Fatal("local contribution graph did not arrive")
	}
	cancel()
	<-done
	srv.Close()
	if !got.Known {
		t.Fatalf("invalid fixture graph: %+v", got)
	}
	return got
}
func nativeCommandSnapshot(t *testing.T) session.ViewState {
	t.Helper()
	return session.ViewState{ClientReady: true, SelectedID: "a", ComposerHostGeneration: 7, ComposerSessionGeneration: 3, PermissionCatalogRevision: 2, NativeContributions: nativeCommandContributionFixture(t, nil), SlashCommandsKnown: true, SelectedPermission: "Read Only", SlashCommands: []api.ListResultItem2{{Name: "permission"}, {Name: "feedback"}}}
}
func nativeChanged(t *testing.T, changed <-chan struct{}) {
	t.Helper()
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("command worker did not publish its receipt")
	}
}
func nativePopupFixture(t *testing.T, text string, rows []NativeCommandOption) (*DraftState, *Model, *ui.Tester, chan struct{}) {
	t.Helper()
	d, m, _ := suggestionFixture(text)
	m.Snapshot = nativeCommandSnapshot(t)
	m.WorkspaceRoot = "fixture"
	changed := make(chan struct{}, 8)
	m.OnNativeCommandChanged = func() { changed <- struct{}{} }
	m.OnLoadNativeCommandOptions = func(context.Context, NativeCommandOwner, string) ([]NativeCommandOption, error) { return rows, nil }
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { ui.Box(c).Grow(1); View(c, m) }) }, 700, 700)
	t.Cleanup(d.RetireNativeCommands)
	return d, &m, tt, changed
}
func openNativePopup(t *testing.T, d *DraftState, m *Model, tt *ui.Tester, changed <-chan struct{}) {
	t.Helper()
	if !d.HandleNativeCommand(m.Snapshot, m.OnNativeCommandAction) {
		t.Fatal("bare command not claimed")
	}
	tt.Frame()
	nativeChanged(t, changed)
	tt.Frame()
	if _, ok := tt.Find("Search command options"); !ok {
		t.Fatal("command popup search absent")
	}
}
func testModelRow(name string, active bool) NativeCommandOption {
	return NativeCommandOption{ID: name, Label: name, Model: &api.ModelCatalogDefault{Provider: "p", Model: name}, Active: active}
}

func TestNativeCommandPortsOnlyDecorateExistingHostRows(t *testing.T) {
	snap := nativeCommandSnapshot(t)
	m := Model{Snapshot: snap, OnNativeCommandAction: func(NativeCommandOwner, string) bool { return true }}
	rows, err := nativeSlashCatalog(m)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, r := range rows {
		seen[r.Line]++
	}
	for _, name := range []string{"file", "model", "permission", "feedback"} {
		if seen["/"+name] != 1 {
			t.Fatalf("/%s entry count %d", name, seen["/"+name])
		}
	}
	m.Snapshot.SlashCommands = nil
	rows, err = nativeSlashCatalog(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Line == "/permission" || r.Line == "/feedback" {
			t.Fatal("decoration manufactured a Host command")
		}
	}
	for _, name := range []string{"file", "model"} {
		m.Snapshot = snap
		m.Snapshot.SlashCommands = append(append([]api.ListResultItem2(nil), snap.SlashCommands...), api.ListResultItem2{Name: name})
		if _, err = nativeSlashCatalog(m); err == nil || !strings.Contains(err.Error(), "collides") {
			t.Fatalf("Host /%s collision hidden: %v", name, err)
		}
	}
}
func TestNativeCommandUnknownAndArgumentOwnership(t *testing.T) {
	snap := nativeCommandSnapshot(t)
	for _, name := range []string{"file", "model", "permission", "feedback"} {
		d := NewDraftState(" /" + name + " ")
		unknown := snap
		unknown.NativeContributions = dsh.NativeContributions{Error: "loading graph"}
		called := false
		if !d.HandleNativeCommand(unknown, func(NativeCommandOwner, string) bool { called = true; return true }) || called || d.Text != " /"+name+" " || d.Error != "loading graph" {
			t.Fatalf("unknown /%s escaped or lost draft", name)
		}
		d.Text = "/" + name + " argument"
		if d.HandleNativeCommand(snap, nil) {
			t.Fatalf("argument-bearing /%s was decorated", name)
		}
	}
	d := NewDraftState("ordinary input")
	if d.HandleNativeCommand(session.ViewState{}, nil) {
		t.Fatal("unknown graph blocked ordinary text")
	}
	for _, name := range []string{"permission", "feedback"} {
		d.Text = "/" + name
		pending := snap
		pending.SlashCommandsKnown = false
		if !d.HandleNativeCommand(pending, nil) || d.Text != "/"+name || d.Error == "" {
			t.Fatal("unloaded Host directory lost decoration admission")
		}
	}
}
func TestNativeActionMenuAndBareKeepAttachments(t *testing.T) {
	snap := nativeCommandSnapshot(t)
	for _, name := range []string{"file", "feedback"} {
		for _, menu := range []bool{false, true} {
			d, m, _ := suggestionFixture("/" + name)
			m.Snapshot = snap
			d.Attachments = []session.ComposerAttachment{{Path: "already-attached"}}
			calls := 0
			m.OnNativeCommandAction = func(owner NativeCommandOwner, action string) bool {
				calls++
				if owner.SessionID != "a" || action != name || d.Text != "" {
					t.Error("action ran before token consumption or with wrong owner")
				}
				return true
			}
			if menu {
				if !pickSlashCommand(m, suggestItem{insert: "/" + name + " ", command: true}, TokenRange{0, len(d.Text)}) {
					t.Fatal("menu action not claimed")
				}
			} else if !d.HandleNativeCommand(snap, m.OnNativeCommandAction) {
				t.Fatal("bare action not claimed")
			}
			if calls != 1 || len(d.Attachments) != 1 || d.Attachments[0].Path != "already-attached" {
				t.Fatal("action submitted or discarded attachment envelope")
			}
		}
	}
}
func TestNativeCommandChildRoomAndUnregisteredOwners(t *testing.T) {
	snap := nativeCommandSnapshot(t)
	child := snap
	child.Items = []session.Item{{SessionID: "a", Origin: "subagent"}}
	for _, name := range []string{"file", "model", "permission", "feedback"} {
		if nativeCommandAvailable(child, name) {
			t.Fatalf("addressed child exposed /%s", name)
		}
	}
	_, m, _ := suggestionFixture("/model")
	m.Snapshot = snap
	m.GroupMembers = []session.GroupMember{}
	if len(slashMenuItems(m, "")) != 0 || handleModelCommand(m) {
		t.Fatal("room exposed native command source")
	}
	missing := snap
	missing.NativeContributions = nativeCommandContributionFixture(t, nil, "ui-model-selection")
	d := NewDraftState("/model")
	if d.HandleNativeCommand(missing, nil) {
		t.Fatal("unmounted model port still intercepted Host command")
	}
}

func TestNativePopupLoadRetryAndSelectionReceipt(t *testing.T) {
	d, m, tt, changed := nativePopupFixture(t, "/model", []NativeCommandOption{testModelRow("Alpha", true), testModelRow("Beta", false)})
	attempts := 0
	load := m.OnLoadNativeCommandOptions
	m.OnLoadNativeCommandOptions = func(ctx context.Context, o NativeCommandOwner, n string) ([]NativeCommandOption, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("catalog read failed")
		}
		return load(ctx, o, n)
	}
	selected := make(chan api.ModelCatalogDefault, 2)
	release := make(chan error, 2)
	m.OnSelectNativeCommandModel = func(ctx context.Context, _ NativeCommandOwner, v api.ModelCatalogDefault) error {
		selected <- v
		select {
		case err := <-release:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	openNativePopup(t, d, m, tt, changed)
	if !tt.HasText("catalog read failed") || d.Text != "/model" {
		t.Fatal("load failure hid retry or consumed command")
	}
	if err := tt.Click(composerText("重试", "Retry")); err != nil {
		t.Fatal(err)
	}
	nativeChanged(t, changed)
	tt.Frame()
	if err := tt.Click("Search command options"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Beta")
	if err := tt.Click("Beta"); err != nil {
		t.Fatal(err)
	}
	select {
	case v := <-selected:
		if v.Model != "Beta" {
			t.Fatal("wrong opaque model route")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("selection not dispatched")
	}
	if d.Text != "/model" || !d.ModelCommand.Open {
		t.Fatal("popup consumed before Host receipt")
	}
	release <- errors.New("writer held")
	nativeChanged(t, changed)
	tt.Frame()
	if d.Text != "/model" || !tt.HasText("writer held") || !tt.HasText("Beta") || tt.HasText("Alpha") {
		t.Fatal("failed selection lost query, token or retryable popup")
	}
	if err := tt.Click("Beta"); err != nil {
		t.Fatal(err)
	}
	<-selected
	release <- nil
	nativeChanged(t, changed)
	tt.Frame()
	if d.Text != "" || d.ModelCommand.Open {
		t.Fatal("successful selection failed to consume and close")
	}
	if !tt.Focused("Composer draft") {
		t.Fatal("success did not return composer focus")
	}
}

func TestNativePopupMenuCASRejectsUndoABA(t *testing.T) {
	text := "prefix /model suffix"
	d, m, tt, changed := nativePopupFixture(t, text, []NativeCommandOption{testModelRow("Beta", false)})
	m.OnSelectNativeCommandModel = func(context.Context, NativeCommandOwner, api.ModelCatalogDefault) error { return nil }
	d.Client.SetSelectionUTF16(13, 13)
	if !pickSlashCommand(*m, suggestItem{insert: "/model ", command: true}, TokenRange{7, 13}) {
		t.Fatal("inline model menu not claimed")
	}
	tt.Frame()
	nativeChanged(t, changed)
	tt.Frame()
	d.Client.ReplaceText(&ui.TextInputRange{Start: 0, End: ui.UTF16Len(d.Text)}, "changed")
	if !d.Client.Undo() || d.Text != text {
		t.Fatal("fixture did not restore original text through Undo")
	}
	tt.Frame()
	if err := tt.Click("Beta"); err != nil {
		t.Fatal(err)
	}
	nativeChanged(t, changed)
	tt.Frame()
	if d.Text != text || d.ModelCommand.Open {
		t.Fatal("late menu settlement consumed an ABA draft")
	}
}
func TestNativePopupRetirementCancelsLoadAndRejectsOldOwner(t *testing.T) {
	for _, reason := range []string{"hidden", "session", "host", "contribution", "permission-catalog"} {
		t.Run(reason, func(t *testing.T) {
			text := "/model"
			if reason == "permission-catalog" {
				text = "/permission"
			}
			d, m, tt, changed := nativePopupFixture(t, text, nil)
			started := make(chan context.Context, 1)
			m.OnLoadNativeCommandOptions = func(ctx context.Context, _ NativeCommandOwner, _ string) ([]NativeCommandOption, error) {
				started <- ctx
				<-ctx.Done()
				return nil, ctx.Err()
			}
			if !d.HandleNativeCommand(m.Snapshot, nil) {
				t.Fatal("popup not claimed")
			}
			tt.Frame()
			ctx := <-started
			switch reason {
			case "hidden":
				d.RetireNativeCommands()
			case "session":
				m.Snapshot.ComposerSessionGeneration++
			case "host":
				m.Snapshot.ComposerHostGeneration++
			case "contribution":
				m.Snapshot.NativeContributions = nativeCommandContributionFixture(t, map[string]string{"ui-model-selection": "r2"})
			case "permission-catalog":
				m.Snapshot.PermissionCatalogRevision++
			}
			tt.Frame()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				t.Fatal("retired popup kept its worker alive")
			}
			nativeChanged(t, changed)
			tt.Frame()
			if d.Text != text || d.ModelCommand.Open || d.PermissionCommand.Open {
				t.Fatal("retired receipt changed draft or reopened popup")
			}
		})
	}
}
func TestNativePopupPermissionRiskCanCancelAndRequiresAcknowledgement(t *testing.T) {
	row := NativeCommandOption{ID: session.PermissionFullAccess, Label: "Full access", Confirmation: true}
	d, m, tt, changed := nativePopupFixture(t, "/permission", []NativeCommandOption{row})
	calls := make(chan bool, 1)
	m.OnSelectNativeCommandPermission = func(_ context.Context, _ NativeCommandOwner, value string, ack bool) error {
		if value != row.ID {
			t.Error("wrong permission value")
		}
		calls <- ack
		return nil
	}
	openNativePopup(t, d, m, tt, changed)
	label := nativeOptionLabel(row)
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	action := composerText("启用完全权限", "Enable Full access")
	if err := tt.Click(action); err != nil {
		t.Fatal(err)
	}
	select {
	case <-calls:
		t.Fatal("unacknowledged risk submitted")
	default:
	}
	if err := tt.Click(composerText("取消", "Cancel")); err != nil {
		t.Fatal(err)
	}
	if d.Text != "/permission" || !d.PermissionCommand.Open {
		t.Fatal("risk cancel consumed or closed command")
	}
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(composerText("我已了解这些风险，并愿意继续", "I understand these risks and want to continue")); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(action); err != nil {
		t.Fatal(err)
	}
	select {
	case ack := <-calls:
		if !ack {
			t.Fatal("checkbox acknowledgement was not forwarded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("confirmed permission was not submitted")
	}
	nativeChanged(t, changed)
	tt.Frame()
	if d.Text != "" || d.PermissionCommand.Open {
		t.Fatal("confirmed receipt did not close/consume")
	}
}
func TestNativePopupGeometryKeyboardScrollAndOutsideDismiss(t *testing.T) {
	rows := []NativeCommandOption{}
	for i := 0; i < 30; i++ {
		rows = append(rows, testModelRow(fmt.Sprintf("Model %02d", i), i == 0))
	}
	d, m, tt, changed := nativePopupFixture(t, "/model", rows)
	before, ok := tt.Find("Composer card")
	if !ok {
		t.Fatal("composer card absent")
	}
	openNativePopup(t, d, m, tt, changed)
	after, _ := tt.Find("Composer card")
	if before != after {
		t.Fatalf("popup moved composer: before=%+v after=%+v", before, after)
	}
	search, ok := tt.Find("Search command options")
	if !ok || search.X < after.X || search.X+search.W > after.X+after.W+.1 || search.Y+search.H > after.Y {
		t.Fatalf("popup not floating within card width: card=%+v search=%+v", after, search)
	}
	if err := tt.Click("Search command options"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 29; i++ {
		tt.Key(0, ui.KeyDown)
	}
	last, ok := tt.Find("Model 29")
	if !ok || last.Y < search.Y+search.H || last.Y+last.H > after.Y-3 {
		t.Fatalf("keyboard active row escaped viewport: %+v", last)
	}
	exportNativePopupGeometry(t, tt, map[string]ui.Rect{"cardBefore": before, "card": after, "search": search, "keyboardLastRow": last})
	tt.ClickAt(after.X+after.W/2, after.Y+12)
	if d.ModelCommand.Open || d.Text != "/model" {
		t.Fatal("outside pointer failed to dismiss without consuming")
	}
}

func TestNativePopupFilteringKeepsProviderOrderAndPermissionDetails(t *testing.T) {
	popup := &nativeCommandPopup{name: "model", query: "dc", rows: []NativeCommandOption{{ID: "p/first", Label: "deep-chat", Group: "first"}, {ID: "p/second", Label: "dc-prefix", Group: "first"}, {ID: "q/perfect", Label: "dc", Group: "second"}, {ID: "failure/q", Label: "provider failure", Failure: "failed"}}}
	rows := nativePopupRows(popup)
	if len(rows) != 3 || rows[0].ID != "p/second" || rows[1].ID != "p/first" || rows[2].ID != "q/perfect" {
		t.Fatalf("fuzzy ranking crossed provider boundary: %+v", rows)
	}
	popup = &nativeCommandPopup{name: "permission", query: "network", rows: []NativeCommandOption{{ID: "custom", Label: "Custom", Detail: "Allows NETWORK access"}, {ID: "other", Label: "Other"}}}
	rows = nativePopupRows(popup)
	if len(rows) != 1 || rows[0].ID != "custom" {
		t.Fatal("permission search ignored details")
	}
}

func TestNativePopupSendOpensOneShell(t *testing.T) {
	d, m, tt, changed := nativePopupFixture(t, "/model", []NativeCommandOption{testModelRow("Alpha", true)})
	// Use the actual visible Send entry, rather than opening internal state.
	if err := tt.Click(i18n.T(i18n.KeyComposerSend)); err != nil {
		t.Fatal(err)
	}
	nativeChanged(t, changed)
	tt.Frame()
	if _, ok := tt.Find("Search command options"); !ok || d.Text != "/model" {
		t.Fatal("Send did not open retained command popup")
	}
	kind, q, _, quoted, open := activeSuggestion(*m, d.Client)
	if open && len(buildSuggestItemsWithQuote(*m, kind, q, quoted)) > 0 {
		t.Fatal("Send left the slash menu active beside its popup")
	}
}

func TestNativePopupPointerHoverControlsKeyboardSelection(t *testing.T) {
	d, m, tt, changed := nativePopupFixture(t, "/model", []NativeCommandOption{testModelRow("Alpha", true), testModelRow("Beta", false)})
	picked := make(chan string, 1)
	m.OnSelectNativeCommandModel = func(_ context.Context, _ NativeCommandOwner, selection api.ModelCatalogDefault) error {
		picked <- selection.Model
		return nil
	}
	openNativePopup(t, d, m, tt, changed)
	r, ok := tt.Find("Beta")
	if !ok {
		t.Fatal("second model row absent")
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Key(0, ui.KeyEnter)
	select {
	case model := <-picked:
		if model != "Beta" {
			t.Fatalf("hover/Enter selected %q", model)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("hovered model did not receive Enter")
	}
	nativeChanged(t, changed)
	tt.Frame()
	if d.Text != "" || d.ModelCommand.Open {
		t.Fatal("hovered choice receipt did not consume and close")
	}
}

func exportNativePopupGeometry(t *testing.T, tt *ui.Tester, bounds map[string]ui.Rect) {
	t.Helper()
	dir := os.Getenv("SHELL_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "native-command-popup-keyboard.png"))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, tt.Image())
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	raw, err := json.MarshalIndent(map[string]any{"renderer": "MyGo software; not native screenshot or visual parity acceptance", "bounds": bounds}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "native-command-popup-keyboard.geometry.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNativePopupKeyboardDismissReturnsComposerFocus(t *testing.T) {
	for _, key := range []struct {
		name string
		mods ui.Modifiers
		key  ui.Key
	}{{"Escape", 0, ui.KeyEscape}, {"ShiftTab", ui.Shift, ui.KeyTab}} {
		t.Run(key.name, func(t *testing.T) {
			d, m, tt, changed := nativePopupFixture(t, "/model", []NativeCommandOption{testModelRow("Alpha", true)})
			openNativePopup(t, d, m, tt, changed)
			if err := tt.Click("Search command options"); err != nil {
				t.Fatal(err)
			}
			tt.Key(key.mods, key.key)
			if d.ModelCommand.Open || d.Text != "/model" {
				t.Fatal("keyboard dismissal selected or consumed the command")
			}
			if !tt.Focused("Composer draft") {
				t.Fatal("keyboard dismissal did not restore composer focus")
			}
		})
	}
}

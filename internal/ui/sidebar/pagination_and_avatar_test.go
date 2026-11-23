package sidebar

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestFlatGroupingPaginatesAtFiftyAndExpandsCleanly(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	const total = 120
	items := make([]session.Item, total)
	ids := make([]string, total)
	for i := range items {
		id := fmt.Sprintf("s-%03d", i+1)
		ids[i] = id
		items[i] = session.Item{SessionID: id, Title: fmt.Sprintf("Flat Session %03d", i+1)}
	}
	m := Model{
		Snapshot: session.ViewState{
			Items:      items,
			Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", Title: "WS", Path: "/ws", SessionIDs: ids}},
			WorkspaceLayout: session.WorkspaceLayoutState{
				GroupBy:        "flat",
				HostGeneration: 1,
			},
		},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			workspaceRowsView(c, m, items, ui.RGB(20, 20, 20), ui.RGB(110, 110, 110), ui.Transparent)
		})
	}, 320, 4800)

	if !tt.HasText("Flat Session 050") || tt.HasText("Flat Session 051") {
		t.Fatalf("flat mode did not paginate at %d rows", flatSessionPageBase)
	}
	moreLabel := "Show 70 more sessions"
	if !tt.HasText(moreLabel) {
		t.Fatalf("missing flat pagination control %q", moreLabel)
	}
	if err := tt.Click(moreLabel); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Flat Session 100") || tt.HasText("Flat Session 101") {
		t.Fatal("clicking show more did not expand flat page by 50 rows")
	}
	if err := tt.Click("Show 20 more sessions"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Flat Session 120") || !tt.HasText("Show less") {
		t.Fatal("second expansion did not reveal remainder and show less toggle")
	}
	if err := tt.Click("Show less"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Flat Session 051") {
		t.Fatal("Show less did not collapse flat mode back to page base")
	}
}

func TestArchivedSectionPaginatesAtThirtyAndResetsOnCollapse(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	const total = 75
	items := make([]session.Item, total)
	for i := range items {
		items[i] = session.Item{SessionID: fmt.Sprintf("a-%03d", i+1), Title: fmt.Sprintf("Archived Item %03d", i+1)}
	}
	expanded := true
	m := Model{archiveExpanded: &expanded}

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			archivedSection(c, m, items, ui.RGB(20, 20, 20), ui.RGB(110, 110, 110))
		})
	}, 320, 3600)

	if !tt.HasText("Archived Item 030") || tt.HasText("Archived Item 031") {
		t.Fatalf("archived section did not paginate at %d rows", archiveSessionPageBase)
	}
	if err := tt.Click("Show 45 more sessions"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Archived Item 060") || tt.HasText("Archived Item 061") {
		t.Fatal("archived expansion did not reveal next page")
	}

	// Collapsing and reopening resets limit back to archiveSessionPageBase.
	expanded = false
	tt.Frame()
	expanded = true
	tt.Frame()
	if tt.HasText("Archived Item 031") {
		t.Fatal("collapsing and reopening archive section did not reset pagination limit")
	}
}

func TestAccountAvatarBitmapLoadsAsyncAndSharesTheSettingsDecode(t *testing.T) {
	// A picture of its own: the process media cache is shared by every test.
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	onePixelPNG := "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes())
	var firstBmp, bmp *ui.Bitmap
	calls := 0
	tt := ui.NewTester(func(c *ui.Context) {
		got := accountAvatarBitmap(c, onePixelPNG)
		calls++
		if calls == 1 {
			firstBmp = got
		}
		bmp = got
	}, 100, 50)
	if firstBmp != nil {
		t.Fatal("first call decoded synchronously instead of dispatching async loader")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && bmp == nil {
		tt.Frame()
		time.Sleep(5 * time.Millisecond)
	}
	if bmp == nil {
		t.Fatal("async accountAvatarBitmap never populated bitmap")
	}
	if media.Default().Avatar(onePixelPNG).Bitmap() != bmp {
		t.Fatal("settings and sidebar avatars do not share one bitmap")
	}
	// Local filesystem paths are never avatar sources.
	if accountAvatarBitmap(nil, filepath.Join(t.TempDir(), "avatar.png")) != nil || media.Default().Avatar(`C:\avatar.png`) != nil {
		t.Fatal("a local path was accepted as an avatar")
	}
}

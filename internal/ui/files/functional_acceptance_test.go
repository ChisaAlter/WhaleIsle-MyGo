package files

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/bmp"
	gitdesktop "whaleisle/internal/desktop/git"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/pdfpreview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/filelink"
	"whaleisle/internal/ui/i18n"
)

// FileTree has no create/rename/hidden-toggle command in the original. This
// acceptance checks its actual menu and safe reference/callback operations;
// clipboard entries are inspected, never selected against the OS clipboard.
func TestFunctionalFileTreeSourceContract(t *testing.T) {
	root, out := os.Getenv("WHALEISLE_TREE_CONTRACT_ROOT"), os.Getenv("WHALEISLE_TREE_CONTRACT_OUTPUT")
	if root == "" || out == "" {
		t.Skip("explicit isolated tree acceptance paths required")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(out) {
		t.Fatal("absolute paths required")
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal("fresh owned root required:", err)
	}
	folder := filepath.Join(root, "owned folder")
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(folder, "note.txt")
	const content = "owned tree contract; never user data"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	report := map[string]any{"matrix": []string{"WORK-03"}, "root": root, "scope": "real owned FS + production row/context menu with headless pointer input; safe callbacks only", "modelRequests": 0}
	defer func() {
		report["pass"] = !t.Failed()
		raw, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(out, append(raw, '\n'), 0600)
		}
		if err != nil {
			t.Error(err)
		}
	}()
	expanded, selected := map[string]bool{}, "other.txt"
	var reference filelink.Reference
	var revealed, opened string
	m := Model{BindingID: "owned-host", SessionID: "owned-session", WorkspaceRoot: root, Expanded: &expanded, SelectedFile: &selected, CanAddToChat: true,
		OnReferenceFile: func(value filelink.Reference) error { reference = value; return nil },
		OnReveal:        func(full string) { revealed = full },
		OnOpenDoc: func(full string, _ workspacefs.DocKind, web bool) {
			if web {
				t.Error("system command selected webview")
			}
			opened = full
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			rowDir(c, m, "owned folder", "owned folder", 0, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(65, 118, 230))
			rowFile(c, m, "owned folder/note.txt", "note.txt", 1, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80), ui.RGB(65, 118, 230))
		})
	}, 500, 250)
	if err := tt.Click("Folder owned folder"); err != nil {
		t.Fatal(err)
	}
	if !expanded["owned folder"] {
		t.Fatal("directory did not expand")
	}
	tt.Key(0, ui.KeyEnter)
	if expanded["owned folder"] {
		t.Fatal("focused directory did not collapse")
	}
	if err := tt.RightClick("Folder owned folder"); err != nil {
		t.Fatal(err)
	}
	folderMenu := tt.Menu()
	report["directoryMenu"] = folderMenu
	if len(folderMenu) != 4 {
		t.Fatalf("directory menu drifted from original path-only actions: %v", folderMenu)
	}
	if err := tt.ChooseMenuItem("Show in folder"); err != nil {
		t.Fatal(err)
	}
	if revealed != folder || selected != "other.txt" {
		t.Fatal("directory reveal retargeted selection or path")
	}
	if err := tt.Click("Reference file owned folder/note.txt"); err != nil {
		t.Fatal(err)
	}
	if reference.Path != "owned folder/note.txt" || reference.BindingID != m.BindingID || reference.SessionID != m.SessionID || selected != "other.txt" {
		t.Fatal("reference lost owner or selected the file")
	}
	if err := tt.RightClick("File owned folder/note.txt"); err != nil {
		t.Fatal(err)
	}
	fileMenu := tt.Menu()
	report["fileMenu"] = fileMenu
	for _, item := range append(append([]string{}, folderMenu...), fileMenu...) {
		lower := strings.ToLower(item)
		if strings.Contains(lower, "rename") || strings.Contains(lower, "new folder") || strings.Contains(lower, "create") || strings.Contains(lower, "hidden") {
			t.Fatalf("invented file-tree command %q", item)
		}
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeyFilesOpenExternal)); err != nil {
		t.Fatal(err)
	}
	if opened != path || selected != "other.txt" {
		t.Fatal("system-open callback did not retain clicked real path")
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != content {
		t.Fatal("context operation mutated actual source file")
	}
	report["checks"] = []string{"directory expand/collapse", "secondary-click opens shipped directory/file menu", "folder reveal callback receives own absolute path", "file reference retains Host/session/workspace owner without selecting", "file system-open callback receives clicked own absolute path", "source file bytes unchanged"}
	report["notPerformed"] = []string{"actual OS clipboard writes", "Explorer/editor/system-app process launch", "native-window pointer interaction or drag/drop", "ACL mutations"}
}

// Opt-in acceptance drives the shipped controllers against a fresh, retained
// filesystem and real Git/loopback services. It does not launch a Host, contact
// a model, use user data, or claim that headless input is native-window QA.
func awaitAcceptanceFileRead(t *testing.T, m Model, done <-chan struct{}) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for selectedTab(m) != nil && selectedTab(m).readLoading {
		select {
		case <-done:
			syncTextRead(m)
		case <-deadline.C:
			t.Fatal("owned file read did not complete")
		}
	}
}

func TestFunctionalFilesGitAcceptance(t *testing.T) {
	base, reportPath := os.Getenv("WHALEISLE_FILES_ACCEPTANCE_ROOT"), os.Getenv("WHALEISLE_FILES_ACCEPTANCE_REPORT")
	if base == "" || reportPath == "" {
		t.Skip("explicit isolated acceptance paths required")
	}
	if !filepath.IsAbs(base) || !filepath.IsAbs(reportPath) {
		t.Fatal("absolute acceptance paths required")
	}
	if err := os.Mkdir(base, 0700); err != nil {
		t.Fatal("acceptance root must be new:", err)
	}
	previous := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(previous)
	type result struct{ ID, Case, Status, Method, Detail string }
	report := struct {
		Started, SourceRevision, Root string
		ModelRequests, RemotePushes   int
		Results                       []result
	}{Started: time.Now().UTC().Format(time.RFC3339), Root: base}
	if revision, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		report.SourceRevision = strings.TrimSpace(string(revision))
	}
	defer func() {
		data, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(reportPath, append(data, '\n'), 0600)
		}
		if err != nil {
			t.Error(err)
		}
	}()
	run := func(id, name, method, detail string, fn func(*testing.T)) {
		ok := t.Run(name, fn)
		status := "PASS"
		if !ok {
			status = "FAIL"
		}
		report.Results = append(report.Results, result{id, name, status, method, detail})
	}
	write := func(t *testing.T, root, path, text string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	read := func(t *testing.T, path string) string {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	newRoot := func(t *testing.T, name string) string {
		t.Helper()
		root := filepath.Join(base, name)
		if err := os.Mkdir(root, 0700); err != nil {
			t.Fatal(err)
		}
		return root
	}
	newModel := func(root string) (Model, chan struct{}) {
		selected, preview, draft, problem, status := "", "", "", "", ""
		editing := false
		done := make(chan struct{}, 64)
		tabs := &Tabs{}
		m := Model{BindingID: "isolated-acceptance/session", WorkspaceRoot: root, Tabs: tabs, SelectedFile: &selected, Preview: &preview, EditDraft: &draft, PreviewErr: &problem, EditStatus: &status, Editing: &editing, notify: func() {
			select {
			case done <- struct{}{}:
			default:
			}
		}, OnSaveEdit: func(path, text string) error {
			full, err := workspacefs.JoinRoot(root, path)
			if err != nil {
				return err
			}
			return workspacefs.WriteFileText(full, text)
		}}
		tabs.Bind(m.BindingID, root)
		prepareTabs(m)
		return m, done
	}
	run("WORK-03", "real-tree-create-rename-refresh", "real filesystem + DirectoryTree + headless native row input", "Fresh root; async listings, explicit refresh, collapsed child, path boundary; no user files", func(t *testing.T) {
		root := newRoot(t, "tree")
		write(t, root, "nested/first.txt", "first\n")
		tree := &workspacefs.DirectoryTree{}
		tree.Ensure("owned", root)
		defer tree.Cancel()
		wait := func(rel string, want string) {
			t.Helper()
			deadline := time.Now().Add(3 * time.Second)
			for {
				snapshot := tree.Get(rel, nil, func() {})
				if snapshot.Error != "" {
					t.Fatal(snapshot.Error)
				}
				if !snapshot.Busy {
					for _, entry := range snapshot.Entries {
						if entry.Name == want {
							return
						}
					}
				}
				if time.Now().After(deadline) {
					t.Fatalf("%s missing %s: %+v", rel, want, snapshot)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
		wait("", "nested")
		wait("nested", "first.txt")
		write(t, root, "nested/new.txt", "new\n")
		tree.Refresh()
		wait("nested", "new.txt")
		if err := os.Rename(filepath.Join(root, "nested/new.txt"), filepath.Join(root, "nested/renamed.txt")); err != nil {
			t.Fatal(err)
		}
		tree.Refresh()
		wait("nested", "renamed.txt")
		expanded := map[string]bool{}
		selected := ""
		m := Model{WorkspaceRoot: root, Expanded: &expanded, SelectedFile: &selected}
		tt := ui.NewTester(func(c *ui.Context) {
			rowDir(c, m, "nested", "nested", 0, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), ui.RGB(0, 0, 0))
		}, 400, 100)
		if err := tt.Click("Folder nested"); err != nil {
			t.Fatal(err)
		}
		if !expanded["nested"] {
			t.Fatal("row did not expand")
		}
		tt.Key(0, ui.KeyEnter)
		if expanded["nested"] {
			t.Fatal("row did not collapse")
		}
		if _, err := workspacefs.JoinRoot(root, "../outside"); err == nil {
			t.Fatal("escaped root")
		}
		if _, err := workspacefs.ListWorkspaceDir(context.Background(), root, "missing"); err == nil {
			t.Fatal("missing directory reported empty success")
		}
	})
	run("WORK-06", "xls-real-biff-workbook", "generated valid OLE2/BIFF8 file + production XLS reader", "Actual stored numeric cell and worksheet; no external converter or macro execution", func(t *testing.T) {
		root := newRoot(t, "xls")
		write(t, root, "book.xls", string(acceptanceXLS()))
		page, err := workspacefs.ReadWorkbookPage(context.Background(), filepath.Join(root, "book.xls"), workspacefs.XLSXPageRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Sheets) != 1 || len(page.Cells) != 1 || page.Cells[0].Value != "42" {
			t.Fatalf("actual BIFF worksheet/cell mismatch: %+v", page)
		}
	})
	run("WORK-04", "real-save-autosave-conflict-dirty-close", "real files + shipped Tabs/SaveQueue controller + headless close prompt", "Exact persisted bytes; external changes retained, failed save retains draft; keep/discard; workspace and recovery identity", func(t *testing.T) {
		root := newRoot(t, "editor")
		write(t, root, "a.txt", "original\n")
		write(t, root, "b.txt", "other\n")
		m, done := newModel(root)
		defer m.Tabs.Dispose()
		selectFile(m, "a.txt")
		awaitAcceptanceFileRead(t, m, done)
		*m.EditDraft = "saved 中文\n"
		*m.Editing = true
		if err := saveDraft(m); err != nil {
			t.Fatal(err)
		}
		if read(t, filepath.Join(root, "a.txt")) != "saved 中文\n" {
			t.Fatal("explicit save byte mismatch")
		}
		observeAutosave(m) // A displayed editor observes its baseline before typing.
		*m.EditDraft = "automatic\n"
		observeAutosave(m)
		deadline := time.Now().Add(4 * time.Second)
		for read(t, filepath.Join(root, "a.txt")) != "automatic\n" {
			if time.Now().After(deadline) {
				t.Fatal("autosave did not persist")
			}
			select {
			case <-done:
				syncSaveResults(m)
			case <-time.After(10 * time.Millisecond):
			}
		}
		syncSaveResults(m)
		*m.EditDraft = "pending draft 中文\n"
		write(t, root, "a.txt", "external writer\n")
		if err := saveDraft(m); err == nil {
			t.Fatal("external edit overwritten")
		}
		if read(t, filepath.Join(root, "a.txt")) != "external writer\n" || *m.EditDraft != "pending draft 中文\n" {
			t.Fatal("conflict changed disk/draft")
		}
		selectFile(m, "b.txt")
		awaitAcceptanceFileRead(t, m, done)
		selectFile(m, "a.txt")
		awaitAcceptanceFileRead(t, m, done)
		if *m.EditDraft != "pending draft 中文\n" {
			t.Fatal("switch dropped dirty draft")
		}
		reloadSelectedFile(m)
		awaitAcceptanceFileRead(t, m, done)
		if *m.EditDraft != "pending draft 中文\n" {
			t.Fatal("reload discarded draft")
		}
		if closeTab(m, "a.txt", false) {
			t.Fatal("dirty close bypassed confirmation")
		}
		tt := ui.NewTester(func(c *ui.Context) { paintTabs(c, m, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0)) }, 650, 400)
		if err := tt.Click("Keep draft"); err != nil {
			t.Fatal(err)
		}
		if tabsFor(m).PendingClose != "" || findTab(tabsFor(m), "a.txt") == nil {
			t.Fatal("keep draft closed tab")
		}
		data, err := m.Tabs.Export()
		if err != nil {
			t.Fatal(err)
		}
		var restored Tabs
		if err = restored.Restore(data); err != nil {
			t.Fatal(err)
		}
		if findTab(restored.Workspaces[filepath.Clean(root)], "a.txt").Draft != "pending draft 中文\n" {
			t.Fatal("recovery dropped draft")
		}
		restored.Dispose()
		if !closeTab(m, "a.txt", true) || *m.SelectedFile != "b.txt" {
			t.Fatal("discard did not close only requested tab")
		}
		if read(t, filepath.Join(root, "a.txt")) != "external writer\n" {
			t.Fatal("discard wrote file")
		}
	})
	run("WORK-05", "text-code-markdown-paging-and-errors", "real UTF-8 files + asynchronous product readers + line navigation", "No global decoder installation; UTF-8/code/Markdown load, deferred line, missing retry, large-file page and edit guard", func(t *testing.T) {
		root := newRoot(t, "text")
		source := "# 中文标题\n\n```go\npackage main\n```\nlast\n"
		write(t, root, "note.md", source)
		write(t, root, "code.go", "package main\n// 中文\nfunc main() {}\n")
		m, done := newModel(root)
		defer m.Tabs.Dispose()
		selectFile(m, "note.md")
		RequestLine(m, 6)
		awaitAcceptanceFileRead(t, m, done)
		if *m.Preview != source || selectedTab(m).revealLine != 6 || selectedTab(m).RenderedMarkdown {
			t.Fatalf("read/line request mismatch: %q line=%d", *m.Preview, selectedTab(m).revealLine)
		}
		selectFile(m, "code.go")
		awaitAcceptanceFileRead(t, m, done)
		if !strings.Contains(*m.Preview, "// 中文") {
			t.Fatal("code read failed")
		}
		selectFile(m, "later.txt")
		select {
		case <-done:
			syncTextRead(m)
		case <-time.After(3 * time.Second):
			t.Fatal("missing read timeout")
		}
		if *m.PreviewErr == "" {
			t.Fatal("missing read lost error")
		}
		write(t, root, "later.txt", "created after error\n")
		reloadSelectedFile(m)
		awaitAcceptanceFileRead(t, m, done)
		if *m.Preview != "created after error\n" || *m.PreviewErr != "" {
			t.Fatal("retry did not load created file")
		}
		large := strings.Repeat("中文 line\n", 140000)
		write(t, root, "large.txt", large)
		selectFile(m, "large.txt")
		awaitAcceptanceFileRead(t, m, done)
		tab := selectedTab(m)
		if !tab.Truncated || !tab.textPaged {
			t.Fatal("large resource is editable or unpaged")
		}
		before := len(*m.Preview)
		beginNextTextPage(m)
		deadline := time.Now().Add(3 * time.Second)
		for selectedTab(m).readLoading {
			select {
			case <-done:
				syncTextRead(m)
			case <-time.After(10 * time.Millisecond):
			}
			if time.Now().After(deadline) {
				t.Fatal("next page read timeout")
			}
		}
		if len(*m.Preview) <= before {
			t.Fatal("next page did not append")
		}
		if err := saveDraft(m); err == nil {
			t.Fatal("paged file saved over full source")
		}
		if read(t, filepath.Join(root, "large.txt")) != large {
			t.Fatal("paging changed disk bytes")
		}
	})
	run("WORK-06", "native-image-decoders", "actual generated/copied format files + production native decoders", "PNG JPEG BMP GIF ICO SVG AVIF WebP decode from actual files; corrupt PNG rejected", func(t *testing.T) {
		root := newRoot(t, "images")
		img := image.NewRGBA(image.Rect(0, 0, 3, 2))
		img.Set(0, 0, color.RGBA{200, 20, 30, 255})
		for _, format := range []string{"png", "jpg", "bmp", "gif"} {
			var data bytes.Buffer
			var err error
			switch format {
			case "png":
				err = png.Encode(&data, img)
			case "jpg":
				err = jpeg.Encode(&data, img, nil)
			case "bmp":
				err = bmp.Encode(&data, img)
			case "gif":
				err = gif.Encode(&data, img, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			write(t, root, "image."+format, data.String())
		}
		write(t, root, "image.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="3" height="2"><rect width="3" height="2" fill="red"/></svg>`)
		pngData, err := os.ReadFile(filepath.Join(root, "image.png"))
		if err != nil {
			t.Fatal(err)
		}
		icon := make([]byte, 22)
		binary.LittleEndian.PutUint16(icon[2:], 1)
		binary.LittleEndian.PutUint16(icon[4:], 1)
		icon[6], icon[7] = 3, 2
		binary.LittleEndian.PutUint16(icon[10:], 1)
		binary.LittleEndian.PutUint16(icon[12:], 32)
		binary.LittleEndian.PutUint32(icon[14:], uint32(len(pngData)))
		binary.LittleEndian.PutUint32(icon[18:], 22)
		write(t, root, "image.ico", string(append(icon, pngData...)))
		copyFixture := func(from, to string) {
			data, err := os.ReadFile(from)
			if err != nil {
				t.Fatal(err)
			}
			write(t, root, to, string(data))
		}
		copyFixture("../../desktop/imagepreview/testdata/test10.avif", "image.avif")
		copyFixture(filepath.Join(os.Getenv("GOMODCACHE"), "golang.org/x/image@v0.46.0/testdata/blue-purple-pink-large.lossless.webp"), "image.webp")
		for _, format := range []string{"png", "jpg", "bmp", "gif", "ico", "svg", "avif", "webp"} {
			p := loadImagePreview(root, "image."+format)
			if p.err != nil || p.bitmap == nil {
				t.Fatalf("%s decode: %+v", format, p.err)
			}
			w, h := p.bitmap.Size()
			if w < 1 || h < 1 {
				t.Fatal("empty decoded image", format)
			}
		}
		write(t, root, "broken.png", "not an image")
		if p := loadImagePreview(root, "broken.png"); p.err == nil {
			t.Fatal("corrupt image accepted")
		}
	})
	run("WORK-06", "xlsx-csv-tsv-real-pages", "actual workbook archives + native read-only page controller", "Stored values and formulas; sheets and version binding; damaged archives; no Office installed or invoked", func(t *testing.T) {
		root := newRoot(t, "workbooks")
		var archive bytes.Buffer
		zw := zip.NewWriter(&archive)
		for name, body := range map[string]string{"[Content_Types].xml": `<Types/>`, "xl/workbook.xml": `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="中文Sheet" sheetId="1" r:id="rId1"/></sheets></workbook>`, "xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`, "xl/worksheets/sheet1.xml": `<worksheet><dimension ref="A1:B2"/><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>中文</t></is></c><c r="B1"><v>42</v></c></row><row r="2"><c r="A2"><f>SUM(B1)</f><v>42</v></c></row></sheetData></worksheet>`} {
			w, err := zw.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = w.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		write(t, root, "book.xlsx", archive.String())
		page, err := workspacefs.ReadWorkbookPage(context.Background(), filepath.Join(root, "book.xlsx"), workspacefs.XLSXPageRequest{Rows: 2, Columns: 2})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Cells) < 3 || !strings.Contains(page.Text, "中文") {
			t.Fatalf("workbook missing actual stored cells: %+v", page)
		}
		write(t, root, "book.csv", "name,value\n中文,42\n")
		write(t, root, "book.tsv", "name\tvalue\n中文\t42\n")
		for _, name := range []string{"book.csv", "book.tsv"} {
			p, err := workspacefs.ReadWorkbookPage(context.Background(), filepath.Join(root, name), workspacefs.XLSXPageRequest{})
			if err != nil || !strings.Contains(p.Text, "中文") {
				t.Fatalf("%s %v", name, err)
			}
		}
		write(t, root, "book.xlsx", archive.String()+"changed")
		if _, err := workspacefs.ReadWorkbookPage(context.Background(), filepath.Join(root, "book.xlsx"), workspacefs.XLSXPageRequest{Version: page.Version}); err == nil {
			t.Fatal("external workbook change accepted under old version")
		}
		write(t, root, "broken.xlsx", "invalid zip")
		if _, err := workspacefs.ReadWorkbookPage(context.Background(), filepath.Join(root, "broken.xlsx"), workspacefs.XLSXPageRequest{}); err == nil {
			t.Fatal("broken archive accepted")
		}
	})
	run("WORK-06", "html-pdf-local-services", "actual filesystem + product loopback HTTP services", "Bytes and inert HTML/CSP; PDF reader assets; close releases listeners. Browser rendering/Office conversion NOT established by this check", func(t *testing.T) {
		root := newRoot(t, "documents")
		write(t, root, "page.html", `<!doctype html><html><body><h1>中文 preview</h1><script>fetch('https://invalid.example')</script></body></html>`)
		write(t, root, "file.pdf", "%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\n%%EOF\n")
		get := func(url string) ([]byte, http.Header) {
			t.Helper()
			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get(url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil || resp.StatusCode != 200 {
				t.Fatalf("service read status=%d err=%v", resp.StatusCode, err)
			}
			return body, resp.Header
		}
		html, err := htmldocument.OpenWorkspace(root, "page.html", func() bool { return true })
		if err != nil {
			t.Fatal(err)
		}
		defer html.Close()
		body, headers := get(html.URL())
		if !bytes.Contains(body, []byte("中文 preview")) || !strings.Contains(headers.Get("Content-Security-Policy"), "script-src 'none'") {
			t.Fatal("HTML preview bytes/CSP mismatch")
		}
		htmlURL := html.URL()
		html.Close()
		client := &http.Client{Timeout: time.Second}
		if resp, err := client.Get(htmlURL); err == nil {
			resp.Body.Close()
			t.Fatal("HTML listener survived Close")
		}
		pdf, err := pdfpreview.OpenWorkspace(root, "file.pdf", func() bool { return true })
		if err != nil {
			t.Fatal(err)
		}
		defer pdf.Close()
		body, _ = get(pdf.URL())
		if string(body) != read(t, filepath.Join(root, "file.pdf")) {
			t.Fatal("PDF snapshot byte mismatch")
		}
		reader, _ := get(pdf.ViewerURL())
		if len(reader) < 100 {
			t.Fatal("PDF viewer assets unavailable")
		}
		pdfURL := pdf.URL()
		pdf.Close()
		if resp, err := client.Get(pdfURL); err == nil {
			resp.Body.Close()
			t.Fatal("PDF listener survived Close")
		}
	})
	run("WORK-09/10", "real-git-branches-diff-reviewed-commit", "actual isolated Git CLI through product APIs; no remotes", "init/main, staged scope, dirty collision, invalid branch, cancellation, stale review refusal, no-repo, publish validation only", func(t *testing.T) {
		root := newRoot(t, "git")
		if gitdesktop.IsRepository(root) {
			t.Fatal("fresh directory is a repo")
		}
		if _, err := gitdesktop.WorkspaceSnapshot(root); err == nil {
			t.Fatal("nonrepo snapshot accepted")
		}
		if _, err := gitdesktop.Initialize(root); err != nil {
			t.Fatal(err)
		}
		gitRun := func(args ...string) string {
			t.Helper()
			cmd := exec.Command("git", args...)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("git %v: %v %s", args, err, out)
			}
			return string(out)
		}
		gitRun("config", "user.name", "Isolated Acceptance")
		gitRun("config", "user.email", "isolated@local.invalid")
		gitRun("config", "commit.gpgsign", "false")
		write(t, root, "a.txt", "base\n")
		write(t, root, "untouched.txt", "not included\n")
		if diff, err := gitdesktop.DiffScope(root, "a.txt", false); err != nil || !strings.Contains(diff, "+base") {
			t.Fatalf("untracked diff %q %v", diff, err)
		}
		if _, err := gitdesktop.StagePaths(root, "a.txt"); err != nil {
			t.Fatal(err)
		}
		_, token, err := gitdesktop.IndexReview(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = gitdesktop.CommitReviewed(root, "owned initial", token); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(gitRun("show", "--format=", "--name-only", "HEAD"), "untouched.txt") {
			t.Fatal("commit included unstaged file")
		}
		if _, err = gitdesktop.CreateBranch(root, "feature/owned"); err != nil {
			t.Fatal(err)
		}
		write(t, root, "a.txt", "feature\n")
		if _, err = gitdesktop.StagePaths(root, "a.txt"); err != nil {
			t.Fatal(err)
		}
		if _, err = gitdesktop.CommitStaged(root, "feature change"); err != nil {
			t.Fatal(err)
		}
		if _, err = gitdesktop.SwitchBranch(root, "main"); err != nil {
			t.Fatal(err)
		}
		write(t, root, "a.txt", "dirty main\n")
		if _, err = gitdesktop.SwitchBranch(root, "feature/owned"); err == nil {
			t.Fatal("branch switch overwrote dirty collision")
		}
		if read(t, filepath.Join(root, "a.txt")) != "dirty main\n" {
			t.Fatal("failed switch changed file")
		}
		if _, err = gitdesktop.CreateBranch(root, "../invalid"); err == nil {
			t.Fatal("invalid branch accepted")
		}
		if _, err = gitdesktop.StagePaths(root, "a.txt"); err != nil {
			t.Fatal(err)
		}
		_, token, err = gitdesktop.IndexReview(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = gitdesktop.StagePaths(root, "untouched.txt"); err != nil {
			t.Fatal(err)
		}
		if _, err = gitdesktop.CommitReviewed(root, "stale must fail", token); err == nil {
			t.Fatal("stale index review accepted")
		}
		if _, err = gitdesktop.UnstagePaths(root, "untouched.txt"); err != nil {
			t.Fatal(err)
		}
		if _, err = gitdesktop.CommitStaged(root, "dirty main change"); err != nil {
			t.Fatal(err)
		}
		branches, err := gitdesktop.Branches(root)
		if err != nil || len(branches) != 2 {
			t.Fatalf("branches %+v %v", branches, err)
		}
		if diff, err := gitdesktop.CompareBranch(context.Background(), root, "a.txt", "feature/owned"); err != nil || !strings.Contains(diff, "dirty main") {
			t.Fatalf("branch diff %q %v", diff, err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := gitdesktop.CompareBranch(ctx, root, "a.txt", "main"); err == nil {
			t.Fatal("cancelled comparison succeeded")
		}
		if _, err = gitdesktop.PreparePublish(root, gitdesktop.PublishInput{RemoteURL: "file:///not-permitted"}); err == nil {
			t.Fatal("invalid publish URL accepted")
		}
		if remotes := gitRun("remote"); strings.TrimSpace(remotes) != "" {
			t.Fatal("validation created a remote")
		}
		if _, err = gitdesktop.CommitStaged(root, ""); err == nil {
			t.Fatal("empty staged commit accepted")
		}
	})
}

// One OLE2 normal-stream workbook with BIFF8 globals, one sheet and A1=42.
// This is an actual binary file fixture, not a mocked workbook reader result.
func acceptanceXLS() []byte {
	const free, end = uint32(0xffffffff), uint32(0xfffffffe)
	out := make([]byte, 11*512)
	u16 := func(at int, value uint16) { binary.LittleEndian.PutUint16(out[at:], value) }
	u32 := func(at int, value uint32) { binary.LittleEndian.PutUint32(out[at:], value) }
	copy(out, []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1})
	u16(24, 0x3e)
	u16(26, 3)
	u16(28, 0xfffe)
	u16(30, 9)
	u16(32, 6)
	u32(44, 1)
	u32(48, 0)
	u32(56, 4096)
	u32(60, end)
	u32(68, end)
	for at := 76; at < 512; at += 4 {
		u32(at, free)
	}
	u32(76, 1)
	directory := func(at int, name string, kind byte, start, size, child uint32) {
		for i, r := range name {
			u16(at+i*2, uint16(r))
		}
		u16(at+64, uint16((len(name)+1)*2))
		out[at+66], out[at+67] = kind, 1
		u32(at+68, free)
		u32(at+72, free)
		u32(at+76, child)
		u32(at+116, start)
		u32(at+120, size)
	}
	directory(512, "Root Entry", 5, end, 0, 1)
	directory(640, "Workbook", 2, 2, 4096, free)
	for sector := 0; sector < 128; sector++ {
		u32(1024+sector*4, free)
	}
	u32(1024, end)
	u32(1028, 0xfffffffd)
	for sector := 2; sector < 9; sector++ {
		u32(1024+sector*4, uint32(sector+1))
	}
	u32(1024+9*4, end)
	var stream bytes.Buffer
	record := func(id uint16, data []byte) {
		_ = binary.Write(&stream, binary.LittleEndian, id)
		_ = binary.Write(&stream, binary.LittleEndian, uint16(len(data)))
		stream.Write(data)
	}
	bof := make([]byte, 16)
	binary.LittleEndian.PutUint16(bof, 0x600)
	binary.LittleEndian.PutUint16(bof[2:], 5)
	record(0x809, bof)
	bound := make([]byte, 13)
	binary.LittleEndian.PutUint32(bound, 41)
	bound[6] = 5
	copy(bound[8:], "Sheet")
	record(0x85, bound)
	record(0x0a, nil)
	binary.LittleEndian.PutUint16(bof[2:], 0x10)
	record(0x809, bof)
	number := make([]byte, 14)
	binary.LittleEndian.PutUint64(number[6:], math.Float64bits(42))
	record(0x203, number)
	record(0x0a, nil)
	copy(out[1536:], stream.Bytes())
	return out
}

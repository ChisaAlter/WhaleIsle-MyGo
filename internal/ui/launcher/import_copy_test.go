package launcher

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/ui/i18n"
)

func writeImportSource(t *testing.T, root, rel, body string) {
	t.Helper()
	name := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func hasHan(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// importRows scans source the way the import wizard does and returns the texts
// the dialog paints, plus the catalog it painted them from.
func importRows(t *testing.T, locale, source, dest string) ([]string, importer.Catalog) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(locale)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	previousCat := importCatActive
	importCatActive = "sessions"
	t.Cleanup(func() { importCatActive = previousCat })

	cat, err := importer.BuildCatalog(source, dest)
	if err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) { importPanel(c, Model{ImportCatalog: &cat}) }, 1200, 1000)
	return tt.Texts(), cat
}

// assertRowCopy checks the painted rows against the copy rule: the sentence the
// person needs is there, in their language, and nothing technical came with it.
func assertRowCopy(t *testing.T, texts []string, locale string, want []string, leaked []string) {
	t.Helper()
	joined := strings.Join(texts, "\n")
	for _, row := range texts {
		if strings.Contains(row, i18n.T(i18n.KeyLauncherImportUnsupported)) {
			t.Logf("row on screen: %q", row)
		}
	}
	for _, w := range want {
		if !strings.Contains(joined, w) {
			t.Fatalf("%s row copy missing %q:\n%s", locale, w, joined)
		}
	}
	for _, leak := range leaked {
		if strings.Contains(joined, leak) {
			t.Fatalf("%s rows leaked %q:\n%s", locale, leak, joined)
		}
	}
	if locale == i18n.LocaleEN && hasHan(joined) {
		t.Fatalf("english rows still show Chinese:\n%s", joined)
	}
}

// TestSessionRowsSayWhyTheyCannotBeImported renders rows for two sessions a
// scan really classifies: one holding plaintext and compressed logs at once,
// one an old SQLite database.
func TestSessionRowsSayWhyTheyCannotBeImported(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	writeImportSource(t, source, "sessions/mixed/session.v3.jsonl", "{}\n")
	writeImportSource(t, source, "sessions/mixed/session.v3.jsonl.zstd", "inert compressed marker")
	writeImportSource(t, source, "sessions/legacy/session.db", "inert database")
	for _, tc := range []struct {
		locale string
		want   []string
	}{
		{i18n.LocaleZH, []string{"无法确定该导入哪一份", "旧版 SQLite 会话库"}},
		{i18n.LocaleEN, []string{"no single copy to import", "legacy SQLite session database"}},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			texts, _ := importRows(t, tc.locale, source, dest)
			assertRowCopy(t, texts, tc.locale, tc.want,
				[]string{"Mixed plaintext", "Legacy SQLite session database is unsupported", "sessionEncoding", source, "/tmp/"})
		})
	}
}

// TestUnreadableSessionFolderSaysWhatToDo is the failure the review found on
// screen: a directory the scan cannot read used to arrive as the OS message.
// The row now tells the person what to check, and the OS words only reach the
// log.
func TestUnreadableSessionFolderSaysWhatToDo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a 0000 directory still reads on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root reads a 0000 directory")
	}
	source, dest := t.TempDir(), t.TempDir()
	writeImportSource(t, source, "sessions/locked/session.v3.jsonl", "{}\n")
	locked := filepath.Join(source, "sessions", "locked")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	for _, tc := range []struct {
		locale string
		want   string
	}{
		{i18n.LocaleZH, "读不到这个会话目录"},
		{i18n.LocaleEN, "could not be read"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			texts, cat := importRows(t, tc.locale, source, dest)
			if len(cat.Sessions) != 1 || cat.Sessions[0].Blocker != importer.BlockedUnreadable {
				t.Fatalf("scan did not classify the locked folder as unreadable: %+v", cat.Sessions)
			}
			assertRowCopy(t, texts, tc.locale, []string{tc.want},
				[]string{locked, "permission denied", "access is denied", "opendir"})
		})
	}
}

// TestSessionBlockedTextSpeaksBothLocales covers every category the classifier
// can return, including the ones a scan cannot stage here.
func TestSessionBlockedTextSpeaksBothLocales(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, blocker := range []importer.Blocker{
		importer.BlockedUnreadable,
		importer.BlockedMixedEncoding,
		importer.BlockedLegacyDatabase,
	} {
		i18n.SetLocale(i18n.LocaleZH)
		zh := sessionBlockedText(blocker)
		i18n.SetLocale(i18n.LocaleEN)
		en := sessionBlockedText(blocker)
		if zh == "" || en == "" || zh == en {
			t.Fatalf("%v: zh=%q en=%q", blocker, zh, en)
		}
		if hasHan(en) {
			t.Fatalf("%v: english sentence carries Chinese: %q", blocker, en)
		}
		t.Logf("%v on screen: zh %q / en %q", blocker, zh, en)
	}
	if got := sessionBlockedText(importer.Supported); got != "" {
		t.Fatalf("an importable row got a reason: %q", got)
	}
	if got := sessionBlockedText(importer.BlockedRejected); got != "" {
		t.Fatalf("a scanner-rejected row got a claim it cannot support: %q", got)
	}
}

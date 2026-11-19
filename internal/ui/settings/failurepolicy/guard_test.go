// Package failurepolicy holds the check that keeps a settings failure's raw text
// out of the sentence a person reads.
package failurepolicy

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// A settings failure is reported in the product's words: what did not happen and
// what to do next. The store's text — a path, a code, an err.Error() in English —
// goes behind the alert's 详情 disclosure or into the log.
//
// This list is empty on purpose. A site that must show the raw text as its
// sentence is a design decision: raise it with the Captain and change the
// surface instead of adding an exception below.
var allowlist = map[string]string{}

// scanned is the tree the rule covers: the settings surfaces.
const scanned = "internal/ui/settings"

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test's directory")
		}
		dir = parent
	}
}

func TestNoRawFailureSentence(t *testing.T) {
	root := repoRoot(t)
	tree := filepath.Join(root, scanned)
	var found []string
	walkErr := filepath.Walk(tree, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, allowed := allowlist[rel]; allowed {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		found = append(found, Violations(rel, string(src))...)
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	if len(found) > 0 {
		sort.Strings(found)
		t.Fatalf("a settings failure is shown in the product's words: %d site(s) put the store's own text in the sentence.\n"+
			"Fix: write what failed and what to do next, and pass the raw text to settingsAlert's fifth argument "+
			"or AlertOptions.Detail (internal/ui/widgets/alert.go), or to the boot log.\n%s",
			len(found), strings.Join(found, "\n"))
	}
}

// The rule above is only worth its absence of findings if a violation is caught,
// so each shape is driven into the checker directly.
func TestGuardCatchesEachRawSentenceShape(t *testing.T) {
	const header = "package probe\n"
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "alert sentence is a store field",
			body: "func draw(c *ui.Context, s store.Snapshot) {\n\tsettingsAlert(c, \"probe\", s.Err != \"\", s.Err, \"\")\n}\n",
			want: "settingsAlert sentence is the store's own failure text",
		},
		{
			name: "alert sentence is err.Error()",
			body: "func draw(c *ui.Context, err error) {\n\tsettingsAlert(c, \"probe\", err != nil, err.Error(), \"\")\n}\n",
			want: "settingsAlert sentence is the store's own failure text",
		},
		{
			name: "alert still takes the old three-argument form",
			body: "func draw(c *ui.Context, s store.Snapshot) {\n\tsettingsAlert(c, \"probe\", s.Err != \"\", func() string { return s.Err })\n}\n",
			want: "settingsAlert takes the sentence and the store's text as separate arguments",
		},
		{
			name: "alert message is a store field",
			body: "func draw(c *ui.Context, s store.Snapshot) {\n\twidgets.Alert(c, widgets.AlertOptions{ID: \"probe\", Message: s.Err})\n}\n",
			want: "Alert Message is the store's own failure text",
		},
		{
			name: "text draws a store field",
			body: "func draw(c *ui.Context, s store.Snapshot) {\n\tui.Text(c, s.Err)\n}\n",
			want: "Text draws the store's own failure text as the sentence",
		},
		{
			name: "text draws err.Error()",
			body: "func draw(c *ui.Context, err error) {\n\tui.Text(c, err.Error())\n}\n",
			want: "Text draws the store's own failure text as the sentence",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := Violations("probe.go", header+tc.body)
			if len(found) != 1 || !strings.Contains(found[0], tc.want) {
				t.Fatalf("Violations = %q, want one naming %q", found, tc.want)
			}
		})
	}
	for _, clean := range []string{
		"func draw(c *ui.Context, s store.Snapshot) {\n\tsettingsAlert(c, \"probe\", s.Err != \"\", providerCopy(\"读取失败，请重试。\", \"Could not read; try again.\"), s.Err)\n}\n",
		"func draw(c *ui.Context, s store.Snapshot) {\n\twidgets.Alert(c, widgets.AlertOptions{ID: \"probe\", Message: providerCopy(\"读取失败，请重试。\", \"Could not read; try again.\"), Detail: s.Err})\n}\n",
		"func draw(c *ui.Context, sentence, detail string) {\n\tui.Text(c, sentence)\n}\n",
	} {
		if found := Violations("probe.go", header+clean); len(found) != 0 {
			t.Fatalf("clean site was refused: %q\n%s", found, clean)
		}
	}
}

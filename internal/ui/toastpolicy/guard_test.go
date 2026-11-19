// Package toastpolicy holds the check that keeps failures out of the toast
// layer.
package toastpolicy

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// An error is never a floating toast over the content. A failure is shown where
// it happened — the row, panel or dialog whose action failed carries the reason
// and the next step (widgets.Alert), and the raw text goes to the log or the
// alert's 详情 disclosure.
//
// This list is empty on purpose. A site that cannot drop its toast is a design
// decision, not an exception here: raise it with the Captain and change the
// surface, rather than adding a path below.
var allowlist = map[string]string{}

// scanned are the trees the rule covers: the app's own Go sources.
var scanned = []string{"cmd", "internal"}

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

func TestNoErrorToast(t *testing.T) {
	root := repoRoot(t)
	var found []string
	for _, tree := range scanned {
		walkErr := filepath.Walk(filepath.Join(root, tree), func(path string, info os.FileInfo, err error) error {
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
			found = append(found, violations(rel, string(src))...)
			return nil
		})
		if walkErr != nil {
			t.Fatal(walkErr)
		}
	}
	if len(found) > 0 {
		sort.Strings(found)
		t.Fatalf("an error is never a toast: %d site(s) put a failure in the floating layer.\n"+
			"Fix: render widgets.Alert (internal/ui/widgets/alert.go) in the row, panel or dialog whose action failed, "+
			"with the reason and the next step, and keep the raw text in the log or the alert's Detail.\n%s",
			len(found), strings.Join(found, "\n"))
	}
}

// The rules above are only worth its absence of findings if a violation is
// caught, so each shape is driven into the checker directly.
func TestGuardCatchesEachToastShape(t *testing.T) {
	const header = "package probe\n\nfunc draw(c *ui.Context) {\n"
	const footer = "}\n"
	cases := []struct {
		name string
		body string
		want string
	}{
		{"untyped", "c.AddToast(ui.Toast{ID: \"x\", Title: \"saved\"})\n", "AddToast with no Type"},
		{"error kind", "c.AddToast(ui.Toast{ID: \"x\", Title: \"broke\", Type: \"error\"})\n", `"error" toast`},
		{"warning kind", "c.AddToast(ui.Toast{ID: \"x\", Title: \"careful\", Type: \"warning\"})\n", `"warning" toast`},
		{"computed kind", "c.AddToast(ui.Toast{ID: \"x\", Type: kind})\n", "computed Type"},
		{"bare toast", "c.Toast(\"broke\")\n", "untyped floating bar"},
		{"toast action", "c.ToastAction(\"broke\", \"Retry\", retry)\n", "untyped floating bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := violations("probe.go", header+tc.body+footer)
			if len(found) != 1 || !strings.Contains(found[0], tc.want) {
				t.Fatalf("violations = %q, want one naming %q", found, tc.want)
			}
		})
	}
	for _, body := range []string{
		"c.AddToast(ui.Toast{ID: \"x\", Title: \"saved\", Type: \"success\"})\n",
		"c.AddToast(ui.Toast{ID: \"x\", Title: \"why\", Type: \"info\"})\n",
		"widgets.Notice(c, \"copied\", ok, func() string { return \"copied\" })\n",
	} {
		if found := violations("probe.go", header+body+footer); len(found) != 0 {
			t.Fatalf("%q was refused: %q", strings.TrimSpace(body), found)
		}
	}
}

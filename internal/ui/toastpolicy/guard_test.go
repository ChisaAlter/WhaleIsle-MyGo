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

// The product has no floating toast layer over the content. A confirmation is
// shown where the action happened — on its own control, or as a status line in
// the surface (widgets.AckLive/widgets.AckLine/widgets.StatusLine) — and a
// failure carries the reason and the next step where it happened
// (widgets.Alert). The framework's toast primitives stay in third_party/mygo;
// this guard keeps every app use out of cmd/ and internal/.
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
		t.Fatalf("the product has no floating toast layer: %d site(s) use it.\n"+
			"Fix: a confirmation goes on the control that caused it or as a status line "+
			"(widgets.AckLive/widgets.AckLine/widgets.StatusLine in internal/ui/widgets/ack.go); "+
			"a failure renders widgets.Alert (internal/ui/widgets/alert.go) in the row, panel or dialog "+
			"whose action failed, with the reason and the next step.\n%s",
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
		{"untyped", "c.AddToast(ui.Toast{ID: \"x\", Title: \"saved\"})\n", "floating toast layer"},
		{"error kind", "c.AddToast(ui.Toast{ID: \"x\", Title: \"broke\", Type: \"error\"})\n", "floating toast layer"},
		{"success kind", "c.AddToast(ui.Toast{ID: \"x\", Title: \"saved\", Type: \"success\"})\n", "floating toast layer"},
		{"info kind", "c.AddToast(ui.Toast{ID: \"x\", Title: \"why\", Type: \"info\"})\n", "floating toast layer"},
		{"bare toast", "c.Toast(\"broke\")\n", "floating toast layer"},
		{"toast action", "c.ToastAction(\"broke\", \"Retry\", retry)\n", "floating toast layer"},
		{"notice helper", "widgets.Notice(c, \"copied\", ok, func() string { return \"copied\" })\n", "removed with the toast layer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := violations("probe.go", header+tc.body+footer)
			if len(found) == 0 {
				t.Fatalf("no violations for %q", tc.body)
			}
			for _, line := range found {
				if !strings.Contains(line, tc.want) {
					t.Fatalf("violations = %q, want each naming %q", found, tc.want)
				}
			}
		})
	}
	// AddToast's literal is flagged alongside the call.
	if found := violations("probe.go", header+"c.AddToast(ui.Toast{ID: \"x\", Type: \"success\"})\n"+footer); len(found) != 2 {
		t.Fatalf("violations = %q, want the call and the toast literal", found)
	}
	for _, body := range []string{
		"widgets.AckLine(c, \"saved\", s.Notice)\n",
		"widgets.StatusLine(c, \"busy\")\n",
		"widgets.AckLive(c, \"saved\", s.Notice, widgets.AckDuration)\n",
	} {
		if found := violations("probe.go", header+body+footer); len(found) != 0 {
			t.Fatalf("%q was refused: %q", strings.TrimSpace(body), found)
		}
	}
}

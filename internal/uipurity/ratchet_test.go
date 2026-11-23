package uipurity

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the view-purity baseline with current counts")

// TestViewPurityRatchet locks the per-file count of view-layer work that
// belongs in an owning background module instead: denied imports, per-frame
// Snapshot() copies, goroutines and timers. A count above the baseline or a
// new file+rule fails; a lower count fails too, because the improvement must
// be locked by running `go test ./internal/uipurity -update`.
func TestViewPurityRatchet(t *testing.T) {
	uiDir := filepath.Join("..", "..", "internal", "ui")
	baselinePath := filepath.Join("testdata", "baseline.txt")

	actual, err := Scan(uiDir)
	if err != nil {
		t.Fatal(err)
	}
	// Store repo-relative paths: the same baseline applies on every OS.
	for i := range actual {
		actual[i].File = strings.TrimPrefix(actual[i].File, "../../")
	}
	if *update {
		if err := os.MkdirAll(filepath.Dir(baselinePath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(baselinePath, []byte(Format(actual)), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("baseline rewritten: %d entries", len(actual))
		return
	}

	raw, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read baseline: %v (run `go test ./internal/uipurity -update` to create it)", err)
	}
	baseline, err := Parse(string(raw))
	if err != nil {
		t.Fatal(err)
	}

	var regressed, improved []string
	seen := map[string]bool{}
	for _, e := range actual {
		key := e.File + "\x00" + e.Rule
		seen[key] = true
		base, ok := baseline[key]
		switch {
		case !ok:
			regressed = append(regressed, e.File+": new "+e.Rule+" use x"+strconv.Itoa(e.Count))
		case e.Count > base.Count:
			regressed = append(regressed, e.File+": "+e.Rule+" grew "+strconv.Itoa(base.Count)+" -> "+strconv.Itoa(e.Count))
		case e.Count < base.Count:
			improved = append(improved, e.File+": "+e.Rule+" shrank "+strconv.Itoa(base.Count)+" -> "+strconv.Itoa(e.Count))
		}
	}
	for key, base := range baseline {
		if !seen[key] {
			improved = append(improved, base.File+": "+base.Rule+" shrank "+strconv.Itoa(base.Count)+" -> 0")
		}
	}
	sort.Strings(regressed)
	sort.Strings(improved)

	if len(regressed) > 0 {
		t.Fatalf("view-purity regression — new work entered the build path:\n%s\n\nView code must read prepared snapshots; move I/O, deep copies, goroutines and timers into an owning background module (ARCHITECTURE.md §8.2).", strings.Join(regressed, "\n"))
	}
	if len(improved) > 0 {
		t.Fatalf("view-purity improved — lock it by running `go test ./internal/uipurity -update`:\n%s", strings.Join(improved, "\n"))
	}
}

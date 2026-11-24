package cicoverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAllTestedPackagesCoveredByCI is the structural guard: every Go package
// that contains test files must be reachable by a `go test` command in
// .github/workflows/main-quality.yml, or carry an explicit reason in
// UncoveredExemptions. Adding a test package without wiring it into a
// workflow job fails here with the missing list.
func TestAllTestedPackagesCoveredByCI(t *testing.T) {
	root := filepath.Join("..", "..")
	pkgs, err := TestPackages(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) == 0 {
		t.Fatal("scanner found no test packages; the repository root resolution is broken")
	}
	raw, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "main-quality.yml"))
	if err != nil {
		t.Fatal(err)
	}
	patterns := CoveredPatterns(raw)
	if len(patterns) == 0 {
		t.Fatal("no `go test` package arguments parsed from main-quality.yml; the workflow parser is broken")
	}

	var b strings.Builder
	if missing := MissingCoverage(pkgs, patterns, UncoveredExemptions); len(missing) > 0 {
		b.WriteString("test packages not exercised by main-quality.yml:\n")
		for _, p := range missing {
			b.WriteString("  " + p + "\n")
		}
		b.WriteString("add each package to the fitting job (go-quality, ui-windows, windows-native-packages or entry-and-git), or record why it cannot run on a hosted runner in internal/cicoverage UncoveredExemptions\n")
	}
	if stale := StaleExemptions(pkgs, patterns, UncoveredExemptions); len(stale) > 0 {
		b.WriteString("stale UncoveredExemptions entries:\n")
		for _, s := range stale {
			b.WriteString("  " + s + "\n")
		}
	}
	if b.Len() > 0 {
		t.Fatal(b.String())
	}
}

func TestCoveredPatternsParsesBlockAndInlineRuns(t *testing.T) {
	workflow := `
jobs:
  a:
    steps:
      - uses: actions/checkout@0000000000000000000000000000000000000000
      - name: unit
        run: |
          set -euo pipefail
          # a comment mentioning go test ./should/not/count is still a comment
          go test -race -count=1 \
            ./internal/one \
            ./internal/tree/... \
            ./internal/two
      - name: wd
        working-directory: third_party/mygo
        run: go test -tags release .
      - name: inline
        run: go test -count=1 ./internal/three && echo ./not-a-pkg
      - name: subset
        run: go test -count=1 -run TestOnlyOne ./internal/partial
      - name: subset-eq
        run: go test -run=TestOnlyOne ./internal/partial2
`
	pats := CoveredPatterns([]byte(workflow))
	want := []string{
		"./internal/one",
		"./internal/tree/...",
		"./internal/two",
		"./third_party/mygo",
		"./internal/three",
	}
	for _, w := range want {
		if !pats[w] {
			t.Errorf("missing pattern %s (got %v)", w, pats)
		}
	}
	for _, bad := range []string{"./should/not/count", "./not-a-pkg", "./internal/partial", "./internal/partial2"} {
		if pats[bad] {
			t.Errorf("unexpected pattern %s leaked into %v", bad, pats)
		}
	}
}

func TestCoversExpandsWildcardAndExact(t *testing.T) {
	cases := []struct {
		pattern, pkg string
		want         bool
	}{
		{"./internal/store/...", "./internal/store/session", true},
		{"./internal/store/...", "./internal/store/internal/storefs", true},
		{"./internal/store/...", "./internal/storex", false},
		{"./internal/store/...", "./internal/store", true},
		{"./internal/ui/chat", "./internal/ui/chat", true},
		{"./internal/ui/chat", "./internal/ui/chat/extra", false},
		{"./...", "./anything/at/all", true},
	}
	for _, tc := range cases {
		if got := Covers(tc.pattern, tc.pkg); got != tc.want {
			t.Errorf("Covers(%q, %q) = %v, want %v", tc.pattern, tc.pkg, got, tc.want)
		}
	}
}

func TestMissingAndStaleExemptions(t *testing.T) {
	pkgs := []string{"./a", "./b", "./c", "./d"}
	patterns := map[string]bool{"./a": true, "./c": true}
	exemptions := map[string]string{
		"./b": "needs a real display",
		"./c": "covered but still listed",
		"./e": "no longer exists",
	}
	if missing := MissingCoverage(pkgs, patterns, exemptions); len(missing) != 1 || missing[0] != "./d" {
		t.Fatalf("MissingCoverage = %v, want [./d]", missing)
	}
	stale := StaleExemptions(pkgs, patterns, exemptions)
	if len(stale) != 2 {
		t.Fatalf("StaleExemptions = %v, want 2 entries", stale)
	}
	if !strings.Contains(stale[0], "./c") || !strings.Contains(stale[1], "./e") {
		t.Fatalf("unexpected stale entries: %v", stale)
	}
}

func TestTestPackagesFindsAndSkips(t *testing.T) {
	root := t.TempDir()
	mk := func(rel string) {
		t.Helper()
		dir := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "x_test.go"), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("internal/one")
	mk("deep/nested/two")
	mk("third_party/lib")
	mk("tmp/scratch")
	mk("node_modules/dep")
	mk(".toolchain/x")
	mk(".git/hooks")
	mk("internal/one/testdata") // testdata is not a package dir
	got, err := TestPackages(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"./internal/one": true, "./deep/nested/two": true}
	if len(got) != len(want) {
		t.Fatalf("TestPackages = %v, want keys of %v", got, want)
	}
	for _, p := range got {
		if !want[p] {
			t.Fatalf("unexpected package %s in %v", p, got)
		}
	}
}

package cicoverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAllTestedPackagesCoveredByCI is the structural guard: every Go package
// that contains test files must be exercised by a full-package `go test` command
// in .github/workflows/main-quality.yml, or carry an explicit reason in
// UncoveredExemptions. Adding a test package without wiring it into a workflow
// job fails here with the missing list.
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
	patterns, err := ParseCoveredPatterns(raw)
	if err != nil {
		t.Fatalf("ParseCoveredPatterns(main-quality.yml): %v", err)
	}
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

func TestCoveredPatternsHonorsYAMLKeyOrderAndDefaults(t *testing.T) {
	t.Run("working-directory after run", func(t *testing.T) {
		wf := `
jobs:
  gate:
    runs-on: windows-latest
    steps:
      - name: Test the MyGo fork
        run: go test ./...
        working-directory: third_party/mygo
`
		pats := CoveredPatterns([]byte(wf))
		if !pats["./third_party/mygo/..."] {
			t.Fatalf("expected ./third_party/mygo/..., got %v", pats)
		}
		if pats["./..."] || len(MissingCoverage([]string{"./internal/state"}, pats, nil)) == 0 {
			t.Fatalf("mygo working-directory leaked root coverage: %v", pats)
		}
	})

	t.Run("job defaults working-directory", func(t *testing.T) {
		wf := `
jobs:
  gate:
    defaults:
      run:
        working-directory: third_party/mygo
    steps:
      - run: go test ./...
`
		pats := CoveredPatterns([]byte(wf))
		if !pats["./third_party/mygo/..."] || len(MissingCoverage([]string{"./internal/state"}, pats, nil)) == 0 {
			t.Fatalf("job defaults working-directory leaked root coverage: %v", pats)
		}
	})

	t.Run("workflow defaults with step override", func(t *testing.T) {
		wf := `
defaults:
  run:
    working-directory: third_party/mygo
jobs:
  gate:
    steps:
      - run: go test .
      - run: go test ./internal/ok
        working-directory: .
`
		pats := CoveredPatterns([]byte(wf))
		if !pats["./third_party/mygo"] || !pats["./internal/ok"] || pats["./"] {
			t.Fatalf("unexpected patterns from workflow defaults: %v", pats)
		}
	})
}

func TestCoveredPatternsRejectsNonExecutingAndFilteredCommands(t *testing.T) {
	cases := []struct {
		name string
		wf   string
	}{
		{
			name: "compile only -c verbatim voice-native-build",
			wf: `
jobs:
  voice:
    steps:
      - run: |
          GOARCH="$arch" go test -c -mod=readonly -o "$RUNNER_TEMP/voicecapture-$arch.test" ./internal/desktop/voicecapture
`,
		},
		{
			name: "compile only -c with -o",
			wf: `
jobs:
  a:
    steps:
      - run: go test -c -o /dev/null ./internal/state
`,
		},
		{
			name: "dry run -n",
			wf: `
jobs:
  a:
    steps:
      - run: go test -n ./internal/state
`,
		},
		{
			name: "list only space arg",
			wf: `
jobs:
  a:
    steps:
      - run: go test -list . ./internal/state
`,
		},
		{
			name: "list only equals",
			wf: `
jobs:
  a:
    steps:
      - run: go test -list=.* ./internal/state
`,
		},
		{
			name: "broad skip dot",
			wf: `
jobs:
  a:
    steps:
      - run: go test -skip . ./internal/state
`,
		},
		{
			name: "broad skip star",
			wf: `
jobs:
  a:
    steps:
      - run: go test -skip=.* ./internal/state
`,
		},
		{
			name: "broad skip anchored star",
			wf: `
jobs:
  a:
    steps:
      - run: go test -skip '^.*$' ./internal/state
`,
		},
		{
			name: "broad skip Test prefix",
			wf: `
jobs:
  a:
    steps:
      - run: go test -skip '^Test' ./internal/state
`,
		},
		{
			name: "invalid skip regex",
			wf: `
jobs:
  a:
    steps:
      - run: go test -skip='[' ./internal/state
`,
		},
		{
			name: "step if false",
			wf: `
jobs:
  a:
    steps:
      - if: false
        run: go test ./internal/state
`,
		},
		{
			name: "step if expression false",
			wf: `
jobs:
  a:
    steps:
      - if: ${{ false }}
        run: go test ./internal/state
`,
		},
		{
			name: "job if false",
			wf: `
jobs:
  a:
    if: false
    steps:
      - run: go test ./internal/state
`,
		},
		{
			name: "job if expression false",
			wf: `
jobs:
  a:
    if: ${{ false }}
    steps:
      - run: go test ./internal/state
`,
		},
		{
			name: "command substitution filter",
			wf: `
jobs:
  a:
    steps:
      - run: go test $(go list ./... | grep -v internal/state)
`,
		},
		{
			name: "backtick command substitution",
			wf:   "jobs:\n  a:\n    steps:\n      - run: go test `go list ./internal/state`\n",
		},
		{
			name: "echo mentioning go test",
			wf: `
jobs:
  a:
    steps:
      - run: echo go test ./internal/state
`,
		},
		{
			name: "boolean flag-only no package leaked from other args",
			wf: `
jobs:
  a:
    steps:
      - run: go test -o ./internal/state ./internal/other
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pats := CoveredPatterns([]byte(tc.wf))
			for _, forbidden := range []string{"./internal/state", "./internal/desktop/voicecapture", "./"} {
				if pats[forbidden] || len(MissingCoverage([]string{forbidden}, pats, nil)) == 0 {
					if forbidden == "./" && len(pats) == 0 {
						continue
					}
					if pats[forbidden] {
						t.Fatalf("bypass leaked %s in patterns %v", forbidden, pats)
					}
				}
			}
			if len(MissingCoverage([]string{"./internal/state", "./internal/desktop/voicecapture"}, pats, nil)) != 2 {
				t.Fatalf("expected neither state nor voicecapture covered, got %v", pats)
			}
		})
	}
}

func TestCoveredPatternsHandlesArgsBoundaryFlagsSeparatorsAndBadYAML(t *testing.T) {
	t.Run("targeted skip and active condition remain covered", func(t *testing.T) {
		wf := `
jobs:
  a:
    if: ${{ !cancelled() }}
    steps:
      - if: ${{ !cancelled() && steps.go.outcome == 'success' }}
        run: go test -count=1 -skip '^TestAccountAvatarLoadsAsyncCachesNegativeAndBoundsLRU$' ./internal/ui/settings
`
		pats := CoveredPatterns([]byte(wf))
		if !pats["./internal/ui/settings"] || len(pats) != 1 {
			t.Fatalf("got %v, want only ./internal/ui/settings", pats)
		}
	})

	t.Run("args after -args do not count as packages", func(t *testing.T) {
		wf := `
jobs:
  a:
    steps:
      - run: go test ./internal/paths -args ./internal/state
`
		pats := CoveredPatterns([]byte(wf))
		if !pats["./internal/paths"] || pats["./internal/state"] || len(pats) != 1 {
			t.Fatalf("got %v, want only ./internal/paths", pats)
		}
	})

	t.Run("flag value starting with dot does not leak as package", func(t *testing.T) {
		wf := `
jobs:
  a:
    steps:
      - run: go test -o ./internal/fakeout -coverprofile ./cover.out ./internal/paths
`
		pats := CoveredPatterns([]byte(wf))
		if !pats["./internal/paths"] || pats["./internal/fakeout"] || pats["./cover.out"] || len(pats) != 1 {
			t.Fatalf("got %v, want only ./internal/paths", pats)
		}
	})

	t.Run("semicolon separates run subset from full test command", func(t *testing.T) {
		wf := `
jobs:
  a:
    steps:
      - run: go test -run X ./internal/paths; go test ./internal/state
`
		pats := CoveredPatterns([]byte(wf))
		if pats["./internal/paths"] || !pats["./internal/state"] || len(pats) != 1 {
			t.Fatalf("got %v, want only ./internal/state", pats)
		}
	})

	t.Run("malformed YAML returns error and CoveredPatterns fails closed", func(t *testing.T) {
		bad := []byte("jobs:\n  a:\n    steps: [unclosed\n")
		if _, err := ParseCoveredPatterns(bad); err == nil {
			t.Fatal("expected ParseCoveredPatterns error on malformed YAML")
		}
		if pats := CoveredPatterns(bad); len(pats) != 0 {
			t.Fatalf("expected empty patterns on malformed YAML, got %v", pats)
		}
	})
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

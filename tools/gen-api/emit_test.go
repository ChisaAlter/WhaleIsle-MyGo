package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmitMatchesToolchainEndpoints cross-checks the parsed testdata
// descriptors against .toolchain/endpoints.json (the build-time cross-check
// reference) for the packages covered by testdata.
func TestEmitMatchesToolchainEndpoints(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".toolchain", "endpoints.json"))
	if err != nil {
		t.Skipf("endpoints.json not available: %v", err)
	}
	var entries []struct {
		Pkg    string   `json:"pkg"`
		Ep     string   `json:"ep"`
		Mode   string   `json:"mode"`
		Stream bool     `json:"stream"`
		Params []string `json:"params"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("unmarshal endpoints.json: %v", err)
	}
	want := map[string]map[string][]string{} // pkg -> ep -> params
	for _, e := range entries {
		if want[e.Pkg] == nil {
			want[e.Pkg] = map[string][]string{}
		}
		want[e.Pkg][e.Ep] = e.Params
	}
	for _, pkg := range []string{"dsh-llm", "dsh-goal"} {
		pf := parseTestData(t, pkg)
		for _, e := range pf.file.Descriptors {
			ref, ok := want[pkg][e.FullName]
			if !ok {
				// endpoints.json is newer than the vendored build; missing
				// entries are fine, present entries must agree.
				continue
			}
			if len(ref) != len(e.Params) {
				t.Fatalf("%s/%s: %d params, endpoints.json has %d", pkg, e.FullName, len(e.Params), len(ref))
			}
			for i, rp := range ref {
				got := formatParamRef(&e.Params[i])
				if got != rp {
					t.Fatalf("%s/%s param %d: got %q, endpoints.json has %q", pkg, e.FullName, i, got, rp)
				}
			}
		}
	}
}

// formatParamRef renders a parsed parameter the way .toolchain/endpoints.json
// does: wire[@lookup]:typeSymbolTail.
func formatParamRef(p *paramDesc) string {
	s := p.Wire
	if p.Source == "lookup" && p.Lookup != "" {
		s += "@" + p.Lookup
	}
	if p.TypeSymbol != "" {
		tail := p.TypeSymbol
		if i := strings.LastIndex(tail, "#"); i >= 0 {
			tail = tail[i+1:]
		}
		s += ":" + tail
	}
	return s
}

func TestRunGeneratesCompilableGo(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not in PATH")
	}
	tmp := t.TempDir()
	roots := []string{
		filepath.Join("testdata", "dsh-llm"),
		filepath.Join("testdata", "dsh-goal"),
	}
	if err := run(roots, tmp, "api"); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, f := range []string{
		filepath.Join(tmp, "dsh_llm_gen.go"),
		filepath.Join(tmp, "dsh_goal_gen.go"),
		filepath.Join(tmp, "registry_gen.go"),
	} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected output %s: %v", f, err)
		}
	}
	// The generated package must compile on its own.
	mod := "module gencheck\n\ngo 1.27\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
}

func TestRegistryHeaderAndDuplicateHandling(t *testing.T) {
	tmp := t.TempDir()
	// Pass the same dsh-llm build twice via two roots that parse to the same
	// package: the second must be skipped silently.
	roots := []string{
		filepath.Join("testdata", "dsh-llm"),
		filepath.Join("testdata", "dsh-goal"),
		filepath.Join("testdata", "dsh-llm"),
	}
	if err := run(roots, tmp, "api"); err != nil {
		t.Fatalf("run: %v", err)
	}
	reg, err := os.ReadFile(filepath.Join(tmp, "registry_gen.go"))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	for _, want := range []string{"\"llm/discoverModels\"", "\"goals/clear\""} {
		if !strings.Contains(string(reg), want) {
			t.Fatalf("registry missing %s", want)
		}
	}
	if strings.Contains(string(reg), "// NOTE: duplicate endpoint") {
		t.Fatalf("identical duplicate package should be skipped without a note")
	}
}

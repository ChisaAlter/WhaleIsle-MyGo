package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMiniSchema(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "mini-schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	inv, err := parseInventory(data)
	if err != nil {
		t.Fatal(err)
	}
	src, err := renderEventsPackage(inv, "testdata/mini-schema.json", "events")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	out := string(src)
	for _, want := range []string{
		"DO NOT EDIT",
		"package events",
		"type SessionHeader struct",
		"type TurnStart struct",
		"type ToolCall struct",
		"TypeTurnStart",
		"TypeToolCall",
		"EventsByType",
		"type Envelope struct",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("generated source missing %q", want)
		}
	}
}

func TestRunGeneratesCompilableGo(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	schema := filepath.Join("..", "..", "runtime", "vendor", "deepseek-harness", "docs", "persistence-schema.json")
	if _, err := os.Stat(schema); err != nil {
		schema = filepath.Join("testdata", "mini-schema.json")
	}
	dir := t.TempDir()
	outDir := filepath.Join(dir, "events")
	if err := run(schema, outDir, "events"); err != nil {
		t.Fatalf("run: %v", err)
	}
	mod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(mod, []byte("module genevents_test\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./events")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	// Write a tiny test into the package so `go test` exercises the build.
	testSrc := `package events

import (
	"encoding/json"
	"testing"
)

func TestPackageBuilds(t *testing.T) {
	if len(Roots) == 0 {
		t.Fatal("no roots")
	}
	_ = Envelope{}
	_ = json.RawMessage(nil)
}
`
	if err := os.WriteFile(filepath.Join(outDir, "smoke_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}
}

func TestRootGoName(t *testing.T) {
	cases := []struct {
		r    PersistenceRoot
		want string
	}{
		{PersistenceRoot{Kind: "event", Event: "user/message"}, "UserMessage"},
		{PersistenceRoot{Kind: "event", Event: "tool-workflow/agent-end"}, "ToolWorkflowAgentEnd"},
		{PersistenceRoot{Kind: "header", Key: "SessionHeader"}, "SessionHeader"},
		{PersistenceRoot{Kind: "envelope", Key: "SessionEventEnvelope"}, "SessionEventEnvelope"},
	}
	for _, c := range cases {
		if got := rootGoName(c.r); got != c.want {
			t.Fatalf("rootGoName(%+v) = %q, want %q", c.r, got, c.want)
		}
	}
}

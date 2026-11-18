package pluginmanager

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNativeMCPPlanPreservesTransportOptionsAndEnableChoice(t *testing.T) {
	op := action{kind: "install", capabilityKind: CapabilityMCP, mcpSpecs: []json.RawMessage{json.RawMessage(`{"id":"demo","command":"node","args":["srv.js",""],"env":{"TOKEN":"secret","LANG":"zh"},"cwd":"C:/work","enabled":true}`)}}
	if err := prepareNativeAction(&op); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(op.mcpSpecs[0], &got)
	if got["transport"] != "stdio" || got["serverName"] != "demo" || got["enabled"] != true || got["cwd"] != "C:/work" || !reflect.DeepEqual(got["args"], []any{"srv.js", ""}) {
		t.Fatalf("lost source semantics: %s", op.mcpSpecs[0])
	}
	disabled := false
	op.options.Enabled = &disabled
	if err := prepareNativeAction(&op); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(op.mcpSpecs[0], &got)
	if got["enabled"] != false {
		t.Fatal("explicit disabled choice was ignored")
	}
	httpSpec, err := normalizeMCP(json.RawMessage(`{"id":"web","type":"http","url":"https://example.invalid/mcp","headers":{"Authorization":"x"}}`), nil)
	if err != nil || !strings.Contains(string(httpSpec), `"transport":"streamable-http"`) {
		t.Fatalf("http: %s %v", httpSpec, err)
	}
	if _, err = normalizeMCP(json.RawMessage(`{"id":"old","type":"sse","url":"https://example.invalid/sse"}`), nil); err == nil {
		t.Fatal("silently changed legacy SSE protocol")
	}
}

func TestNativeMCPExistingIdentityBlocksAllWrites(t *testing.T) {
	writes := 0
	c := Client{fixtureCall(func(_ context.Context, method string, args, out any) error {
		if method == "mygoMCP/capabilities" {
			return reply(out, map[string]any{"createOnly": true})
		}
		if method == "mcpServers/list" {
			return reply(out, map[string]any{"servers": []any{map[string]any{"id": "existing", "spec": map[string]any{"serverName": "demo"}}}})
		}
		writes++
		return nil
	})}
	op := action{kind: "install", capabilityKind: CapabilityMCP, mcpSpecs: []json.RawMessage{json.RawMessage(`{"id":"new","serverName":"demo","command":"node"}`)}}
	if err := prepareNativeAction(&op); err != nil {
		t.Fatal(err)
	}
	change, err := c.applyNative(context.Background(), op)
	if err != nil || change.Application != "failed" || writes != 0 {
		t.Fatalf("conflict wrote: %+v %v writes=%d", change, err, writes)
	}
}

func TestLocalDefinitionsDoNotBecomeBundleJustBecausePackageHasMain(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"fixture-mcp","main":"server.js","mcpServers":{"fixture":{"command":"node","args":["server.js"]}}}`), 0600)
	no := false
	inspection := Inspection{Status: "refused", Bundle: &no, Problem: "not-a-bundle"}
	enrichInspectionCapabilities(dir, &inspection)
	if inspection.Status != "accepted" || !nativeCapabilityInstallReady(&inspection) {
		t.Fatalf("local definition lost behind package main: %+v", inspection)
	}
	inspection.Capabilities.HasBundle = true
	if nativeCapabilityInstallReady(&inspection) {
		t.Fatal("bundle installation was stolen by local capability adapter")
	}
}

func TestLocalSkillPlanCarriesAllResourcesAndSeparatesFrontmatter(t *testing.T) {
	dir := t.TempDir()
	_ = os.Mkdir(filepath.Join(dir, "scripts"), 0700)
	markdown := "---\nname: resource-skill\ndescription: A skill\nmetadata:\n  owner: retained\n---\n\nUse scripts/run.sh\n"
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(markdown), 0600)
	_ = os.WriteFile(filepath.Join(dir, "scripts", "run.sh"), []byte("#!/bin/sh\nprintf result\n"), 0700)
	caps, err := InspectLocalPackage(dir)
	if err != nil || len(caps.LocalSkills) != 1 {
		t.Fatal(caps, err)
	}
	skill := caps.LocalSkills[0]
	if strings.Contains(skill.Content, "metadata:") {
		t.Fatal("frontmatter leaked into instruction body")
	}
	files, err := readSkillFiles(context.Background(), skill)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("lost resources: %+v", files)
	}
	for _, file := range files {
		if file.Path == "SKILL.md" && string(file.Data) != markdown {
			t.Fatal("rewrote existing metadata")
		}
	}
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(markdown+"changed"), 0600)
	if _, err = readSkillFiles(context.Background(), skill); err == nil {
		t.Fatal("accepted a skill edited after inspection")
	}
}

func TestNativeSkillCreateUsesHostEnvelopeAndRoot(t *testing.T) {
	c := Client{fixtureCall(func(_ context.Context, method string, args, out any) error {
		raw, _ := json.Marshal(args)
		var got struct {
			Request map[string]any `json:"request"`
		}
		_ = json.Unmarshal(raw, &got)
		if method != "skillInventory/create" || got.Request["root"] != "user-dsh" || got.Request["content"] != "# Body" {
			t.Errorf("invalid wire: %s %s", method, raw)
		}
		return nil
	})}
	if err := c.CreateSkill(context.Background(), LocalSkillManifest{Name: "plain", Description: "Plain", Content: "# Body", ModelInvocable: true, UserInvocable: true}); err != nil {
		t.Fatal(err)
	}
}

func TestSkillFrontmatterPreservesWindowsBodyAndExactFence(t *testing.T) {
	body := "\r\nFirst line\r\nSecond line\r\n"
	meta, got, err := skillMarkdown("---\r\nname: windows-skill\r\ndescription: Fixture\r\n---\r\n" + body)
	if err != nil || meta["name"] != "windows-skill" || got != body {
		t.Fatalf("Windows skill changed content: %q %v", got, err)
	}
	if _, _, err := skillMarkdown("---\nname: fixture\n---not-a-fence\nbody"); err == nil {
		t.Fatal("accepted a partial frontmatter closing fence")
	}
}

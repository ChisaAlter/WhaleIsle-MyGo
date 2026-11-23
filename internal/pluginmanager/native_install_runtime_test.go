//go:build functionalacceptance

package pluginmanager_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	manager "whaleisle/internal/pluginmanager"
)

// Uses the selected read-only runtime and a fresh profile. The fixture creates
// no model/account calls, external network dependencies or user installation.
func TestFunctionalNativeCapabilityImports(t *testing.T) {
	stage := os.Getenv("WHALEISLE_FUNCTIONAL_RUNTIME")
	if !filepath.IsAbs(stage) {
		t.Fatal("explicit WHALEISLE_FUNCTIONAL_RUNTIME required")
	}
	root := t.TempDir()
	data := filepath.Join(root, "data")
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(paths.EnvUserData, data)
	t.Setenv(paths.EnvVendor, filepath.Join(stage, "vendor"))
	t.Setenv(paths.EnvNodeDir, filepath.Join(stage, "node"))
	home, vendor := paths.DshHome(), paths.VendorDir()
	node := filepath.Join(stage, "node", "node.exe")
	probe := filepath.Join(root, "native-probe.mjs")
	probeSource, err := os.ReadFile(filepath.Join("testdata", "native-capability-probe.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(probe, probeSource, 0600); err != nil {
		t.Fatal(err)
	}
	probeURL := (&url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(probe)}).String()
	quotedProbe, _ := json.Marshal(probeURL)
	patch := filepath.Join(root, "native-probe.patch.yml")
	if err := os.WriteFile(patch, []byte("- insert:\n    - id: fixture-native-probe\n      name: "+string(quotedProbe)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sup := harness.New(harness.Options{Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: filepath.Join(vendor, "deepseek-harness"), VendorDir: vendor, Home: home, Host: "127.0.0.1", Port: 36842, PatchFiles: []string{patch}}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: filepath.Join(vendor, "deepseek-harness"), ProfileDir: paths.WebProfileDir(), DshHome: home, RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 90 * time.Second})
	sup.Start()
	defer sup.Stop()
	defer func() {
		if t.Failed() {
			t.Log("Host diagnostics:\n" + strings.Join(sup.CurrentStartLogs(), "\n"))
		}
	}()
	var client *dsh.Client
	deadline := time.Now().Add(100 * time.Second)
	for time.Now().Before(deadline) {
		status := sup.Status()
		if status.State == harness.Failed {
			t.Fatal(status.Err)
		}
		if status.State == harness.Ready {
			client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: status.Endpoint.Origin, Cookie: status.Endpoint.Cookie}, HTTP: &http.Client{Timeout: 30 * time.Second}}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if client == nil {
		t.Fatal("Host did not become ready")
	}
	install := func(dir string) {
		t.Helper()
		controller := manager.New()
		defer controller.Detach()
		controller.Attach(client, nil)
		actions := controller.Actions(controller.Snapshot().Value.Owner)
		actions.OpenInstall()
		actions.EditInstall(dir, "")
		actions.Inspect()
		until := time.Now().Add(30 * time.Second)
		for time.Now().Before(until) {
			state := controller.Snapshot().Value.Install
			if state.Phase == "done" {
				return
			}
			if state.Error != "" {
				t.Fatal(state.Error)
			}
			time.Sleep(25 * time.Millisecond)
		}
		t.Fatalf("native import did not settle: %+v", controller.Snapshot().Value.Install)
	}
	mcpDir := filepath.Join(root, "mcp")
	_ = os.Mkdir(mcpDir, 0700)
	mcpSource, err := os.ReadFile(filepath.Join("testdata", "native-capability-mcp.cjs"))
	if err != nil {
		t.Fatal(err)
	}
	mcpScript := filepath.Join(root, "fixture-mcp.cjs")
	if err := os.WriteFile(mcpScript, mcpSource, 0600); err != nil {
		t.Fatal(err)
	}
	spec := map[string]any{"mcpServers": map[string]any{"fixture": map[string]any{"command": node, "args": []string{mcpScript}, "enabled": false, "env": map[string]string{"TEST_TOKEN": "fixture-secret"}, "cwd": workspace}}}
	raw, _ := json.Marshal(spec)
	_ = os.WriteFile(filepath.Join(mcpDir, ".mcp.json"), raw, 0600)
	install(mcpDir)
	var servers struct {
		Servers []struct {
			ID      string `json:"id"`
			Enabled bool   `json:"enabled"`
		} `json:"servers"`
	}
	if err := client.Call(context.Background(), "mcpServers/list", nil, &servers); err != nil {
		t.Fatal(err)
	}
	if len(servers.Servers) != 1 || servers.Servers[0].ID != "fixture" || servers.Servers[0].Enabled {
		t.Fatalf("MCP config mismatch: %+v", servers)
	}
	skillDir := filepath.Join(root, "skill")
	_ = os.MkdirAll(filepath.Join(skillDir, "scripts"), 0700)
	markdown := "---\nname: fixture-directory-skill\ndescription: Owned fixture\nmetadata:\n  source: preserved\n---\n\nUse scripts/task.js\n"
	_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(markdown), 0600)
	_ = os.WriteFile(filepath.Join(skillDir, "scripts", "task.js"), []byte("console.log('fixture resource')\n"), 0700)
	install(skillDir)
	for _, rel := range []string{"SKILL.md", "scripts/task.js"} {
		source, err := os.ReadFile(filepath.Join(skillDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		installed, err := os.ReadFile(filepath.Join(home, "skills", "fixture-directory-skill", filepath.FromSlash(rel)))
		if err != nil || string(source) != string(installed) {
			t.Fatalf("skill resource %s changed: %v", rel, err)
		}
	}
	var session api.SessionCreateValue
	if err := client.Call(context.Background(), "session/create", map[string]any{"request": map[string]any{"cwd": workspace, "agentPreset": "standard"}}, &session); err != nil {
		t.Fatal(err)
	}
	var detail struct{ Name, Content string }
	if err := client.Call(context.Background(), "skillInventory/get", map[string]any{"request": map[string]string{"name": "fixture-directory-skill", "sessionId": session.SessionId, "cwd": workspace}}, &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Name != "fixture-directory-skill" || detail.Content == "" {
		t.Fatal("Host cannot read imported skill")
	}
	if err := client.Call(context.Background(), "mcpServers/setEnabled", map[string]any{"request": api.McpServerEnableRequest{Id: "fixture", Enabled: true}}, nil); err != nil {
		t.Fatal(err)
	}
	connected := false
	until := time.Now().Add(20 * time.Second)
	for time.Now().Before(until) {
		var state api.McpServerSnapshot
		if err := client.Call(context.Background(), "mcpServers/list", nil, &state); err != nil {
			t.Fatal(err)
		}
		for _, row := range state.Servers {
			if row.Id == "fixture" && row.Connection != nil && row.Connection.Health == "connected" && len(row.Connection.Tools) == 1 {
				connected = true
			}
		}
		if connected {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !connected {
		t.Fatal("MCP fixture did not register its tool")
	}
	var consumed json.RawMessage
	if err := client.Call(context.Background(), "fixtureNativeCapability/run", map[string]any{"request": map[string]string{"sessionId": session.SessionId, "cwd": workspace}}, &consumed); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(consumed), "fixture:ok:workspace:true") || !strings.Contains(string(consumed), "fixture resource") {
		t.Fatalf("imported capabilities did not execute: %s", consumed)
	}
	if err := client.Call(context.Background(), "mcpServers/setEnabled", map[string]any{"request": api.McpServerEnableRequest{Id: "fixture", Enabled: false}}, nil); err != nil {
		t.Fatal(err)
	}
	t.Log("real rc.4 MCP import, enablement, tool execution and complete local Skill resource execution confirmed; model requests=0")
}

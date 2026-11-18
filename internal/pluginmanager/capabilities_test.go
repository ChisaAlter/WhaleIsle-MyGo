package pluginmanager

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClassifyPackageManifestKinds(t *testing.T) {
	cases := []struct {
		name         string
		raw          string
		skillMD      bool
		mcpFile      bool
		wantKind     CapabilityKind
		wantNativeOK bool
	}{
		{
			name:         "pure-host",
			raw:          `{"name":"host-ext","main":"./lib/index.js","dsh":{"bundle":{"patch":"cordis.patch.yml"}}}`,
			wantKind:     CapabilityPureHost,
			wantNativeOK: true,
		},
		{
			name:         "hybrid",
			raw:          `{"name":"hybrid-ext","exports":{".":"./lib/index.js","./client":"./client.js"},"dsh":{"bundle":{"patch":"cordis.patch.yml"},"client":"./client.js"}}`,
			wantKind:     CapabilityHybrid,
			wantNativeOK: true,
		},
		{
			name:         "web-ui-only",
			raw:          `{"name":"ui-only","exports":{"./client":"./client.js"},"dsh":{"client":"./client.js"}}`,
			wantKind:     CapabilityWebUIOnly,
			wantNativeOK: false,
		},
		{
			name:         "mcp-from-manifest",
			raw:          `{"name":"mcp-pack","mcpServers":{"fs":{"command":"npx","args":["-y","server"]}}}`,
			wantKind:     CapabilityMCP,
			wantNativeOK: true,
		},
		{
			name:         "skill-from-md",
			raw:          `{"name":"skill-pack"}`,
			skillMD:      true,
			wantKind:     CapabilitySkill,
			wantNativeOK: true,
		},
		{
			name:         "unknown",
			raw:          `{"name":"plain-pkg","version":"1.0.0"}`,
			wantKind:     CapabilityUnknown,
			wantNativeOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyPackageManifestJSON([]byte(tc.raw), tc.skillMD, tc.mcpFile)
			if got.Kind != tc.wantKind || got.NativeCompatible != tc.wantNativeOK {
				t.Fatalf("got %+v, want kind=%q native=%v", got, tc.wantKind, tc.wantNativeOK)
			}
			if got.CompatibilityLabel == "" {
				t.Fatal("missing compatibility label")
			}
		})
	}
}

func TestInstallUnblocksLocalMCPAndSkillWithoutBundle(t *testing.T) {
	mcpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(mcpDir, ".mcp.json"), []byte(`{"mcpServers":{"demo":{"command":"node","args":["srv.js"]}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	skillDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: sample-skill\ndescription: Native skill\n---\n\n# Body\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, spec, wantRPC string
	}{
		{"local-mcp", mcpDir, "mygoMCP/import"},
		{"local-skill", skillDir, "mygoSkills/import"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var calls []string
			var installedMCP json.RawMessage
			var installedSkill bool
			done := make(chan struct{})
			c := New()
			defer c.Detach()
			c.Attach(fixtureCall(func(_ context.Context, method string, args, out any) error {
				mu.Lock()
				calls = append(calls, method)
				mu.Unlock()
				switch method {
				case "pluginManager/registries":
					return reply(out, RegistryInfo{})
				case "pluginInventory/list", "pluginManager/listPlugins":
					return reply(out, map[string]any{})
				case "pluginManager/listBundles":
					return reply(out, []Bundle{})
				case "pluginManager/inspect":
					no := false
					return reply(out, Inspection{Status: "refused", Problem: "not-a-bundle", Reason: "declares no dsh.bundle", Bundle: &no})
				case "mygoMCP/capabilities":
					return reply(out, map[string]any{"createOnly": true})
				case "mygoMCP/import":
					raw, _ := json.Marshal(args)
					var envelope struct {
						Request struct {
							Spec json.RawMessage `json:"spec"`
						} `json:"request"`
					}
					if json.Unmarshal(raw, &envelope) != nil || len(envelope.Request.Spec) == 0 {
						t.Error("missing MCP request envelope")
					}
					installedMCP = envelope.Request.Spec
					return nil
				case "mcpServers/list":
					if len(installedMCP) == 0 {
						return reply(out, map[string]any{"servers": []any{}})
					}
					return reply(out, map[string]any{"servers": []any{map[string]any{"id": "demo", "spec": installedMCP}}})
				case "skillInventory/list":
					return reply(out, map[string]any{"skills": []any{}})
				case "mygoSkills/capabilities":
					return reply(out, map[string]bool{"importDirectory": true})
				case "mygoSkills/import":
					installedSkill = true
					return nil
				case "mygoSkills/status":
					return reply(out, map[string]any{"installed": installedSkill, "detail": map[string]any{"name": "sample-skill", "content": "# Body\n", "modelInvocable": true, "userInvocable": true}})
				case "skillInventory/get":
					return reply(out, map[string]any{"name": "sample-skill", "content": "# Body\n", "modelInvocable": true, "userInvocable": true})
				default:
					t.Errorf("unexpected method %s", method)
					return nil
				}
			}), func() {
				if c.Snapshot().Install.Phase == "done" {
					select {
					case <-done:
					default:
						close(done)
					}
				}
			})

			act := c.Actions(c.Snapshot().Owner)
			act.OpenInstall()
			act.EditInstall(tc.spec, "")
			act.Inspect()

			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatalf("install did not complete natively, state=%+v", c.Snapshot().Install)
			}
			mu.Lock()
			gotCalls := append([]string(nil), calls...)
			mu.Unlock()
			found := false
			for _, m := range gotCalls {
				if m == tc.wantRPC {
					found = true
				}
				if m == "pluginManager/installBundle" {
					t.Fatalf("must not invoke installBundle for %s: calls=%v", tc.name, gotCalls)
				}
			}
			if !found {
				t.Fatalf("expected %s RPC, calls=%v", tc.wantRPC, gotCalls)
			}
		})
	}
}

func TestRemoteMCPNotABundleReplacesBlanketRefusalWithGuidance(t *testing.T) {
	done := make(chan struct{})
	c := New()
	c.Attach(fixtureCall(func(_ context.Context, method string, _, out any) error {
		switch method {
		case "pluginManager/registries":
			return reply(out, RegistryInfo{})
		case "pluginManager/inspect":
			no := false
			return reply(out, Inspection{Status: "refused", Problem: "not-a-bundle", Name: "mcp-server-filesystem", Description: "Filesystem MCP server", Bundle: &no, Reason: "mcp-server-filesystem declares no dsh.bundle"})
		default:
			return nil
		}
	}), func() {
		s := c.Snapshot().Install
		if s.Phase == "idle" && s.Inspection != nil {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	act := c.Actions(c.Snapshot().Owner)
	act.OpenInstall()
	act.EditInstall("mcp-server-filesystem", "")
	act.Inspect()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("inspect did not settle")
	}
	inst := c.Snapshot().Install
	if inst.CapabilityKind != CapabilityMCP || strings.Contains(inst.Error, "The package does not declare a plugin bundle.") {
		t.Fatalf("expected MCP capability guidance instead of blanket bundle refusal: %+v", inst)
	}
	if inst.Guidance == "" || !strings.Contains(inst.Error, "MCP") {
		t.Fatalf("missing MCP guidance: %+v", inst)
	}
}

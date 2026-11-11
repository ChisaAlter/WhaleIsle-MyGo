package harness

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func dshbotRoomFixture(t *testing.T) (PrepareOptions, DesktopPlugin) {
	t.Helper()
	root := t.TempDir()
	o := PrepareOptions{VendorDir: filepath.Join(root, "vendor"), ProfileDir: filepath.Join(root, "profile"), Dshbot: true, RemoteSet: true}
	o.HarnessRoot = filepath.Join(o.VendorDir, "deepseek-harness")
	files := map[string]string{
		"deepseek-harness/packages/preset/agent-preset-registry/package.json": `{"name":"@deepseek-ai/dsh-agent-preset-registry","version":"0.2.1-rc.2"}`,
		"dshbot/package.json":                         `{"name":"dshbot","version":"0.2.0","type":"module","main":"lib/index.js","engines":{"node":">=24"},"peerDependencies":{"@deepseek-ai/dsh-agent":"0.2.1-rc.2"},"exports":{".":"./lib/index.js"},"dsh":{"bundle":{"patch":"./cordis.patch.yml"},"client":{"platform":"web","inject":["original-client-dependency"]}}}`,
		"dshbot/presets/dshbot-room/preset.yml":       "description: Original room metadata\norder: 80\nhidden: true\n",
		"dshbot/presets/dshbot-room/agent.cordis.yml": "- id: persona\n  name: '@deepseek-ai/dsh-persona'\n  config: {includeRuntimeContext: false, complete: true, prefix: ''}\n- id: ask-participant\n  name: dshbot/ask-participant\n",
		"dshbot/client/client.js":                     "/* original compiled client bytes */\n",
		"dshbot/lib/index.js":                         `export const name='dsh-bot';export const inject=['tools'];export const Config={original:true};export function apply(ctx,config){ctx.applied=config;ctx.toolsRegistered=true;return 'original-result';}`,
	}
	for name, body := range files {
		filename := filepath.Join(o.VendorDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return o, optionalPlugins(o)[0]
}

func TestDshbotRoomCompatibilityPreservesCompositionAndAdmission(t *testing.T) {
	o, p := dshbotRoomFixture(t)
	source := filepath.Join(o.VendorDir, "dshbot", "package.json")
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := writeDshbotRoomOverlay(o, p)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(patch)
	module, err := os.ReadFile(filepath.Join(dir, "room-adapter.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"id":"dshbot-room"`, `"hidden":true`, `"order":80`, `"complete":true`, `"includeRuntimeContext":false`, `"prefix":""`, `"name":"dshbot/ask-participant"`} {
		if !strings.Contains(string(module), want) {
			t.Fatalf("original room definition was changed: missing %s", want)
		}
	}
	var original, adapter map[string]json.RawMessage
	if err = json.Unmarshal(before, &original); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &adapter); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "version", "engines", "peerDependencies"} {
		var a, b any
		_ = json.Unmarshal(original[key], &a)
		_ = json.Unmarshal(adapter[key], &b)
		x, _ := json.Marshal(a)
		y, _ := json.Marshal(b)
		if !bytes.Equal(x, y) {
			t.Fatalf("original compatibility field changed: %s", key)
		}
	}
	var dsh map[string]json.RawMessage
	_ = json.Unmarshal(adapter["dsh"], &dsh)
	if len(dsh["bundle"]) != 0 || len(dsh["client"]) == 0 {
		t.Fatal("adapter changes bundle activation or drops client metadata")
	}
	client, err := os.ReadFile(filepath.Join(dir, "client", "client.js"))
	if err != nil || string(client) != "/* original compiled client bytes */\n" {
		t.Fatal("compiled client bytes changed", err)
	}
	if after, _ := os.ReadFile(source); !bytes.Equal(before, after) {
		t.Fatal("original vendor manifest changed")
	}
	data, _ := os.ReadFile(patch)
	if strings.Count(string(data), "    - id:") != 1 || !strings.Contains(string(data), "id: dsh-bot") {
		t.Fatal("compatibility added another activation entry")
	}
}

func TestDshbotRoomCompatibilityOnlyForSelectedRc2Plugin(t *testing.T) {
	o, p := dshbotRoomFixture(t)
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im"} {
		dir := filepath.Join(o.VendorDir, name)
		_ = os.MkdirAll(dir, 0700)
		_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0600)
	}
	o.Dshbot = false
	patches, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(patches, "\n"), "desktop-dshbot.patch.yml") {
		t.Fatal("disabled dshbot was mounted")
	}
	if _, err = os.Stat(filepath.Join(o.ProfileDir, "desktop-plugins", "dshbot")); !os.IsNotExist(err) {
		t.Fatal("disabled plugin generated a registrar")
	}
	_ = os.WriteFile(filepath.Join(o.HarnessRoot, "packages", "preset", "agent-preset-registry", "package.json"), []byte(`{"name":"@deepseek-ai/dsh-agent-preset-registry","version":"0.2.1-alpha.1"}`), 0600)
	patch, err := writeDshbotRoomOverlay(o, p)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(patch)
	if !strings.Contains(string(data), `name: "dshbot"`) {
		t.Fatal("non-rc.2 profile was rewritten")
	}
}

func TestDshbotRoomCompatibilityRc4KeepsHiddenComposition(t *testing.T) {
	o, p := dshbotRoomFixture(t)
	if err := os.WriteFile(filepath.Join(o.HarnessRoot, "packages", "preset", "agent-preset-registry", "package.json"), []byte(`{"name":"@deepseek-ai/dsh-agent-preset-registry","version":"0.2.1-rc.4"}`), 0600); err != nil {
		t.Fatal(err)
	}
	patch, err := writeDshbotRoomOverlay(o, p)
	if err != nil {
		t.Fatal(err)
	}
	module, err := os.ReadFile(filepath.Join(filepath.Dir(patch), "room-adapter.mjs"))
	if err != nil || !strings.Contains(string(module), `"hidden":true`) || !strings.Contains(string(module), `"name":"dshbot/ask-participant"`) {
		t.Fatalf("rc.4 lost its hidden room registrar: %v", err)
	}
}

func TestDshbotRoomAdapterAwaitsRegistrationAndDisposes(t *testing.T) {
	node := os.Getenv("WHALEISLE_DSHBOT_ROOM_TEST_NODE")
	if node == "" {
		t.Skip("explicit bundled Node required")
	}
	o, p := dshbotRoomFixture(t)
	patch, err := writeDshbotRoomOverlay(o, p)
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(t.TempDir(), "adapter-check.mjs")
	if err = os.WriteFile(script, []byte(roomAdapterCheck), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, script, filepath.Join(filepath.Dir(patch), "room-adapter.mjs"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("adapter lifecycle failed: %v: %s", err, output)
	}
}

const roomAdapterCheck = `
import assert from 'node:assert/strict';import {pathToFileURL} from 'node:url';
const adapter=await import(pathToFileURL(process.argv[2]));
assert.equal(adapter.name,'dsh-bot');assert.equal(adapter.Config.original,true);assert.deepEqual(adapter.inject,['tools','agentPresets']);
const disposed=[];let release;const barrier=new Promise(resolve=>release=resolve);let registered=false,unregistered=0,finished=false;
const ctx={effect(fn){disposed.push(fn())},agentPresets:{async register(definition){assert.equal(ctx.toolsRegistered,true);assert.equal(definition.id,'dshbot-room');assert.equal(definition.hidden,true);assert.equal(definition.plugins[0].config.complete,true);registered=true;await barrier;return ()=>{unregistered++}}}};
const config={maxSpeaks:1,maxRounds:1};const active=adapter.apply(ctx,config).then(value=>{finished=true;return value});
await Promise.resolve();assert.equal(registered,true);assert.equal(finished,false);assert.equal(ctx.applied,config);release();assert.equal(await active,'original-result');assert.equal(disposed.length,1);await disposed[0]();assert.equal(unregistered,1);
`

package pluginui

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"whaleisle/internal/processutil"
)

func TestBundledRegistrantIsKnownToThePageAndTheNativeFallbacks(t *testing.T) {
	if !strings.Contains(string(publicSlotsSource), "'"+BundledRegistrant+"'") {
		t.Fatalf("public_slots.js must report bundled entries as %q", BundledRegistrant)
	}
	if !nativeDefaultRegistrant(BundledRegistrant) || nativeDefaultRegistrant("some-plugin") {
		t.Fatal("only the bundled registrant is a native default")
	}
}

// Harness bundles its own UI under one minified registrant; plugins keep theirs.
func TestPageRecoversTheBundledRegistrantFromTheSettingsSections(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(string(publicSlotsSource))
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const context=vm.createContext({assert});vm.runInContext(` + string(source) + `,context);
vm.runInContext('(' + function(){
 const stock=['general','appearance','interface','models','plugins','skills','mcp','agent-presets','usage-stats','about'];
 const section=(id,registrant)=>({registrant,options:{id}});
 const entriesWith=rows=>slot=>slot==='settings.section'?rows:[];
 const bundled=stock.map(id=>section(id,'KC'));

 assert.equal(whaleBundledRegistrant(entriesWith(bundled)),'KC');
 assert.equal(whaleBundledRegistrant(entriesWith([...bundled,section('fixture','whaleisle-fixture-plugin')])),'KC');
 assert.equal(whaleStockReady(entriesWith(bundled)),true);

 // A plugin cannot pose as the bundle with a couple of look-alike sections.
 assert.equal(whaleBundledRegistrant(entriesWith([section('general','evil'),section('about','evil')])),'');
 // Before the bundle registered anything nothing is stock, and nothing is ready.
 assert.equal(whaleBundledRegistrant(entriesWith([])),'');
 assert.equal(whaleStockReady(entriesWith([section('fixture','whaleisle-fixture-plugin')])),false);
 // Unminified builds keep package names: nothing to recover, the name lists apply.
 const named=stock.map(id=>section(id,'@deepseek-ai/dsh-client-ui-settings-'+id));
 assert.equal(whaleBundledRegistrant(entriesWith(named)),'');
 assert.equal(whaleStockReady(entriesWith(named)),true);

 assert.equal(whaleReportedRegistrant({registrant:'KC'},entriesWith(bundled)),'whaleisle:bundled');
 assert.equal(whaleReportedRegistrant({registrant:'dsh-whale'},entriesWith(bundled)),'dsh-whale');
 assert.equal(whaleReportedRegistrant({registrant:''},entriesWith(bundled)),'');
 // The conversation package names two of its own fibers; they are stock too, with or without the bundle's registrant.
 assert.equal(whaleReportedRegistrant({registrant:'conversation-todo-dock'},entriesWith(bundled)),'whaleisle:bundled');
 assert.equal(whaleReportedRegistrant({registrant:'conversation-queue-dock'},entriesWith([])),'whaleisle:bundled');
 assert.equal(whaleReportedRegistrant({registrant:'conversation-todo-dock-fork'},entriesWith(bundled)),'conversation-todo-dock-fork');
}.toString() + ')()',context);`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bundled registrant recovery: %v\n%s", err, out)
	}
}

// The bundle's own settings.plugins.tab ("all", the inventory) used to be hosted
// like a plugin tab: it hid the native inventory behind "Loading plugin settings".
func TestStockPluginsTabIsNotHostedButAPluginsTabIs(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(string(navigationSource))
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const context=vm.createContext({assert});vm.runInContext(` + string(source) + `,context);
vm.runInContext('(' + function(){
 const none={stock:false,nativeDefault:false,nativeSetting:false};
 assert.equal(whaleHostsEntry('settings.plugins.tab',{...none,stock:true}),false);
 assert.equal(whaleHostsEntry('settings.plugins.tab',none),true);
 // A tab is hosted even when its registrant is a native default: only the bundle is stock.
 assert.equal(whaleHostsEntry('settings.plugins.tab',{...none,nativeDefault:true}),true);
 assert.equal(whaleHostsEntry('settings.section',{...none,stock:true}),false);
 assert.equal(whaleHostsEntry('settings.section',{...none,nativeSetting:true}),false);
 assert.equal(whaleHostsEntry('settings.section',none),true);
 assert.equal(whaleHostsEntry('main',{...none,nativeDefault:true}),false);
 assert.equal(whaleHostsEntry('main',{...none,nativeSetting:true}),true);
}.toString() + ')()',context);`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hosted entries: %v\n%s", err, out)
	}
}

// The stock composer is the fallback of the composer chain; its input seats are
// additive, so a plugin's entry there must not be dropped as part of the fallback,
// and the native seat keeps its own controls next to it. JS and Go name the same seats.
func TestComposerInputSeatsAreAdditiveInPageAndNative(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(string(sessionSource))
	if err != nil {
		t.Fatal(err)
	}
	slots := []string{"conversation.composer", "conversation.composer.bar", "conversation.composer.dock", "conversation.input.dock", "conversation.input.overlay", "conversation.input.left", "conversation.input.right", "conversation.input.activity", "conversation.input.plan", "conversation.input.permission", "conversation.input.model", "conversation.input.managed", "conversation.input.attachments", "conversation.chat.node", "conversation.session.header.utilities"}
	list, err := json.Marshal(slots)
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm'),assert=require('node:assert/strict');
const context=vm.createContext({assert,slots:` + string(list) + `});vm.runInContext(` + string(source) + `,context);
console.log(JSON.stringify(vm.runInContext('slots.map(slot=>whaleAdditiveComposerSlot(slot))',context)));`
	cmd := exec.Command(node, "-e", script)
	processutil.Background(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("additive composer slots: %v\n%s", err, out)
	}
	var fromPage []bool
	if err := json.Unmarshal(out, &fromPage); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for i, slot := range slots {
		if fromPage[i] != AdditiveNativeSlot(slot) {
			t.Errorf("%s: page says additive=%v, native says %v", slot, fromPage[i], AdditiveNativeSlot(slot))
		}
	}
	if AdditiveNativeSlot("conversation.composer") || AdditiveNativeSlot("conversation.composer.bar") || AdditiveNativeSlot("conversation.chat.node") {
		t.Error("chain and message slots keep replace semantics")
	}
	if !AdditiveNativeSlot("conversation.input.left") || !AdditiveNativeSlot("conversation.composer.dock") {
		t.Error("input seats and the composer dock are additive")
	}
}

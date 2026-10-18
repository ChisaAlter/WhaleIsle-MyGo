package startuprec

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func fixtureInventory() []Plugin {
	return []Plugin{{Name: "@fixture/broken", Enabled: true, Mutable: true}, {Name: "@fixture/healthy", Enabled: true, Mutable: true}, {Name: "@fixture/locked", Enabled: true, Mutable: false}}
}
func TestForensicsMapsOnlyCurrentVerifiedPlugin(t *testing.T) {
	d := Diagnose([]string{"failed to apply loader entry fixture (@fixture/broken/client) token=syntheticToken"}, "", fixtureInventory(), true)
	if !reflect.DeepEqual(d.RepairNames, []string{"@fixture/broken"}) {
		t.Fatalf("wrong repair targets: %+v", d)
	}
	if strings.Contains(d.Evidence[0].Line, "syntheticToken") {
		t.Fatal("evidence contains credential")
	}
	if !d.CanRepair || d.Receipt == "" {
		t.Fatal("verified target not repairable")
	}
	changed := fixtureInventory()
	changed[0].Enabled = false
	if Diagnose([]string{"failed to apply loader entry fixture (@fixture/broken/client)"}, "", changed, true).Receipt == d.Receipt {
		t.Fatal("inventory change did not expire receipt")
	}
}
func TestForensicsGenericFailureSuppressesPluginBlame(t *testing.T) {
	for _, reason := range []string{"heap out of memory", "listen EACCES", "EADDRINUSE", "cannot find node", "domain 'session_projcache': does not match its schema"} {
		d := Diagnose([]string{"Cannot find package '@fixture/broken'"}, reason, fixtureInventory(), true)
		if d.Kind == "plugins" || d.CanRepair || len(d.RepairNames) != 0 {
			t.Fatalf("generic fault disabled plugin: %+v", d)
		}
	}
}
func TestForensicsUnknownReadonlyAndRuntimeAreNotRepairTargets(t *testing.T) {
	for _, name := range []string{"@fixture/missing", "@fixture/locked", "@deepseek-ai/dsh-client-ui-files"} {
		d := Diagnose([]string{"Cannot find package '" + name + "'"}, "", fixtureInventory(), true)
		if d.CanRepair || len(d.RepairNames) != 0 {
			t.Fatalf("unsafe target %s: %+v", name, d)
		}
		if name == "@deepseek-ai/dsh-client-ui-files" && d.Kind != "runtime" {
			t.Fatal("builtin damage not recognized")
		}
	}
	d := Diagnose([]string{"Cannot find package '@fixture/broken'"}, "", fixtureInventory(), false)
	if d.CanRepair {
		t.Fatal("offline stale inventory is repairable")
	}
}
func TestForensicsAuditOnlyWithinHeader(t *testing.T) {
	bare := Diagnose([]string{"@fixture/broken: failed"}, "", fixtureInventory(), true)
	if len(bare.Evidence) > 0 {
		t.Fatal("ordinary status treated as failure")
	}
	audit := Diagnose([]string{"web boot: 1 entry did not activate", "@fixture/broken: failed"}, "", fixtureInventory(), true)
	if len(audit.RepairNames) != 1 {
		t.Fatal("boot audit ignored")
	}
}
func TestTargetedRepairStopsOnFailureAndNeverRestarts(t *testing.T) {
	var calls []string
	actions := RepairActions{Check: func() error { return nil }, Disable: func(n string) error { calls = append(calls, n); return errors.New("synthetic failure") }, ClearRecovery: func() error { calls = append(calls, "clear"); return nil }, Restart: func() error { calls = append(calls, "restart"); return nil }}
	done, err := Repair([]string{"first", "second"}, actions)
	if err == nil || len(done) != 0 || !reflect.DeepEqual(calls, []string{"first"}) {
		t.Fatalf("unsafe repair: %v %v %v", done, calls, err)
	}
}
func TestTargetedRepairRevalidatesAndPreservesRecoveryOnChange(t *testing.T) {
	checked := 0
	var calls []string
	actions := RepairActions{Check: func() error {
		checked++
		if checked == 3 {
			return errors.New("changed")
		}
		return nil
	}, Disable: func(n string) error { calls = append(calls, n); return nil }, ClearRecovery: func() error { calls = append(calls, "clear"); return nil }, Restart: func() error { calls = append(calls, "restart"); return nil }}
	done, err := Repair([]string{"first", "second"}, actions)
	if err == nil || !reflect.DeepEqual(done, []string{"first", "second"}) || !reflect.DeepEqual(calls, done) {
		t.Fatalf("unsafe completion: %v %v %v", done, calls, err)
	}
}
func TestTargetedRepairClearsOnlyAfterAllSavedThenRestarts(t *testing.T) {
	var calls []string
	actions := RepairActions{Check: func() error { return nil }, Disable: func(n string) error { calls = append(calls, n); return nil }, ClearRecovery: func() error { calls = append(calls, "clear"); return nil }, Restart: func() error { calls = append(calls, "restart"); return nil }}
	done, err := Repair([]string{"first", "second"}, actions)
	if err != nil || len(done) != 2 || !reflect.DeepEqual(calls, []string{"first", "second", "clear", "restart"}) {
		t.Fatalf("bad order: %v %v", calls, err)
	}
}

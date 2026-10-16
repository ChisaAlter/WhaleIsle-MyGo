package main

import (
	"os"
	"strings"
	"testing"
)

func TestBootRecoveryControlIsInMountedDialogue(t *testing.T) {
	body, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	mounted := strings.Index(source, "// Body under the title bar must Grow")
	fallback := strings.Index(source, "boot.View(c, boot.Model{")
	if mounted < 0 || fallback <= mounted {
		t.Fatal("dialogue/fallback boundaries missing")
	}
	dialogue := source[mounted:fallback]
	control := strings.Index(dialogue, "boot.RecoveryControl(c, boot.Model{")
	if control < 0 {
		t.Fatal("recovery control only exists in unreachable fallback")
	}
	wiring := dialogue[control:]
	end := strings.Index(wiring, "connBanner :=")
	if end < 0 {
		t.Fatal("recovery control is not above dialogue content")
	}
	wiring = wiring[:end]
	for _, required := range []string{"Status: st", "sup.CancelRecovery()", "state.restartUnlessImport(sup)"} {
		if !strings.Contains(wiring, required) {
			t.Fatalf("missing mounted recovery wiring %q", required)
		}
	}
}

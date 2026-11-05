package phoneremote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOlderSelectedPhoneComponentKeepsQRContract(t *testing.T) {
	user, vendor := t.TempDir(), t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	component := filepath.Join(user, "components", "dshd-remote")
	root := filepath.Join(component, "versions", "older-fixture")
	write(filepath.Join(component, "installed.json"), `{"version":"older-fixture"}`)
	for _, rel := range append(requiredPhoneResources(false), "node.exe") {
		write(filepath.Join(root, filepath.FromSlash(rel)), "inert fixture, never executed")
	}
	if got, err := ResolveResources(user, vendor); err != nil || got.Module != filepath.Join(root, "src", "main", "dshd-remote.js") {
		t.Fatalf("older component rejected: %v %v", got, err)
	}
	newUser := t.TempDir()
	bundled := filepath.Join(vendor, "dshd-remote")
	for _, rel := range append(requiredPhoneResources(false), "node.exe") {
		write(filepath.Join(bundled, filepath.FromSlash(rel)), "inert fixture, never executed")
	}
	if _, err := ResolveResources(newUser, vendor); err == nil || !strings.Contains(err.Error(), "relay-pairing-link-store.js") {
		t.Fatalf("new bundle accepted an incomplete link dependency: %v", err)
	}
	child := &deviceControlChild{writes: make(chan []byte, 1)}
	runtime := NewRuntime()
	runtime.child = child
	runtime.state = RuntimeState{Listening: true, DevicesLoaded: true, PairingURL: "fictional-short-offer"}
	if runtime.DeviceAction("revoke-link", "", "") == nil {
		t.Fatal("old runtime accepted an unsupported link mutation")
	}
	if runtime.Snapshot().DeviceAction.Busy {
		t.Fatal("unsupported new action locked the old QR/device workflow")
	}
}

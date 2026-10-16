package main

import (
	"os"
	"testing"
)

func TestManagedLauncherStopBindsExecutable(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if !managedLauncherStop([]string{"--managed-launcher-stop", exe}) {
		t.Fatal("matching installation rejected")
	}
	for _, args := range [][]string{nil, {"--managed-launcher-stop"}, {"--managed-launcher-stop", "relative.exe"}, {"--managed-launcher-stop", exe, "extra"}, {"--other", exe}} {
		if managedLauncherStop(args) {
			t.Fatalf("invalid command accepted: %v", args)
		}
	}
}

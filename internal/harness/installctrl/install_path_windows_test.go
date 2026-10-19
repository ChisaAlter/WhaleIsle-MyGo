//go:build windows

package installctrl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPnpmSelectionPreservesPathDirectoryPrecedence(t *testing.T) {
	for _, names := range [][2]string{{"pnpm.exe", "pnpm.cmd"}, {"pnpm.cmd", "pnpm.exe"}} {
		t.Run(names[0], func(t *testing.T) {
			first, later := t.TempDir(), t.TempDir()
			for i, directory := range []string{first, later} {
				if err := os.WriteFile(filepath.Join(directory, names[i]), []byte("lookup fixture; never execute"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", first+string(os.PathListSeparator)+later)
			t.Setenv("PATHEXT", ".EXE;.CMD")
			if actual := findPnpmBin(); actual != filepath.Join(first, names[0]) {
				t.Fatalf("selected %q instead of the first PATH directory", actual)
			}
			if actual := prependPnpmToPath(InstallConfig{}); actual != first {
				t.Fatalf("child PATH would promote a later pnpm directory: %q", actual)
			}
		})
	}
}

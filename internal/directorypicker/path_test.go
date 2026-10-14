package directorypicker

import "testing"

func TestSelectablePath(t *testing.T) {
	for _, path := range []string{"", "  ", `\\.\dsh-computer`, `\\.\DSH-COMPUTER\`, `//./dsh-computer/`} {
		if SelectablePath(path) {
			t.Errorf("virtual or empty path is selectable: %q", path)
		}
	}
	for _, path := range []string{`C:\`, `D:\projects`, `\\server\share`, "/home/project", "/"} {
		if !SelectablePath(path) {
			t.Errorf("filesystem path is blocked: %q", path)
		}
	}
}

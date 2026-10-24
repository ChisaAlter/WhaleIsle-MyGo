package webview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSanitizeDownloadNameMatchesNative ports the table from
// reference/electron/src/main/download-path.js:1-28 so a page cannot pick a name
// Windows rejects or a DOS device name.
func TestSanitizeDownloadNameMatchesNative(t *testing.T) {
	cases := map[string]string{
		"report.pdf":              "report.pdf",
		"  spaced.txt  ":          "spaced.txt",
		"a<b>c:d\"e|f?g*h.txt":    "a_b_c_d_e_f_g_h.txt",
		"dir/sub/file.txt":        "file.txt",
		`C:\tmp\win\file.txt`:     "file.txt",
		"trailing...":             "trailing",
		"trailing. . .":           "trailing",
		"":                        "download",
		".":                       "download",
		"..":                      "download",
		"con":                     "_con",
		"CON.txt":                 "_CON.txt",
		"nul.log":                 "_nul.log",
		"com1":                    "_com1",
		"lpt9.dat":                "_lpt9.dat",
		"console.txt":             "console.txt",
		"ctrl\x01name.txt":        "ctrl_name.txt",
		"emoji-🐋.txt":            "emoji-🐋.txt",
	}
	for in, want := range cases {
		if got := SanitizeDownloadName(in); got != want {
			t.Errorf("SanitizeDownloadName(%q) = %q, want %q", in, got, want)
		}
	}
	// The 180-character cap counts runes, not bytes, and never leaves a trailing
	// dot or an empty name.
	long := strings.Repeat("鲸", 200) + ".txt"
	got := SanitizeDownloadName(long)
	if n := len([]rune(got)); n != 180 {
		t.Fatalf("long name has %d runes, want 180", n)
	}
	if strings.HasSuffix(got, ".") || strings.HasSuffix(got, " ") {
		t.Fatalf("long name ends with %q", got)
	}
}

// TestUniqueDownloadPathNumbersTakenNames mirrors the engine's dedupe.
func TestUniqueDownloadPathNumbersTakenNames(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "pet.png")
	if err := os.WriteFile(first, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := UniqueDownloadPath(dir, "pet.png"); got != filepath.Join(dir, "pet (2).png") {
		t.Fatalf("taken name gave %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "pet (2).png"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := UniqueDownloadPath(dir, "pet.png"); got != filepath.Join(dir, "pet (3).png") {
		t.Fatalf("second taken name gave %q", got)
	}
	if got := UniqueDownloadPath(dir, "free.png"); got != filepath.Join(dir, "free.png") {
		t.Fatalf("free name gave %q", got)
	}
}

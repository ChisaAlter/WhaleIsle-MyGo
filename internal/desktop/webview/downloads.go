package webview

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo"
)

// Download names follow the native shell's download-path.js: a leaf-only name,
// control characters and Windows-reserved characters replaced, trailing dots and
// spaces trimmed, Windows device names prefixed, and at most 180 characters. The
// engine's own default (filepath.Base) keeps characters Windows rejects, so a
// harness page download could fail or land under a name the OS rewrites.
const (
	defaultDownloadName    = "download"
	maxDownloadNameChars   = 180
)

// windowsDeviceNames are the DOS device names Windows refuses as plain files.
var windowsDeviceNames = []string{
	"con", "prn", "aux", "nul", "conin$", "conout$",
	"com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8", "com9",
	"lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9",
}

// SanitizeDownloadName ports sanitizeDownloadFilename from
// reference/electron/src/main/download-path.js:1-28.
func SanitizeDownloadName(suggested string) string {
	leaf := suggested
	leaf = strings.ReplaceAll(leaf, `\`, "/")
	if i := strings.LastIndexByte(leaf, '/'); i >= 0 {
		leaf = leaf[i+1:]
	}
	var b strings.Builder
	for _, r := range leaf {
		switch {
		case r < 0x20 || r == 0x7f:
			b.WriteByte('_')
		case strings.ContainsRune(`<>:"/\|?*`, r):
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	name := strings.TrimSpace(b.String())
	name = strings.TrimRight(name, ". ")
	if name == "" || name == "." || name == ".." {
		name = defaultDownloadName
	}
	lower := strings.ToLower(name)
	base := lower
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	for _, device := range windowsDeviceNames {
		if base == device {
			name = "_" + name
			break
		}
	}
	if utf8.RuneCountInString(name) > maxDownloadNameChars {
		runes := []rune(name)
		name = strings.TrimRight(string(runes[:maxDownloadNameChars]), ". ")
		if name == "" {
			name = defaultDownloadName
		}
	}
	return name
}

// UniqueDownloadPath numbers a path like "name (2).ext" when the name is taken,
// matching the engine's own dedupe so a sanitized name keeps that behaviour.
func UniqueDownloadPath(dir, name string) string {
	path := filepath.Join(dir, name)
	if _, err := os.Lstat(path); err != nil {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		candidate := base + " (" + strconv.Itoa(i) + ")" + ext
		if _, err := os.Lstat(candidate); err != nil {
			return candidate
		}
	}
}

// saveDownload gives the page's download a sanitized file name in the user's
// Downloads folder, replacing whatever the engine suggested.
func saveDownload(e *mygo.DownloadEvent) {
	name := SanitizeDownloadName(e.SuggestedName)
	dir, err := mygo.App.Path(mygo.PathDownloads)
	if err != nil || dir == "" {
		if e.Path == "" {
			e.PreventDefault()
			return
		}
		dir = filepath.Dir(e.Path)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.PreventDefault()
		return
	}
	e.Path = UniqueDownloadPath(dir, name)
}

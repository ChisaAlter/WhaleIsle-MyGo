package workspacefs

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DocKind classifies office/document files that need external or WebView preview.
type DocKind string

const (
	DocNone DocKind = ""
	DocPDF  DocKind = "pdf"
	DocXLSX DocKind = "xlsx"
	DocOther DocKind = "other"
)

// ClassifyDoc returns a document kind for preview routing.
func ClassifyDoc(path string) DocKind {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return DocPDF
	case ".xlsx", ".xls":
		return DocXLSX
	case ".docx", ".doc", ".pptx", ".ppt":
		return DocOther
	default:
		return DocNone
	}
}

// FileURL builds a file:// URL for local absolute paths (WebView2 / browser).
func FileURL(absPath string) string {
	absPath = filepath.Clean(absPath)
	if runtime.GOOS == "windows" {
		p := filepath.ToSlash(absPath)
		if len(p) >= 2 && p[1] == ':' {
			return "file:///" + p
		}
		return "file://" + p
	}
	return "file://" + absPath
}

// OpenExternal opens a local file with the OS default handler.
func OpenExternal(absPath string) error {
	absPath = filepath.Clean(absPath)
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", absPath).Start()
	case "darwin":
		return exec.Command("open", absPath).Start()
	default:
		return exec.Command("xdg-open", absPath).Start()
	}
}

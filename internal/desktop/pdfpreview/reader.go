package pdfpreview

import (
	"embed"
	"net/http"
	"strings"
)

// Only immutable, product-owned reader code is executable. The PDF is input
// bytes to PDF.js, never HTML, an annotation script, or a native PDF plugin.
//
//go:embed assets/* reader.html reader.js reader.css
var readerFiles embed.FS

func (d *Document) ViewerURL() string {
	return d.origin + strings.TrimSuffix(d.route, "document.pdf") + "reader.html"
}

func (d *Document) readerResource(path string) (string, string, bool) {
	base := strings.TrimSuffix(d.route, "document.pdf")
	if !strings.HasPrefix(path, base) {
		return "", "", false
	}
	name := strings.TrimPrefix(path, base)
	switch name {
	case "reader.html":
		return name, "text/html; charset=utf-8", true
	case "reader.js":
		return name, "text/javascript; charset=utf-8", true
	case "reader.css":
		return name, "text/css; charset=utf-8", true
	case "pdf.mjs", "pdf.worker.mjs", "pdf_viewer.mjs":
		return "assets/" + name, "text/javascript; charset=utf-8", true
	case "pdf_viewer.css":
		return "assets/" + name, "text/css; charset=utf-8", true
	case "assets.json":
		return "assets/" + name, "application/json", true
	}
	return "", "", false
}

func (d *Document) serveReader(w http.ResponseWriter, r *http.Request) bool {
	file, kind, ok := d.readerResource(r.URL.Path)
	if !ok {
		return false
	}
	data, err := readerFiles.ReadFile(file)
	if err != nil {
		http.Error(w, "Reader unavailable", http.StatusInternalServerError)
		return true
	}
	// No inline JS, eval, frames, object/plugin, remote resources, or forms.
	// wasm-unsafe-eval is only for the bundled image decoders, not JS eval.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'wasm-unsafe-eval'; worker-src 'self'; connect-src 'self'; style-src 'self' 'unsafe-inline'; img-src data: blob:; font-src blob:; object-src 'none'; frame-src 'none'; frame-ancestors 'none'; form-action 'none'; base-uri 'none'")
	w.Header().Set("Content-Type", kind)
	if r.Method != http.MethodHead {
		_, _ = w.Write(data)
	}
	return true
}

func (d *Document) isReaderPath(path string) bool { _, _, ok := d.readerResource(path); return ok }

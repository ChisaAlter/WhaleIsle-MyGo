package platform

import "strings"

// LocalPDFViewerAllowed authorizes only the fixed product reader entry point.
// Reuse the canonical loopback/token parser without admitting user HTML.
func LocalPDFViewerAllowed(expected, target string) bool {
	if !strings.HasSuffix(expected, "/reader.html") {
		return false
	}
	base, fragment, hasFragment := strings.Cut(target, "#")
	if base != expected || len(fragment) > 1024 {
		return false
	}
	canonical := strings.TrimSuffix(expected, "reader.html") + "document.html"
	if hasFragment {
		return LocalHTMLAllowed(canonical, canonical+"#"+fragment)
	}
	return LocalHTMLAllowed(canonical, canonical)
}

// Exact immutable resource inventory. This is never an origin/directory grant.
func LocalPDFViewerResourceAllowed(expected, target string) bool {
	if !LocalPDFViewerAllowed(expected, expected) {
		return false
	}
	base := strings.TrimSuffix(expected, "reader.html")
	for _, name := range []string{"reader.html", "reader.js", "reader.css", "pdf.mjs", "pdf.worker.mjs", "pdf_viewer.mjs", "pdf_viewer.css", "assets.json", "document.pdf"} {
		if target == base+name {
			return true
		}
	}
	return false
}

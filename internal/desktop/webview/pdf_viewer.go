package webview

import "strings"

func validLocalPDFViewerURL(raw string) bool {
	return strings.HasSuffix(raw, "/reader.html") && validLocalHTMLURL(strings.TrimSuffix(raw, "reader.html")+"document.html")
}

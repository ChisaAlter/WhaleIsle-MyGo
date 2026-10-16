package pdfpreview

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestProductReaderExactResourcesAndLifetime(t *testing.T) {
	var active atomic.Bool
	active.Store(true)
	d, err := OpenBytes("test.pdf", []byte("%PDF-1.7\n%%EOF\n"), active.Load)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	base := strings.TrimSuffix(d.ViewerURL(), "reader.html")
	for _, name := range []string{"reader.html", "reader.js", "reader.css", "pdf.mjs", "pdf.worker.mjs", "pdf_viewer.mjs", "pdf_viewer.css", "assets.json", "document.pdf"} {
		response, err := http.Get(base + name)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != 200 {
			t.Fatalf("%s: %d", name, response.StatusCode)
		}
		if response.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("MIME sniffing enabled")
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if len(body) == 0 {
			t.Fatal("empty resource", name)
		}
		if name == "reader.html" && (!strings.Contains(response.Header.Get("Content-Security-Policy"), "object-src 'none'") || strings.Contains(string(body), "<iframe")) {
			t.Fatal("reader is not constrained")
		}
	}
	for _, name := range []string{"../reader.js", "pdf.sandbox.mjs", "quickjs-eval.js", "other.pdf", "reader.js?x=1", "assets/LICENSE", "%72eader.js"} {
		response, err := http.Get(base + name)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode == 200 {
			t.Fatal("unexpected resource", name)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, d.ViewerURL(), strings.NewReader("x"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 405 {
		t.Fatal("reader accepted write")
	}
	req, _ = http.NewRequest(http.MethodGet, d.ViewerURL(), nil)
	req.Header.Set("Origin", "https://evil.test")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatal("cross origin accepted")
	}
	active.Store(false)
	if d.Valid() {
		t.Fatal("stale owner still valid")
	}
}

func TestReaderDoesNotInstallPDFScriptsOrHostBridge(t *testing.T) {
	b, _ := readerFiles.ReadFile("reader.js")
	s := string(b)
	for _, token := range []string{"PDFScriptingManager", "getJSActions(", "eval(", "chrome.webview", "window.mygo", "innerHTML"} {
		if strings.Contains(s, token) {
			t.Fatal("untrusted execution path", token)
		}
	}
	for _, token := range []string{"new Worker(", "PDFWorker.create({port:worker})", "TextLayerBuilder", "enableXfa:false", "useWorkerFetch:false", "history.replaceState", "selectionchange", "scrollTop"} {
		if !strings.Contains(s, token) {
			t.Fatal("reader missing", token)
		}
	}
}

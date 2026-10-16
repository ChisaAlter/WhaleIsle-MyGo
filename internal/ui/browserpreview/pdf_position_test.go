package browserpreview

import (
	"encoding/json"
	"testing"
)

func TestPDFReadingPositionSurvivesWorkspaceAndLease(t *testing.T) {
	b := NewController(nil)
	b.OpenDocumentSource(DocumentSource{Path: "report.pdf", Kind: DocumentPDF, SessionID: "a"})
	tab := b.current()
	tab.pdfPosition = &PDFReadingPosition{Page: 7, Zoom: 150, Left: 20, Top: 300}
	b.releaseSourceDocument(tab)
	data, err := json.Marshal(b.WorkspaceSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	var saved WorkspaceSnapshot
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	restored := NewController(nil)
	restored.RestoreWorkspace(&saved)
	got := restored.current().pdfPosition
	if got == nil || *got != *tab.pdfPosition {
		t.Fatalf("lost position: %s", data)
	}
	if restored.current().pdfViewerURL("http://127.0.0.1:99/new/document.pdf") != "http://127.0.0.1:99/new/document.pdf#page=7&zoom=150,20,300" {
		t.Fatal("position not applied to new lease")
	}
	restored.current().pdfPosition.Page = 9
	if tab.pdfPosition.Page != 7 {
		t.Fatal("tabs share mutable reading state")
	}
}

func TestPDFPositionParsesOnlyCurrentLease(t *testing.T) {
	base := "http://127.0.0.1:99/token/document.pdf"
	p, ok := pdfPositionFromURL(base, base+"#page=4&zoom=125,10,20")
	if !ok || p.Page != 4 || p.Zoom != 125 || p.Left != 10 || p.Top != 20 {
		t.Fatalf("position: %+v %v", p, ok)
	}
	if _, ok = pdfPositionFromURL(base, "http://127.0.0.1:99/other/document.pdf#page=90"); ok {
		t.Fatal("other lease accepted")
	}
	if _, ok = pdfPositionFromURL(base, base+"#page=0&zoom=oops"); ok {
		t.Fatal("invalid position accepted")
	}
	if (&Tab{}).pdfViewerURL(base) != base {
		t.Fatal("new document changed")
	}
}

func TestPDFSnapshotCopiesPosition(t *testing.T) {
	b := NewController(nil)
	b.OpenDocumentSource(DocumentSource{Path: "report.pdf", Kind: DocumentPDF})
	b.current().pdfPosition = &PDFReadingPosition{Page: 3, Zoom: 125}
	snapshot := b.WorkspaceSnapshot()
	b.current().pdfPosition.Page = 8
	if snapshot.Tabs[0].PDFPosition.Page != 3 {
		t.Fatal("snapshot aliases tab reading state")
	}
	b.OpenDocumentSource(DocumentSource{Path: "second.pdf", Kind: DocumentPDF})
	if b.current().pdfPosition != nil {
		t.Fatal("another document inherited reading state")
	}
}

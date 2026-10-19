package files

import "testing"

func TestDelimitedPathsKeepDesktopTextAndBinaryWorkbookRoutes(t *testing.T) {
	for _, path := range []string{"book.xls", "book.XLSX"} {
		if !isTablePreviewPath(path) {
			t.Fatalf("%s bypassed table reader", path)
		}
	}
	for _, path := range []string{"data.csv", "data.TSV", "notes.txt"} {
		if isTablePreviewPath(path) || !isAsyncTextPath(path) {
			t.Fatalf("%s left the desktop source editor", path)
		}
	}
}

package files

import "testing"

func TestDelimitedPathsUseReadOnlyTablePreview(t *testing.T) {
	for _, path := range []string{"data.csv", "data.TSV", "book.xls", "book.xlsx"} {
		if !isTablePreviewPath(path) {
			t.Fatalf("%s bypassed table reader", path)
		}
	}
	if isTablePreviewPath("notes.txt") {
		t.Fatal("text incorrectly routed to table")
	}
}

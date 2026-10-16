package workspacefs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDelimitedWorkbookPages(t *testing.T) {
	for _, tc := range []struct{ name, data string }{
		{"data.csv", "\ufeffname,value\r\n\"hello, world\",0012\r\n\"two\nlines\",=SUM(A1)\r\nlast\r\n"},
		{"data.tsv", "\ufeffname\tvalue\nhello, world\t0012\n\"two\nlines\"\t=SUM(A1)\nlast\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.name)
			if err := os.WriteFile(path, []byte(tc.data), 0600); err != nil {
				t.Fatal(err)
			}
			page, err := ReadWorkbookPage(context.Background(), path, XLSXPageRequest{Row: 1, Column: 1, Rows: 2, Columns: 1})
			if err != nil {
				t.Fatal(err)
			}
			if page.TotalRows != 4 || page.TotalColumns != 2 || len(page.Cells) != 2 {
				t.Fatalf("wrong viewport: %+v", page)
			}
			if page.Cells[0].Address != "B2" || page.Cells[0].Value != "0012" || page.Cells[1].Value != "=SUM(A1)" {
				t.Fatalf("values not preserved: %+v", page.Cells)
			}
			if page.TSV != "0012\n=SUM(A1)\n" || page.DisplayTSV != page.TSV {
				t.Fatalf("copy: %q", page.TSV)
			}
			req := page.Request
			if err := os.WriteFile(path, []byte(tc.data+"new\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err = ReadWorkbookPage(context.Background(), path, req); err == nil {
				t.Fatal("changed snapshot accepted")
			}
		})
	}
}

func TestNamedRemoteDelimitedWorkbook(t *testing.T) {
	page, err := ReadNamedWorkbookBytesPage(context.Background(), "remote.TSV", []byte("left\tright\n"), XLSXPageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Cells) != 2 || page.Cells[1].Value != "right" {
		t.Fatalf("remote delimiter: %+v", page)
	}
	empty, err := ReadNamedWorkbookBytesPage(context.Background(), "empty.csv", nil, XLSXPageRequest{})
	if err != nil || empty.TotalRows != 0 {
		t.Fatalf("empty table: %+v %v", empty, err)
	}
}

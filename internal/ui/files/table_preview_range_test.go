package files

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"whaleisle/internal/desktop/workspacefs"
)

// One focused fixture. All files are synthetic; no Host or user document is read.
func TestTablePreviewRangeFixture(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, parts [][2]string) string {
		t.Helper()
		var body bytes.Buffer
		z := zip.NewWriter(&body)
		for _, part := range parts {
			w, e := z.Create(part[0])
			if e != nil {
				t.Fatal(e)
			}
			if _, e = w.Write([]byte(part[1])); e != nil {
				t.Fatal(e)
			}
		}
		if e := z.Close(); e != nil {
			t.Fatal(e)
		}
		file := filepath.Join(dir, name)
		if e := os.WriteFile(file, body.Bytes(), 0600); e != nil {
			t.Fatal(e)
		}
		return file
	}
	sheet := func(rows string) string { return `<worksheet><sheetData>` + rows + `</sheetData></worksheet>` }
	long := strings.Repeat("完整", 80) + ", quoted\tcell\nlast line"
	parts := [][2]string{
		{"xl/worksheets/sheet2.xml", sheet(`<row r="1"><c r="A1" t="inlineStr"><is><t>second sheet</t></is></c></row>`)},
		{"xl/workbook.xml", `<workbook xmlns:r="urn:r"><sheets><sheet name="First" r:id="r1"/><sheet name="Second" r:id="r2"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<Relationships><Relationship Id="r1" Type="urn:worksheet/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="r2" Type="urn:worksheet/worksheet" Target="worksheets/sheet2.xml"/></Relationships>`},
		{"xl/sharedStrings.xml", `<sst><si><t>` + long + `</t></si></sst>`},
		{"xl/worksheets/sheet1.xml", sheet(`<row r="1"><c r="A1" t="s"><v>0</v></c><c r="Q1" t="inlineStr"><is><t>column 17</t></is></c></row><row r="61"><c r="A61" t="inlineStr"><is><t>row 61</t></is></c><c r="Q61" t="inlineStr"><is><t>row 61 column 17</t></is></c></row>`)},
	}
	file := write("range.xlsx", parts)
	before, _ := os.ReadFile(file)
	read := func(req workspacefs.XLSXPageRequest) *workspacefs.XLSXPage {
		t.Helper()
		p, e := readTablePreview(context.Background(), file, req)
		if e != nil {
			t.Fatal(e)
		}
		return p
	}
	first := read(workspacefs.XLSXPageRequest{})
	if len(first.Sheets) != 2 || first.Sheets[0].Name != "First" || first.TotalRows != 61 || first.TotalColumns != 17 || first.Request.Rows != 60 || first.Request.Columns != 16 {
		t.Fatalf("wrong workbook/extent: %+v", first)
	}
	if len(first.Cells) != 1 || first.Cells[0].Value != long || !strings.Contains(first.Text, "last line") || strings.Contains(first.Text, "…") {
		t.Fatal("long cell silently clipped or altered")
	}
	r := csv.NewReader(strings.NewReader(first.TSV))
	r.Comma = '\t'
	records, e := r.ReadAll()
	if e != nil || records[0][0] != long {
		t.Fatal("TSV lost original cell bytes")
	}
	for _, test := range []struct {
		row, col int
		want     string
	}{{0, 16, "column 17"}, {60, 0, "row 61"}, {60, 16, "row 61 column 17"}} {
		p := read(workspacefs.XLSXPageRequest{Row: test.row, Column: test.col, Version: first.Version})
		if len(p.Cells) != 1 || p.Cells[0].Value != test.want {
			t.Fatalf("later range unreachable: %+v", p)
		}
	}
	second := read(workspacefs.XLSXPageRequest{Sheet: first.Sheets[1].ID, Version: first.Version})
	if len(second.Cells) != 1 || second.Cells[0].Value != "second sheet" {
		t.Fatal("later worksheet unreachable")
	}
	if _, e := readTablePreview(context.Background(), file, workspacefs.XLSXPageRequest{Version: "obsolete"}); e == nil {
		t.Fatal("changed file accepted")
	}
	if _, e := readTablePreview(context.Background(), file, workspacefs.XLSXPageRequest{Sheet: "../../escape"}); e == nil {
		t.Fatal("foreign worksheet accepted")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := readTablePreview(cancelled, file, workspacefs.XLSXPageRequest{}); !errors.Is(e, context.Canceled) {
		t.Fatal("cancelled read executed")
	}
	if _, e := readTablePreview(context.Background(), filepath.Join(dir, "not-read.xls"), workspacefs.XLSXPageRequest{}); e == nil || !strings.Contains(e.Error(), "Legacy XLS") {
		t.Fatal("legacy binary not rejected before read")
	}
	if !isTablePreviewPath("a.XLSX") || isTablePreviewPath("data.csv") || isTablePreviewPath("data.tsv") {
		t.Fatal("text routes expanded")
	}
	big := filepath.Join(dir, "oversize.xlsx")
	f, e := os.Create(big)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.Truncate((16 << 20) + 1); e != nil {
		t.Fatal(e)
	}
	f.Close()
	if _, e := readTablePreview(context.Background(), big, workspacefs.XLSXPageRequest{}); e == nil || !strings.Contains(e.Error(), "16 MiB") {
		t.Fatal("file bound absent")
	}
	xmlBig := write("xml-limit.xlsx", [][2]string{{"xl/worksheets/sheet1.xml", strings.Repeat("x", (8<<20)+1)}})
	if _, e := readTablePreview(context.Background(), xmlBig, workspacefs.XLSXPageRequest{}); e == nil || !strings.Contains(e.Error(), "XML limit") {
		t.Fatal("XML bound absent")
	}
	cellBig := write("cell-limit.xlsx", [][2]string{{"xl/worksheets/sheet1.xml", sheet(`<row r="1"><c r="A1" t="inlineStr"><is><t>` + strings.Repeat("x", (64<<10)+1) + `</t></is></c></row><row r="61"><c r="A61"><v>61</v></c></row>`)}})
	p, e := readTablePreview(context.Background(), cellBig, workspacefs.XLSXPageRequest{})
	if e != nil || !strings.Contains(p.Problem, "64 KiB") || p.TSV != "" || len(p.Cells) != 0 || p.TotalRows != 61 {
		t.Fatal("cell limit silent or disables navigation")
	}
	p, e = readTablePreview(context.Background(), cellBig, workspacefs.XLSXPageRequest{Row: 60, Version: p.Version})
	if e != nil || p.Problem != "" || len(p.Cells) != 1 || p.Cells[0].Value != "61" {
		t.Fatal("page after oversized cell unreachable")
	}
	var dense strings.Builder
	for row := 1; row <= 2; row++ {
		fmt.Fprintf(&dense, `<row r="%d">`, row)
		for col := 0; col < 16; col++ {
			fmt.Fprintf(&dense, `<c r="%c%d" t="inlineStr"><is><t>%s</t></is></c>`, rune('A'+col), row, strings.Repeat("x", 40000))
		}
		dense.WriteString(`</row>`)
	}
	pageBig := write("page-limit.xlsx", [][2]string{{"xl/worksheets/sheet1.xml", sheet(dense.String())}})
	p, e = readTablePreview(context.Background(), pageBig, workspacefs.XLSXPageRequest{})
	if e != nil || !strings.Contains(p.Problem, "1 MiB") || p.TSV != "" || len(p.Cells) != 0 {
		t.Fatal("page content bound absent")
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(before, after) {
		t.Fatal("preview mutated workbook")
	}
	q := &textReadQueue{}
	defer q.stop(true)
	started, release, settled := make(chan struct{}), make(chan struct{}), make(chan struct{}, 4)
	var once sync.Once
	defer once.Do(func() { close(release) })
	q.setNotify(func() { settled <- struct{}{} })
	a := textReadOwner{binding: "host-a:epoch1", root: dir, session: "s", path: "range.xlsx", tab: 1}
	q.request(a, func(context.Context) textReadResult {
		close(started)
		<-release
		return textReadResult{table: first, text: first.Text}
	})
	wait := func(ch <-chan struct{}) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatal("queue did not settle")
		}
	}
	wait(started)
	b := a
	b.binding = "host-b:epoch2"
	b.tab = 2
	q.request(b, func(context.Context) textReadResult { return textReadResult{table: second, text: second.Text} })
	once.Do(func() { close(release) })
	wait(settled)
	wrong := b
	wrong.path = "other.xlsx"
	for _, owner := range []textReadOwner{a, wrong} {
		if _, ok := q.take(owner); ok {
			t.Fatal("stale owner accepted")
		}
	}
	result, ok := q.take(b)
	if !ok || result.table != second || result.text != second.Text {
		t.Fatal("new page lost to old read")
	}
	q.stop(false)
	if _, ok := q.take(b); ok {
		t.Fatal("cancelled page retained")
	}
	q.request(b, func(context.Context) textReadResult { return textReadResult{table: first} })
	wait(settled)
	q.stop(false)
	if _, ok := q.take(b); ok {
		t.Fatal("cancel after completion retained page")
	}
	q.stop(true)
	t.Log("later worksheets/rows/columns; exact long-cell/TSV copies; local version and cancellation; XLS/file/XML/cell/page limits; no writes; stale Host/file/tab ownership verified")
}

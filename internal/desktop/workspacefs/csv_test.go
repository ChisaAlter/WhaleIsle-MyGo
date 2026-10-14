package workspacefs

import (
	"strings"
	"testing"
)

func TestFormatCSVTable(t *testing.T) {
	got, ok := FormatCSVTable("a,b\n1,22\n", 50, 12)
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(got, "a") || !strings.Contains(got, "22") {
		t.Fatalf("got %q", got)
	}
}

func TestIsCSVPath(t *testing.T) {
	if !IsCSVPath(`C:\x\data.csv`) || !IsCSVPath("a.TSV") {
		t.Fatal("csv/tsv")
	}
	if IsCSVPath("a.txt") {
		t.Fatal("txt")
	}
}

package workspacefs

import (
	"runtime"
	"strings"
	"testing"
)

func TestClassifyDoc(t *testing.T) {
	if ClassifyDoc("a.PDF") != DocPDF {
		t.Fatal("pdf")
	}
	if ClassifyDoc("b.xlsx") != DocXLSX {
		t.Fatal("xlsx")
	}
	if ClassifyDoc("c.go") != DocNone {
		t.Fatal("go")
	}
}

func TestFileURL(t *testing.T) {
	if runtime.GOOS == "windows" {
		u := FileURL(`C:\tmp\x.pdf`)
		if !strings.HasPrefix(u, "file:///C:/") {
			t.Fatalf("%q", u)
		}
	} else {
		u := FileURL("/tmp/x.pdf")
		if u != "file:///tmp/x.pdf" {
			t.Fatalf("%q", u)
		}
	}
}

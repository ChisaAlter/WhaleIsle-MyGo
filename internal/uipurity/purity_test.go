package uipurity

import (
	"strings"
	"testing"
)

func TestCountSourceRules(t *testing.T) {
	src := `package sample

import (
	"fmt"
	"net/http"
	"os"
	"time"

	xsys "golang.org/x/sys/windows"
)

var _ = fmt.Sprint
var _ = http.DefaultClient
var _ = os.Getenv
var _ = xsys.Beep

type store struct{}

func (s *store) Snapshot() int { return 0 }

func build(s *store, other interface{ Snapshot(int) int }) {
	s.Snapshot()
	s.Snapshot()
	other.Snapshot(1)        // one arg does not count
	go helper()
	go func() {}()
	time.AfterFunc(0, helper)
	t := time.NewTicker(0)
	_ = time.NewTimer(0)
	_ = time.Tick(0)
	_ = t
	// s.Snapshot() in a comment is not a call.
	text := "store.Snapshot() and time.NewTimer in a string are not calls"
	_ = text
}

func helper() {}
`
	counts, err := countSource("sample.go", src)
	if err != nil {
		t.Fatal(err)
	}
	want := Counts{RuleImport: 3, RuleSnapshot: 2, RuleGo: 2, RuleTimer: 4}
	for rule, n := range want {
		if counts[rule] != n {
			t.Errorf("%s: got %d, want %d", rule, counts[rule], n)
		}
	}
}

func TestCountSourceStringsAndCommentsDoNotCount(t *testing.T) {
	src := `package sample

// m.Snapshot() time.NewTimer(0) go run() — all in a comment.
var text = ` + "`s.Snapshot() go f() time.AfterFunc(0,nil)`" + `

func build() {}
`
	counts, err := countSource("sample.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 0 {
		t.Fatalf("expected zero counts, got %v", counts)
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	entries := []Entry{
		{File: "internal/ui/a.go", Rule: RuleGo, Count: 1},
		{File: "internal/ui/a.go", Rule: RuleSnapshot, Count: 3},
		{File: "internal/ui/b.go", Rule: RuleImport, Count: 2},
	}
	parsed, err := Parse(Format(entries))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != len(entries) {
		t.Fatalf("parsed %d entries, want %d", len(parsed), len(entries))
	}
	for _, e := range entries {
		got, ok := parsed[e.File+"\x00"+e.Rule]
		if !ok || got != e {
			t.Errorf("round trip lost %+v", e)
		}
	}
	if !strings.HasSuffix(Format(entries), "\n") {
		t.Error("format must end with a newline")
	}
}

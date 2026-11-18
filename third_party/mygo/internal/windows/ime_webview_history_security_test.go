//go:build windows

package windows

import (
	"fmt"
	"testing"
)

func TestImeStringNullHandleSafe(t *testing.T) {
	if got := imeString(0, 0); got != "" {
		t.Fatalf("imeString(0, 0) = %q, want empty", got)
	}
}

func TestToastServiceHistoryBoundedFIFO(t *testing.T) {
	s := &toastService{}
	total := maxToastHistory + 44
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("toast-%d", i)
		s.trackActive(id, &notification{id: id})
		if seen := s.activationSeen(id); seen {
			t.Fatalf("first activationSeen(%q) = true, want false", id)
		}
	}
	if len(s.active) != maxToastHistory {
		t.Fatalf("len(active) = %d, want %d", len(s.active), maxToastHistory)
	}
	if len(s.activated) != maxToastHistory {
		t.Fatalf("len(activated) = %d, want %d", len(s.activated), maxToastHistory)
	}
	if s.active["toast-0"] != nil {
		t.Fatal("oldest active notification toast-0 was not evicted")
	}
	if s.active[fmt.Sprintf("toast-%d", total-1)] == nil {
		t.Fatal("newest active notification missing")
	}
	if !s.activationSeen(fmt.Sprintf("toast-%d", total-1)) {
		t.Fatal("newest activation should remain deduplicated")
	}
	if s.activationSeen("toast-0") {
		t.Fatal("evicted activation toast-0 unexpectedly still marked seen")
	}
}

func TestWindowLoadHTMLFiltersBoundedAndDeduplicated(t *testing.T) {
	w := &window{}
	w.trackHTMLDocument("https://dup.localhost/app", "<html/>", true)
	w.trackHTMLDocument("https://dup.localhost/app", "<html/>", true)
	if len(w.htmlOrder) != 1 || len(w.htmlFilters) != 1 {
		t.Fatalf("duplicate filter was not deduplicated: order=%d filters=%d", len(w.htmlOrder), len(w.htmlFilters))
	}

	total := maxLoadHTMLFilters + 10
	for i := 0; i < total; i++ {
		w.trackHTMLDocument(fmt.Sprintf("https://t%d.localhost/doc", i), "<html/>", true)
	}
	if len(w.htmlFilters) != maxLoadHTMLFilters {
		t.Fatalf("len(htmlFilters) = %d, want %d", len(w.htmlFilters), maxLoadHTMLFilters)
	}
	if w.htmlFilters["https://dup.localhost/app"] {
		t.Fatal("oldest LoadHTML filter was not evicted")
	}
	if !w.htmlFilters[fmt.Sprintf("https://t%d.localhost/doc", total-1)] {
		t.Fatal("newest LoadHTML filter missing")
	}
}

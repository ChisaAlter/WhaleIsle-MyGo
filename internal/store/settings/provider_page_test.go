package settings

import (
	"encoding/json"
	"testing"
)

// The card-owner counter belongs to the store it retires cards for. A store
// dropped with its counter cannot leave a stale owner behind in a package-level
// map, so a fresh store starts at page 0 and leaving one store's forms never
// retires a card on another store.
func TestProviderPageCounterBelongsToItsStore(t *testing.T) {
	newStore := func() *Store {
		s := New()
		s.namespaces = []Namespace{{
			NS:     "llm-pi-ai",
			Value:  json.RawMessage(`{}`),
			Fields: []Field{{Key: "baseURL", Kind: FieldString}},
		}}
		return s
	}
	first, second := newStore(), newStore()

	card := first.BindProviderEditor("llm-pi-ai", "alpha", nil)
	if card == nil {
		t.Fatal("first store: no provider card")
	}
	if !card.currentLocked() {
		t.Fatal("first store: a fresh card is not current")
	}
	first.LeaveProviderForms()
	if card.currentLocked() {
		t.Fatal("first store: leaving forms did not retire the card")
	}

	other := second.BindProviderEditor("llm-pi-ai", "alpha", nil)
	if other == nil || !other.currentLocked() {
		t.Fatal("second store: its card is not current on its own store")
	}
	if page := other.providerDraft.page; page != 0 {
		t.Fatalf("second store starts at page %d, want 0: the counter outlived its store", page)
	}
}

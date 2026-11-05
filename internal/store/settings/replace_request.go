package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sync"
	"whaleisle/internal/dsh"
)

// replaceDraftReceipt freezes the user-layer candidate displayed by a replace
// confirmation. Confirming cannot target another Host, namespace or later edit.
type replaceDraftReceipt struct {
	client                  *dsh.Client
	generation              uint64
	namespace               string
	revision, draftRevision float64
	drafts                  map[string]string
	dirty, resets           map[string]bool
	fields                  []Field
	err                     error
}

func (r *replaceDraftReceipt) currentLocked(s *Store, name string) bool {
	ns := s.findNamespaceLocked(name)
	return ns != nil && s.client == r.client && s.generation == r.generation && s.selectedNS == r.namespace && name == r.namespace && ns.Revision == r.revision && s.draftRevisionLocked(*ns) == r.draftRevision && maps.Equal(s.drafts[name], r.drafts) && maps.Equal(s.dirty[name], r.dirty) && maps.Equal(s.resets[name], r.resets) && reflect.DeepEqual(ns.Fields, r.fields)
}

// PrepareReplaceFromDrafts is side-effect free until its returned confirmation
// is invoked. It retains the existing replace writer and revision contract.
func (s *Store) PrepareReplaceFromDrafts(ctx context.Context) func() {
	s.mu.Lock()
	ns := s.findNamespaceLocked(s.selectedNS)
	if s.client == nil || ns == nil || s.saving || s.loading || s.themeWriteInFlightLocked(s.selectedNS) {
		s.mu.Unlock()
		return nil
	}
	receipt := &replaceDraftReceipt{client: s.client, generation: s.generation, namespace: ns.NS, revision: ns.Revision, draftRevision: s.draftRevisionLocked(*ns), drafts: copyDrafts(s.drafts[ns.NS]), dirty: copyFlags(s.dirty[ns.NS]), resets: copyFlags(s.resets[ns.NS]), fields: append([]Field(nil), ns.Fields...)}
	section, err := replacementFromDrafts(*ns, receipt.drafts, receipt.dirty, receipt.resets)
	receipt.err = err
	// Freeze the serialized section independently of subsequent form snapshots.
	frozen := map[string]json.RawMessage{}
	for key, value := range section {
		frozen[key] = append(json.RawMessage(nil), value...)
	}
	s.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { s.saveReplacePrepared(ctx, receipt.namespace, frozen, receipt) }) }
}
func (s *Store) replaceReceiptErrorLocked(name string, receipt *replaceDraftReceipt) error {
	if receipt == nil {
		return nil
	}
	if !receipt.currentLocked(s, name) {
		return fmt.Errorf("Settings changed while confirmation was open; review the current section and request replacement again")
	}
	return receipt.err
}

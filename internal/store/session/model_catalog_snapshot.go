package session

import (
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// CachedModelCatalog returns Host metadata already loaded by the normal model
// picker. It does not resolve credentials, discover providers or send requests,
// and remains independent of whichever Session is currently selected.
func (s *Store) CachedModelCatalog(client *dsh.Client) ([]ModelOption, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client == nil || s.client != client {
		return nil, false
	}
	options := append([]ModelOption(nil), s.modelOptions...)
	for i, o := range options {
		if o.Reasoning != nil {
			r := *o.Reasoning
			r.Efforts = append([]api.ModelCatalogGroupsItemModelsItemReasoningEffortsItem(nil), r.Efforts...)
			if r.DefaultEffort != nil {
				v := *r.DefaultEffort
				r.DefaultEffort = &v
			}
			options[i].Reasoning = &r
		}
	}
	return options, s.modelState.CatalogReady
}

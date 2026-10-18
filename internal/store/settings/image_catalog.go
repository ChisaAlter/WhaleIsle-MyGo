package settings

import "whaleisle/internal/petlook"

// ImageCatalogSnapshot reads only model-catalog readiness and image routes.
// Pet configuration must not depend on the global vision namespace's value,
// presence, write access or an unrelated settings save.
func (s *Store) ImageCatalogSnapshot() petlook.Catalog {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := petlook.Catalog{Connected: s.client != nil, Loading: s.visionLoading, Err: s.visionErr}
	for _, route := range s.visionRoutes {
		v.Routes = append(v.Routes, petlook.Route{Provider: route.Provider, Model: route.Model, Label: route.Label})
	}
	return v
}

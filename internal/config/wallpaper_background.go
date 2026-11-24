package config

import (
	"encoding/json"
	"whaleisle/internal/desktop/wallpapers"
)

const backgroundKey = "goWallpaperBackground"

// HasWallpaperBackground reports whether the profile has saved appearance values,
// so a fresh profile can take the product default without overriding an explicit
// "none" effect.
func (s *Store) HasWallpaperBackground() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.raw[backgroundKey]) > 0
}

func (s *Store) WallpaperBackground() (wallpapers.Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := wallpapers.Defaults()
	if raw := s.raw[backgroundKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return p, err
		}
		// A stored backgroundEffect key — including "none" — is an explicit
		// choice, so it survives the flat no-wallpaper default.
		var stored map[string]json.RawMessage
		if err := json.Unmarshal(raw, &stored); err == nil {
			_, p.EffectExplicit = stored["backgroundEffect"]
		}
	}
	return p, nil
}

var (
	wallpaperPrefsFields  = structJSONFields[wallpapers.Preferences]()
	wallpaperCropFields   = structJSONFields[wallpapers.Crop]()
	wallpaperSourceFields = structJSONFields[wallpapers.Source]()
	wallpaperItemFields   = structJSONFields[wallpapers.Item]()
)

// SaveWallpaperBackground commits native appearance preferences without changing unrelated config
// and preserves unknown fields inside goWallpaperBackground and its nested objects.
func (s *Store) SaveWallpaperBackground(p wallpapers.Preferences) error {
	unlock, err := s.beginSyncWrite()
	if err != nil {
		return err
	}
	defer unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return err
	}
	mergedObj, err := mergeKnownFieldsObject(s.raw[backgroundKey], encoded, wallpaperPrefsFields, func(existing, out map[string]json.RawMessage) error {
		if prevCrop := existing["wallpaperCrop"]; len(prevCrop) > 0 {
			if nextCrop, ok := out["wallpaperCrop"]; ok {
				if mergedCrop, err := mergeKnownFieldsObject(prevCrop, nextCrop, wallpaperCropFields, nil); err == nil {
					out["wallpaperCrop"] = mergedCrop
				}
			}
		}
		if prevSources := existing["wallpaperSources"]; len(prevSources) > 0 {
			if nextSources, ok := out["wallpaperSources"]; ok {
				if mergedSources, err := mergeObjectArrayByID(prevSources, nextSources, wallpaperSourceFields, nil); err == nil {
					out["wallpaperSources"] = mergedSources
				}
			}
		}
		if prevFavs := existing["wallpaperFavorites"]; len(prevFavs) > 0 {
			if nextFavs, ok := out["wallpaperFavorites"]; ok {
				if mergedFavs, err := mergeObjectArrayByID(prevFavs, nextFavs, wallpaperItemFields, nil); err == nil {
					out["wallpaperFavorites"] = mergedFavs
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	merged := cloneRawMap(s.raw)
	merged[backgroundKey] = mergedObj
	baseline, err := s.commitRawLocked(s.baseline, merged)
	if err != nil {
		return err
	}
	s.raw = merged
	s.baseline = baseline
	return nil
}

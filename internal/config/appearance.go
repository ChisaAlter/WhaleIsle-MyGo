package config

import (
	"encoding/json"
	"whaleisle/internal/ui/theme"
)

const appearanceKey = "goAppearance"

func (s *Store) Appearance() (theme.Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := theme.Defaults()
	if raw := s.raw[appearanceKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return p, err
		}
	}
	return p.Normalized(), nil
}

var (
	appearancePrefsFields = structJSONFields[theme.Preferences]()
	fontPrefFields        = structJSONFields[theme.FontPreference]()
	themeFamilyFields     = structJSONFields[theme.Family]()
	themeSeedFields       = structJSONFields[theme.Seed]()
	appearanceFontKeys    = [5]string{"interface", "chat", "code", "composer", "terminal"}
)

// SaveAppearance commits native appearance preferences without changing unrelated config
// and preserves unknown fields inside goAppearance, its fonts, and matched custom themes.
func (s *Store) SaveAppearance(p theme.Preferences) error {
	unlock, err := s.beginSyncWrite()
	if err != nil {
		return err
	}
	defer unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return err
	}
	mergedObj, err := mergeKnownFieldsObject(s.raw[appearanceKey], encoded, appearancePrefsFields, func(existing, out map[string]json.RawMessage) error {
		for _, fontKey := range appearanceFontKeys {
			if prevFont := existing[fontKey]; len(prevFont) > 0 {
				if nextFont, ok := out[fontKey]; ok {
					if mergedFont, err := mergeKnownFieldsObject(prevFont, nextFont, fontPrefFields, nil); err == nil {
						out[fontKey] = mergedFont
					}
				}
			}
		}
		if prevCustom := existing["custom"]; len(prevCustom) > 0 {
			if nextCustom, ok := out["custom"]; ok {
				mergedCustom, err := mergeObjectArrayByID(prevCustom, nextCustom, themeFamilyFields, func(existingItem, _, mergedItem map[string]json.RawMessage) error {
					for _, seedKey := range [2]string{"light", "dark"} {
						if prevSeed := existingItem[seedKey]; len(prevSeed) > 0 {
							if nextSeed, ok := mergedItem[seedKey]; ok {
								if mergedSeed, err := mergeKnownFieldsObject(prevSeed, nextSeed, themeSeedFields, nil); err == nil {
									mergedItem[seedKey] = mergedSeed
								}
							}
						}
					}
					return nil
				})
				if err == nil {
					out["custom"] = mergedCustom
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	merged := cloneRawMap(s.raw)
	merged[appearanceKey] = mergedObj
	baseline, err := s.commitRawLocked(s.baseline, merged)
	if err != nil {
		return err
	}
	s.raw = merged
	s.baseline = baseline
	return nil
}

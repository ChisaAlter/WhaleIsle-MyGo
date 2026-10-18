package previewprocess

import (
	"encoding/json"
	"errors"
	"whaleisle/internal/desktop/htmlpreview"
)

// SetAppearance is the default-palette convenience path.
func (s *Session) SetAppearance(appearance string) error { return s.SetTheme(appearance, nil) }

// SetTheme transmits current application tokens over the parent-only pipe.
// It cannot evaluate caller-supplied script or reload the document.
func (s *Session) SetTheme(appearance string, variables map[string]string) error {
	if _, err := htmlpreview.ProcessThemeTokensScript(appearance, variables); err != nil {
		return err
	}
	encoded, _ := json.Marshal(struct {
		Appearance string
		Variables  map[string]string
	}{appearance, variables})
	key := string(encoded)
	s.surfaceMu.Lock()
	defer s.surfaceMu.Unlock()
	if s.theme == key {
		return nil
	}
	select {
	case <-s.Done():
		return errors.New("preview closed")
	default:
	}
	if s.sendTheme == nil {
		return errors.New("preview theme channel unavailable")
	}
	if err := s.sendTheme(appearance, variables); err != nil {
		return err
	}
	s.theme = key
	return nil
}

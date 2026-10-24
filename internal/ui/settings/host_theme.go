package settings

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"whaleisle/internal/desktop/wallpapers"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

type HostThemeBinding struct{ binding *store.ThemeBinding }

func (h *HostThemeBinding) Pending() bool { return h.binding.Pending() }
func (h *HostThemeBinding) Flush()        { h.binding.Flush() }

type HostThemeState struct {
	Ready, Writable, Loading, Saving, Pending bool
	Owner, Error, Scheme, WallpaperImage      string
	WallpaperImageExplicit                    bool
	Preferences                               theme.Preferences
	Background                                wallpapers.Preferences
	Revision                                  float64
}

func BindHostTheme(s *store.Store) *HostThemeBinding {
	if s == nil {
		return nil
	}
	b := s.BindTheme()
	if b == nil {
		return nil
	}
	return &HostThemeBinding{b}
}

func (h *HostThemeBinding) Snapshot() HostThemeState {
	s := h.binding.Snapshot()
	result := HostThemeState{Ready: s.ClientReady && len(s.Namespaces) > 0, Writable: s.Writable, Loading: s.Loading, Saving: s.Saving, Pending: h.binding.Pending(), Owner: h.binding.Owner(), Error: s.Err}
	if len(s.Namespaces) == 0 {
		return result
	}
	ns := s.Namespaces[0]
	value, err := theme.ReadHostSettings(ns.Value)
	if err != nil {
		result.Ready = false
		result.Error = err.Error()
		return result
	}
	result.Preferences, result.Background, result.Revision = value.Preferences(), value.Background(), ns.Revision
	result.Scheme, result.WallpaperImage, result.WallpaperImageExplicit = value.Preference, value.WallpaperImage, value.WallpaperImageExplicit
	return result
}

func (h *HostThemeBinding) SaveTypography(p theme.Preferences) error {
	if err := p.Validate(); err != nil {
		return err
	}
	current, err := theme.ReadHostSettings(h.binding.EffectiveValue())
	if err != nil {
		return err
	}
	return h.queueProjected(theme.TypographyFields(p), theme.TypographyFields(current.Preferences()))
}
func (h *HostThemeBinding) SaveBackground(p wallpapers.Preferences) error {
	current, err := theme.ReadHostSettings(h.binding.EffectiveValue())
	if err != nil {
		return err
	}
	return h.queueProjected(theme.BackgroundFields(p), theme.BackgroundFields(current.Background()))
}

func (h *HostThemeBinding) queueProjected(values, projection map[string]any) error {
	changed := map[string]any{}
	for key, value := range values {
		left, _ := json.Marshal(value)
		right, _ := json.Marshal(projection[key])
		if string(left) != string(right) {
			changed[key] = value
		}
	}
	if len(changed) == 0 {
		return nil
	}
	return h.binding.Queue(changed)
}

func (h *HostThemeBinding) SaveScheme(scheme string) error {
	switch scheme {
	case "deepseek.light", "light":
		scheme = "light"
	case "deepseek.dark", "dark":
		scheme = "dark"
	case "system", "deepseek":
		scheme = "system"
	default:
		return fmt.Errorf("Unsupported color scheme")
	}
	return h.queueChanged(map[string]any{"preference": scheme})
}

// ApplyWallpaper encodes an explicitly selected/cropped image for the Host's
// existing wallpaperImage field. There is no automatic import of local config.
func (h *HostThemeBinding) ApplyWallpaper(path string) error {
	image := ""
	if path != "" {
		if strings.HasPrefix(path, "data:image/") {
			image = path
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err = wallpapers.Decode(data); err != nil {
				return err
			}
			mime := "image/png"
			lower := strings.ToLower(path)
			if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") {
				mime = "image/jpeg"
			}
			if strings.HasSuffix(lower, ".webp") {
				mime = "image/webp"
			}
			if strings.HasSuffix(lower, ".gif") {
				mime = "image/gif"
			}
			image = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
		}
	}
	return h.queueChanged(map[string]any{"wallpaperImage": image})
}

func (h *HostThemeBinding) queueChanged(values map[string]any) error {
	s := h.binding.Snapshot()
	if !s.ClientReady || !s.Writable || len(s.Namespaces) == 0 {
		return fmt.Errorf("Host appearance is unavailable or read-only")
	}
	var current map[string]json.RawMessage
	_ = json.Unmarshal(h.binding.EffectiveValue(), &current)
	changed := map[string]any{}
	for key, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var left, right any
		_ = json.Unmarshal(raw, &left)
		_ = json.Unmarshal(current[key], &right)
		leftJSON, _ := json.Marshal(left)
		rightJSON, _ := json.Marshal(right)
		if string(leftJSON) != string(rightJSON) {
			changed[key] = value
		}
	}
	if len(changed) == 0 {
		return nil
	}
	return h.binding.Queue(changed)
}

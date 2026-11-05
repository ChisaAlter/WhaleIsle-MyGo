package settings

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

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

// hostSettingsMemo keeps the decoded Host settings for the raw document the
// binding last exposed. Snapshot runs on every frame and the document is a large
// JSON object decoded into maps and arrays; an unchanged document is answered
// from here (tmp/stalls, round 2026-10-26).
var hostSettingsMemo struct {
	mu    sync.Mutex
	raw   []byte
	value theme.HostSettings
	err   error
	ok    bool
}

func readHostSettingsCached(raw json.RawMessage) (theme.HostSettings, error) {
	hostSettingsMemo.mu.Lock()
	defer hostSettingsMemo.mu.Unlock()
	if hostSettingsMemo.ok && bytes.Equal(hostSettingsMemo.raw, raw) {
		return hostSettingsMemo.value, hostSettingsMemo.err
	}
	value, err := theme.ReadHostSettings(raw)
	hostSettingsMemo.raw = append(hostSettingsMemo.raw[:0], raw...)
	hostSettingsMemo.value, hostSettingsMemo.err, hostSettingsMemo.ok = value, err, true
	return value, err
}

func (h *HostThemeBinding) Snapshot() HostThemeState {
	ready, writable, loading, saving, pending, owner, errStr, revision, rawValue, fields := h.binding.ReadOnlyThemeView()
	result := HostThemeState{Ready: ready, Writable: writable, Loading: loading, Saving: saving, Pending: pending, Owner: owner, Error: errStr}
	if !ready {
		return result
	}
	value, err := readHostSettingsCached(rawValue)
	if err != nil {
		result.Ready = false
		result.Error = err.Error()
		return result
	}
	result.Preferences, result.Background, result.Revision = value.Preferences(), hostThemeBackground(value, fields), revision
	result.Scheme, result.WallpaperImage, result.WallpaperImageExplicit = value.Preference, value.WallpaperImage, value.WallpaperImageExplicit
	return result
}

// hostThemeBackground applies the served ui-theme default to an omitted effect.
// Older Hosts that do not expose this field keep the local flat fallback.
func hostThemeBackground(value theme.HostSettings, fields []store.Field) wallpapers.Preferences {
	if !value.BackgroundEffectExplicit {
		for _, field := range fields {
			if field.Key == "backgroundEffect" {
				value.BackgroundEffect = "gradient"
				value.BackgroundEffectExplicit = true
				break
			}
		}
	}
	return value.Background()
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
	s := h.binding.Snapshot()
	if !s.ClientReady || len(s.Namespaces) == 0 {
		return fmt.Errorf("Host appearance is unavailable or read-only")
	}
	return h.queueProjected(theme.BackgroundFields(p), theme.BackgroundFields(hostThemeBackground(current, s.Namespaces[0].Fields)))
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
		var data []byte
		var err error
		if strings.HasPrefix(path, "data:image/") {
			_, raw, ok := strings.Cut(path, ",")
			if !ok || len(path) > wallpapers.MaxDataURLChars {
				return fmt.Errorf("wallpaper exceeds the image limit")
			}
			data, err = base64.StdEncoding.DecodeString(raw)
		} else {
			data, err = wallpapers.ReadImportFile(path)
		}
		if err != nil {
			return err
		}
		_, encoded, err := wallpapers.PrepareImport(data)
		if err != nil {
			return err
		}
		image = wallpapers.ImageDataURL(encoded)
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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"whaleisle/internal/shortcutprefs"
	"whaleisle/internal/shortcuts"
)

func shortcutPlatform() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}
func (s *mainState) pageShortcutPreferences() *shortcutprefs.Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shortcutPreferences != nil {
		return s.shortcutPreferences
	}
	cfg := s.cfgStore
	if cfg == nil {
		return shortcutprefs.New("desktop:"+shortcutPlatform(), nil, nil)
	}
	profile := "desktop:" + shortcutPlatform()
	raw, initErr := cfg.ShortcutDocument()
	if initErr == nil && len(raw) == 0 {
		saved, err := cfg.PageShortcuts(shortcuts.IDs())
		initErr = err
		doc := shortcutprefs.Document{SchemaVersion: 2, Profiles: map[string]map[string]*shortcutprefs.Binding{profile: {}}}
		if initErr == nil {
			for id, value := range saved {
				b, e := shortcuts.SharedBinding(value)
				if e != nil {
					initErr = e
					break
				}
				doc.Profiles[profile][shortcuts.CanonicalID(id)] = b
			}
		}
		if initErr == nil {
			data, e := json.Marshal(doc)
			initErr = e
			if e == nil {
				initErr = cfg.InitializeShortcutDocument(data)
			}
		}
	}
	read := cfg.ShortcutDocument
	if initErr != nil {
		read = func() ([]byte, error) { return nil, initErr }
	}
	prefs := shortcutprefs.New(profile, read, cfg.SaveShortcutDocument)
	prefs.SetDefinitions("native", shortcuts.SharedDefinitions(profile))
	s.shortcutPreferences = prefs
	return prefs
}
func sharedNativeValues(prefs *shortcutprefs.Store) map[string]string {
	values := map[string]string{}
	byID := map[string]shortcutprefs.EffectiveShortcut{}
	for _, row := range prefs.Effective("native") {
		byID[row.ID] = row
	}
	for _, id := range shortcuts.IDs() {
		row, ok := byID[shortcuts.CanonicalID(id)]
		if !ok {
			values[id] = ""
			continue
		}
		values[id] = shortcuts.SharedDisplay(row.Binding, shortcutPlatform())
	}
	return values
}
func (s *mainState) pageShortcutEditor() *shortcuts.Editor {
	prefs := s.pageShortcutPreferences()
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := prefs.Get()
	if s.shortcutEditor == nil {
		var loadErr error
		if snapshot.Status != "ready" {
			loadErr = errors.New("shared shortcut preferences unavailable")
		}
		s.shortcutRevision = snapshot.Revision
		s.shortcutEditor = shortcuts.NewEditor(sharedNativeValues(prefs), loadErr, func(patch map[string]string) error {
			var edits []shortcutprefs.Edit
			for id, raw := range patch {
				if raw == "\x00reset" {
					edits = append(edits, shortcutprefs.Edit{Type: "reset", ID: shortcuts.CanonicalID(id)})
					continue
				}
				b, err := shortcuts.SharedBinding(raw)
				if err != nil {
					return err
				}
				edits = append(edits, shortcutprefs.Edit{Type: "set", ID: shortcuts.CanonicalID(id), Binding: b})
			}
			result := prefs.Apply("native", s.shortcutRevision, edits)
			if result.Status != "saved" {
				return fmt.Errorf("%s; reload shortcuts before retrying", result.Status)
			}
			s.shortcutRevision = result.Snapshot.Revision
			return nil
		})
		s.shortcutEditor.UseShared(func() error {
			var edits []shortcutprefs.Edit
			for _, row := range prefs.Effective("native") {
				// Native reset does not erase advanced bindings that this editor cannot represent.
				if row.Binding != nil {
					if _, err := shortcuts.Parse(shortcuts.SharedDisplay(row.Binding, shortcutPlatform())); err != nil {
						continue
					}
				}
				edits = append(edits, shortcutprefs.Edit{Type: "reset", ID: row.ID})
			}
			result := prefs.Apply("native", s.shortcutRevision, edits)
			if result.Status != "saved" {
				return fmt.Errorf("%s; reload shortcuts before retrying", result.Status)
			}
			s.shortcutRevision = result.Snapshot.Revision
			return nil
		})
	}
	if s.shortcutEditor.Reconcile(sharedNativeValues(prefs)) {
		s.shortcutRevision = snapshot.Revision
	}
	return s.shortcutEditor
}
func (s *mainState) pageShortcutBindings() map[string]string {
	prefs := s.pageShortcutPreferences()
	values := sharedNativeValues(prefs)
	if prefs.Get().Status != "ready" {
		for id := range values {
			values[id] = ""
		}
		return values
	}
	byID := map[string]string{}
	for _, id := range shortcuts.IDs() {
		byID[shortcuts.CanonicalID(id)] = id
	}
	for _, row := range prefs.Effective("native") {
		if row.Issue != nil || len(row.Conflicts) > 0 {
			values[byID[row.ID]] = ""
		}
	}
	return values
}

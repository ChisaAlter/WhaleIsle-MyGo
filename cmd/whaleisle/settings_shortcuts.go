package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/shortcutprefs"
	"whaleisle/internal/shortcuts"
)

func shortcutPlatform() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

func initializeNativeShortcutDocument(cfg *config.Store, profile string) error {
	raw, err := cfg.ShortcutDocument()
	if err != nil || len(raw) != 0 {
		return err
	}
	saved, err := cfg.PageShortcuts(shortcuts.IDs())
	if err != nil {
		return err
	}
	doc := shortcutprefs.Document{SchemaVersion: 2, Profiles: map[string]map[string]*shortcutprefs.Binding{profile: {}}}
	for id, value := range saved {
		binding, err := shortcuts.LegacySharedBinding(value)
		if err != nil {
			return err
		}
		doc.Profiles[profile][shortcuts.CanonicalID(id)] = binding
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return cfg.InitializeShortcutDocument(data)
}
func (s *mainState) pageShortcutPreferences() *shortcutprefs.Store {
	s.mu.Lock()
	if s.shortcutPreferences != nil {
		prefs := s.shortcutPreferences
		s.mu.Unlock()
		return prefs
	}
	cfg := s.cfgStore
	if cfg == nil {
		prefs := shortcutprefs.New("desktop:"+shortcutPlatform(), nil, nil)
		s.mu.Unlock()
		return prefs
	}
	s.mu.Unlock()
	profile := "desktop:" + shortcutPlatform()
	initErr := initializeNativeShortcutDocument(cfg, profile)
	read := cfg.ShortcutDocument
	if initErr != nil {
		read = func() ([]byte, error) { return nil, initErr }
	}
	prefs := shortcutprefs.New(profile, read, cfg.SaveShortcutDocument)
	prefs.SetDefinitions("native", shortcuts.SharedDefinitions(profile))
	s.mu.Lock()
	if s.shortcutPreferences != nil {
		existing := s.shortcutPreferences
		s.mu.Unlock()
		return existing
	}
	s.shortcutPreferences = prefs
	s.mu.Unlock()
	// The document is the shared authority with another shell, so it still has
	// to be noticed when that shell writes it. Reading it here, on a timer, keeps
	// that check off the render path: the UI thread only compares the accepted
	// revision. tmp/stalls captured the per-frame read inside the frame.
	go watchShortcutDocument(cfg, prefs, func() *mygo.Window {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.win
	})
	return prefs
}

// watchShortcutDocument rereads the document periodically and asks for one
// frame when its accepted revision advances. It never touches the render path.
func watchShortcutDocument(cfg *config.Store, prefs *shortcutprefs.Store, window func() *mygo.Window) {
	if cfg == nil {
		return
	}
	var last []byte
	if raw, err := cfg.ShortcutDocument(); err == nil {
		last = raw
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		raw, err := cfg.ShortcutDocument()
		if err != nil || bytes.Equal(raw, last) {
			continue
		}
		last = raw
		before := prefs.Accepted().Sequence
		prefs.Get()
		if prefs.Accepted().Sequence == before {
			continue
		}
		if win := window(); win != nil {
			win.Update(func() {})
		}
	}
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
	cfg := s.cfgStore
	snapshot := prefs.Accepted()
	if s.shortcutEditor == nil {
		var loadErr error
		if snapshot.Status != "ready" {
			loadErr = errors.New(i18n.T(i18n.KeyCmdSharedShortcutsUnavailable))
		}
		s.shortcutRevision = snapshot.Revision
		s.shortcutEditorSeq = snapshot.Sequence
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
				return fmt.Errorf(i18n.T(i18n.KeyCmdReloadShortcutsRetry), result.Status)
			}
			s.shortcutRevision = result.Snapshot.Revision
			return nil
		})
		s.shortcutEditor.UseShared(func() error {
			var edits []shortcutprefs.Edit
			for _, row := range prefs.Effective("native") {
				// Native reset only touches this owner's representable catalog rows.
				if row.Binding != nil {
					if _, err := shortcuts.Parse(shortcuts.SharedDisplay(row.Binding, shortcutPlatform())); err != nil {
						continue
					}
				}
				edits = append(edits, shortcutprefs.Edit{Type: "reset", ID: row.ID})
			}
			result := prefs.Apply("native", s.shortcutRevision, edits)
			if result.Status != "saved" {
				return fmt.Errorf(i18n.T(i18n.KeyCmdReloadShortcutsRetry), result.Status)
			}
			s.shortcutRevision = result.Snapshot.Revision
			return nil
		})
		s.shortcutEditor.SetDescribe(func(id, raw string) string {
			binding, err := shortcuts.SharedBinding(raw)
			if err != nil {
				return err.Error()
			}
			issue, conflicts := prefs.Describe("native", shortcuts.CanonicalID(id), binding)
			if issue != "" {
				return issue
			}
			if len(conflicts) > 0 {
				return fmt.Sprintf(i18n.T(i18n.KeyCmdShortcutConflicts), conflicts)
			}
			return ""
		})
		s.shortcutEditor.SetReload(func() (map[string]string, error) {
			if cfg == nil {
				return nil, errors.New(i18n.T(i18n.KeyCmdShortcutPersistenceUnavailable))
			}
			if _, err := cfg.ReloadShortcutDocument(); err != nil {
				return nil, err
			}
			if err := initializeNativeShortcutDocument(cfg, "desktop:"+shortcutPlatform()); err != nil {
				return nil, err
			}
			snapshot := prefs.ReloadSource(cfg.ShortcutDocument)
			if snapshot.Status != "ready" {
				return nil, errors.New(i18n.T(i18n.KeyCmdSharedShortcutsUnavailable))
			}
			s.mu.Lock()
			s.shortcutRevision = snapshot.Revision
			s.shortcutEditorSeq = snapshot.Sequence
			s.mu.Unlock()
			return sharedNativeValues(prefs), nil
		})
	}
	if s.shortcutEditorSeq != snapshot.Sequence || s.shortcutRevision != snapshot.Revision {
		if s.shortcutEditor.Reconcile(sharedNativeValues(prefs)) {
			s.shortcutRevision = snapshot.Revision
			s.shortcutEditorSeq = snapshot.Sequence
		}
	}
	return s.shortcutEditor
}

// pageShortcutBindings caches the accepted bindings for one state. The values
// only change when the preferences revision advances or the config store writes
// the document, so a frame reuses them instead of reading config.json: that read
// was captured on the UI thread in tmp/stalls. The resolved chords and their
// encoded form come from the same entry: resolving compares every command pair,
// which allocated thousands of objects on every frame.
type shortcutBindingsCache struct {
	prefs      *shortcutprefs.Store
	generation uint64
	sequence   uint64
	values     map[string]string
	chords     map[string]shortcuts.Chord
	encoded    string
}

func (s *mainState) pageShortcutBindings() map[string]string {
	return s.pageShortcutState().values
}

func (s *mainState) pageShortcutChords() map[string]shortcuts.Chord {
	return s.pageShortcutState().chords
}

func (s *mainState) pageShortcutEncoded() string {
	return s.pageShortcutState().encoded
}

type shortcutState struct {
	values  map[string]string
	chords  map[string]shortcuts.Chord
	encoded string
}

func (s *mainState) pageShortcutState() shortcutState {
	prefs := s.pageShortcutPreferences()
	s.mu.Lock()
	cfg := s.cfgStore
	s.mu.Unlock()
	var generation uint64
	if cfg != nil {
		generation = cfg.ShortcutDocumentGeneration()
	}
	s.shortcutBindingsMu.Lock()
	defer s.shortcutBindingsMu.Unlock()
	cache := &s.shortcutBindingsCache
	if cache.prefs == prefs && cache.generation == generation && cache.values != nil {
		if snap := prefs.Accepted(); snap.Sequence == cache.sequence {
			return shortcutState{values: cache.values, chords: cache.chords, encoded: cache.encoded}
		}
	}
	snap := prefs.Accepted()
	if cache.prefs != prefs || cache.generation != generation || cache.values == nil {
		// A write through the config store (initialize, save, reload) can change
		// the document without touching the preferences store: read it once.
		snap = prefs.Get()
	}
	values := shortcutBindingValues(prefs, snap)
	chords := shortcuts.Resolve(values)
	encoded, _ := json.Marshal(values)
	*cache = shortcutBindingsCache{prefs: prefs, generation: generation, sequence: snap.Sequence, values: values, chords: chords, encoded: string(encoded)}
	return shortcutState{values: values, chords: chords, encoded: string(encoded)}
}

// shortcutBindingValues applies one accepted snapshot to the native catalog.
func shortcutBindingValues(prefs *shortcutprefs.Store, snap shortcutprefs.Snapshot) map[string]string {
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
		if row.Issue != nil || len(row.Conflicts) > 0 {
			values[id] = ""
		}
	}
	if snap.Status != "ready" {
		for id := range values {
			values[id] = ""
		}
	}
	return values
}

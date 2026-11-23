// Package custompanel stores declarative templates and independent per-Bot copies.
// Definitions contain no executable code or privileged Host API bindings.
package custompanel

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"whaleisle/internal/state"
	"whaleisle/internal/store/storelocale"
)

type Scope struct {
	Host string `json:"host"`
	Bot  string `json:"bot"`
}
type Widget struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Label    string   `json:"label,omitempty"`
	Text     string   `json:"text,omitempty"`
	Options  []string `json:"options,omitempty"`
	Value    float64  `json:"value,omitempty"`
	FontSize int      `json:"fontSize,omitempty"`
}
type Definition struct {
	ID      string   `json:"id,omitempty"`
	Title   string   `json:"title"`
	Width   int      `json:"width"`
	Height  int      `json:"height"`
	Columns int      `json:"columns"`
	Widgets []Widget `json:"widgets"`
}
type Instance struct {
	Scope      Scope             `json:"scope"`
	Definition Definition        `json:"definition"`
	State      map[string]string `json:"state"`
}
type Snapshot struct {
	Library   []Definition
	Instances []Instance
}
type document struct {
	Version   int          `json:"version"`
	Next      uint64       `json:"next"`
	Library   []Definition `json:"library"`
	Instances []Instance   `json:"instances"`
}

// Store keeps the panel document and one published, immutable view per
// scope it has been asked about. Writers compute under the state lock,
// persist outside it through persistMu so disk latency never reaches a
// reader, and publish the new version only when the write succeeded.
type Store struct {
	mu        sync.RWMutex
	path      string
	data      document
	views     map[Scope]*state.Published[Snapshot]
	persistMu sync.Mutex
	writeFn   func(document) error
}

func Parse(raw string) (Definition, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```json\n") && strings.HasSuffix(raw, "```") {
		raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "```json\n"), "```"))
	}
	if len(raw) > 256*1024 {
		return Definition{}, refuse(ProblemTooLarge, "panel JSON exceeds 256 KiB")
	}
	var d Definition
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return d, refuse(ProblemJSON, "%s", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return d, refuse(ProblemTrailingContent, "expected one Panel JSON object")
	}
	return d, Validate(d)
}
func Validate(d Definition) error {
	if strings.TrimSpace(d.Title) == "" {
		return refuse(ProblemMissingTitle, "panel title is required")
	}
	if d.Width < 240 || d.Width > 1600 || d.Height < 120 || d.Height > 1600 || d.Columns < 1 || d.Columns > 4 {
		return refuse(ProblemSize, "panel width 240-1600, height 120-1600, columns 1-4 required")
	}
	if len(d.Widgets) == 0 || len(d.Widgets) > 64 {
		return refuse(ProblemWidgetCount, "panel needs 1-64 widgets")
	}
	seen := map[string]bool{}
	for _, w := range d.Widgets {
		if w.ID == "" || seen[w.ID] {
			return refuse(ProblemWidgetID, "widget IDs must be nonempty and unique")
		}
		seen[w.ID] = true
		switch w.Kind {
		case "text", "input", "checkbox", "progress", "select", "action":
		default:
			return refuse(ProblemWidgetKind, "unsupported widget kind %q", w.Kind)
		}
		if w.FontSize != 0 && (w.FontSize < 10 || w.FontSize > 32) {
			return refuse(ProblemFontSize, "font size must be 10-32")
		}
		if w.Kind == "progress" && (w.Value < 0 || w.Value > 1) {
			return refuse(ProblemProgressRange, "progress must be 0-1")
		}
		if w.Kind == "select" && len(w.Options) == 0 {
			return refuse(ProblemSelectOptions, "select requires options")
		}
	}
	return nil
}
func Open(userData string) (*Store, error) {
	s := &Store{path: filepath.Join(userData, "assistant", "panels.json"), data: document{Version: 1}, views: map[Scope]*state.Published[Snapshot]{}}
	s.writeFn = func(d document) error { return writeDocument(s.path, d) }
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &s.data); err != nil {
		return nil, err
	}
	if s.data.Version != 1 {
		return nil, refuse(ProblemLibraryVersion, "unsupported panel library version %d", s.data.Version)
	}
	for _, d := range s.data.Library {
		if err = Validate(d); err != nil {
			return nil, err
		}
	}
	for _, i := range s.data.Instances {
		if err = Validate(i.Definition); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Snapshot returns the scope's published view. The same *Version is shared
// by every reader until a write publishes the next revision; its Value is
// immutable — a caller that edits before writing back uses MutableCopy.
func (s *Store) Snapshot(scope Scope) *state.Version[Snapshot] {
	s.mu.RLock()
	p := s.views[scope]
	s.mu.RUnlock()
	if p != nil {
		return p.Load()
	}
	s.mu.Lock()
	p = s.views[scope]
	if p == nil {
		p = &state.Published[Snapshot]{}
		p.Publish(s.buildSnapshotLocked(scope))
		s.views[scope] = p
	}
	s.mu.Unlock()
	return p.Load()
}

func (s *Store) buildSnapshotLocked(scope Scope) Snapshot {
	out := Snapshot{Library: s.data.Library}
	for _, i := range s.data.Instances {
		if i.Scope == scope {
			out.Instances = append(out.Instances, i)
		}
	}
	return out
}

func (s *Store) publishLocked() {
	for scope, p := range s.views {
		p.Publish(s.buildSnapshotLocked(scope))
	}
}

// mutate runs the edit under persistMu so writers serialize end to end,
// computes the next document under the state lock, writes it without that
// lock held, and publishes only on success — the disk stays the last word
// on whether a change happened, while readers never wait on it.
func (s *Store) mutate(edit func(d *document) error) error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.Lock()
	d := s.data
	if err := edit(&d); err != nil {
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()
	if err := s.writeFn(d); err != nil {
		return err
	}
	s.mu.Lock()
	s.data = d
	s.publishLocked()
	s.mu.Unlock()
	return nil
}

// copyDefinition and copyInstance give writers and editors a deep-enough
// copy to mutate without JSON round-trips or touching published values.
func copyDefinition(d Definition) Definition {
	d.Widgets = slices.Clone(d.Widgets)
	for n := range d.Widgets {
		d.Widgets[n].Options = slices.Clone(d.Widgets[n].Options)
	}
	return d
}

func copyInstance(i Instance) Instance {
	i.Definition = copyDefinition(i.Definition)
	i.State = maps.Clone(i.State)
	return i
}

// MutableCopy is the deep copy a caller edits before writing back — the
// widget editor's State map, the test's independent copies.
func MutableCopy(i Instance) Instance { return copyInstance(i) }

func writeDocument(path string, d document) error {
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".panels-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func validScope(scope Scope) error {
	if scope.Host == "" || scope.Bot == "" {
		return refuse(ProblemScope, "open a canonical Bot conversation first")
	}
	return nil
}

// Create atomically saves a reusable template and the creating Bot's independent copy.
func (s *Store) Create(scope Scope, def Definition) (string, error) {
	if err := validScope(scope); err != nil {
		return "", err
	}
	if err := Validate(def); err != nil {
		return "", err
	}
	def = copyDefinition(def)
	var id string
	err := s.mutate(func(d *document) error {
		d.Next++
		id = fmt.Sprintf("panel-%d", d.Next)
		def.ID = id
		d.Library = append(slices.Clone(d.Library), def)
		d.Instances = append(slices.Clone(d.Instances), Instance{Scope: scope, Definition: def, State: map[string]string{}})
		return nil
	})
	return id, err
}
func (s *Store) Add(scope Scope, id string) error {
	if err := validScope(scope); err != nil {
		return err
	}
	return s.mutate(func(d *document) error {
		for _, i := range d.Instances {
			if i.Scope == scope && i.Definition.ID == id {
				return refuse(ProblemAlreadyAdded, "this Bot already has this panel")
			}
		}
		for _, def := range d.Library {
			if def.ID == id {
				d.Instances = append(slices.Clone(d.Instances), Instance{Scope: scope, Definition: def, State: map[string]string{}})
				return nil
			}
		}
		return refuse(ProblemTemplateMissing, "panel template not found %q", id)
	})
}
func (s *Store) Update(scope Scope, instance Instance) error {
	if err := Validate(instance.Definition); err != nil {
		return err
	}
	instance = copyInstance(instance)
	return s.mutate(func(d *document) error {
		for n, i := range d.Instances {
			if i.Scope == scope && i.Definition.ID == instance.Definition.ID {
				instance.Scope = scope
				d.Instances = slices.Clone(d.Instances)
				d.Instances[n] = instance
				return nil
			}
		}
		return refuse(ProblemInstanceMissing, "panel instance not found %q", instance.Definition.ID)
	})
}

// SaveTemplate is an explicit snapshot replacement; existing Bot copies never change.
func (s *Store) SaveTemplate(def Definition) error {
	if err := Validate(def); err != nil {
		return err
	}
	def = copyDefinition(def)
	return s.mutate(func(d *document) error {
		for n, v := range d.Library {
			if v.ID == def.ID {
				d.Library = slices.Clone(d.Library)
				d.Library[n] = def
				return nil
			}
		}
		return refuse(ProblemTemplateMissing, "panel template not found %q", def.ID)
	})
}
func (s *Store) Move(scope Scope, id string, delta int) error {
	return s.mutate(func(d *document) error {
		var indices []int
		current := -1
		for n, i := range d.Instances {
			if i.Scope == scope {
				if i.Definition.ID == id {
					current = len(indices)
				}
				indices = append(indices, n)
			}
		}
		next := current + delta
		if current < 0 || next < 0 || next >= len(indices) {
			return nil
		}
		d.Instances = slices.Clone(d.Instances)
		a, b := indices[current], indices[next]
		d.Instances[a], d.Instances[b] = d.Instances[b], d.Instances[a]
		return nil
	})
}
func (s *Store) Remove(scope Scope, id string) error {
	return s.mutate(func(d *document) error {
		for n, i := range d.Instances {
			if i.Scope == scope && i.Definition.ID == id {
				d.Instances = slices.Delete(slices.Clone(d.Instances), n, n+1)
				return nil
			}
		}
		return refuse(ProblemInstanceMissing, "panel instance not found %q", id)
	})
}

const Example = `{"title":"Daily dashboard","width":420,"height":320,"columns":1,"widgets":[{"id":"heading","kind":"text","text":"My daily plan","fontSize":18},{"id":"notes","kind":"input","label":"Notes","text":""},{"id":"done","kind":"checkbox","label":"Complete"}]}`

func CreationPrompt(requirements string) string {
	return storelocale.T(storelocale.KeyStoreCustompanelCustomizeAReusableRightSidePanelForMy) + Example + storelocale.T(storelocale.KeyStoreCustompanelNwidgetsSupportTextInputCheckboxProgres) + requirements
}

// Package custompanel stores declarative templates and independent per-Bot copies.
// Definitions contain no executable code or privileged Host API bindings.
package custompanel

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
type Store struct {
	mu   sync.Mutex
	path string
	data document
}

func Parse(raw string) (Definition, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```json\n") && strings.HasSuffix(raw, "```") {
		raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "```json\n"), "```"))
	}
	if len(raw) > 256*1024 {
		return Definition{}, fmt.Errorf("Panel JSON exceeds 256 KiB")
	}
	var d Definition
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return d, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return d, fmt.Errorf("Expected one Panel JSON object")
	}
	return d, Validate(d)
}
func Validate(d Definition) error {
	if strings.TrimSpace(d.Title) == "" {
		return fmt.Errorf("Panel title is required")
	}
	if d.Width < 240 || d.Width > 1600 || d.Height < 120 || d.Height > 1600 || d.Columns < 1 || d.Columns > 4 {
		return fmt.Errorf("Panel width 240–1600, height 120–1600, columns 1–4 required")
	}
	if len(d.Widgets) == 0 || len(d.Widgets) > 64 {
		return fmt.Errorf("Panel needs 1–64 widgets")
	}
	seen := map[string]bool{}
	for _, w := range d.Widgets {
		if w.ID == "" || seen[w.ID] {
			return fmt.Errorf("Widget IDs must be nonempty and unique")
		}
		seen[w.ID] = true
		switch w.Kind {
		case "text", "input", "checkbox", "progress", "select", "action":
		default:
			return fmt.Errorf("Unsupported widget kind %q", w.Kind)
		}
		if w.FontSize != 0 && (w.FontSize < 10 || w.FontSize > 32) {
			return fmt.Errorf("Font size must be 10–32")
		}
		if w.Kind == "progress" && (w.Value < 0 || w.Value > 1) {
			return fmt.Errorf("Progress must be 0–1")
		}
		if w.Kind == "select" && len(w.Options) == 0 {
			return fmt.Errorf("Select requires options")
		}
	}
	return nil
}
func Open(userData string) (*Store, error) {
	s := &Store{path: filepath.Join(userData, "assistant", "panels.json"), data: document{Version: 1}}
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
		return nil, fmt.Errorf("Unsupported Panel library version")
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
func clone[T any](v T) T {
	raw, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(raw, &out)
	return out
}
func (s *Store) Snapshot(scope Scope) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Snapshot{Library: clone(s.data.Library)}
	for _, i := range s.data.Instances {
		if i.Scope == scope {
			out.Instances = append(out.Instances, clone(i))
		}
	}
	return out
}
func (s *Store) commit(d document) error {
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".panels-*.json")
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
	if err = os.Rename(f.Name(), s.path); err != nil {
		return err
	}
	s.data = d
	return nil
}
func validScope(scope Scope) error {
	if scope.Host == "" || scope.Bot == "" {
		return fmt.Errorf("Open a canonical Bot conversation first")
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
	s.mu.Lock()
	defer s.mu.Unlock()
	d := clone(s.data)
	d.Next++
	def.ID = fmt.Sprintf("panel-%d", d.Next)
	d.Library = append(d.Library, clone(def))
	d.Instances = append(d.Instances, Instance{Scope: scope, Definition: clone(def), State: map[string]string{}})
	return def.ID, s.commit(d)
}
func (s *Store) Add(scope Scope, id string) error {
	if err := validScope(scope); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d := clone(s.data)
	for _, i := range d.Instances {
		if i.Scope == scope && i.Definition.ID == id {
			return fmt.Errorf("This Bot already has this Panel")
		}
	}
	for _, def := range d.Library {
		if def.ID == id {
			d.Instances = append(d.Instances, Instance{Scope: scope, Definition: clone(def), State: map[string]string{}})
			return s.commit(d)
		}
	}
	return fmt.Errorf("Panel template not found")
}
func (s *Store) Update(scope Scope, instance Instance) error {
	if err := Validate(instance.Definition); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d := clone(s.data)
	for n, i := range d.Instances {
		if i.Scope == scope && i.Definition.ID == instance.Definition.ID {
			instance.Scope = scope
			d.Instances[n] = clone(instance)
			return s.commit(d)
		}
	}
	return fmt.Errorf("Panel instance not found")
}

// SaveTemplate is an explicit snapshot replacement; existing Bot copies never change.
func (s *Store) SaveTemplate(def Definition) error {
	if err := Validate(def); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d := clone(s.data)
	for n, v := range d.Library {
		if v.ID == def.ID {
			d.Library[n] = clone(def)
			return s.commit(d)
		}
	}
	return fmt.Errorf("Panel template not found")
}
func (s *Store) Move(scope Scope, id string, delta int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := clone(s.data)
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
	a, b := indices[current], indices[next]
	d.Instances[a], d.Instances[b] = d.Instances[b], d.Instances[a]
	return s.commit(d)
}
func (s *Store) Remove(scope Scope, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := clone(s.data)
	for n, i := range d.Instances {
		if i.Scope == scope && i.Definition.ID == id {
			d.Instances = append(d.Instances[:n], d.Instances[n+1:]...)
			return s.commit(d)
		}
	}
	return fmt.Errorf("Panel instance not found")
}

const Example = `{"title":"Daily dashboard","width":420,"height":320,"columns":1,"widgets":[{"id":"heading","kind":"text","text":"My daily plan","fontSize":18},{"id":"notes","kind":"input","label":"Notes","text":""},{"id":"done","kind":"checkbox","label":"Complete"}]}`

func CreationPrompt(requirements string) string {
	return storelocale.T(storelocale.KeyStoreCustompanelCustomizeAReusableRightSidePanelForMy) + Example + storelocale.T(storelocale.KeyStoreCustompanelNwidgetsSupportTextInputCheckboxProgres) + requirements
}

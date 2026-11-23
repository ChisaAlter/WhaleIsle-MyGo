package lang

import (
	"errors"
)

// Field is a form field value written by an input that is not bound to a
// $variable (spec section 5.3).
type Field struct {
	Value         Value
	ComponentType string
}

func unwrapField(v Value) Value {
	if f, ok := v.(*Field); ok {
		return f.Value
	}
	return v
}

// Store is the reactive state store: $variables as top-level keys, page
// fields as top-level *Field values and form fields as *Object values of
// *Field keyed by form name.
type Store struct {
	vals      *Object
	pristine  map[string]bool
	persisted map[string]bool
	version   uint64
	maxKeys   int
	inited    bool
}

func newStore(maxKeys int) *Store {
	return &Store{vals: NewObject(8), pristine: map[string]bool{}, persisted: map[string]bool{}, maxKeys: maxKeys}
}

// Get returns the stored value (fields are not unwrapped).
func (s *Store) Get(name string) Value { return s.vals.Value(name) }

// Version increases on every change.
func (s *Store) Version() uint64 { return s.version }

var errTooManyKeys = errors.New("state store key budget exceeded")

// Set writes a value. pristine marks a restored declared default that a
// later parse may still replace.
func (s *Store) Set(name string, v Value, pristine bool) error {
	if pristine {
		s.pristine[name] = true
		delete(s.persisted, name)
	} else {
		delete(s.pristine, name)
		s.persisted[name] = true
	}
	old, present := s.vals.Get(name)
	if present && shallowEqual(old, v) {
		return nil
	}
	if !present && s.maxKeys > 0 && s.vals.Len() >= s.maxKeys {
		return errTooManyKeys
	}
	s.vals.Set(name, v)
	s.version++
	return nil
}

func shallowEqual(a, b Value) bool {
	if SameValue(a, b) {
		return true
	}
	if fa, ok := a.(*Field); ok {
		fb, ok := b.(*Field)
		return ok && fa.ComponentType == fb.ComponentType && SameValue(fa.Value, fb.Value)
	}
	oa, ok1 := a.(*Object)
	ob, ok2 := b.(*Object)
	if !ok1 || !ok2 || oa.Len() != ob.Len() {
		return false
	}
	for _, k := range oa.keys {
		bv, ok := ob.Get(k)
		if !ok || !SameValue(oa.vals[k], bv) {
			return false
		}
	}
	return true
}

// Initialize applies declared defaults and, on first call, persisted
// values. Defaults replace only pristine values; keys are never deleted.
func (s *Store) Initialize(defaults, persisted *Object) error {
	changed := false
	if !s.inited {
		s.inited = true
		for _, k := range persisted.Keys() {
			v := persisted.vals[k]
			delete(s.pristine, k)
			s.persisted[k] = true
			if old, ok := s.vals.Get(k); !ok || !SameValue(old, v) {
				if !ok && s.maxKeys > 0 && s.vals.Len() >= s.maxKeys {
					return errTooManyKeys
				}
				s.vals.Set(k, v)
				changed = true
			}
		}
	}
	for _, k := range defaults.Keys() {
		v := defaults.vals[k]
		old, ok := s.vals.Get(k)
		switch {
		case !ok:
			if s.maxKeys > 0 && s.vals.Len() >= s.maxKeys {
				return errTooManyKeys
			}
			s.vals.Set(k, v)
			s.pristine[k] = true
			changed = true
		case s.pristine[k] && !SameValue(old, v) && !deepEqualData(old, v):
			s.vals.Set(k, v)
			changed = true
		}
	}
	if changed {
		s.version++
	}
	return nil
}

// Snapshot returns a copy of all values.
func (s *Store) Snapshot() *Object { return s.vals.Clone() }

// Persisted returns the values a user or the host produced (not pristine
// defaults), the content of the persisted state document.
func (s *Store) Persisted() *Object {
	o := NewObject(len(s.persisted))
	for _, k := range s.vals.keys {
		if s.persisted[k] {
			o.Set(k, s.vals.vals[k])
		}
	}
	return o
}

// deepEqualData compares plain data structurally.
func deepEqualData(a, b Value) bool {
	switch x := a.(type) {
	case []Value:
		y, ok := b.([]Value)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !deepEqualData(x[i], y[i]) {
				return false
			}
		}
		return true
	case *Object:
		y, ok := b.(*Object)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for _, k := range x.keys {
			yv, ok := y.Get(k)
			if !ok || !deepEqualData(x.vals[k], yv) {
				return false
			}
		}
		return true
	case *Field:
		y, ok := b.(*Field)
		return ok && x.ComponentType == y.ComponentType && deepEqualData(x.Value, y.Value)
	}
	return SameValue(a, b)
}

package pluginmanager

import (
	"reflect"
	"sync"

	"whaleisle/internal/state"
)

// The controller's working State is edited in place across many paths
// (BusyKeys entries, install runs, discover listings), so the published
// version is a deep copy taken once per change instead of the JSON
// round-trip every reader used to pay. unlockPublish compares the working
// state with the last published value and publishes only when something
// changed, so read-only critical sections never bump the revision.

// Snapshot returns the current published version. The same *Version is
// shared by every reader until a change publishes the next revision; its
// Value is immutable.
func (c *Controller) Snapshot() *state.Version[State] {
	if v := c.pub.Load(); v != nil {
		return v
	}
	c.mu.Lock()
	c.unlockPublish()
	return c.pub.Load()
}

// Changed closes when the next version is published.
func (c *Controller) Changed() <-chan struct{} { return c.pub.Changed() }

// unlockPublish publishes the working state if it differs from the last
// published version, then releases c.mu. Every critical section that may
// have edited c.state ends with it.
func (c *Controller) unlockPublish() {
	if v := c.pub.Load(); v == nil || !reflect.DeepEqual(v.Value, c.state) {
		c.pub.Publish(cloneState(c.state))
	}
	c.mu.Unlock()
}

// cloneState deep-copies every slice, map and pointer reachable from s, so
// later in-place edits of the working state never reach a published value.
func cloneState(s State) State {
	var out State
	deepCopy(reflect.ValueOf(&out).Elem(), reflect.ValueOf(s))
	return out
}

func deepCopy(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Pointer:
		if src.IsNil() {
			dst.SetZero()
			return
		}
		p := reflect.New(src.Elem().Type())
		deepCopy(p.Elem(), src.Elem())
		dst.Set(p)
	case reflect.Slice:
		if src.IsNil() {
			dst.SetZero()
			return
		}
		s := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
		if isFlat(src.Type().Elem()) {
			reflect.Copy(s, src)
		} else {
			for i := 0; i < src.Len(); i++ {
				deepCopy(s.Index(i), src.Index(i))
			}
		}
		dst.Set(s)
	case reflect.Map:
		if src.IsNil() {
			dst.SetZero()
			return
		}
		m := reflect.MakeMapWithSize(src.Type(), src.Len())
		elem := src.Type().Elem()
		iter := src.MapRange()
		for iter.Next() {
			v := reflect.New(elem).Elem()
			deepCopy(v, iter.Value())
			m.SetMapIndex(iter.Key(), v)
		}
		dst.Set(m)
	case reflect.Struct:
		dst.Set(src)
		for i := 0; i < src.NumField(); i++ {
			if f := dst.Field(i); f.CanSet() && !isFlat(f.Type()) {
				deepCopy(f, src.Field(i))
			}
		}
	default:
		// Scalars, strings, interfaces and funcs are copied by value;
		// State holds no interface or func fields.
		dst.Set(src)
	}
}

var flatTypes sync.Map // reflect.Type -> bool

// isFlat reports whether a value of t holds no reference that needs copying.
func isFlat(t reflect.Type) bool {
	if v, ok := flatTypes.Load(t); ok {
		return v.(bool)
	}
	flat := computeFlat(t)
	flatTypes.Store(t, flat)
	return flat
}

func computeFlat(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface, reflect.Chan, reflect.Func:
		return false
	case reflect.Array:
		return computeFlat(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !computeFlat(t.Field(i).Type) {
				return false
			}
		}
	}
	return true
}

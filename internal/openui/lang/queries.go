package lang

import (
	"fmt"
	"strings"
)

// FetchRequest asks the host to call a tool for a Query statement.
type FetchRequest struct {
	QueryID string
	Tool    string
	Args    Value
	// Key identifies this fetch; pass it back to ResolveFetch.
	Key string
	// RefreshSeconds is the query's refresh interval (0 for none). The host
	// owns the timer and calls Refetch.
	RefreshSeconds float64
}

// ToolNotFoundError reports a missing host tool (spec section 6.4).
type ToolNotFoundError struct {
	Tool      string
	Available []string
}

func (e *ToolNotFoundError) Error() string {
	avail := strings.Join(e.Available, ", ")
	if avail == "" {
		avail = "(none)"
	}
	return fmt.Sprintf("No handler for tool %q. Available: %s", e.Tool, avail)
}

type queryEntry struct {
	id, tool    string
	args        Value
	defaults    Value
	key, prev   string
	refresh     float64
	loading     bool
	needsFetch  bool
	needsRefire bool
	err         *Diagnostic
}

type cacheEntry struct {
	data     Value
	has      bool
	inFlight bool
}

// MutationResult is the value a mutation reference reads.
type MutationResult struct {
	Status string // idle, loading, success, error
	Data   Value
	Error  Value
}

type mutationEntry struct {
	tool   string
	result MutationResult
	err    *Diagnostic
}

type queryManager struct {
	in        *Instance
	queries   map[string]*queryEntry
	order     []string
	cache     map[string]*cacheEntry
	mutations map[string]*mutationEntry
	resolving map[string]bool
	version   uint64
}

func newQueryManager(in *Instance) *queryManager {
	return &queryManager{in: in, queries: map[string]*queryEntry{}, cache: map[string]*cacheEntry{}, mutations: map[string]*mutationEntry{}, resolving: map[string]bool{}}
}

func (q *queryManager) enabled() bool {
	return q.in.lib == nil || q.in.lib.Profile.AllowData
}

func stableKey(tool string, args Value, deps *Object) string {
	k := tool + "::" + string(EncodeJSON(sortedData(args)))
	if deps != nil {
		k += "::" + string(EncodeJSON(sortedData(deps)))
	}
	return k
}

func sortedData(v Value) Value {
	switch t := v.(type) {
	case *Object:
		keys := append([]string(nil), t.keys...)
		sortStrings(keys)
		o := NewObject(len(keys))
		for _, k := range keys {
			o.Set(k, sortedData(t.Value(k)))
		}
		return o
	case []Value:
		out := make([]Value, len(t))
		for i, x := range t {
			out[i] = sortedData(x)
		}
		return out
	}
	return dataOnly(v)
}

// sync evaluates the query statements and schedules fetches (upstream
// evaluateQueries, run when streaming has ended and on state changes).
func (q *queryManager) sync() {
	res := q.in.result
	if res == nil {
		return
	}
	if len(res.Queries) == 0 && len(res.Mutations) == 0 && len(q.queries) == 0 && len(q.mutations) == 0 {
		return
	}
	q.version++
	env := q.in.newEnv(q.in.actRand)
	active := map[string]bool{}
	q.order = q.order[:0]
	for _, qs := range res.Queries {
		active[qs.ID] = true
		q.order = append(q.order, qs.ID)
		var tool string
		var args, defaults Value
		refresh := 0.0
		func() {
			defer func() { _ = recover() }()
			if qs.Tool != nil {
				tool, _ = env.eval(qs.Tool).(string)
			}
			if qs.Args != nil {
				args = dataOnly(env.eval(qs.Args))
			}
			if qs.Defaults != nil {
				defaults = dataOnly(env.eval(qs.Defaults))
			}
			if qs.Refresh != nil {
				refresh = ToNumber(env.eval(qs.Refresh))
			}
		}()
		var deps *Object
		if len(qs.Deps) > 0 {
			deps = NewObject(len(qs.Deps))
			for _, d := range qs.Deps {
				deps.Set(d, dataOnly(unwrapField(q.in.store.Get(d))))
			}
		}
		key := stableKey(tool, args, deps)
		e := q.queries[qs.ID]
		if e == nil {
			e = &queryEntry{id: qs.ID}
			q.queries[qs.ID] = e
		} else if e.key != key {
			e.prev = e.key
		}
		e.tool, e.args, e.defaults, e.key, e.refresh = tool, args, defaults, key, refresh
		c := q.cache[key]
		if q.enabled() && (c == nil || !c.has) && (c == nil || !c.inFlight) {
			e.needsFetch = true
		}
	}
	for id := range q.queries {
		if !active[id] {
			delete(q.queries, id)
		}
	}
	active = map[string]bool{}
	for _, ms := range res.Mutations {
		active[ms.ID] = true
		tool := ""
		func() {
			defer func() { _ = recover() }()
			if ms.Tool != nil {
				tool, _ = env.eval(ms.Tool).(string)
			}
		}()
		if m := q.mutations[ms.ID]; m != nil {
			m.tool = tool
		} else {
			q.mutations[ms.ID] = &mutationEntry{tool: tool, result: MutationResult{Status: "idle"}}
		}
	}
	for id := range q.mutations {
		if !active[id] {
			delete(q.mutations, id)
		}
	}
}

func (q *queryManager) result(id string) Value {
	if m := q.mutations[id]; m != nil {
		return ObjectFrom("status", m.result.Status, "data", m.result.Data, "error", m.result.Error)
	}
	e := q.queries[id]
	if e == nil {
		// Before the stream ends no query is registered: read its defaults.
		if res := q.in.result; res != nil {
			for _, qs := range res.Queries {
				if qs.ID == id && qs.Defaults != nil {
					if q.resolving[id] {
						return nil
					}
					q.resolving[id] = true
					defer delete(q.resolving, id)
					env := q.in.newEnv(q.in.actRand)
					return dataOnly(env.eval(qs.Defaults))
				}
			}
			for _, ms := range res.Mutations {
				if ms.ID == id {
					return ObjectFrom("status", "idle", "data", nil, "error", nil)
				}
			}
		}
		return nil
	}
	if c := q.cache[e.key]; c != nil && c.has {
		return c.data
	}
	if e.prev != "" {
		if c := q.cache[e.prev]; c != nil && c.has {
			return c.data
		}
	}
	return e.defaults
}

func (q *queryManager) errors() []Diagnostic {
	var out []Diagnostic
	for _, id := range q.order {
		if e := q.queries[id]; e != nil && e.err != nil {
			out = append(out, *e.err)
		}
	}
	for _, m := range q.mutations {
		if m.err != nil {
			out = append(out, *m.err)
		}
	}
	return out
}

// PendingFetches returns the tool calls the host should make now and marks
// them in flight. Query results arrive through ResolveFetch.
func (in *Instance) PendingFetches() []FetchRequest {
	q := in.queries
	var out []FetchRequest
	for _, id := range q.order {
		e := q.queries[id]
		if e == nil || !e.needsFetch {
			continue
		}
		e.needsFetch = false
		c := q.cache[e.key]
		if c == nil {
			c = &cacheEntry{}
			q.cache[e.key] = c
		}
		if c.inFlight {
			e.needsRefire = true
			continue
		}
		c.inFlight = true
		e.loading = true
		out = append(out, FetchRequest{QueryID: id, Tool: e.tool, Args: e.args, Key: e.key, RefreshSeconds: e.refresh})
	}
	return out
}

// ResolveFetch delivers a tool result (or error) for a FetchRequest. A
// result for arguments that are no longer current is cached but not shown.
func (in *Instance) ResolveFetch(req FetchRequest, data Value, err error) {
	q := in.queries
	q.version++
	c := q.cache[req.Key]
	if c == nil {
		c = &cacheEntry{}
		q.cache[req.Key] = c
	}
	c.inFlight = false
	e := q.queries[req.QueryID]
	if e == nil || e.key != req.Key {
		return
	}
	e.loading = false
	if err != nil {
		d := &Diagnostic{Source: SourceQuery, StatementID: req.QueryID, Component: "Query", ToolName: req.Tool}
		if tnf, ok := err.(*ToolNotFoundError); ok {
			d.Code = "tool-not-found"
			d.Message = fmt.Sprintf("Query tool %q not found", req.Tool)
			if len(tnf.Available) > 0 {
				d.Hint = "Available tools: " + strings.Join(tnf.Available, ", ")
			}
		} else {
			d.Code = "tool-error"
			d.Message = fmt.Sprintf("Query %q failed: %s", req.Tool, err.Error())
		}
		e.err = d
	} else {
		c.data, c.has = dataOnly(data), true
		e.err = nil
		if e.prev != "" && e.prev != req.Key {
			delete(q.cache, e.prev)
			e.prev = ""
		}
	}
	if e.needsRefire {
		e.needsRefire = false
		e.needsFetch = true
	}
}

// Refetch schedules the named queries again (all when ids is empty).
func (in *Instance) Refetch(ids ...string) {
	q := in.queries
	if !q.enabled() {
		return
	}
	q.version++
	targets := ids
	if len(targets) == 0 {
		targets = q.order
	}
	for _, id := range targets {
		if e := q.queries[id]; e != nil {
			if c := q.cache[e.key]; c != nil && c.inFlight {
				e.needsRefire = true
			} else {
				e.needsFetch = true
			}
		}
	}
}

// QueryLoading reports queries with a fetch in flight.
func (in *Instance) QueryLoading() []string {
	var out []string
	for _, id := range in.queries.order {
		if e := in.queries.queries[id]; e != nil && e.loading {
			out = append(out, id)
		}
	}
	return out
}

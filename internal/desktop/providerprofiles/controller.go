// Package providerprofiles manages removal of user-owned provider profiles.
// Only credential metadata is read; credential literals never enter this model.
package providerprofiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"whaleisle/internal/dsh/api"
)

type Caller interface {
	Call(context.Context, string, any, any) error
}
type Row struct {
	ID, Provider, DisplayName, Namespace, CredentialRef string
	Path                                                []string
	Removable                                           bool
}
type Snapshot struct {
	Rows            []Row
	Busy, Ready     bool
	Pending         *Row
	Err, LastAction string
}
type Controller struct {
	mu    sync.Mutex
	state Snapshot
}

func cloneRow(r Row) Row { r.Path = append([]string(nil), r.Path...); return r }
func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.state
	s.Rows = append([]Row(nil), s.Rows...)
	for i := range s.Rows {
		s.Rows[i] = cloneRow(s.Rows[i])
	}
	if s.Pending != nil {
		r := cloneRow(*s.Pending)
		s.Pending = &r
	}
	return s
}
func (c *Controller) RequestDelete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Busy || !c.state.Ready {
		return
	}
	for _, r := range c.state.Rows {
		if r.ID == id && r.Removable {
			r = cloneRow(r)
			c.state.Pending = &r
			c.state.Err = ""
			c.state.LastAction = ""
			return
		}
	}
}
func (c *Controller) CancelDelete() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.state.Busy {
		c.state.Pending = nil
	}
}
func (c *Controller) Refresh(ctx context.Context, call Caller) {
	c.mu.Lock()
	if c.state.Busy {
		c.mu.Unlock()
		return
	}
	c.state.Busy = true
	c.state.Pending = nil
	c.state.LastAction = ""
	c.mu.Unlock()
	rows, err := load(ctx, call)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Busy = false
	c.state.Ready = err == nil
	c.state.Err = ""
	if err != nil {
		c.state.Err = "Provider refresh failed; availability is unverified. " + err.Error()
		return
	}
	c.state.Rows = rows
}
func (c *Controller) DeleteExpected(ctx context.Context, call Caller, id string) {
	c.mu.Lock()
	if c.state.Busy || c.state.Pending == nil || id == "" || c.state.Pending.ID != id {
		c.mu.Unlock()
		return
	}
	r := cloneRow(*c.state.Pending)
	c.state.Busy = true
	c.state.Err = ""
	c.state.LastAction = ""
	c.mu.Unlock()
	credentialRemoved, profileRemoved := false, false
	var err error
	if call == nil {
		err = errors.New("Host unavailable")
	} else {
		if r.CredentialRef != "" {
			err = call.Call(ctx, "credentials/unset", map[string]any{"ref": r.CredentialRef}, nil)
			credentialRemoved = err == nil
		}
		if err == nil {
			err = call.Call(ctx, "settings/mutate", map[string]any{"ns": r.Namespace, "ops": []map[string]any{{"op": "unset", "path": r.Path}}}, nil)
			profileRemoved = err == nil
		}
	}
	var rows []Row
	if err == nil {
		rows, err = load(ctx, call)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Busy = false
	if err != nil {
		c.state.Ready = false
		switch {
		case profileRemoved:
			c.state.Pending = nil
			c.state.Err = "Provider profile removed, but refresh failed; displayed rows are stale and availability is unverified. "
		case credentialRemoved:
			c.state.Err = "Associated credential removed, but profile removal failed. This profile may no longer work; retry removal or refresh before reconfiguring. "
		default:
			c.state.Err = "Removal failed; profile deletion was not completed and credential state is unverified. "
		}
		c.state.Err += err.Error()
		return
	}
	c.state.Rows = rows
	c.state.Ready = true
	c.state.Pending = nil
	c.state.LastAction = "Provider profile removed and refreshed."
}

var nonKey = regexp.MustCompile(`[^A-Z0-9]+`)

func keyRef(provider string) string {
	return nonKey.ReplaceAllString(strings.ToUpper(provider), "_") + "_API_KEY"
}
func pathValue(raw json.RawMessage, path []string) (any, bool) {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil, false
	}
	for _, key := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return v, true
}
func load(ctx context.Context, call Caller) ([]Row, error) {
	if call == nil {
		return nil, errors.New("Host unavailable")
	}
	var directory api.ListConfigurableProvidersResult
	if err := call.Call(ctx, "llm/listConfigurableProviders", map[string]any{}, &directory); err != nil {
		return nil, err
	}
	var settings api.SettingsDescribeValue
	if err := call.Call(ctx, "settings/describe", map[string]any{}, &settings); err != nil {
		return nil, err
	}
	refs := []string{}
	rows := []Row{}
	for _, entry := range directory {
		if entry.Provider == "deepseek-account" {
			continue
		}
		for _, ns := range settings.Namespaces {
			if ns.Ns != entry.SettingsNs {
				continue
			}
			profile, configured := pathValue(ns.Value, entry.SettingsPath)
			if !configured {
				continue
			}
			_, user := pathValue(ns.User, entry.SettingsPath)
			_, base := pathValue(ns.Base, entry.SettingsPath)
			identity, _ := json.Marshal([]any{entry.Provider, entry.SettingsNs, entry.SettingsPath})
			r := Row{ID: string(identity), Provider: entry.Provider, DisplayName: entry.DisplayName, Namespace: entry.SettingsNs, Path: append([]string(nil), entry.SettingsPath...), Removable: settings.Writable && len(entry.SettingsPath) > 0 && user && !base}
			if m, ok := profile.(map[string]any); ok {
				if ref, ok := m["apiKeyEnv"].(string); ok && ref == keyRef(entry.Provider) {
					r.CredentialRef = ref
					refs = append(refs, ref)
				}
			}
			rows = append(rows, r)
		}
	}
	var credentials api.DescribeResult
	if err := call.Call(ctx, "credentials/describe", map[string]any{"refs": refs}, &credentials); err != nil {
		return nil, fmt.Errorf("credential metadata unavailable: %w", err)
	}
	for i := range rows {
		info := credentials[rows[i].CredentialRef]
		if !info.Configured || !info.Writable {
			rows[i].CredentialRef = ""
		}
	}
	return rows, nil
}

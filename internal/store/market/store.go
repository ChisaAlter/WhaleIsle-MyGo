// Package market loads pluginManager/listPlugins when the codegen registry
// includes dsh-plugin-manager RPCs.
package market

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// Entry is one row in the installed-plugin list.
type Entry struct {
	Label         string
	Detail        string
	Enabled       *bool  // from listBundles when present
	Removable     bool   // from listBundles when present
	Source        string // listBundles source or empty
	Version       string
	Installed     bool
	IdentityKnown bool
}

// ViewState is a UI-safe snapshot of the store.
type ViewState struct {
	ClientReady    bool
	Loading        bool
	Err            string
	HasListRPC     bool
	ListMethod     string
	Methods        []string
	Rows           []Entry
	RawListPreview string
	RegistriesNote string
	ExemptionsNote string
}

// Store fetches pluginManager/listPlugins when available.
type Store struct {
	mu         sync.Mutex
	client     *dsh.Client
	generation uint64
	refreshID  uint64
	discovery  discoveryState
	updates    updateState
	personal   personalState
	notify     func()

	loading        bool
	err            error
	hasListRPC     bool
	listMethod     string
	methods        []string
	rows           []Entry
	rawListPreview string
	registriesNote string
	exemptionsNote string
}

// New returns an empty market store.
func New() *Store {
	dir, err := os.UserConfigDir()
	if err != nil {
		return NewWithPersonalPath("")
	}
	return NewWithPersonalPath(filepath.Join(dir, "whaleisle", "market-personal.json"))
}

// Attach binds a dsh client and an optional UI refresh hook.
func (s *Store) Attach(client *dsh.Client, notify func()) {
	s.mu.Lock()
	if s.updates.cancel != nil {
		s.updates.cancel()
	}
	s.generation++
	s.refreshID++
	s.rows = nil
	s.loading = false
	s.client = client
	s.notify = notify
	s.err = nil
	s.mu.Unlock()
}

// Detach clears the client when harness stops.
func (s *Store) Detach() {
	s.mu.Lock()
	if s.updates.cancel != nil {
		s.updates.cancel()
	}
	s.generation++
	s.refreshID++
	s.client = nil
	s.loading = false
	s.hasListRPC = false
	s.listMethod = ""
	s.methods = nil
	s.rows = nil
	s.rawListPreview = ""
	s.registriesNote = ""
	s.exemptionsNote = ""
	s.mu.Unlock()
	s.fire()
}

// Snapshot copies observable state for one frame.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{
		ClientReady:    s.client != nil,
		Loading:        s.loading,
		HasListRPC:     s.hasListRPC,
		ListMethod:     s.listMethod,
		RawListPreview: s.rawListPreview,
		RegistriesNote: s.registriesNote,
		ExemptionsNote: s.exemptionsNote,
	}
	if s.err != nil {
		st.Err = s.err.Error()
	}
	st.Methods = append([]string(nil), s.methods...)
	st.Rows = append([]Entry(nil), s.rows...)
	return st
}

// Refresh probes pluginManager/* registry entries and calls listPlugins when present.
func (s *Store) Refresh(ctx context.Context) {
	methods := api.PluginManagerMethods()
	listMethod := api.ListPluginsMethod()
	hasRPC := listMethod != ""

	s.mu.Lock()
	client := s.client
	if client == nil {
		s.mu.Unlock()
		return
	}
	if s.updates.cancel != nil {
		s.updates.cancel()
	}
	s.refreshID++
	generation, refreshID := s.generation, s.refreshID
	s.loading = true
	s.err = nil
	s.hasListRPC = hasRPC
	s.listMethod = listMethod
	s.methods = append([]string(nil), methods...)
	s.mu.Unlock()
	s.fire()

	if !hasRPC {
		s.finishRefresh(client, generation, refreshID, nil, nil, "", "", "")
		return
	}

	go func() {
		regNote := ""
		var regs api.PluginRegistries
		if err := client.Call(ctx, "pluginManager/registries", map[string]any{}, &regs); err == nil {
			regNote = regs.Resolved
			if regNote == "" {
				regNote = regs.Registry
			}
			if len(regs.FallbackRegistries) > 0 {
				regNote += " (+" + fmt.Sprint(len(regs.FallbackRegistries)) + " fallback)"
			}
		}
		var fastest api.FastestResult
		if err := client.Call(ctx, "pluginRegistryProbe/fastest", map[string]any{}, &fastest); err == nil && strings.TrimSpace(fastest) != "" {
			if regNote != "" {
				regNote += " · "
			}
			regNote += "fastest=" + strings.TrimSpace(fastest)
		}
		exNote := ""
		var ex api.ListVersionExemptionsResult
		if err := client.Call(ctx, "pluginManager/listVersionExemptions", map[string]any{}, &ex); err == nil {
			n := 0
			for _, vers := range ex.Exemptions {
				n += len(vers)
			}
			if n > 0 {
				exNote = fmt.Sprintf("%d package(s)", n)
			} else if len(ex.Warnings) > 0 {
				exNote = strings.Join(ex.Warnings, "; ")
			} else {
				exNote = "none"
			}
		}
		var bundles api.ListBundlesResult
		if err := client.Call(ctx, "pluginManager/listBundles", map[string]any{}, &bundles); err == nil {
			rows := entriesFromBundles(bundles)
			s.finishRefresh(client, generation, refreshID, nil, rows, "", regNote, exNote)
			return
		}
		var raw json.RawMessage
		if err := client.Call(ctx, "pluginManager/listPlugins", nil, &raw); err != nil {
			s.finishRefresh(client, generation, refreshID, err, nil, "", regNote, exNote)
			return
		}
		rows, preview := parsePluginList(raw)
		s.finishRefresh(client, generation, refreshID, nil, rows, preview, regNote, exNote)
	}()
}

func entriesFromBundles(bundles api.ListBundlesResult) []Entry {
	out := make([]Entry, 0, len(bundles))
	for _, b := range bundles {
		en := b.Enabled
		detail := ""
		if b.Version != nil && *b.Version != "" {
			detail = *b.Version
		}
		if b.Source != nil && *b.Source != "" {
			if detail != "" {
				detail += " · "
			}
			detail += *b.Source
		}
		if b.Installed {
			if detail != "" {
				detail += " · "
			}
			detail += "installed"
		}
		if !b.Enabled {
			if detail != "" {
				detail += " · "
			}
			detail += "disabled"
		}
		if b.Description != nil && *b.Description != "" {
			if detail != "" {
				detail += " · "
			}
			detail += *b.Description
		}
		src := ""
		if b.Source != nil {
			src = *b.Source
		}
		out = append(out, Entry{
			Label:         b.Name,
			Detail:        detail,
			Enabled:       &en,
			Removable:     b.Removable,
			Source:        src,
			Version:       valueOrEmpty(b.Version),
			Installed:     b.Installed,
			IdentityKnown: true,
		})
	}
	return out
}

func (s *Store) finishRefresh(client *dsh.Client, generation, refreshID uint64, err error, rows []Entry, preview, regNote, exNote string) {
	s.mu.Lock()
	if s.client != client || s.generation != generation || s.refreshID != refreshID {
		s.mu.Unlock()
		return
	}
	s.loading = false
	s.err = err
	if err == nil {
		s.rows = rows
		s.rawListPreview = preview
		s.registriesNote = regNote
		s.exemptionsNote = exNote
	}
	s.mu.Unlock()
	s.fire()
}

func parsePluginList(raw json.RawMessage) ([]Entry, string) {
	preview := truncateJSON(string(raw), 360)
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		if rows := entriesFromObjectList(raw); len(rows) > 0 {
			return rows, preview
		}
		return nil, preview
	}
	if len(arr) == 0 {
		return nil, preview
	}
	return entriesFromArray(arr), preview
}

func entriesFromObjectList(raw json.RawMessage) []Entry {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil
	}
	for _, key := range []string{"plugins", "items", "list"} {
		if chunk, ok := top[key]; ok {
			var arr []json.RawMessage
			if err := json.Unmarshal(chunk, &arr); err == nil {
				if rows := entriesFromArray(arr); len(rows) > 0 {
					return rows
				}
			}
		}
	}
	return nil
}

func entriesFromArray(arr []json.RawMessage) []Entry {
	out := make([]Entry, 0, len(arr))
	for i, item := range arr {
		label, detail := labelDetailFromObject(item)
		if label == "" {
			label = fmt.Sprintf("#%d", i+1)
		}
		out = append(out, Entry{Label: label, Detail: detail})
	}
	return out
}

func labelDetailFromObject(raw json.RawMessage) (string, string) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return truncateJSON(string(raw), 80), ""
	}
	labelKeys := []string{"displayName", "name", "id", "packageName", "package", "spec"}
	var label string
	for _, k := range labelKeys {
		if v, ok := obj[k]; ok {
			label = jsonString(v)
			if label != "" {
				break
			}
		}
	}
	detailKeys := []string{"version", "source", "kind", "enabled", "status"}
	var detail string
	for _, k := range detailKeys {
		if v, ok := obj[k]; ok {
			part := jsonString(v)
			if part != "" {
				if detail != "" {
					detail += " · "
				}
				detail += part
			}
		}
	}
	return label, detail
}

func jsonString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		if b {
			return "true"
		}
		return "false"
	}
	return stringsTrim(string(raw))
}

func stringsTrim(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func truncateJSON(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func (s *Store) fire() {
	s.mu.Lock()
	fn := s.notify
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// RuntimeToken changes on every attachment, including reconnect to the same client.
func (s *Store) RuntimeToken() uint64 { s.mu.Lock(); defer s.mu.Unlock(); return s.generation }

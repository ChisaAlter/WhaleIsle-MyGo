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
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/pluginmanager"
	"whaleisle/internal/state"
)

// Entry is one row in the installed-plugin list.
type Entry struct {
	Label                    string
	Detail                   string
	Title                    json.RawMessage
	Official                 bool
	Enabled                  *bool  // from listBundles when present
	Removable                bool   // from listBundles when present
	Source                   string // listBundles source or empty
	Version                  string
	Installed                bool
	IdentityKnown            bool
	ReadOnlyReason           string
	Error                    string
	CapabilityKind           desktopmarket.CapabilityKind
	NativeCompatible         bool
	NativeCompatibilityLabel string
}

// ViewState is a UI-safe snapshot of the store. A published ViewState is
// immutable: Rows and Methods are replaced wholesale, never edited in place.
type ViewState struct {
	ClientReady    bool
	Loading        bool
	Err            *state.Fault
	HasListRPC     bool
	ListMethod     string
	Methods        []string
	Rows           []Entry
	RawListPreview string
	RegistriesNote string
	ExemptionsNote string
	// Runtime is the attachment's RuntimeToken, published so owners can
	// fence on it without taking the store lock.
	Runtime uint64
}

// Store fetches pluginManager/listPlugins when available. Its working
// fields live under mu; every change publishes an immutable ViewState and
// Discovery version. The runtime lease (RuntimeToken) is renewed on every
// attachment, the inventory lease on every attachment and inventory read;
// catalog lists and details hold their own leases, so a superseded read is
// cancelled and its completion dropped. Local favorites/history writes
// serialize through persistMu and touch the disk without mu held.
type Store struct {
	mu               sync.Mutex
	persistMu        sync.Mutex
	client           *dsh.Client
	inventoryCaller  pluginmanager.Caller
	inventoryVersion uint64
	runtime          state.LeaseOwner
	runtimeL         state.Lease
	inventory        state.LeaseOwner
	inventoryL       state.Lease
	lists            state.LeaseOwner
	listL            state.Lease
	details          state.LeaseOwner
	detailL          state.Lease
	completedRefresh uint64
	discovery        discoveryState
	updates          updateState
	updateProfileDir string
	management       managementState
	personal         personalState
	notify           func()
	pub              state.Published[ViewState]
	disc             state.Published[Discovery]
	now              func() time.Time

	loading         bool
	inventoryLoaded bool
	err             *state.Fault
	hasListRPC      bool
	listMethod      string
	methods         []string
	rows            []Entry
	rawListPreview  string
	registriesNote  string
	exemptionsNote  string
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
	s.runtimeL = s.runtime.Renew(context.Background())
	s.inventoryL = s.inventory.Renew(context.Background())
	s.completedRefresh++
	s.clearConfirmationLocked()
	s.rows = nil
	s.inventoryLoaded = false
	s.loading = false
	s.hasListRPC = false
	s.client = client
	s.inventoryCaller = nil
	s.inventoryVersion = 0
	s.notify = notify
	s.err = nil
	s.publishLocked()
	s.mu.Unlock()
}

// SetInventoryCaller shares Host inventory with the native manager. Independent
// Store users keep the existing direct-client behavior when no source is set.
func (s *Store) SetInventoryCaller(caller pluginmanager.Caller) {
	s.mu.Lock()
	s.inventoryCaller = caller
	s.mu.Unlock()
}

// EnsureInventory is the visible market's demand boundary; a failed read stays
// visible until an explicit Refresh rather than retrying on every frame.
func (s *Store) EnsureInventory(ctx context.Context) {
	s.mu.Lock()
	stale := false
	if source, ok := s.inventoryCaller.(interface{ InventoryVersion() uint64 }); ok {
		stale = s.inventoryVersion != source.InventoryVersion()
	}
	needed := s.client != nil && (!s.inventoryLoaded || stale) && !s.loading && (s.err == nil || stale)
	s.mu.Unlock()
	if needed {
		s.refresh(ctx, false)
	}
}

// RefreshShared reads the generation its owner invalidated for a write/event.
func (s *Store) RefreshShared(ctx context.Context) { s.refresh(ctx, false) }

// InventoryRequested includes initial errors so real events can recover them
// while unused market catalogs stay unloaded.
func (s *Store) InventoryRequested() bool { s.mu.Lock(); defer s.mu.Unlock(); return s.hasListRPC }

// Detach clears the client when harness stops.
func (s *Store) Detach() {
	s.mu.Lock()
	if s.updates.cancel != nil {
		s.updates.cancel()
	}
	s.runtimeL = s.runtime.Renew(context.Background())
	s.inventoryL = s.inventory.Renew(context.Background())
	s.completedRefresh++
	s.clearConfirmationLocked()
	s.client = nil
	s.inventoryCaller = nil
	s.inventoryLoaded = false
	s.loading = false
	s.hasListRPC = false
	s.listMethod = ""
	s.methods = nil
	s.rows = nil
	s.rawListPreview = ""
	s.registriesNote = ""
	s.exemptionsNote = ""
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}

// Snapshot returns the current published version. The same *Version is
// shared by every reader until a change publishes the next revision; its
// Value is immutable.
func (s *Store) Snapshot() *state.Version[ViewState] {
	if v := s.pub.Load(); v != nil {
		return v
	}
	s.mu.Lock()
	if s.pub.Load() == nil {
		s.publishLocked()
	}
	s.mu.Unlock()
	return s.pub.Load()
}

// Changed closes when the next ViewState or Discovery version is published.
func (s *Store) Changed() <-chan struct{} { return s.pub.Changed() }

// publishLocked publishes the inventory view and the discovery projection
// built from the current working state. Callers hold mu.
func (s *Store) publishLocked() {
	s.pub.Publish(ViewState{
		ClientReady:    s.client != nil,
		Loading:        s.loading,
		Err:            s.err,
		HasListRPC:     s.hasListRPC,
		ListMethod:     s.listMethod,
		Methods:        s.methods,
		Rows:           s.rows,
		RawListPreview: s.rawListPreview,
		RegistriesNote: s.registriesNote,
		ExemptionsNote: s.exemptionsNote,
		Runtime:        s.runtimeL.Generation(),
	})
	s.disc.Publish(s.buildDiscoveryLocked())
}

// Refresh probes pluginManager/* registry entries and calls listPlugins when present.
func (s *Store) Refresh(ctx context.Context) {
	s.refresh(ctx, true)
}

func (s *Store) refresh(ctx context.Context, invalidate bool) {
	methods := api.PluginManagerMethods()
	listMethod := api.ListPluginsMethod()
	hasRPC := listMethod != ""

	s.mu.Lock()
	client := s.client
	inventoryCaller := s.inventoryCaller
	if inventoryCaller == nil && client != nil {
		inventoryCaller = client
	}
	if client == nil {
		s.mu.Unlock()
		return
	}
	if !invalidate && s.loading {
		s.mu.Unlock()
		return
	}
	if source, ok := inventoryCaller.(interface{ InvalidateInventory() }); ok && invalidate {
		source.InvalidateInventory()
	}
	if source, ok := inventoryCaller.(interface{ InventoryVersion() uint64 }); ok {
		s.inventoryVersion = source.InventoryVersion()
	}
	if s.updates.cancel != nil {
		s.updates.cancel()
	}
	s.inventoryL = s.inventory.Renew(context.Background())
	runtime, lease := s.runtimeL.Generation(), s.inventoryL
	s.loading = true
	s.err = nil
	s.hasListRPC = hasRPC
	s.listMethod = listMethod
	s.methods = append([]string(nil), methods...)
	s.publishLocked()
	s.mu.Unlock()
	s.fire()

	if !hasRPC {
		s.finishRefresh(client, runtime, lease, nil, nil, "", "", "")
		return
	}

	go func() {
		// A superseding read renews the inventory lease, which cancels this one.
		ctx, cancel := context.WithCancel(ctx)
		stop := context.AfterFunc(lease.Context(), cancel)
		defer func() { stop(); cancel() }()
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
		if err := inventoryCaller.Call(ctx, "pluginManager/listBundles", map[string]any{}, &bundles); err == nil {
			rows := entriesFromBundles(bundles)
			s.finishRefresh(client, runtime, lease, nil, rows, "", regNote, exNote)
			return
		}
		var raw json.RawMessage
		if err := inventoryCaller.Call(ctx, "pluginManager/listPlugins", nil, &raw); err != nil {
			s.finishRefresh(client, runtime, lease, err, nil, "", regNote, exNote)
			return
		}
		rows, preview := parsePluginList(raw)
		s.finishRefresh(client, runtime, lease, nil, rows, preview, regNote, exNote)
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
		var title json.RawMessage
		if b.Meta != nil && len(b.Meta.Title) > 0 {
			title = append(json.RawMessage(nil), b.Meta.Title...)
		}
		caps := classifyBundleEntry(b)
		out = append(out, Entry{
			Label:                    b.Name,
			Detail:                   detail,
			Title:                    title,
			Official:                 b.Official,
			Enabled:                  &en,
			Removable:                b.Removable,
			Source:                   src,
			Version:                  valueOrEmpty(b.Version),
			Installed:                b.Installed,
			IdentityKnown:            true,
			ReadOnlyReason:           valueOrEmpty(b.ReadOnlyReason),
			Error:                    bundleError(b.Error),
			CapabilityKind:           caps.Kind,
			NativeCompatible:         caps.NativeCompatible,
			NativeCompatibilityLabel: caps.CompatibilityLabel,
		})
	}
	return out
}

func classifyBundleEntry(b api.ListBundlesResultItem) desktopmarket.PackageCapabilities {
	rows := make([]pluginmanager.BundleRow, 0, len(b.Rows))
	for _, r := range b.Rows {
		rows = append(rows, pluginmanager.BundleRow{RowID: r.RowId, ModuleName: r.ModuleName})
	}
	desc := ""
	if b.Description != nil {
		desc = *b.Description
	}
	src := ""
	if b.Source != nil {
		src = *b.Source
	}
	hasClient, hasHost := false, false
	for _, r := range rows {
		lower := strings.ToLower(r.ModuleName)
		if strings.Contains(lower, "client-ui-") || strings.HasSuffix(lower, "/client") || strings.Contains(lower, "-client-") {
			hasClient = true
		} else {
			hasHost = true
		}
	}
	var caps pluginmanager.PackageCapabilities
	switch {
	case hasHost && hasClient:
		caps.Kind = pluginmanager.CapabilityHybrid
		caps.NativeCompatible = true
	case hasClient:
		caps.Kind = pluginmanager.CapabilityWebUIOnly
		caps.NativeCompatible = false
	case hasHost || b.Installed:
		caps.Kind = pluginmanager.CapabilityPureHost
		caps.NativeCompatible = true
	default:
		caps = pluginmanager.InferCapabilityFromCatalogMetadata("", b.Name, src, desc)
	}
	if caps.CompatibilityLabel == "" {
		caps.CompatibilityLabel = pluginmanager.CompatibilityLabelFor(caps.Kind)
	}
	return caps
}

func (s *Store) finishRefresh(client *dsh.Client, runtime uint64, lease state.Lease, err error, rows []Entry, preview, regNote, exNote string) {
	s.mu.Lock()
	if s.client != client || s.runtimeL.Generation() != runtime || !lease.Valid() {
		s.mu.Unlock()
		return
	}
	s.loading = false
	s.err = state.FaultOf(err)
	s.inventoryLoaded = err == nil
	s.completedRefresh++
	if err == nil {
		s.rows = rows
		s.rawListPreview = preview
		s.registriesNote = regNote
		s.exemptionsNote = exNote
	}
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	if err == nil {
		s.checkUpdatesWhenReady()
	}
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
func (s *Store) RuntimeToken() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runtimeL.Generation()
}

// CompletedRefreshID increments whenever a host attach, detach, or Refresh completes.
func (s *Store) CompletedRefreshID() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.completedRefresh
}

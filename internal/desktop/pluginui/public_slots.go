package pluginui

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// PublicSlotEntry is one original mounted occurrence or registry-only catalog
// record. Catalog records elect native recipes before lazy owners mount; they
// are never rendered. Native marks an existing native builtin, not a copy.
type PublicSlotEntry struct {
	Slot       string       `json:"slot"`
	OwnerKey   string       `json:"ownerKey,omitempty"`
	Handle     string       `json:"handle"`
	ID         string       `json:"id,omitempty"`
	Registrant string       `json:"registrant"`
	Label      string       `json:"label,omitempty"`
	Order      float64      `json:"order"`
	Native     bool         `json:"native,omitempty"`
	Catalog    bool         `json:"catalog,omitempty"`
	Nodes      []NativeNode `json:"nodes,omitempty"`
}

// PublicMenuAction is a capability for the actual mounted owner, not a label
// turned into a guessed command. Page/revision are attached when read by native.
type PublicMenuAction struct {
	Slot           string  `json:"slot"`
	OwnerKey       string  `json:"ownerKey,omitempty"`
	Handle         string  `json:"handle"`
	EntryHandle    string  `json:"entryHandle,omitempty"`
	ID             string  `json:"id,omitempty"`
	Registrant     string  `json:"registrant,omitempty"`
	Order          float64 `json:"order"`
	Label          string  `json:"label"`
	Disabled       bool    `json:"disabled"`
	Page, Revision uint64  `json:"-"`
	owner          *Owner
}

// PublicContributions is the frame/sidebar/settings seam. Before inserts the
// external bucket immediately before a native builtin ID; "" is the tail.
// NativeWinner gates each native recipe and its callbacks after same-ID election.
// This preserves list order without remounting or duplicating native rows.
type PublicContributions struct {
	Slot             func(*ui.Context, string, string, func())
	Before           func(*ui.Context, string, string, string)
	Entries          func(string, string) []PublicSlotEntry
	RenderEntry      func(*ui.Context, string, string, string) bool
	MenuActions      func(string, string) []PublicMenuAction
	Invoke           func(PublicMenuAction)
	RowContext       func(string, bool, bool)
	WorkspaceContext func(string, bool)
	KeepNative       func(string, string, string) bool
}

func (o *Owner) RenderPublicSlotBefore(c *ui.Context, slot, ownerKey, beforeID string) {
	entries := o.PublicSlotEntries(slot, ownerKey)
	bucket := []PublicSlotEntry{}
	for _, entry := range entries {
		if entry.Native {
			if entry.ID == beforeID {
				for _, external := range bucket {
					o.RenderPublicSlotEntry(c, slot, ownerKey, external.Handle)
				}
				return
			}
			bucket = nil
		} else {
			bucket = append(bucket, entry)
		}
	}
	if beforeID == "" {
		for _, external := range bucket {
			o.RenderPublicSlotEntry(c, slot, ownerKey, external.Handle)
		}
	}
}

var publicSlots = map[string]bool{
	"sidebar.brand.mark": true, "sidebar.brand.name": true, "sidebar.toggle.badge": true,
	"sidebar.panellist": true, "sidebar.footer.action": true, "sidebar.settings": true,
	"sidebar.workspaces": true, "sidebar.workspaces.sections": true,
	"sidebar.session.row.leading": true, "sidebar.session.row.badge": true, "sidebar.session.row.hover": true,
	"sidebar.workspaces.session.row.action": true, "sidebar.workspaces.session.menu.item": true,
	"settings.header": true, "settings.action": true, "settings.close": true, "settings.trigger": true,
	"settings.launcher": true, "settings.launcher.action": true,
	"settings.general.item": true, "settings.interface.item": true, "settings.appearance.item": true,
	"settings.conversation.item": true, "settings.features.item": true, "settings.pet.item": true, "settings.usage.item": true,
}
var publicSingleSlots = map[string]bool{
	"sidebar.brand.mark": true, "sidebar.brand.name": true, "sidebar.toggle.badge": true,
	"sidebar.settings": true, "sidebar.workspaces": true,
	"settings.header": true, "settings.close": true, "settings.trigger": true, "settings.launcher": true,
}

func validPublicProjection(documents []PublicSlotEntry, menus []PublicMenuAction) bool {
	if len(documents) > 4096 || len(menus) > 1024 {
		return false
	}
	seen := map[string]bool{}
	for _, d := range documents {
		if !publicSlots[d.Slot] || d.Handle == "" || len(d.Handle) > 64 || len(d.OwnerKey) > 8192 || len(d.ID) > 8192 || len(d.Registrant) > 8192 || len(d.Label) > 8192 || !validNativeNodes(d.Nodes) || seen[d.Handle] {
			return false
		}
		seen[d.Handle] = true
	}
	seen = map[string]bool{}
	for _, a := range menus {
		if (a.Slot != "settings.launcher.action" && a.Slot != "sidebar.workspaces.session.menu.item") || a.Handle == "" || len(a.Handle) > 64 || len(a.OwnerKey) > 8192 || len(a.EntryHandle) > 64 || len(a.ID) > 8192 || len(a.Registrant) > 8192 || len(a.Label) > 8192 || seen[a.Handle] {
			return false
		}
		seen[a.Handle] = true
	}
	return true
}

func (o *Owner) PublicSlotEntries(slot, ownerKey string) []PublicSlotEntry {
	s := o.Snapshot()
	if !s.Connected || s.Subject == nil || s.Unavailable != "" {
		return nil
	}
	result := []PublicSlotEntry{}
	for _, d := range s.PublicDocuments {
		if !d.Catalog && d.Slot == slot && d.OwnerKey == ownerKey {
			result = append(result, d)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Order < result[j].Order })
	return result
}

func (o *Owner) RenderPublicSlotEntry(c *ui.Context, slot, ownerKey, handle string) bool {
	s := o.Snapshot()
	s.Documents = nil
	s.Outlets = nil
	for _, d := range s.PublicDocuments {
		if !d.Catalog && !d.Native && d.Slot == slot && d.OwnerKey == ownerKey && (handle == "" || d.Handle == handle) {
			s.Documents = append(s.Documents, NativeDocument{Slot: slot, Handle: d.Handle, Nodes: d.Nodes})
		}
	}
	return o.renderNativeSnapshot(c, s, slot, "", -1, -1, "")
}

// RenderPublicSlot preserves single election and list membership. For a list
// containing native builtin rows, use PublicSlotEntries at the row anchors to
// interleave by the original entry order rather than appending to a whole page.
func (o *Owner) RenderPublicSlot(c *ui.Context, slot, ownerKey string, fallback func()) {
	if publicSingleSlots[slot] {
		for _, entry := range o.PublicSlotEntries(slot, ownerKey) {
			if !entry.Native {
				o.RenderPublicSlotEntry(c, slot, ownerKey, entry.Handle)
				return
			}
		}
		if fallback != nil {
			fallback()
		}
		return
	}
	if fallback != nil {
		fallback()
	}
	o.RenderPublicSlotEntry(c, slot, ownerKey, "")
}

func (o *Owner) PublicMenuActions(slot, ownerKey string) []PublicMenuAction {
	s := o.Snapshot()
	if !s.Connected || s.Subject == nil || s.Unavailable != "" {
		return nil
	}
	result := []PublicMenuAction{}
	for _, a := range s.PublicMenus {
		if a.Slot == slot && a.OwnerKey == ownerKey {
			a.Page, a.Revision, a.owner = s.Page, s.Revision, o
			result = append(result, a)
		}
	}
	return result
}
func (o *Owner) InvokePublicMenuAction(action PublicMenuAction) error {
	if action.owner != o {
		return errors.New("plugin UI: public menu owner changed")
	}
	s := o.Snapshot()
	if s.Page != action.Page || s.Revision != action.Revision {
		return errors.New("plugin UI: public menu owner changed")
	}
	found := false
	for _, a := range s.PublicMenus {
		if a.Slot == action.Slot && a.OwnerKey == action.OwnerKey && a.Handle == action.Handle && !a.Disabled {
			found = true
			break
		}
	}
	if !found {
		return errors.New("plugin UI: public menu action withdrawn")
	}
	o.mu.Lock()
	page, valid := o.page, !o.closed && !o.masked && !o.incompatible && s.Subject != nil && o.selected.equal(*s.Subject)
	o.mu.Unlock()
	if !valid {
		return errors.New("plugin UI: public menu owner unavailable")
	}
	return o.sendToPage(page, map[string]any{"type": "publicAction", "revision": s.Revision, "subject": s.Subject, "handle": action.Handle, "slot": action.Slot, "ownerKey": action.OwnerKey})
}

type publicRowContext struct {
	page, frame, sentPage            uint64
	menu, hover, sentMenu, sentHover bool
}

func (o *Owner) SetPublicRowContext(sessionID string, menuOpen, hoverOpen bool) {
	if o == nil || sessionID == "" {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	page := o.page
	if o.closed || o.masked || page == nil {
		return
	}
	if o.publicContexts == nil {
		o.publicContexts = map[string]publicRowContext{}
	}
	state := o.publicContexts[sessionID]
	if state.frame != o.rich.frame || state.page != page.id {
		state.menu, state.hover = false, false
	}
	state.page, state.frame = page.id, o.rich.frame
	state.menu, state.hover = state.menu || menuOpen, state.hover || hoverOpen
	o.publicContexts[sessionID] = state
}
func (o *Owner) flushPublicRowContexts() {
	o.mu.Lock()
	page, frame := o.page, o.rich.frame
	updates := []map[string]any{}
	for id, state := range o.publicContexts {
		if state.frame != frame {
			state.menu, state.hover = false, false
		}
		if page != nil && (state.sentPage != page.id || state.sentMenu != state.menu || state.sentHover != state.hover) {
			updates = append(updates, map[string]any{"type": "publicRowContext", "sessionId": id, "menu": state.menu, "hover": state.hover})
			state.sentPage, state.sentMenu, state.sentHover = page.id, state.menu, state.hover
		}
		if state.frame != frame {
			delete(o.publicContexts, id)
		} else {
			o.publicContexts[id] = state
		}
	}
	o.mu.Unlock()
	for _, update := range updates {
		_ = o.sendToPage(page, update)
	}
}

// Keep the original WorkspaceBrowser's local search/archive owner aligned with
// the native view so its child slots receive the actual query and mounted rows.
func (o *Owner) SetPublicWorkspaceContext(query string, archived bool) {
	if o == nil {
		return
	}
	o.mu.Lock()
	page := o.page
	if o.closed || o.masked || page == nil {
		o.mu.Unlock()
		return
	}
	context := map[string]any{"type": "publicWorkspaceContext", "query": query, "archived": archived}
	encoded, _ := json.Marshal(context)
	key := strconv.FormatUint(page.id, 10) + ":" + string(encoded)
	changed := o.publicWorkspaceContext != key
	if changed {
		o.publicWorkspaceContext = key
	}
	o.mu.Unlock()
	if changed {
		_ = o.sendToPage(page, context)
	}
}

// PublicMenuActionCurrent also guards native replacements of one exact shipped
// action: a menu from an earlier Host/page cannot open a new owner's popup.
func (o *Owner) PublicMenuActionCurrent(action PublicMenuAction) bool {
	if o == nil || action.owner != o {
		return false
	}
	for _, current := range o.PublicMenuActions(action.Slot, action.OwnerKey) {
		if current.Handle == action.Handle && current.Page == action.Page && current.Revision == action.Revision && !current.Disabled {
			o.mu.Lock()
			valid := !o.closed && !o.masked && !o.incompatible
			o.mu.Unlock()
			return valid
		}
	}
	return false
}

// NativeWinner applies list ID shadowing to native recipes, not only to the
// projected external fragments. A missing private source keeps existing native
// callers usable; a known external winner suppresses that builtin everywhere.
func (p PublicContributions) NativeWinner(slot, ownerKey, id string) bool {
	if p.KeepNative != nil {
		return p.KeepNative(slot, ownerKey, id)
	}
	if p.Entries != nil {
		for _, entry := range p.Entries(slot, ownerKey) {
			if entry.ID == id {
				return entry.Native
			}
		}
	}
	return true
}

func (o *Owner) PublicNativeWinner(slot, ownerKey, id string) bool {
	s := o.Snapshot()
	if !s.Connected || s.Subject == nil || s.Unavailable != "" {
		return true
	}
	for _, entry := range s.PublicDocuments {
		if !entry.Catalog && entry.Slot == slot && entry.OwnerKey == ownerKey && entry.ID == id {
			return entry.Native
		}
	}
	// Registry-only metadata is available before a menu opens, without mounting
	// its component or fabricating per-session owner props.
	for _, entry := range s.PublicDocuments {
		if entry.Catalog && entry.Slot == slot && entry.ID == id {
			return entry.Native
		}
	}
	return true
}

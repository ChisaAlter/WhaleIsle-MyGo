package pluginui

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/desktop/webview"
)

const bridgePath = "/__whale_plugin_ui/events"
const bridgeProtocol = "whaleisle.plugin-ui.v1"

var slotNames = map[string]bool{
	"plugins.tab":  true,
	"plugins.item": true, "plugins.bundle.config": true, "plugins.row.config": true,
	"plugins.detail.actions": true, "plugins.detail.badge": true, "plugins.detail.section": true,
	"plugins.add.actions": true, "plugins.bundle.activation": true,
}

type Subject struct {
	Kind      string `json:"kind"`
	Name      string `json:"name,omitempty"`
	RowID     string `json:"rowId,omitempty"`
	ItemID    string `json:"itemId,omitempty"`
	EntryID   string `json:"entryId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Epoch     string `json:"epoch,omitempty"`
}

func (s Subject) equal(other Subject) bool {
	return s.Kind == other.Kind && s.Name == other.Name && s.RowID == other.RowID && s.ItemID == other.ItemID && s.SessionID == other.SessionID && s.Epoch == other.Epoch
}

// Entry is a winning live SlotRegistry entry, not a package manifest claim.
// No component, inject function, script, URL or RPC method crosses this channel.
type Entry struct {
	Slot       string `json:"slot"`
	ID         string `json:"id"`
	Key        string `json:"key"`
	Label      string `json:"label"`
	Registrant string `json:"registrant"`
	Order      int    `json:"order"`
	Summary    string `json:"summary"`
}

type Action struct {
	Handle   string `json:"handle"`
	Label    string `json:"label"`
	Disabled bool   `json:"disabled"`
}

type Snapshot struct {
	Connected             bool                       `json:"-"`
	Page                  uint64                     `json:"-"`
	Revision              uint64                     `json:"revision"`
	Locale                string                     `json:"locale"`
	Entries               []Entry                    `json:"entries"`
	Subject               *Subject                   `json:"subject"`
	Navigation            *Subject                   `json:"navigation,omitempty"`
	Actions               []Action                   `json:"actions"`
	Badges                []string                   `json:"badges"`
	Section               bool                       `json:"section"`
	Configuration         bool                       `json:"configuration"`
	Surface               string                     `json:"surface"`
	SessionSurfaceHandle  string                     `json:"sessionSurfaceHandle,omitempty"`
	SessionSurfaceRequest string                     `json:"sessionSurfaceRequest,omitempty"`
	Height                int                        `json:"height"`
	Error                 string                     `json:"-"`
	Unavailable           string                     `json:"unavailable,omitempty"`
	Outlets               []SessionOutlet            `json:"outlets,omitempty"`
	SessionControls       []SessionControl           `json:"sessionControls,omitempty"`
	Observation           bool                       `json:"observation,omitempty"`
	RightPaneTabs         []RightPaneTab             `json:"rightPaneTabs,omitempty"`
	RightPaneKinds        []RightPaneKind            `json:"rightPaneKinds,omitempty"`
	ConversationViews     []ConversationView         `json:"conversationViews,omitempty"`
	Views                 []NativeView               `json:"views,omitempty"`
	GlobalDocuments       []NativeDocument           `json:"globalDocuments,omitempty"`
	Documents             []NativeDocument           `json:"documents,omitempty"`
	ModelDocuments        []NativeDocument           `json:"modelDocuments,omitempty"`
	PublicDocuments       []PublicSlotEntry          `json:"publicDocuments,omitempty"`
	PublicMenus           []PublicMenuAction         `json:"publicMenus,omitempty"`
	DirectoryActions      []WorkspaceDirectoryAction `json:"directoryActions,omitempty"`
	WorkspaceNavigation   []WorkspaceNavigation      `json:"workspaceNavigation,omitempty"`
}

type bridgePage struct {
	socket *websocket.Conn
	id     uint64
}

// Owner owns one temporary exact-Host gateway and one binding-free browser.
// The caller must close it before changing Host generation/session or route.
type Owner struct {
	mu                     sync.Mutex
	gateway                *pluginslots.Gateway
	host                   *webview.Host
	page                   *bridgePage
	nextPage               uint64
	closed                 bool
	masked                 bool
	incompatible           bool
	state                  Snapshot
	selected               Subject
	changed                func()
	composer               ComposerBridge
	composerLast           string
	composerError          string
	off                    []func()
	mirror                 *mirrorState
	rich                   richState
	rightPane              rightPaneState
	workspaceOps           workspaceOperationState
	navigationSession      string
	modelTargets           map[string]bool
	modelTargetsLast       string
	publicContexts         map[string]publicRowContext
	publicWorkspaceContext string
}

type Options struct {
	Parent           *mygo.Window
	Origin, Cookie   string
	Changed          func()
	OnFailure        func(webview.PageFailure)
	Composer         ComposerBridge
	PermissionPolicy func(mygo.PermissionRequest) bool
	Session          bool
	Navigation       bool
	// Inline retains the full original Session page as a single hidden source.
	Inline bool
}

func New(opts Options) (*Owner, error) {
	if opts.Parent == nil {
		return nil, errors.New("plugin UI: parent window required")
	}
	if opts.Inline && !opts.Session {
		return nil, errors.New("plugin UI: inline source requires a Session")
	}
	o := &Owner{changed: opts.Changed, composer: opts.Composer, selected: Subject{Kind: "list"}}
	if opts.Inline {
		o.mirror = &mirrorState{}
	}
	var responseFilter func(*http.Response) error
	if opts.Session || opts.Navigation {
		responseFilter = o.sessionResponse
	} else {
		responseFilter = nativePadOnlyResponse
	}
	gateway, err := pluginslots.NewGatewayWithResponseFilter(opts.Origin, opts.Cookie, o.serve, responseFilter)
	if err != nil {
		return nil, err
	}
	o.gateway = gateway
	gateway.SetClientModulesChanged(opts.Changed)
	mode := "desktopNativePlugins"
	if opts.Session {
		mode = "desktopNativeSession"
		o.selected = Subject{Kind: "session"}
	}
	raw := strings.TrimRight(opts.Origin, "/") + "/?" + mode + "=1"
	if opts.Navigation {
		raw = strings.TrimRight(opts.Origin, "/") + "/?desktopNativeNavigation=1"
		o.selected = Subject{Kind: "navigation"}
	}
	if opts.Inline {
		raw += "&desktopNativeInline=1"
	}
	entry, err := gateway.EntryURL(raw)
	if err != nil {
		gateway.Close()
		return nil, err
	}
	preload := ""
	if opts.Inline {
		preload = string(mirrorTextGuardSource) + "\n" + string(mirrorPortalGuardSource) + "\n" + string(mirrorGuardSource)
	}
	host, err := webview.NewPluginSurface(webview.HostOptions{
		PreloadScript: preload,
		Parent:        opts.Parent, URL: entry, Title: "Plugin contributions", Untrusted: true,
		AllowNavigation: gateway.AllowsBrowserURL, OnChange: opts.Changed, OnClosed: o.Close,
		PermissionPolicy: opts.PermissionPolicy, OnFailure: opts.OnFailure,
	}, gateway)
	if err != nil {
		gateway.Close()
		return nil, err
	}
	o.host = host
	o.off = []func(){opts.Parent.OnClosed(o.Close), opts.Parent.OnHide(o.Close), opts.Parent.OnMinimize(o.Close)}
	return o, nil
}

func (o *Owner) Alive() bool {
	if o == nil {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return !o.closed && o.host != nil && o.host.Alive()
}

func (o *Owner) Snapshot() Snapshot {
	if o == nil {
		return Snapshot{}
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	s := o.state
	if o.incompatible {
		s.Subject = nil
		s.Actions = nil
		s.SessionControls = nil
		s.Outlets = nil
		s.Documents = nil
		s.GlobalDocuments = nil
		s.ConversationViews = nil
		s.RightPaneTabs = nil
		s.RightPaneKinds = nil
		s.Surface = ""
		s.SessionSurfaceHandle = ""
		s.SessionSurfaceRequest = ""
		s.Unavailable = "session-version-incompatible"
	}
	s.Entries = append([]Entry(nil), s.Entries...)
	s.Actions = append([]Action(nil), s.Actions...)
	s.Badges = append([]string(nil), s.Badges...)
	s.Outlets = append([]SessionOutlet(nil), s.Outlets...)
	s.SessionControls = append([]SessionControl(nil), s.SessionControls...)
	s.Documents = append([]NativeDocument(nil), s.Documents...)
	s.GlobalDocuments = append([]NativeDocument(nil), s.GlobalDocuments...)
	s.ModelDocuments = append([]NativeDocument(nil), s.ModelDocuments...)
	s.PublicDocuments = append([]PublicSlotEntry(nil), s.PublicDocuments...)
	s.PublicMenus = append([]PublicMenuAction(nil), s.PublicMenus...)
	s.DirectoryActions = append([]WorkspaceDirectoryAction(nil), s.DirectoryActions...)
	s.WorkspaceNavigation = append([]WorkspaceNavigation(nil), s.WorkspaceNavigation...)
	s.Views = append([]NativeView(nil), s.Views...)
	s.ConversationViews = append([]ConversationView(nil), s.ConversationViews...)
	s.RightPaneTabs = append([]RightPaneTab(nil), s.RightPaneTabs...)
	s.RightPaneKinds = append([]RightPaneKind(nil), s.RightPaneKinds...)
	if s.Subject != nil {
		subject := *s.Subject
		s.Subject = &subject
	}
	return s
}

func (o *Owner) notify() {
	if o.changed != nil {
		o.changed()
	}
}

func (o *Owner) send(value any) error {
	o.mu.Lock()
	page, closed := o.page, o.closed
	o.mu.Unlock()
	if closed || page == nil {
		return errors.New("plugin UI: page unavailable")
	}
	return o.sendToPage(page, value)
}

func (o *Owner) sendToPage(page *bridgePage, value any) error {
	if page == nil {
		return errors.New("plugin UI: page unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, page.socket, value); err != nil {
		o.disconnect(page)
		return errors.New("plugin UI: page disconnected")
	}
	return nil
}

// Select only navigates through the original Plugins owner and its controls.
func (o *Owner) Select(subject Subject) {
	if o == nil {
		return
	}
	o.mu.Lock()
	changed := !o.selected.equal(subject)
	o.selected = subject
	if changed {
		o.retireWorkspaceOperationsLocked()
		o.retireRightPaneOpensLocked()
		o.composerError = ""
		o.retireMirrorTextLocked()
		o.invalidateMirrorLocked()
		o.state.Subject = nil
		o.state.Actions = nil
		o.state.Badges = nil
		o.state.Section = false
		o.state.Configuration = false
		o.state.Surface = ""
		o.state.SessionSurfaceHandle = ""
		o.state.SessionSurfaceRequest = ""
		o.state.Outlets = nil
		o.state.Documents = nil
		o.state.GlobalDocuments = nil
		o.state.ConversationViews = nil
		o.state.RightPaneTabs = nil
		o.state.RightPaneKinds = nil
		o.state.SessionControls = nil
		o.state.Unavailable = ""
	}
	o.mu.Unlock()
	if changed {
		o.Hide()
		_ = o.send(map[string]any{"type": "select", "subject": subject})
	}
}

// TakeNavigation consumes original activation Open details or tab openPackage.
// The native manager still verifies that this package exists in its inventory.
func (o *Owner) TakeNavigation() *Subject {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed || (o.selected.Kind != "list" && o.selected.Kind != "pluginTab") || o.state.Navigation == nil {
		return nil
	}
	next := *o.state.Navigation
	o.state.Navigation = nil
	return &next
}

// Click addresses a control in the captured page/revision/subject, never a
// renderer-supplied RPC string. The browser rechecks connectivity and DOM owner.
func (o *Owner) Click(pageID, revision uint64, subject Subject, handle string) error {
	if o == nil {
		return errors.New("plugin UI: page unavailable")
	}
	o.mu.Lock()
	s := o.state
	valid := !o.closed && s.Connected && s.Page == pageID && s.Revision == revision && s.Subject != nil && s.Subject.equal(subject) && o.selected.equal(subject)
	found := false
	for _, a := range s.Actions {
		if a.Handle == handle && !a.Disabled {
			found = true
			break
		}
	}
	o.mu.Unlock()
	if !valid || !found {
		return errors.New("plugin UI: action is no longer available")
	}
	return o.send(map[string]any{"type": "click", "revision": revision, "handle": handle})
}

func (o *Owner) Show(slot string) {
	if o == nil || (slot != "" && !slotNames[slot] && !sessionSlotNames[slot]) {
		return
	}
	_ = o.send(map[string]any{"type": "surface", "slot": slot})
	if slot == "" {
		o.Hide()
	}
}

// EnableActivation delegates the one list enable gesture to the original page.
// It never follows a native RPC, so the original switch writes exactly once.
func (o *Owner) EnableActivation(page, revision uint64, name string) error {
	if o == nil {
		return errors.New("plugin UI: page unavailable")
	}
	o.mu.Lock()
	valid := !o.closed && o.state.Connected && o.state.Page == page && o.state.Revision == revision && o.selected.Kind == "list"
	registered := false
	for _, entry := range o.state.Entries {
		if entry.Slot == "plugins.bundle.activation" && entry.Key == name {
			registered = true
			break
		}
	}
	o.mu.Unlock()
	if !valid || !registered {
		return errors.New("plugin UI: activation is no longer available")
	}
	return o.send(map[string]any{"type": "enableActivation", "revision": revision, "name": name})
}

func (o *Owner) AddActions() {
	if o != nil {
		_ = o.send(map[string]any{"type": "add"})
	}
}

// Present accepts the full viewport and its independent native paint clip.
// Scroll clipping removes pixels and input without resizing/reflowing the DOM.
func (o *Owner) Present(rect, clip mygo.Rectangle, holes []mygo.Rectangle) {
	if o == nil {
		return
	}
	o.mu.Lock()
	h, capability, current := o.host, o.gateway, !o.closed && !o.incompatible && !o.masked && o.state.Surface != ""
	o.mu.Unlock()
	if h == nil {
		return
	}
	if !current || rect.Width <= 0 || rect.Height <= 0 {
		h.SetVisible(false)
		return
	}
	if _, err := h.PresentPlugin(rect, clip, holes, capability); err != nil {
		o.mu.Lock()
		o.state.Error = err.Error()
		o.mu.Unlock()
		h.SetVisible(false)
	}
}

// Mask keeps native menus and dialogs in front of browser pixels and input.
func (o *Owner) Mask(masked bool) {
	if o == nil {
		return
	}
	o.mu.Lock()
	o.masked = masked
	if masked {
		o.retireMirrorTextLocked()
		o.invalidateMirrorLocked()
	}
	o.mu.Unlock()
	if masked {
		o.Hide()
	}
}

func (o *Owner) Hide() {
	if o == nil {
		return
	}
	o.mu.Lock()
	h := o.host
	o.mu.Unlock()
	if h != nil {
		h.SetVisible(false)
	}
}

func (o *Owner) Close() {
	if o == nil {
		return
	}
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return
	}
	o.closed = true
	o.retireWorkspaceOperationsLocked()
	o.retireRightPaneOpensLocked()
	o.retireMirrorTextLocked()
	o.invalidateMirrorLocked()
	h, gateway, page, off := o.host, o.gateway, o.page, o.off
	o.host, o.gateway, o.page, o.off = nil, nil, nil, nil
	o.state = Snapshot{}
	o.mu.Unlock()
	if page != nil {
		_ = page.socket.CloseNow()
	}
	if gateway != nil {
		gateway.Close()
	}
	if h != nil {
		_ = h.Close()
	}
	for _, f := range off {
		f()
	}
}

func (o *Owner) disconnect(expected *bridgePage) {
	o.mu.Lock()
	if o.page != expected {
		o.mu.Unlock()
		return
	}
	o.page = nil
	o.retireWorkspaceOperationsLocked()
	o.retireRightPaneOpensLocked()
	o.retireMirrorTextLocked()
	o.invalidateMirrorLocked()
	o.state = Snapshot{}
	o.mu.Unlock()
	_ = expected.socket.CloseNow()
	o.Hide()
	o.notify()
}

func (o *Owner) serve(w http.ResponseWriter, r *http.Request) bool {
	if o.serveComposerInsert(w, r) {
		return true
	}
	if o.serveComposerFiles(w, r) {
		return true
	}
	if r.URL.Path != bridgePath {
		return false
	}
	origin := "http://" + r.Host
	if r.Method != http.MethodGet || r.Header.Get("Origin") != origin || r.URL.RawQuery != "" || !hasProtocol(r.Header.Get("Sec-WebSocket-Protocol")) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return true
	}
	socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{bridgeProtocol}})
	if err != nil {
		return true
	}
	socket.SetReadLimit(512 * 1024)
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		_ = socket.CloseNow()
		return true
	}
	old := o.page
	o.retireWorkspaceOperationsLocked()
	o.retireRightPaneOpensLocked()
	o.retireMirrorTextLocked()
	o.nextPage++
	o.modelTargetsLast = ""
	page := &bridgePage{socket: socket, id: o.nextPage}
	o.page = page
	o.state = Snapshot{}
	o.composerLast = ""
	o.composerError = ""
	subject := o.selected
	navigationSession := o.navigationSession
	o.mu.Unlock()
	if old != nil {
		_ = old.socket.CloseNow()
	}
	defer o.disconnect(page)
	_ = o.send(map[string]any{"type": "select", "subject": subject})
	if navigationSession != "" {
		_ = o.send(map[string]any{"type": "navigationContext", "sessionId": navigationSession})
	}
	for {
		var message struct {
			Type string `json:"type"`
			Snapshot
			ComposerRequest *ComposerRequest `json:"composer"`
			Token           uint64           `json:"token"`
			OK              bool             `json:"ok"`
			Request         string           `json:"request"`
			Handle          string           `json:"handle"`
			OpenError       string           `json:"error"`
		}
		if err := wsjson.Read(r.Context(), socket, &message); err != nil {
			return true
		}
		if message.Type == "composer" {
			o.handleComposer(page, message.ComposerRequest)
			continue
		}
		if message.Type == "richAck" {
			o.ackRichLayout(page, message.Token, message.Revision, message.OK)
			continue
		}
		if message.Type == "rightPaneOpened" {
			if len(message.Request) > 64 || len(message.Handle) > 64 || len(message.OpenError) > 8192 {
				return true
			}
			o.finishRightPaneOpen(page, message.Request, message.Handle, message.OpenError, message.Subject)
			continue
		}
		if message.Type == "workspaceResult" {
			if len(message.Request) > 64 || len(message.OpenError) > 8192 {
				return true
			}
			o.finishWorkspaceOperation(page, message.Request, message.OpenError)
			continue
		}
		if message.Type != "state" || !validSnapshot(message.Snapshot) {
			return true
		}
		o.mu.Lock()
		if o.closed || o.page != page {
			o.mu.Unlock()
			return true
		}
		if message.Revision <= o.state.Revision {
			o.mu.Unlock()
			continue
		}
		s := message.Snapshot
		if o.incompatible {
			s.Subject = nil
			s.Outlets = nil
			s.Documents = nil
			s.GlobalDocuments = nil
			s.ConversationViews = nil
			s.RightPaneTabs = nil
			s.RightPaneKinds = nil
			s.Actions = nil
			s.SessionControls = nil
			s.Surface = ""
			s.SessionSurfaceHandle = ""
			s.SessionSurfaceRequest = ""
			s.Unavailable = "session-version-incompatible"
		}
		s.Connected = true
		s.Page = page.id
		if s.Subject == nil || !s.Subject.equal(o.selected) {
			s.Subject = nil
			s.SessionSurfaceHandle = ""
			s.SessionSurfaceRequest = ""
			s.Actions = nil
			s.Badges = nil
			s.Section = false
			s.Configuration = false
			s.Outlets = nil
			s.Documents = nil
			s.GlobalDocuments = nil
			s.ConversationViews = nil
			s.RightPaneTabs = nil
			s.RightPaneKinds = nil
			s.SessionControls = nil
			if o.selected.Kind != "list" || (s.Surface != "plugins.add.actions" && s.Surface != "plugins.bundle.activation") {
				s.Surface = ""
			}
		}
		if o.selected.Kind != "list" && o.selected.Kind != "pluginTab" {
			s.Navigation = nil
		}
		o.state = s
		o.invalidateMirrorLocked()
		o.mu.Unlock()
		if s.Surface == "" {
			o.Hide()
		}
		o.notify()
	}
}

func hasProtocol(value string) bool {
	for _, item := range strings.Split(value, ",") {
		if strings.TrimSpace(item) == bridgeProtocol {
			return true
		}
	}
	return false
}

func validSnapshot(s Snapshot) bool {
	if !validPublicProjection(s.PublicDocuments, s.PublicMenus) {
		return false
	}
	if !validWorkspaceProjection(s.DirectoryActions, s.WorkspaceNavigation) {
		return false
	}
	if len(s.ModelDocuments) > 1024 {
		return false
	}
	for _, document := range s.ModelDocuments {
		if (document.Slot != "settings.models.provider-card" && document.Slot != "settings.models.footer") || len(document.Handle) > 16384 || len(document.Provider) > 8192 || len(document.Namespace) > 8192 || !validNativeNodes(document.Nodes) {
			return false
		}
	}
	if !validRightPaneKinds(s.RightPaneKinds) {
		return false
	}
	if !validRightPaneTabs(s.RightPaneTabs, s.Outlets) {
		return false
	}
	if !validConversationViews(s.ConversationViews) {
		return false
	}
	if !validNativeViews(s.Views) {
		return false
	}
	if len(s.GlobalDocuments) > 128 {
		return false
	}
	for _, document := range s.GlobalDocuments {
		if !globalSlots[document.Slot] || len(document.Handle) > 16384 || !validNativeNodes(document.Nodes) {
			return false
		}
	}
	if len(s.Documents) > 16 {
		return false
	}
	for _, document := range s.Documents {
		if (!slotNames[document.Slot] && !navigationSlots[document.Slot]) || !validNativeNodes(document.Nodes) {
			return false
		}
	}
	if s.Navigation != nil && (s.Navigation.Kind != "bundle" || s.Navigation.Name == "" || len(s.Navigation.Name) > 8192) {
		return false
	}
	if s.Revision == 0 || len(s.Entries) > 2048 || len(s.Actions) > 128 || len(s.Badges) > 128 || s.Height < 0 || s.Height > 1000 || (s.Surface != "" && !slotNames[s.Surface] && !sessionSlotNames[s.Surface]) || !validSessionOutlets(s.Outlets) || !validSessionControls(s.SessionControls, s.Outlets) || len(s.Unavailable) > 8192 || len(s.SessionSurfaceHandle) > 64 {
		return false
	}
	if s.SessionSurfaceHandle != "" {
		found := false
		for _, row := range s.Outlets {
			if row.Handle == s.SessionSurfaceHandle && row.Available && row.Slot == s.Surface {
				found = true
			}
		}
		if !found || s.Subject == nil || s.Subject.Kind != "session" || s.Unavailable != "" {
			return false
		}
	}
	if len(s.SessionSurfaceRequest) > 64 || s.SessionSurfaceRequest != "" && s.SessionSurfaceHandle == "" {
		return false
	}
	for _, e := range s.Entries {
		if (!slotNames[e.Slot] && !sessionSlotNames[e.Slot]) || len(e.ID) > 8192 || len(e.Key) > 8192 || len(e.Label) > 8192 || len(e.Registrant) > 8192 || len(e.Summary) > 8192 {
			return false
		}
	}
	for _, a := range s.Actions {
		if a.Handle == "" || len(a.Handle) > 64 || a.Label == "" || len(a.Label) > 8192 {
			return false
		}
	}
	for _, badge := range s.Badges {
		if len(badge) > 8192 {
			return false
		}
	}
	if s.Subject != nil {
		x := s.Subject
		if !((x.Kind == "navigation" && (x.Name == "" || x.Name == "models" || strings.HasPrefix(x.Name, "settings:") && len(x.Name) <= 256)) || (x.Kind == "list" && x.Name == "" && x.ItemID == "" && x.RowID == "") || (x.Kind == "pluginTab" && x.Name != "" && len(x.Name) <= 8192) || (x.Kind == "view" && x.Name != "" && len(x.Name) <= 16384) || (x.Kind == "bundle" && x.Name != "") || (x.Kind == "row" && x.Name != "" && x.RowID != "") || (x.Kind == "item" && x.ItemID != "") || (x.Kind == "session" && x.SessionID != "" && x.Epoch != "" && len(x.SessionID) <= 256 && len(x.Epoch) <= 256)) {
			return false
		}
	}
	return true
}

// LogicalURL is available for diagnostics without exposing the gateway port,
// browser capability or Host credential.
func (o *Owner) LogicalURL() string {
	if o == nil {
		return ""
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.host == nil || o.gateway == nil {
		return ""
	}
	return o.gateway.LogicalURL(o.host.URL())
}

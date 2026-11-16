package assistantctl

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
)

const assistantVendorProbeTTL = 5 * time.Second

type vendorDirCache struct {
	root      string
	gen       uint64
	loading   bool
	checkedAt time.Time
	whale     bool
	dshbot    bool
}

type whaleKeyCache struct {
	client     *dsh.Client
	origin     string
	cookie     string
	generation int
	digest     string
}

// Manager aliases Service for main-shell wiring symmetry across internal/app packages.
type Manager = Service

// Service centrally owns all assistant, Dshbot, room, routine, proactive, and
// Whale controllers and coordinates lifecycle, lock ordering, and host snapshots.
//
// Lock ordering rule: Service.mu is leaf-guarded against RuntimeHost and external
// store mutexes. Never hold Service.mu across RuntimeHost calls, HTTP/RPC/disk I/O,
// or store method calls. When acquiring live-poller locks after Service.mu, the
// strict downward order is Service.mu -> groupLive.mu -> contacts.mu.
type Service struct {
	host RuntimeHost

	mu                     sync.Mutex
	assistant              *assiststore.Store
	dshbotRoster           *dshbotstore.Store
	open                   bool
	conversationGeneration uint64
	draft                  string
	lastAct                string
	pluginBusy             bool
	pluginDetail           string
	vendorProbe            vendorDirCache

	botNotifications        *botNotificationRun
	botNotificationMessage  string
	botNotificationShowing  bool
	botNotificationDenied   bool
	botNotificationReadSlot chan struct{}

	whaleOwnerKey     whaleKeyCache
	whaleNavName      string
	whaleNavKey       string
	whaleNavSerial    uint64
	whaleNavBusy      bool
	whaleNavFetchedAt time.Time

	proactivePanel proactivePanelState

	assistantTasks         dshbotstore.TaskControls
	assistantRoutines      dshbotstore.RoutineControls
	assistantRoutineCreate dshbotstore.RoutineCreator
	assistantRoomCreator   dshbotstore.RoomCreator
	assistantBotVisibility dshbotstore.BotVisibilityManager
	assistantContacts      dshbotstore.ContactControls
	assistantBotCreator    dshbotstore.BotCreator
	assistantBotDeleter    dshbotstore.BotDeleter
	assistantBotDuplicator dshbotstore.BotDuplicator
	assistantRoomEditor    dshbotstore.RoomEditor
	assistantRoomRuntime   dshbotstore.RoomRuntimeReader
	assistantGroupLive     groupLiveState
	whaleSettings          assiststore.WhaleSettingsEditor
	whaleTasks             assiststore.WhaleTaskPanel
}

// NewService constructs an assistant domain Service backed by HostDeps.
func NewService(deps HostDeps) *Service {
	return &Service{host: wrapHostDeps(deps)}
}

// NewManager constructs an assistant domain Manager backed by HostDeps.
func NewManager(deps HostDeps) *Manager {
	return NewService(deps)
}

// NewServiceWithHost constructs an assistant domain Service backed by a RuntimeHost.
func NewServiceWithHost(host RuntimeHost) *Service {
	if host == nil {
		host = wrapHostDeps(HostDeps{})
	}
	return &Service{host: host}
}

func (s *Service) safeHost() RuntimeHost {
	if s == nil || s.host == nil {
		return wrapHostDeps(HostDeps{})
	}
	return s.host
}

// AttachStores attaches cold-started stores after window creation. Any nil store
// argument is ignored safely without panicking or clobbering existing state.
func (s *Service) AttachStores(assist *assiststore.Store, proactiveSvc *proactive.Service, roster *dshbotstore.Store) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if assist != nil {
		s.assistant = assist
	}
	if proactiveSvc != nil {
		s.proactivePanel.service = proactiveSvc
		s.proactivePanel.loadError = ""
	}
	if roster != nil {
		s.dshbotRoster = roster
	}
	s.mu.Unlock()
}

// AttachAssistantStore attaches or replaces the assistant mode/history store.
func (s *Service) AttachAssistantStore(store *assiststore.Store) {
	if s == nil || store == nil {
		return
	}
	s.mu.Lock()
	s.assistant = store
	s.mu.Unlock()
}

// AttachDshbotRoster attaches or replaces the cached Dshbot roster store.
func (s *Service) AttachDshbotRoster(roster *dshbotstore.Store) {
	if s == nil || roster == nil {
		return
	}
	s.mu.Lock()
	s.dshbotRoster = roster
	s.mu.Unlock()
}

// AttachProactive attaches the proactive task service and optional open error.
func (s *Service) AttachProactive(service *proactive.Service, loadErr error) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.proactivePanel.service = service
	if loadErr != nil {
		s.proactivePanel.loadError = loadErr.Error()
	} else {
		s.proactivePanel.loadError = ""
	}
	s.mu.Unlock()
}

// AssistantStore returns the currently attached assistant store (may be nil).
func (s *Service) AssistantStore() *assiststore.Store {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.assistant
}

// EnsureAssistantStore returns the attached store or creates an in-memory fallback.
func (s *Service) EnsureAssistantStore() *assiststore.Store {
	if s == nil {
		return assiststore.New()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.assistant == nil {
		s.assistant = assiststore.New()
	}
	return s.assistant
}

// DshbotRoster returns the attached Dshbot roster store (may be nil).
func (s *Service) DshbotRoster() *dshbotstore.Store {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dshbotRoster
}

// ProactiveService returns the attached proactive service (may be nil).
func (s *Service) ProactiveService() *proactive.Service {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.proactivePanel.service
}

// Open reports whether the assistant panel is currently open.
func (s *Service) Open() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.open
}

// SetOpen toggles assistant panel visibility and bumps conversation generation
// so stale async completions from previous opens cannot hijack navigation.
func (s *Service) SetOpen(open bool) uint64 {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversationGeneration++
	s.open = open
	return s.conversationGeneration
}

// ConversationGeneration returns the current monotonic assistant panel generation.
func (s *Service) ConversationGeneration() uint64 {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conversationGeneration
}

// BumpConversationGeneration increments and returns the conversation generation.
func (s *Service) BumpConversationGeneration() uint64 {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversationGeneration++
	return s.conversationGeneration
}

// Draft returns the current unsent assistant composer draft.
func (s *Service) Draft() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.draft
}

// SetDraft updates the assistant panel draft text.
func (s *Service) SetDraft(draft string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.draft = draft
	s.mu.Unlock()
}

// WhaleNavName returns the cached Whale display label for sidebar navigation.
func (s *Service) WhaleNavName() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.whaleNavName
}

// PluginStatus returns a snapshot of the assistant plugin RPC/open busy state.
func (s *Service) PluginStatus() PluginStatus {
	if s == nil {
		return PluginStatus{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return PluginStatus{
		Busy:       s.pluginBusy,
		Detail:     s.pluginDetail,
		LastAction: s.lastAct,
	}
}

// SuspendViews suspends all interactive Dshbot/room view controllers and scopes
// without stopping background proactive monitoring or bot activity notifications.
func (s *Service) SuspendViews() {
	if s == nil {
		return
	}
	s.assistantBotCreator.Suspend()
	s.assistantBotVisibility.Suspend()
	s.assistantContacts.Suspend()
	s.assistantBotDeleter.Suspend()
	s.assistantBotDuplicator.Suspend()
	s.assistantRoomCreator.Suspend()
	s.assistantRoomEditor.Suspend()
	s.assistantRoomRuntime.Suspend()
	s.assistantRoutines.Bind(dshbotstore.RoutineScope{})
	s.assistantTasks.Bind(dshbotstore.TaskScope{})
}

// SuspendAll unifies suspension across all assistant/Bot/room/Whale/proactive
// controllers, pollers, and timers (used by shutdown, StopDSH, and teardown).
func (s *Service) SuspendAll() {
	if s == nil {
		return
	}
	s.StopProactiveMonitor()
	s.StopProactiveRunner()
	s.StopBotNotifications()
	s.StopGroupRuntime()
	s.whaleSettings.Suspend()
	s.whaleTasks.Suspend()
	s.SuspendViews()
}

// OnWindowHide handles OS window hide events by suspending interactive views.
func (s *Service) OnWindowHide() {
	s.SuspendViews()
}

// OnWindowMinimize handles OS window minimize events by suspending interactive views.
func (s *Service) OnWindowMinimize() {
	s.SuspendViews()
}

// OnWindowClosed retires window-scoped Whale settings, bot notifications,
// group/contact pollers, and interactive views when the main window closes.
func (s *Service) OnWindowClosed() {
	if s == nil {
		return
	}
	s.whaleSettings.Suspend()
	s.whaleTasks.Suspend()
	s.StopBotNotifications()
	s.StopGroupRuntime()
	s.SuspendViews()
}

// OnWindowFocus clears unsubmitted bot reminder queues once the window is focused.
func (s *Service) OnWindowFocus() {
	s.ClearFocusedBotNotifications()
}

// StopDSH disconnects Host-bound proactive runner and interactive Dshbot views.
func (s *Service) StopDSH() {
	if s == nil {
		return
	}
	s.StopProactiveRunner()
	s.StopGroupRuntime()
	s.SuspendViews()
}

// ObserveFrame reconciles per-frame rendering flags and suspends unmounted
// assistant controllers and observers.
func (s *Service) ObserveFrame(obs FrameObservation) {
	if s == nil {
		return
	}
	s.ObserveBotRoute()
	s.ObserveRoomRuntime(obs.RoomRuntimeRendered)
	s.ObserveRoomEditor(obs.RoomEditRendered)
	s.ObserveRoomCreator(obs.RoomCreateRendered)
	s.ObserveBotCreator(obs.RoomCreateRendered)
	s.ObserveBotVisibility(obs.RoomCreateRendered)
	s.ObserveContacts(obs.RoomCreateRendered)
	s.ObserveBotDeleter(obs.RoomCreateRendered)
	s.ObserveBotDuplicator(obs.RoomCreateRendered)
	s.ObserveBotNotifications(obs.NotificationsRendered)
	s.ObserveWhaleSettings()
}

// WhaleAssistantEnabled reports whether desktop configuration enables Whale.
func (s *Service) WhaleAssistantEnabled() bool {
	cfg := s.safeHost().Config()
	return cfg != nil && cfg.Get().WhaleAssistantEnabled
}

// GateNavigation keeps sidebar assistant navigation gated strictly by committed
// Whale desktop preference, never by Host readiness.
func (s *Service) GateNavigation(hasRoute bool) bool {
	if !hasRoute {
		return false
	}
	return s.WhaleAssistantEnabled()
}

// CachedVendors returns cached vendor directory presence and triggers a non-blocking
// async filesystem probe when stale.
func (s *Service) CachedVendors() (whaleVendor, dshbotVendor bool) {
	if s == nil {
		return false, false
	}
	host := s.safeHost()
	root := host.VendorDir()
	ctx := host.LifecycleContext()
	s.mu.Lock()
	if s.vendorProbe.root != root {
		s.vendorProbe = vendorDirCache{root: root, gen: s.vendorProbe.gen}
	}
	stale := s.vendorProbe.checkedAt.IsZero() || time.Since(s.vendorProbe.checkedAt) >= assistantVendorProbeTTL
	var launchGen uint64
	launch := stale && !s.vendorProbe.loading
	if launch {
		s.vendorProbe.gen++
		launchGen = s.vendorProbe.gen
		s.vendorProbe.loading = true
	}
	whaleVendor, dshbotVendor = s.vendorProbe.whale, s.vendorProbe.dshbot
	s.mu.Unlock()
	if launch {
		go s.probeVendorsAsync(ctx, launchGen, root)
	}
	return whaleVendor, dshbotVendor
}

func (s *Service) probeVendorsAsync(ctx context.Context, gen uint64, root string) {
	if ctx.Err() != nil {
		s.mu.Lock()
		if s.vendorProbe.gen == gen {
			s.vendorProbe.loading = false
		}
		s.mu.Unlock()
		return
	}
	var whaleVendor, dshbotVendor bool
	if info, err := os.Stat(filepath.Join(root, PluginWhale)); err == nil && info.IsDir() {
		whaleVendor = true
	}
	if info, err := os.Stat(filepath.Join(root, PluginDshbot)); err == nil && info.IsDir() {
		dshbotVendor = true
	}
	s.mu.Lock()
	if s.vendorProbe.gen != gen || s.vendorProbe.root != root {
		s.mu.Unlock()
		return
	}
	s.vendorProbe.loading = false
	if ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	changed := s.vendorProbe.whale != whaleVendor || s.vendorProbe.dshbot != dshbotVendor
	s.vendorProbe.whale = whaleVendor
	s.vendorProbe.dshbot = dshbotVendor
	s.vendorProbe.checkedAt = time.Now()
	s.mu.Unlock()
	if changed {
		host := s.safeHost()
		host.Invalidate(host.Window())
	}
}

// RefreshWhaleSettingsUI invalidates the host window so Whale/proactive changes repaint.
func (s *Service) RefreshWhaleSettingsUI() {
	if s == nil {
		return
	}
	host := s.safeHost()
	host.Invalidate(host.Window())
}

// UpdateAssistantTasks notifies the host of assistant task/control state changes.
func (s *Service) UpdateAssistantTasks() {
	if s == nil {
		return
	}
	host := s.safeHost()
	host.Invalidate(host.Window())
	host.NotifyTaskChanged()
}

// RunPluginRPC executes a single non-overlapping plugin RPC action and updates
// PluginStatus while preserving existing busy gating.
func (s *Service) RunPluginRPC(win any, ep harness.Endpoint, label string, fn func(context.Context, *pluginrpc.Client) (string, error)) {
	if s == nil || fn == nil {
		return
	}
	s.mu.Lock()
	if s.pluginBusy {
		s.mu.Unlock()
		return
	}
	s.pluginBusy = true
	s.pluginDetail = label + "…"
	s.lastAct = s.pluginDetail
	s.mu.Unlock()
	host := s.safeHost()
	if win == nil {
		win = host.Window()
	}
	if win != nil {
		host.WindowUpdate(win, func() {})
	}
	rpc := host.PluginRPCClient(ep)
	go func() {
		detail, err := fn(context.Background(), rpc)
		msg := detail
		if err != nil {
			msg = label + ": " + err.Error()
		} else if msg == "" {
			msg = label + ": ok"
		} else {
			msg = label + ": " + truncateRunes(msg, 180)
		}
		s.mu.Lock()
		s.pluginBusy = false
		s.pluginDetail = msg
		s.lastAct = msg
		s.mu.Unlock()
		if win != nil {
			host.WindowUpdate(win, func() {})
		}
	}()
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func endpointScope(parts ...string) string {
	h := sha256.New()
	for i, part := range parts {
		if i > 0 {
			_, _ = h.Write([]byte{0})
		}
		_, _ = h.Write([]byte(part))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func clientBinding(client *dsh.Client) string {
	if client == nil {
		return ""
	}
	return fmt.Sprintf("%p", client)
}

func checkWriteAuthority(ctx context.Context, client *dsh.Client, live func() bool, staleMsg string) (bool, error) {
	if client == nil || (live != nil && !live()) {
		return false, fmt.Errorf("%s", staleMsg)
	}
	var authority struct {
		Writable bool `json:"writable"`
	}
	err := client.Call(ctx, "settings/describe", nil, &authority)
	if live != nil && !live() {
		return false, fmt.Errorf("%s", staleMsg)
	}
	return err == nil && authority.Writable, err
}

func readBotCapabilityStatus(ctx context.Context, client *dsh.Client, rpc *pluginrpc.Client, botID string, live func() bool) (pluginrpc.DshbotCapabilityStatus, bool, error) {
	var out pluginrpc.DshbotCapabilityStatus
	writable, authErr := checkWriteAuthority(ctx, client, live, "Bot view changed")
	if authErr != nil && live != nil && !live() {
		return out, false, authErr
	}
	out, err := rpc.DshbotCapabilities(ctx, botID)
	if err != nil {
		return out, false, err
	}
	if live != nil && !live() {
		return out, false, fmt.Errorf("Bot view changed")
	}
	return out, authErr == nil && writable, nil
}

func (s *Service) updateRosterItems(origin string, revision int, items []pluginrpc.DshbotCatalogItem) {
	roster := s.DshbotRoster()
	if roster == nil {
		return
	}
	roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{
		View: pluginrpc.DshbotSettingsView{
			NS:       "dsh-bot",
			Revision: revision,
			Value:    pluginrpc.DshbotCatalogValue{Items: items},
		},
	}, false, origin)
}

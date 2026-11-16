package assistantctl

import (
	"context"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

const (
	// PluginWhale is the canonical DSH plugin ID for the Whale assistant.
	PluginWhale = "dsh-whale"
	// PluginDshbot is the canonical DSH plugin ID for personal/room Bots.
	PluginDshbot = "dshbot"
)

// WindowState captures the minimal native window visibility and focus state
// needed by assistant notifications and room/bot observers without importing mygo.
type WindowState struct {
	Exists    bool
	Visible   bool
	Minimized bool
	Focused   bool
	Destroyed bool
}

// FileFilter mirrors a native open-file dialog filter without importing mygo.
type FileFilter struct {
	Name       string
	Extensions []string
}

// FileDialogOptions configures an interactive file-picker call on RuntimeHost.
type FileDialogOptions struct {
	Title    string
	Multiple bool
	Filters  []FileFilter
}

// NotificationOptions describes a desktop notification emitted by proactive or
// bot activity monitors.
type NotificationOptions struct {
	Title string
	Body  string
	Group string
}

// NotificationHandle represents a single desktop notification lifecycle.
type NotificationHandle interface {
	Show() error
	Close()
	OnClick(func())
}

// GroupReplyTarget abstracts a composer draft capable of entering thread-reply mode.
type GroupReplyTarget interface {
	BeginGroupReply(threadID string) bool
}

// PluginStatus reports the shared plugin RPC / session-open activity state.
type PluginStatus struct {
	Busy       bool
	Detail     string
	LastAction string
}

// ProactiveStatus exposes a read-only snapshot of proactive panel state.
type ProactiveStatus struct {
	Open             bool
	Busy             bool
	SourceBusy       bool
	DeliveryBusy     bool
	Title            string
	Description      string
	Selected         string
	Autonomy         int
	Error            string
	LoadError        string
	RunnerError      string
	MonitorError     string
	SourceRepository string
	SourceError      string
	DeliveryError    string
	RunnerActive     bool
	MonitorActive    bool
}

// FrameObservation summarizes which assistant sub-panels actually painted in
// the current UI frame so hidden controllers can suspend promptly.
type FrameObservation struct {
	RoomCreateRendered    bool
	RoomEditRendered      bool
	RoomRuntimeRendered   bool
	NotificationsRendered bool
}

// PanelBuildOptions carries read-only per-frame shell values into BuildPanelModel.
type PanelBuildOptions struct {
	Endpoint          harness.Endpoint
	ThemeID           string
	SystemDark        bool
	LastWebViewURL    string
	SessionLastAction string
	SessionMessages   []struct{ Role, Body string }
}

// RuntimeHost decouples package assistantctl from mainState, mygo windows, and
// platform dialogs so all domain logic runs cleanly in pure Go tests.
type RuntimeHost interface {
	RunOnMain(fn func())
	Window() any
	WindowUpdate(win any, fn func())
	Invalidate(win any)
	WindowStatus(win any) WindowState
	ShowWindow(win any)
	RestoreWindow(win any)
	FocusWindow(win any)

	PickImageFile(win any, title string) (string, error)
	OpenFileDialog(win any, opts FileDialogOptions) ([]string, error)
	WriteClipboard(text string)

	SystemNotificationsSupported() bool
	NewNotification(opts NotificationOptions) NotificationHandle
	IsNotificationsDenied(err error) bool

	CurrentClient() *dsh.Client
	BootEndpoint() (harness.Endpoint, harness.State, int)
	LifecycleContext() context.Context
	Sessions() *session.Store
	Config() *config.Store
	PluginRPCClient(ep harness.Endpoint) *pluginrpc.Client
	DshHome() string
	VendorDir() string

	Quitting() bool
	OtherSurfaceBlocking() bool
	PetSettingsActive() bool

	ClearResumePendingSession()
	HasSessionDraftConflict(sid string) bool
	StageSessionDraft(sid, text string) bool
	SubmitComposer(win any, sessions *session.Store, sid, cwd string, hostGen uint64)
	RevokePhoneGit()
	NotifyUser(win any, text string)
	NotifyTaskChanged()

	OpenWebViewHostPlugin(origin, pluginID string)
	MarkWebViewOpen()
	RevealMainConversation()
	SelectMainPanel(panel string)
	SetAssistantFlag(whale, dshbot *bool)
}

// HostDeps is the callback-backed implementation of RuntimeHost wired by cmd/whaleisle
// or unit tests. Any unset callback degrades safely without panicking.
type HostDeps struct {
	RunOnMain                  func(fn func())
	Window                     func() any
	WindowUpdate               func(win any, fn func())
	Invalidate                 func(win any)
	WindowStatus               func(win any) WindowState
	ShowWindow                 func(win any)
	RestoreWindow              func(win any)
	FocusWindow                func(win any)
	PickImageFile              func(win any, title string) (string, error)
	OpenFileDialog             func(win any, opts FileDialogOptions) ([]string, error)
	WriteClipboard             func(text string)
	SystemNotificationsSupport func() bool
	NewNotification            func(opts NotificationOptions) NotificationHandle
	IsNotificationsDenied      func(err error) bool
	CurrentClient              func() *dsh.Client
	BootEndpoint               func() (harness.Endpoint, harness.State, int)
	LifecycleContext           func() context.Context
	Sessions                   func() *session.Store
	Config                     func() *config.Store
	PluginRPCClient            func(ep harness.Endpoint) *pluginrpc.Client
	DshHome                    func() string
	VendorDir                  func() string
	Quitting                   func() bool
	OtherSurfaceBlocking       func() bool
	PetSettingsActive          func() bool
	ClearResumePendingSession  func()
	HasSessionDraftConflict    func(sid string) bool
	StageSessionDraft          func(sid, text string) bool
	SubmitComposer             func(win any, sessions *session.Store, sid, cwd string, hostGen uint64)
	RevokePhoneGit             func()
	NotifyUser                 func(win any, text string)
	NotifyTaskChanged          func()
	OpenWebViewHostPlugin      func(origin, pluginID string)
	MarkWebViewOpen            func()
	RevealMainConversation     func()
	SelectMainPanel            func(panel string)
	SetAssistantFlag           func(whale, dshbot *bool)
}

type hostDepsAdapter struct {
	deps HostDeps
}

func wrapHostDeps(deps HostDeps) RuntimeHost {
	return hostDepsAdapter{deps: deps}
}

func (h hostDepsAdapter) RunOnMain(fn func()) {
	if fn == nil {
		return
	}
	if h.deps.RunOnMain != nil {
		h.deps.RunOnMain(fn)
		return
	}
	fn()
}

func (h hostDepsAdapter) Window() any {
	if h.deps.Window != nil {
		return h.deps.Window()
	}
	return nil
}

func (h hostDepsAdapter) WindowUpdate(win any, fn func()) {
	if fn == nil {
		return
	}
	if h.deps.WindowUpdate != nil {
		h.deps.WindowUpdate(win, fn)
		return
	}
	if h.deps.RunOnMain != nil {
		h.deps.RunOnMain(fn)
		return
	}
	fn()
}

func (h hostDepsAdapter) Invalidate(win any) {
	if h.deps.Invalidate != nil {
		h.deps.Invalidate(win)
	}
}

func (h hostDepsAdapter) WindowStatus(win any) WindowState {
	if h.deps.WindowStatus != nil {
		return h.deps.WindowStatus(win)
	}
	if win == nil {
		return WindowState{}
	}
	return WindowState{Exists: true, Visible: true}
}

func (h hostDepsAdapter) ShowWindow(win any) {
	if h.deps.ShowWindow != nil {
		h.deps.ShowWindow(win)
	}
}

func (h hostDepsAdapter) RestoreWindow(win any) {
	if h.deps.RestoreWindow != nil {
		h.deps.RestoreWindow(win)
	}
}

func (h hostDepsAdapter) FocusWindow(win any) {
	if h.deps.FocusWindow != nil {
		h.deps.FocusWindow(win)
	}
}

func (h hostDepsAdapter) PickImageFile(win any, title string) (string, error) {
	if h.deps.PickImageFile != nil {
		return h.deps.PickImageFile(win, title)
	}
	if h.deps.OpenFileDialog != nil {
		selected, err := h.deps.OpenFileDialog(win, FileDialogOptions{
			Title:   title,
			Filters: []FileFilter{{Name: "JPEG / PNG", Extensions: []string{"jpg", "jpeg", "png"}}},
		})
		if err != nil || len(selected) == 0 {
			return "", err
		}
		return selected[0], nil
	}
	return "", nil
}

func (h hostDepsAdapter) OpenFileDialog(win any, opts FileDialogOptions) ([]string, error) {
	if h.deps.OpenFileDialog != nil {
		return h.deps.OpenFileDialog(win, opts)
	}
	if h.deps.PickImageFile != nil && !opts.Multiple {
		picked, err := h.deps.PickImageFile(win, opts.Title)
		if err != nil || picked == "" {
			return nil, err
		}
		return []string{picked}, nil
	}
	return nil, nil
}

func (h hostDepsAdapter) WriteClipboard(text string) {
	if h.deps.WriteClipboard != nil {
		h.deps.WriteClipboard(text)
	}
}

func (h hostDepsAdapter) SystemNotificationsSupported() bool {
	if h.deps.SystemNotificationsSupport != nil {
		return h.deps.SystemNotificationsSupport()
	}
	return false
}

func (h hostDepsAdapter) NewNotification(opts NotificationOptions) NotificationHandle {
	if h.deps.NewNotification != nil {
		return h.deps.NewNotification(opts)
	}
	return nil
}

func (h hostDepsAdapter) IsNotificationsDenied(err error) bool {
	if h.deps.IsNotificationsDenied != nil {
		return h.deps.IsNotificationsDenied(err)
	}
	return false
}

func (h hostDepsAdapter) CurrentClient() *dsh.Client {
	if h.deps.CurrentClient != nil {
		return h.deps.CurrentClient()
	}
	return nil
}

func (h hostDepsAdapter) BootEndpoint() (harness.Endpoint, harness.State, int) {
	if h.deps.BootEndpoint != nil {
		return h.deps.BootEndpoint()
	}
	return harness.Endpoint{}, harness.Stopped, 0
}

func (h hostDepsAdapter) LifecycleContext() context.Context {
	if h.deps.LifecycleContext != nil {
		if ctx := h.deps.LifecycleContext(); ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

func (h hostDepsAdapter) Sessions() *session.Store {
	if h.deps.Sessions != nil {
		return h.deps.Sessions()
	}
	return nil
}

func (h hostDepsAdapter) Config() *config.Store {
	if h.deps.Config != nil {
		return h.deps.Config()
	}
	return nil
}

func (h hostDepsAdapter) PluginRPCClient(ep harness.Endpoint) *pluginrpc.Client {
	if h.deps.PluginRPCClient != nil {
		if cli := h.deps.PluginRPCClient(ep); cli != nil {
			return cli
		}
	}
	return &pluginrpc.Client{BaseURL: ep.Origin, Cookie: ep.Cookie}
}

func (h hostDepsAdapter) DshHome() string {
	if h.deps.DshHome != nil {
		return h.deps.DshHome()
	}
	return paths.DshHome()
}

func (h hostDepsAdapter) VendorDir() string {
	if h.deps.VendorDir != nil {
		return h.deps.VendorDir()
	}
	return paths.VendorDir()
}

func (h hostDepsAdapter) Quitting() bool {
	if h.deps.Quitting != nil {
		return h.deps.Quitting()
	}
	return false
}

func (h hostDepsAdapter) OtherSurfaceBlocking() bool {
	if h.deps.OtherSurfaceBlocking != nil {
		return h.deps.OtherSurfaceBlocking()
	}
	return false
}

func (h hostDepsAdapter) PetSettingsActive() bool {
	if h.deps.PetSettingsActive != nil {
		return h.deps.PetSettingsActive()
	}
	return false
}

func (h hostDepsAdapter) ClearResumePendingSession() {
	if h.deps.ClearResumePendingSession != nil {
		h.deps.ClearResumePendingSession()
	}
}

func (h hostDepsAdapter) HasSessionDraftConflict(sid string) bool {
	if h.deps.HasSessionDraftConflict != nil {
		return h.deps.HasSessionDraftConflict(sid)
	}
	return false
}

func (h hostDepsAdapter) StageSessionDraft(sid, text string) bool {
	if h.deps.StageSessionDraft != nil {
		return h.deps.StageSessionDraft(sid, text)
	}
	return h.deps.SubmitComposer != nil
}

func (h hostDepsAdapter) SubmitComposer(win any, sessions *session.Store, sid, cwd string, hostGen uint64) {
	if h.deps.SubmitComposer != nil {
		h.deps.SubmitComposer(win, sessions, sid, cwd, hostGen)
	}
}

func (h hostDepsAdapter) RevokePhoneGit() {
	if h.deps.RevokePhoneGit != nil {
		h.deps.RevokePhoneGit()
	}
}

func (h hostDepsAdapter) NotifyUser(win any, text string) {
	if h.deps.NotifyUser != nil {
		h.deps.NotifyUser(win, text)
	}
}

func (h hostDepsAdapter) NotifyTaskChanged() {
	if h.deps.NotifyTaskChanged != nil {
		h.deps.NotifyTaskChanged()
	}
}

func (h hostDepsAdapter) OpenWebViewHostPlugin(origin, pluginID string) {
	if h.deps.OpenWebViewHostPlugin != nil {
		h.deps.OpenWebViewHostPlugin(origin, pluginID)
	}
}

func (h hostDepsAdapter) MarkWebViewOpen() {
	if h.deps.MarkWebViewOpen != nil {
		h.deps.MarkWebViewOpen()
	}
}

func (h hostDepsAdapter) RevealMainConversation() {
	if h.deps.RevealMainConversation != nil {
		h.deps.RevealMainConversation()
	}
}

func (h hostDepsAdapter) SelectMainPanel(panel string) {
	if h.deps.SelectMainPanel != nil {
		h.deps.SelectMainPanel(panel)
	}
}

func (h hostDepsAdapter) SetAssistantFlag(whale, dshbot *bool) {
	if h.deps.SetAssistantFlag != nil {
		h.deps.SetAssistantFlag(whale, dshbot)
	}
}

func hostNotReadyToast() string {
	return i18n.T(i18n.KeyCmdNotConnectedToTheHostYetTryAgainShor)
}

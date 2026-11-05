package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/pad"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/files"
	padui "whaleisle/internal/ui/pad"
)

// Child editors are durable draft consumers, but never alias an ordinary
// session ID or another child address's draft/checkpoint entry.
func (s *mainState) childComposerDraft(binding string, address session.AgentSessionAddress) *composer.DraftState {
	raw, _ := json.Marshal(address)
	key := "\x00child-draft:" + base64.RawURLEncoding.EncodeToString(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.composerDraftLocked(key)
	if s.childComposerBindings == nil {
		s.childComposerBindings = map[string]string{}
	}
	if old, ok := s.childComposerBindings[key]; ok && old != binding {
		d.RetireBinding()
	}
	s.childComposerBindings[key] = binding
	return d
}

func (s *mainState) boundReferenceQuery(sessions *session.Store, snapshot session.ViewState) func(context.Context, string, bool) ([]session.ReferenceCandidate, error) {
	current := func() bool {
		s.mu.Lock()
		same := s.sessions == sessions && !s.quitting
		s.mu.Unlock()
		v := sessions.SelectionSummary()
		return same && v.SelectedID == snapshot.SelectedID && v.ComposerHostGeneration == snapshot.ComposerHostGeneration && sessions.ComposerSessionGeneration() == snapshot.ComposerSessionGeneration
	}
	return func(ctx context.Context, q string, quoted bool) ([]session.ReferenceCandidate, error) {
		if !current() {
			return nil, context.Canceled
		}
		rows, err := sessions.QueryReferenceCandidates(ctx, q, quoted)
		if !current() {
			return nil, context.Canceled
		}
		return rows, err
	}
}

func (s *mainState) saveDefaultAgentPreset(id string) {
	s.mu.Lock()
	manager, sessions := s.agentPresetManager, s.sessions
	s.mu.Unlock()
	if manager == nil {
		return
	}
	var apply func(context.Context, string)
	if sessions != nil {
		apply = sessions.CaptureBlankAgentPresetDefaultTarget()
	}
	go func() {
		ctx := context.Background()
		manager.SaveDefault(ctx, id)
		confirmed := manager.Snapshot()
		if !confirmed.Busy && confirmed.Err == "" && confirmed.DefaultID == id && apply != nil {
			apply(ctx, id)
		}
	}()
}

func (s *mainState) syncPadDesktop() *pad.Desktop {
	s.mu.Lock()
	desktop, win, st, quitting := s.padDesktop, s.win, s.bootStatus, s.quitting
	attempted := s.padDesktopAttempted
	s.mu.Unlock()
	if win == nil || quitting {
		return desktop
	}
	if desktop == nil && !attempted && st.State == harness.Ready {
		s.mu.Lock()
		s.padDesktopAttempted = true
		s.mu.Unlock()
		created, err := pad.NewDesktop(pad.Options{Parent: win, UserData: paths.UserDataDir(), VendorDir: paths.VendorDir(), Changed: win.Invalidate, OnAsk: s.askFromPad, OnOpenURL: mygo.Shell.OpenExternal, OnFailure: s.onEmbeddedPageFailure})
		if err != nil {
			s.notifyUser(win, resumeText("面板初始化失败：", "Panel initialization failed: ")+err.Error())
			return nil
		}
		s.mu.Lock()
		s.padDesktop = created
		s.mu.Unlock()
		desktop = created
	}
	if desktop != nil {
		origin, cookie := "", ""
		if st.State == harness.Ready {
			origin, cookie = st.Endpoint.Origin, st.Endpoint.Cookie
		}
		desktop.Sync(origin, cookie, uint64(st.Generation))
	}
	return desktop
}

func (s *mainState) paintPad(c *ui.Context) {
	d := s.syncPadDesktop()
	if d == nil {
		return
	}
	s.mu.Lock()
	masked := s.settingsOpen || s.launcherOpen || s.closingOverlay || s.confirmAsk != nil || s.dirPickOpen || s.quitting
	themeID := s.themeID
	s.mu.Unlock()
	d.SetMasked(masked)
	if !masked {
		padui.View(c, padui.Model{Desktop: d, ThemeID: themeID, SystemDark: mygo.Theme.IsDark()})
	}
}

func (s *mainState) padReservedWidth(c *ui.Context) float32 {
	s.mu.Lock()
	d := s.padDesktop
	masked := s.settingsOpen || s.launcherOpen || s.closingOverlay || s.confirmAsk != nil || s.dirPickOpen || s.quitting
	s.mu.Unlock()
	if d == nil || masked {
		return 0
	}
	width, _ := c.Size()
	return d.ReservedWidth(width)
}

// Both a native click and a card's websocket request arrive here. RunOnMain is
// reentrant on the UI thread, unlike posting Window.Update then waiting on it.
func (s *mainState) askFromPad(text string, insert bool) error {
	if strings.TrimSpace(text) == "" {
		return errors.New(resumeText("请输入面板需求。", "Enter what you want the panel to do."))
	}
	s.mu.Lock()
	sessions, host := s.sessions, s.bootStatus
	s.mu.Unlock()
	if sessions == nil || host.State != harness.Ready {
		return errors.New(resumeText("会话连接尚未就绪。", "The conversation connection is not ready."))
	}
	target, err := sessions.CaptureComposerTarget(sessions.SelectedSessionID())
	if err != nil {
		return err
	}
	result := errors.New(resumeText("主窗口已关闭。", "The main window has closed."))
	mygo.RunOnMain(func() {
		s.mu.Lock()
		current := !s.quitting && s.sessions == sessions && s.bootStatus.Generation == host.Generation && s.bootStatus.Endpoint == host.Endpoint && s.bootStatus.State == harness.Ready
		win := s.win
		s.mu.Unlock()
		if !current || win == nil || !target.NavigationCurrent() {
			result = errors.New(resumeText("会话已更改，请重新操作。", "The conversation changed. Try again."))
			return
		}
		v := sessions.Snapshot()
		s.mu.Lock()
		d := s.composerDraftLocked(v.SelectedID)
		s.mu.Unlock()
		if result = applyPadComposerDraft(d, text, insert); result != nil {
			return
		}
		s.revealMainConversation()
		d.EditFocus = true
		win.Invalidate()
	})
	return result
}

// Pad hands authored text to the native editor. It never submits a prompt.
// insertText replaces the captured UTF-16 selection, as the original input hub
// does; ask fills an empty draft and refuses to overwrite existing user work.
func applyPadComposerDraft(d *composer.DraftState, text string, insert bool) error {
	if d.Sending || d.Preparing > 0 || d.EditSeq > 0 {
		return errors.New(resumeText("输入框正在发送、准备附件或编辑消息，请稍后重试。", "The composer is sending, preparing attachments or editing a message. Try again when it is ready."))
	}
	if !insert && (strings.TrimSpace(d.Text) != "" || len(d.Attachments) != 0) {
		return errors.New(resumeText("输入框已有草稿，请先发送或清空后再提问。", "The composer has a draft. Send or clear it before asking."))
	}
	if insert {
		selection := d.Client.Selection().Range
		d.Client.ReplaceText(&selection, text)
	} else {
		d.Client.ReplaceText(&ui.TextInputRange{End: ui.UTF16Len(d.Text)}, text)
	}
	return nil
}

type fileOpenWithOwner struct {
	sessions              *session.Store
	client                *dsh.Client
	endpoint              harness.Endpoint
	generation            int
	sessionID, root, path string
	epoch                 uint64
}

func (s *mainState) fileOpenWithSourceCurrent(o fileOpenWithOwner) bool {
	s.mu.Lock()
	valid := s.sessions == o.sessions && s.bootStatus.Generation == o.generation && s.bootStatus.Endpoint == o.endpoint && s.bootStatus.State == harness.Ready && !s.quitting
	s.mu.Unlock()
	s.dshMu.Lock()
	sameClient := s.dshClient == o.client
	s.dshMu.Unlock()
	if !valid || !sameClient || o.sessions == nil {
		return false
	}
	v := o.sessions.SelectionSummary()
	return v.SelectedID == o.sessionID && v.SelectedCwd == o.root && o.sessions.ComposerSessionGeneration() == o.epoch
}

func (s *mainState) fileOpenWithCurrent(o fileOpenWithOwner) bool {
	s.mu.Lock()
	same := s.fileOpenWithOwner == o
	s.mu.Unlock()
	return same && s.fileOpenWithSourceCurrent(o)
}

func (s *mainState) refreshFileOpenWith(win *mygo.Window, o fileOpenWithOwner, force bool) {
	if o.sessions == nil || o.client == nil || o.sessionID == "" || o.root == "" || o.path == "" {
		return
	}
	if !s.fileOpenWithSourceCurrent(o) {
		return
	}
	s.mu.Lock()
	if !force && s.fileOpenWithOwner == o {
		s.mu.Unlock()
		return
	}
	s.fileOpenWithOwner = o
	s.fileOpenWithFor = o.path
	s.fileOpenWithApps = nil
	s.fileOpenWithState = files.OpenWithState{Loading: true}
	s.mu.Unlock()
	if !s.fileOpenWithCurrent(o) {
		return
	}
	target, err := o.sessions.CaptureComposerTarget(o.sessionID)
	if err != nil {
		s.mu.Lock()
		if s.fileOpenWithOwner == o {
			s.fileOpenWithState = files.OpenWithState{Error: err.Error()}
		}
		s.mu.Unlock()
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var apps api.WorkspacePathApplicationsResult
		path := strings.ReplaceAll(o.path, `\`, "/")
		if !strings.HasPrefix(path, "/") && !(len(path) > 1 && path[1] == ':') {
			path = strings.TrimRight(strings.ReplaceAll(o.root, `\`, "/"), "/") + "/" + path
		}
		err := o.client.Call(ctx, "session/workspacePathApplications", map[string]any{"request": api.WorkspacePathApplicationsRequest{Path: path}}, &apps)
		if !s.fileOpenWithCurrent(o) || !target.NavigationCurrent() {
			return
		}
		publish := func() {
			if !s.fileOpenWithCurrent(o) || !target.NavigationCurrent() {
				return
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			s.fileOpenWithState = files.OpenWithState{}
			if err != nil {
				s.fileOpenWithState.Error = err.Error()
				return
			}
			for _, a := range apps {
				name := strings.TrimSpace(a.Name)
				if name == "" {
					name = a.Id
				}
				s.fileOpenWithApps = append(s.fileOpenWithApps, files.OpenWithApp{ID: a.Id, Name: name, Default: a.Default})
			}
		}
		if win != nil {
			win.Update(publish)
		}
	}()
}

func (s *mainState) openWorkspacePathWithOwner(o fileOpenWithOwner, path, app string) {
	if !s.fileOpenWithCurrent(o) {
		return
	}
	rel, ok := files.WorkspaceRelativePath(o.root, path)
	if !ok || strings.ReplaceAll(rel, `\`, "/") != strings.ReplaceAll(o.path, `\`, "/") {
		return
	}
	target, err := o.sessions.CaptureComposerTarget(o.sessionID)
	if err != nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var out api.SessionOpenWorkspacePathValue
		request := api.SessionOpenWorkspacePathRequest{Path: path}
		if app != "" {
			request.Application = &app
		}
		if !s.fileOpenWithCurrent(o) || !target.NavigationCurrent() {
			return
		}
		err := o.client.Call(ctx, "session/openWorkspacePath", map[string]any{"request": request}, &out)
		if !s.fileOpenWithCurrent(o) || !target.NavigationCurrent() {
			return
		}
		if err != nil {
			o.sessions.SetLastAction(err.Error())
		} else if !out.Opened {
			o.sessions.SetLastAction(resumeText("未能使用所选应用打开文件。", "The selected application could not open the file."))
		}
	}()
}

func (s *mainState) workspaceImageReader(sessions *session.Store, client *dsh.Client, st harness.Status, sid, root string) func(context.Context, string, int64) ([]byte, error) {
	reader, valid, err := sessions.MediaPreviewSource(sid)
	return func(ctx context.Context, path string, limit int64) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		current := func() bool {
			s.mu.Lock()
			ok := s.sessions == sessions && s.bootStatus.Generation == st.Generation && s.bootStatus.Endpoint == st.Endpoint && !s.quitting
			s.mu.Unlock()
			s.dshMu.Lock()
			same := s.dshClient == client
			s.dshMu.Unlock()
			v := sessions.SelectionSummary()
			return ok && same && valid() && v.SelectedID == sid && v.SelectedCwd == root
		}
		if limit <= 0 || limit > int64(^uint(0)>>1) {
			return nil, fmt.Errorf("invalid image byte limit")
		}
		return mediapreview.ReadSnapshot(ctx, reader, sid, path, int(limit), current)
	}
}

func (s *mainState) managedMarketProfile(st harness.Status) string {
	s.mu.Lock()
	sup := s.sup
	s.mu.Unlock()
	if sup == nil || st.State != harness.Ready || st.Generation == 0 {
		return ""
	}
	owned := sup.Status()
	if owned.State != harness.Ready || owned.Generation != st.Generation || owned.Endpoint != st.Endpoint || !strings.HasPrefix(owned.Source, "vendor ") {
		return ""
	}
	return paths.WebProfileDir()
}

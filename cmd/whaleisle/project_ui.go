package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	projectctl "whaleisle/internal/app/project"
	"whaleisle/internal/config"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/desktop/sshremote"
	"whaleisle/internal/desktop/taskcontrol"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/workspacepicker"
)

func projectWindowRef(w *mygo.Window) any {
	if w == nil {
		return nil
	}
	return w
}

func (s *mainState) projectManager() *projectctl.Manager {
	if s == nil {
		return projectctl.NewManager(projectctl.HostDeps{})
	}
	s.initCtrlMu.Lock()
	if s.projectCtl == nil {
		s.projectCtl = projectctl.NewManager(mainProjectDeps{s: s}.hostDeps())
	}
	mgr := s.projectCtl
	s.initCtrlMu.Unlock()
	return mgr
}

func (s *mainState) projectLaunchAccess(initial config.Config) (func(context.Context, map[string]any) map[string]any, func(context.Context) ([]string, error)) {
	return s.projectManager().LaunchAccess(initial)
}

func (s *mainState) projectPreferenceAccess() *settings.ProjectPreferenceAccess {
	return s.projectManager().PreferenceAccess()
}

func (s *mainState) setProjectPreference(next bool) {
	s.projectManager().SetPreference(next)
}

func (s *mainState) ordinaryTeamModel(win *mygo.Window) projects.OrdinaryTeamModel {
	return s.projectManager().OrdinaryTeamModel(projectWindowRef(win))
}

func (s *mainState) projectDirectoryActions(win *mygo.Window) []workspacepicker.DirectoryAction {
	return s.projectManager().DirectoryActions(projectWindowRef(win))
}

func (s *mainState) bindProjectModel(win *mygo.Window, m *frame.Model) {
	if m == nil {
		return
	}
	m.Projects = s.projectManager().BindModel(projectWindowRef(win), m.ThemeID, m.SystemDark)
}

func (s *mainState) beginProjectPresentationFrame() func() {
	return s.projectManager().BeginPresentationFrame()
}

func (s *mainState) bindProjectNativeConsumers(win *mygo.Window, m *projects.Model) {
	s.projectManager().BindNativeConsumers(projectWindowRef(win), m)
}

func (s *mainState) capturePRViewOwner(root string) func() bool {
	return s.projectManager().CapturePRViewOwner(root)
}

func (s *mainState) projectFileAccessCurrent(sessions *session.Store, client *dsh.Client, host harness.Status) bool {
	return s.projectManager().FileAccessCurrent(sessions, client, host)
}

func projectMountEnabled(cfg config.Config, environment string) bool {
	return projectctl.MountEnabled(cfg, environment)
}

func projectDevelopmentOverride() string {
	return projectctl.DevelopmentOverride()
}

func validateProjectRelativeFile(root, canonicalPath string) (string, error) {
	fail := errors.New(resumeText("项目文件路径不属于报告授权的工作目录。", "The Project file is outside the report's authorized root."))
	rel, ok := files.WorkspaceRelativePath(root, canonicalPath)
	if !ok || rel == "." {
		return "", fail
	}
	if err := workspacefs.ValidateRelativePath(rel); err != nil {
		return "", err
	}
	return rel, nil
}

func projectTargetRelative(t projectflow.FileTarget) (string, error) {
	return projectctl.TargetRelative(t, validateProjectRelativeFile)
}

func projectAbsolutePath(root, value string) string {
	value = strings.ReplaceAll(value, `\`, "/")
	if strings.HasPrefix(value, "/") || len(value) > 1 && value[1] == ':' {
		return value
	}
	return strings.TrimRight(strings.ReplaceAll(root, `\`, "/"), "/") + "/" + value
}

// --- projectctl.HostDeps ---

type mainProjectDeps struct{ s *mainState }

func (d mainProjectDeps) hostDeps() projectctl.HostDeps {
	return projectctl.HostDeps{
		HostSnapshot: func() projectctl.HostSnapshot {
			d.s.dshMu.Lock()
			client, mux := d.s.dshClient, d.s.dshMux
			d.s.dshMu.Unlock()

			d.s.mu.Lock()
			st := d.s.bootStatus
			sup := d.s.sup
			quitting := d.s.quitting
			overlay := d.s.maintenanceOpen || d.s.settingsOpen || d.s.closingOverlay
			importBusy := d.s.launcherImportBusy
			tunnel := d.s.sshTunnel
			d.s.mu.Unlock()

			var supID any
			var supStatus harness.Status
			if sup != nil {
				supID = sup
				supStatus = sup.Status()
			}
			remoteConnected := false
			if tunnel != nil {
				if _, status, _ := tunnel.Snapshot(); status == sshremote.TunnelConnected {
					remoteConnected = true
				}
			}
			return projectctl.HostSnapshot{
				Client:           client,
				Mux:              mux,
				Status:           st,
				HasSupervisor:    sup != nil,
				SupervisorID:     supID,
				SupervisorStatus: supStatus,
				Quitting:         quitting,
				OverlayOpen:      overlay,
				ImportBusy:       importBusy,
				RemoteConnected:  remoteConnected,
			}
		},
		LifecycleContext: d.s.lifecycleContext,
		Sessions: func() *session.Store {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			return d.s.sessions
		},
		Config: func() *config.Store {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			return d.s.cfgStore
		},
		MainWindow: func() any {
			d.s.mu.Lock()
			w := d.s.win
			d.s.mu.Unlock()
			return projectWindowRef(w)
		},
		RunOnMain: mygo.RunOnMain,
		UpdateWindow: func(win any, fn func()) {
			if fn == nil {
				return
			}
			if w, ok := win.(*mygo.Window); ok {
				if w == nil {
					fn()
					return
				}
				if !w.IsDestroyed() {
					w.Update(fn)
				}
				return
			}
			if win == nil {
				fn()
			}
		},
		Invalidate: func(win any) {
			if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
				w.Invalidate()
			}
		},
		OpenDirectoryDialog: func(win any, title, defaultPath string) ([]string, error) {
			w, _ := win.(*mygo.Window)
			return mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent:      w,
				Title:       title,
				DefaultPath: defaultPath,
				Directory:   true,
			})
		},
		OpenWorkspaceDirectory: desktopcomponents.OpenInFileManager,
		EnsureWorkspace: func(ctx context.Context, ep harness.Endpoint, dir string) error {
			return harness.EnsureWorkspace(ctx, nil, ep, dir)
		},
		OpenModelSettings: func() {
			d.s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings, Section: "models"})
			d.s.mu.Lock()
			set := d.s.settings
			d.s.mu.Unlock()
			if set != nil {
				set.RefreshDescribe(context.Background())
			}
		},
		ValidateRelativeFile: validateProjectRelativeFile,
		OpenAuthorizedFile:   d.openAuthorizedFile,
		ManagedMarketProfile: d.s.managedMarketProfile,
		CaptureWorkspaceAdoption: func(sessions *session.Store) any {
			op := d.s.captureWorkspaceAdoption(sessions)
			return &op
		},
		ApplyWorkspaceAdoption: func(adoptionAny any, _ *session.Store, _ string, leadSessionID string) bool {
			op, ok := adoptionAny.(*workspaceAdoption)
			if !ok || op == nil || !op.carry {
				return true
			}
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			if d.s.composerDrafts[op.sourceID] == op.draft &&
				workspaceSessionDraftRevision(op.draft) == op.revision &&
				bytes.Equal(op.draft.SaveDraft(), op.snapshot) {
				if leadSessionID == op.sourceID || !workspaceSessionDraftOccupied(d.s.composerDrafts[leadSessionID]) {
					d.s.composerDrafts[leadSessionID] = op.draft
					op.draft.TargetID = leadSessionID
					if leadSessionID != op.sourceID {
						delete(d.s.composerDrafts, op.sourceID)
					}
					return true
				}
				return false
			}
			return true
		},
		PlaceLeadDraft: func(sessionID, text string) error {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			target := d.s.composerDraftLocked(sessionID)
			if target.Text != "" || len(target.Attachments) > 0 || target.Sending || target.Preparing > 0 || target.EditSeq > 0 {
				return errors.New(i18n.T(i18n.KeyCmdTheLeadHasADraftOrPendingSendItWas))
			}
			target.Text = text
			target.Client.Sync()
			return nil
		},
		SubmitLeadDraft: func(win any, sessions *session.Store, sessionID, text string) error {
			w, _ := win.(*mygo.Window)
			if sessions == nil || strings.TrimSpace(text) == "" {
				return fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectRecoveryUnavailable))
			}
			d.s.mu.Lock()
			draft := d.s.composerDraftLocked(sessionID)
			if draft.Text != "" || len(draft.Attachments) > 0 || draft.Sending || draft.Preparing > 0 || draft.EditSeq > 0 || draft.EditSessionID != "" {
				d.s.mu.Unlock()
				return fmt.Errorf("%s", i18n.T(i18n.KeyCmdProjectLeadDraftPending))
			}
			draft.Text = text
			d.s.mu.Unlock()
			draft.Client.Sync()
			snap := sessions.Snapshot()
			d.s.submitComposer(w, sessions, draft, sessionID, snap.ScratchCwd, false, snap.ComposerHostGeneration)
			return nil
		},
		OccupiedDraftSessionIDs: func() []string {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			var excluded []string
			for id, draft := range d.s.composerDrafts {
				if workspaceSessionDraftOccupied(draft) {
					excluded = append(excluded, id)
				}
			}
			return excluded
		},
		AcquireExternalMaintenance: func() (func(), error) {
			return importer.AcquireExternalMaintenance(paths.DshHome())
		},
		ProtectTaskAction: func(ctx context.Context, win any, operation string, commit func() error) (bool, error) {
			w, _ := win.(*mygo.Window)
			d.s.mu.Lock()
			sup := d.s.sup
			d.s.mu.Unlock()
			did, err := d.s.protectTaskAction(ctx, w, sup, operation, commit)
			if errors.Is(err, taskcontrol.ErrCancelled) {
				return false, nil
			}
			return did, err
		},
		RestartHost: func(ctx context.Context) error {
			d.s.mu.Lock()
			sup := d.s.sup
			d.s.mu.Unlock()
			if sup == nil {
				return errors.New("desktop supervisor unavailable")
			}
			if err := d.s.closeTaskProtectedResources(); err != nil {
				return err
			}
			d.s.syncHarnessFromConfig()
			if err := sup.RestartAndWait(ctx); err != nil {
				return fmt.Errorf("Project restart failed: %w", err)
			}
			return nil
		},
		VisualReaderAvailable: func() (bool, string) {
			caps := harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{})
			return caps.VisualReplyReader, caps.Reason
		},
		CaptureVisualHTML: func(ctx context.Context, html string, width int, appearance string) (projectctl.VisualCaptureResult, error) {
			capture, err := captureHTMLInHelper(ctx, html, width, appearance)
			if err != nil {
				return projectctl.VisualCaptureResult{}, err
			}
			return projectctl.VisualCaptureResult{
				PNG:            capture.PNG,
				ContentHeight:  capture.ContentHeight,
				CapturedHeight: capture.CapturedHeight,
			}, nil
		},
		DecodeVisualBitmap: func(preview *projects.VisualPagePreview, png []byte) error {
			bmp, err := ui.DecodeBitmap(png)
			if err != nil {
				return err
			}
			preview.Bitmap = bmp
			return nil
		},
		NewVisualSurface: func(ctx context.Context, win any, opts projectctl.VisualSurfaceOptions) (projectctl.VisualSurface, error) {
			w, _ := win.(*mygo.Window)
			exe, err := os.Executable()
			if err != nil {
				return nil, err
			}
			handle, err := previewprocess.OpenExperimental(ctx, previewprocess.Options{
				Enabled:    true,
				HelperPath: previewprocess.HelperPath(exe),
			}, previewprocess.Document{
				HTML:       opts.HTML,
				Width:      opts.Width,
				Appearance: opts.Appearance,
				Inline:     opts.Inline,
			})
			if err != nil {
				return nil, err
			}
			return newPublishedHTMLSurface(handle, w), nil
		},
	}
}

func (d mainProjectDeps) openAuthorizedFile(win any, req projectctl.AuthorizedFileRequest) error {
	w, _ := win.(*mygo.Window)
	target := req.Target
	client := req.Client
	model := files.Model{ThemeID: req.ThemeID, SystemDark: req.SystemDark, ImageGeneration: uint64(req.HostGeneration)}
	if req.Local {
		model.OnSaveEdit = func(path, text string) error {
			if !req.AccessCurrent() {
				return errors.New(resumeText("项目文件授权已失效，草稿已保留。", "Project file access expired. Your draft was kept."))
			}
			full, err := workspacefs.JoinRoot(target.Root, path)
			if err != nil {
				return err
			}
			if req.CheckPath != nil {
				if err = req.CheckPath(full); err != nil {
					return err
				}
			}
			return workspacefs.WriteFileText(full, text)
		}
	} else {
		model.ReadTextPageContext = func(ctx context.Context, path string, offset int64) (workspacefs.TextPage, error) {
			full := projectAbsolutePath(target.Root, path)
			if req.CheckPath != nil {
				if err := req.CheckPath(full); err != nil {
					return workspacefs.TextPage{}, err
				}
			}
			return workspacefs.ReadRemoteTextPage(ctx, client, target.CoordinatorSessionID, full, offset)
		}
		model.ListRemoteContext = func(ctx context.Context, path string) ([]workspacefs.Entry, error) {
			if !req.AccessCurrent() {
				return nil, errors.New(resumeText("项目文件授权已失效。", "Project file access expired."))
			}
			return workspacefs.ListRemoteDirScoped(ctx, client, target.CoordinatorSessionID, projectAbsolutePath(target.Root, path))
		}
		model.ReadImageContext = func(ctx context.Context, path string, limit int64) ([]byte, error) {
			full := projectAbsolutePath(target.Root, path)
			if req.CheckPath != nil {
				if err := req.CheckPath(full); err != nil {
					return nil, err
				}
			}
			return mediapreview.ReadSnapshot(ctx, client, target.CoordinatorSessionID, full, int(limit), req.AccessCurrent)
		}
	}
	model.OpenDocumentViewer = func(path string) (files.DocumentSurface, error) {
		full := projectAbsolutePath(target.Root, path)
		if req.CheckPath != nil {
			if err := req.CheckPath(full); err != nil {
				return nil, err
			}
		}
		docURL, err := desktopwebview.DocumentURL(client.Endpoint.Origin, target.CoordinatorSessionID, full)
		if err != nil {
			return nil, err
		}
		var onChange func()
		if w != nil {
			onChange = w.Invalidate
		}
		return desktopwebview.NewHost(desktopwebview.HostOptions{
			Parent:                w,
			URL:                   docURL,
			Title:                 full,
			Untrusted:             true,
			SessionOrigin:         client.Endpoint.Origin,
			SessionCookie:         client.Endpoint.Cookie,
			OnChange:              onChange,
			OnFailure:             d.s.onEmbeddedPageFailure,
			ShortcutPreferences:   d.s.pageShortcutPreferences(),
			OnShortcutCloseWindow: d.s.queueProductShortcutClose,
		})
	}
	d.s.mu.Lock()
	if d.s.layout.Workspace == nil {
		d.s.layout.Workspace = &frame.WorkspacePanes{}
	}
	panes := d.s.layout.Workspace
	d.s.mu.Unlock()
	if err := panes.OpenAuthorizedProjectFile(target, req.Binding, frame.ProjectFileAccess{Valid: req.AccessCurrent, Model: model}); err != nil {
		return err
	}
	d.s.selectMainPanel("")
	d.s.mu.Lock()
	d.s.filesPaneOpen, d.s.rightPaneTab = true, "files"
	d.s.mu.Unlock()
	return nil
}

var _ = time.Second

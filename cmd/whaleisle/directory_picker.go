package main

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/dirpick"
	"whaleisle/internal/ui/i18n"
	remoteui "whaleisle/internal/ui/remote"
	"whaleisle/internal/ui/theme"
)

func (s *mainState) openDirectoryPicker(sessions *session.Store, win *mygo.Window, mode string) {
	if sessions == nil {
		return
	}
	if mode == "" {
		mode = "workspace"
	}
	host := sessions.ComposerHostGeneration()
	s.mu.Lock()
	cfgStore := s.cfgStore
	s.mu.Unlock()
	useNative := runtime.GOOS == "windows" && cfgStore != nil && cfgStore.Get().DirectoryPickerUseNative
	s.mu.Lock()
	s.cancelDirectoryScanLocked()
	s.dirPickOpen, s.dirPickMode, s.dirPickHostGeneration = true, mode, host
	s.dirPickListing = api.DirectoryListing{}
	s.dirPickBrowser = dirpick.BrowserState{UseNative: useNative}
	s.dirPickNewName, s.dirPickErr = "", ""
	s.mu.Unlock()
	if useNative {
		updateDirectoryWindow(win)
	} else {
		s.scanDirectoryPicker(sessions, win, "", "navigate")
	}
}

// The preference is shared by every directory entry point. A failed save keeps
// the current mode, so the displayed choice never promises an unpersisted mode.
func (s *mainState) setDirectoryPickerNativeMode(sessions *session.Store, win *mygo.Window, enabled bool) {
	if runtime.GOOS != "windows" {
		return
	}
	s.mu.Lock()
	cfgStore, request := s.cfgStore, s.dirPickRequest
	allowed := s.dirPickOpen && !s.dirPickBrowser.NativeBusy && s.dirPickBrowser.FolderDraft == nil
	s.mu.Unlock()
	if !allowed || cfgStore == nil {
		return
	}
	err := cfgStore.Update(func(cfg *config.Config) { cfg.DirectoryPickerUseNative = enabled })
	s.mu.Lock()
	if !s.dirPickOpen || s.dirPickRequest != request {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.dirPickErr = directoryFailureText(err)
		s.mu.Unlock()
		updateDirectoryWindow(win)
		return
	}
	s.cancelDirectoryScanLocked()
	s.dirPickBrowser.UseNative, s.dirPickBrowser.NativeOpened = enabled, false
	s.dirPickBrowser.Revision++
	s.dirPickErr = ""
	unopened := s.dirPickBrowser.Parent == nil
	s.mu.Unlock()
	if !enabled && unopened {
		s.scanDirectoryPicker(sessions, win, "", "navigate")
	} else {
		updateDirectoryWindow(win)
	}
}

func (s *mainState) cancelDirectoryScanLocked() {
	s.dirPickRequest++
	if s.dirPickCancel != nil {
		s.dirPickCancel()
		s.dirPickCancel = nil
	}
	s.dirPickLoading = false
	s.dirPickBrowser.SlowScan = false
}

func (s *mainState) closeDirectoryPicker(win *mygo.Window) {
	s.mu.Lock()
	s.cancelDirectoryScanLocked()
	s.dirPickOpen = false
	s.dirPickNewName, s.dirPickErr = "", ""
	s.dirPickBrowser.PathDraft, s.dirPickBrowser.FolderDraft = nil, nil
	s.dirPickBrowser.CreateError = ""
	s.dirPickBrowser.NativeBusy = false
	remote := s.remoteMachines
	s.mu.Unlock()
	if remote != nil {
		remote.CancelDirectoryChoice()
	}
	updateDirectoryWindow(win)
}

func updateDirectoryWindow(win *mygo.Window) {
	if win != nil {
		win.Update(func() {})
	}
}

// One scan owns a captured transport, Host generation, cancellation and intent.
// This never resolves the Store again to issue a request on a replacement Host.
type directoryScan struct {
	state         *mainState
	sessions      *session.Store
	win           *mygo.Window
	client        *dsh.Client
	ctx           context.Context
	cancel        context.CancelFunc
	request, host uint64
	boot          int
}

func (s *mainState) beginDirectoryScan(sessions *session.Store, win *mygo.Window) *directoryScan {
	if sessions == nil {
		return nil
	}
	host := sessions.ComposerHostGeneration()
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirPickOpen || s.dirPickHostGeneration != host {
		return nil
	}
	s.cancelDirectoryScanLocked()
	ctx, cancel := context.WithCancel(context.Background())
	s.dirPickCancel = cancel
	return &directoryScan{s, sessions, win, client, ctx, cancel, s.dirPickRequest, host, s.bootStatus.Generation}
}

func (op *directoryScan) current() bool {
	if op.ctx.Err() != nil || op.sessions.ComposerHostGeneration() != op.host {
		return false
	}
	s := op.state
	s.mu.Lock()
	valid := s.dirPickOpen && s.dirPickRequest == op.request && s.dirPickHostGeneration == op.host && s.bootStatus.Generation == op.boot
	s.mu.Unlock()
	s.dshMu.Lock()
	valid = valid && s.dshClient == op.client
	s.dshMu.Unlock()
	return valid
}

func (op *directoryScan) commit(change func(*dirpick.BrowserState)) bool {
	if !op.current() {
		return false
	}
	s := op.state
	s.mu.Lock()
	valid := s.dirPickOpen && s.dirPickRequest == op.request && s.bootStatus.Generation == op.boot
	if valid {
		change(&s.dirPickBrowser)
		s.dirPickBrowser.Revision++
		if current := s.dirPickBrowser.Current(); current != nil {
			s.dirPickListing = *current
		}
	}
	s.mu.Unlock()
	if valid {
		updateDirectoryWindow(op.win)
	}
	return valid
}

func (op *directoryScan) finish() {
	s := op.state
	s.mu.Lock()
	if s.dirPickRequest == op.request {
		s.dirPickCancel = nil
	}
	s.mu.Unlock()
	op.cancel()
}

// Each leg has the source's 300ms silent window. Fast scans never shift rows.
func (op *directoryScan) list(path string) (api.DirectoryListing, error) {
	var window uint64
	op.commit(func(b *dirpick.BrowserState) { b.SlowScan = false; b.ScanWindow++; window = b.ScanWindow })
	go func() {
		timer := time.NewTimer(300 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-op.ctx.Done():
		case <-timer.C:
			op.commit(func(b *dirpick.BrowserState) {
				if op.state.dirPickLoading && b.ScanWindow == window {
					b.SlowScan = true
				}
			})
		}
	}()
	var listing api.DirectoryListing
	if op.client == nil {
		return listing, session.ErrNoClient
	}
	if !op.current() {
		return listing, context.Canceled
	}
	args := map[string]any{}
	// Source list(path?: string) forwards a nonblank submitted path verbatim.
	if path != "" {
		args["path"] = path
	}
	err := op.client.Call(op.ctx, "directoryPicker/list", args, &listing)
	return listing, err
}

func (s *mainState) requestDirectoryPicker(sessions *session.Store, win *mygo.Window, path, createName string) {
	if createName != "" {
		s.createDirectoryPicker(sessions, win, path, createName)
		return
	}
	s.scanDirectoryPicker(sessions, win, path, "navigate")
}

// navigate/preview share a landing; select instead keeps the clicked left pane.
func (s *mainState) scanDirectoryPicker(sessions *session.Store, win *mygo.Window, path, kind string) {
	op := s.beginDirectoryScan(sessions, win)
	if op == nil {
		return
	}
	op.commit(func(b *dirpick.BrowserState) {
		s.dirPickLoading = true
		if kind != "preview" {
			s.dirPickErr = ""
			b.FailureToast = ""
			s.dirPickPath = path
		}
		if kind == "navigate" {
			b.RetrySelection = nil
		}
	})
	go func() {
		defer op.finish()
		target, err := op.list(path)
		if err != nil {
			op.commit(func(b *dirpick.BrowserState) {
				s.dirPickLoading = false
				b.SlowScan = false
				if kind != "preview" {
					s.dirPickErr = directoryFailureText(err)
				}
				if kind == "select" {
					b.Selected = nil
					b.Child = nil
					b.FocusEdit = true
				}
			})
			return
		}
		if kind == "select" {
			op.commit(func(b *dirpick.BrowserState) { b.Child = &target; s.dirPickLoading = false; b.SlowScan = false })
			return
		}
		settle := func(b *dirpick.BrowserState) {
			s.dirPickLoading = false
			b.SlowScan = false
			if kind == "preview" {
				b.ScannedDirectory = path
				b.ScannedPath = target.Path
				b.FocusPath = true
				s.dirPickErr = ""
			} else {
				b.PathDraft = nil
				b.FocusEdit = true
			}
		}
		single := func() {
			op.commit(func(b *dirpick.BrowserState) { b.Parent = &target; b.Selected = nil; b.Child = nil; settle(b) })
		}
		if len(target.Crumbs) < 2 {
			single()
			return
		}
		parentPath := target.Crumbs[len(target.Crumbs)-2].Path
		type parentReply struct {
			listing api.DirectoryListing
			err     error
		}
		result := make(chan parentReply, 1)
		go func() { listing, err := op.list(parentPath); result <- parentReply{listing, err} }()
		landParent := func(reply parentReply) {
			match := dirpick.MatchEntry(reply.listing, target.Path)
			if reply.err != nil || match == nil {
				single()
				return
			}
			op.commit(func(b *dirpick.BrowserState) {
				b.Parent = &reply.listing
				b.Selected = match
				b.Child = &target
				settle(b)
			})
		}
		if kind == "preview" {
			select {
			case reply := <-result:
				landParent(reply)
			case <-op.ctx.Done():
			}
			return
		}
		timer := time.NewTimer(200 * time.Millisecond)
		defer timer.Stop()
		select {
		case reply := <-result:
			landParent(reply)
		case <-timer.C:
			single()
			select {
			case reply := <-result:
				landParent(reply)
			case <-op.ctx.Done():
			}
		case <-op.ctx.Done():
		}
	}()
}

func (s *mainState) selectDirectoryPicker(sessions *session.Store, win *mygo.Window, entry api.DirectoryListingCrumbsItem, right bool) {
	s.mu.Lock()
	b := &s.dirPickBrowser
	if !s.dirPickOpen || b.FolderDraft != nil {
		s.mu.Unlock()
		return
	}
	if right {
		if b.Child == nil {
			s.mu.Unlock()
			return
		}
		b.Parent = b.Child
	}
	b.FocusSelection = b.PathDraft != nil
	b.PathDraft = nil
	b.Selected = &entry
	b.Child = nil
	b.RetrySelection = &entry
	s.mu.Unlock()
	s.scanDirectoryPicker(sessions, win, entry.Path, "select")
}

func (s *mainState) beginDirectoryPath(sessions *session.Store, win *mygo.Window) {
	s.mu.Lock()
	s.cancelDirectoryScanLocked()
	b := &s.dirPickBrowser
	base := b.TargetPath()
	if base != "" && !strings.HasSuffix(base, dirpick.Separator(b.Parent)) {
		base += dirpick.Separator(b.Parent)
	}
	b.PathDraft = &base
	b.PathHadFocus = false
	b.PreviewSuspended = false
	b.FocusPath = true
	s.mu.Unlock()
	updateDirectoryWindow(win)
}

func (s *mainState) editDirectoryPath(sessions *session.Store, win *mygo.Window, draft string) {
	s.mu.Lock()
	if !s.dirPickOpen || s.dirPickBrowser.FolderDraft != nil {
		s.mu.Unlock()
		return
	}
	s.cancelDirectoryScanLocked()
	s.dirPickBrowser.PathDraft = &draft
	s.dirPickBrowser.PreviewSuspended = false
	ctx, cancel := context.WithCancel(context.Background())
	s.dirPickCancel = cancel
	request, host := s.dirPickRequest, s.dirPickHostGeneration
	s.mu.Unlock()
	updateDirectoryWindow(win)
	go func() {
		timer := time.NewTimer(250 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if sessions.ComposerHostGeneration() != host {
			return
		}
		s.mu.Lock()
		b := s.dirPickBrowser.Clone()
		valid := s.dirPickOpen && s.dirPickRequest == request && !b.PreviewSuspended && b.PathDraft != nil && *b.PathDraft == draft
		s.mu.Unlock()
		if !valid {
			return
		}
		directory := dirpick.DirectoryPart(b.Current(), draft)
		_, answered := b.Prefix()
		if b.Current() != nil && directory != "" && !answered {
			s.scanDirectoryPicker(sessions, win, directory, "preview")
		}
	}()
}

func (s *mainState) cancelDirectoryPath(sessions *session.Store, win *mygo.Window) {
	s.mu.Lock()
	s.cancelDirectoryScanLocked()
	b := &s.dirPickBrowser
	b.PathDraft = nil
	b.FocusEdit = true
	if b.Child == nil {
		b.Selected = nil
	}
	s.dirPickErr = ""
	unopened := b.Parent == nil
	s.mu.Unlock()
	if unopened {
		s.scanDirectoryPicker(sessions, win, "", "navigate")
	} else {
		updateDirectoryWindow(win)
	}
}

func (s *mainState) createDirectoryPicker(sessions *session.Store, win *mygo.Window, parent, name string) {
	if !directorypicker.SelectablePath(parent) || strings.TrimSpace(name) == "" {
		return
	}
	s.mu.Lock()
	creating := s.dirPickBrowser.Creating
	s.mu.Unlock()
	if creating {
		return
	}
	op := s.beginDirectoryScan(sessions, win)
	if op == nil {
		return
	}
	op.commit(func(b *dirpick.BrowserState) {
		b.Creating = true
		b.CreateError = ""
		b.CreateToast = ""
		b.FolderDraft = &name
	})
	go func() {
		defer op.finish()
		var created api.CreateDirectoryResult
		var err error
		if op.client == nil {
			err = session.ErrNoClient
		} else if op.current() {
			// workspace-controller/src/directory-picker.ts validates the single
			// segment; DirectoryBrowser forwards its original whitespace.
			err = op.client.Call(op.ctx, "directoryPicker/createDirectory", map[string]any{"path": parent, "name": name}, &created)
		}
		if err != nil {
			op.commit(func(b *dirpick.BrowserState) { b.Creating = false; b.CreateError = directoryFailureText(err) })
			return
		}
		if !op.commit(func(b *dirpick.BrowserState) {
			b.Creating = false
			b.FolderDraft = nil
			s.dirPickLoading = true
			s.dirPickErr = ""
			s.dirPickPath = parent
			b.RetrySelection = nil
		}) {
			return
		}
		level, err := op.list(parent)
		if err != nil {
			op.commit(func(b *dirpick.BrowserState) {
				s.dirPickLoading = false
				b.SlowScan = false
				s.dirPickErr = directoryFailureText(err)
			})
			return
		}
		entry := api.DirectoryListingCrumbsItem{Name: name, Path: string(created), Hidden: false}
		if !op.commit(func(b *dirpick.BrowserState) {
			b.Parent = &level
			b.Selected = &entry
			b.Child = nil
			b.RetrySelection = &entry
			s.dirPickPath = entry.Path
		}) {
			return
		}
		child, err := op.list(entry.Path)
		op.commit(func(b *dirpick.BrowserState) {
			s.dirPickLoading = false
			b.SlowScan = false
			if err != nil {
				s.dirPickErr = directoryFailureText(err)
				b.Selected = nil
				b.Child = nil
			} else {
				b.Child = &child
			}
		})
	}()
}

// The root wiring stays stable; browser state remains owned by this open/Host.
func (s *mainState) directoryPickerView(c *ui.Context, win *mygo.Window, sessions *session.Store, themeID string, systemDark bool, onPick func(mode, path string)) {
	s.mu.Lock()
	open, host := s.dirPickOpen, s.dirPickHostGeneration
	s.mu.Unlock()
	if !open {
		c.CloseToast("directory-browse-error")
		c.CloseToast("directory-create-error")
		return
	}
	if sessions == nil || sessions.ComposerHostGeneration() != host {
		s.closeDirectoryPicker(win)
		return
	}
	s.mu.Lock()
	if s.dirPickBrowser.UseNative && !s.dirPickBrowser.Remote && !s.dirPickBrowser.NativeOpened && win != nil {
		s.dirPickBrowser.NativeOpened = true
		s.mu.Unlock()
		s.nativeDirectoryPicker(sessions, win, onPick)
		s.mu.Lock()
	}
	browser := s.dirPickBrowser.Clone()
	before := browser.Clone()
	hadFolder := browser.FolderDraft != nil
	loading, errorText, mode, request, attemptedPath := s.dirPickLoading, s.dirPickErr, s.dirPickMode, s.dirPickRequest, s.dirPickPath
	s.mu.Unlock()
	w, h := c.Size()
	colors := theme.Resolve(themeID).Active(systemDark)
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Key("directory-picker-dialog").Column().Width(min(float32(680), max(float32(0), w-48))).Height(min(float32(500), max(float32(0), h-48))).MinWidth(0).MinHeight(0).Padding(0).Radius(28).Background(chrome.Color(colors.Field)).Children(func() {
			s.mu.Lock()
			remote, cfgStore := s.remoteMachines, s.cfgStore
			s.mu.Unlock()
			remoteEnabled := false
			if cfgStore != nil {
				remoteEnabled = cfgStore.Get().RemoteWorkspaceEnabled
			}
			if mode == "workspace" && remote != nil && remoteEnabled {
				selected := 0
				if browser.Remote {
					selected = 1
				}
				if ui.Tabs(c.Key("directory-picker-tabs"), &selected, directoryPickerText("本地", "Local"), directoryPickerText("远程", "Remote")).FillWidth().Shrink(0).Padding(12, 24, 0, 24).Disabled(browser.NativeBusy || hadFolder).Changed() {
					s.mu.Lock()
					s.cancelDirectoryScanLocked()
					s.dirPickBrowser.Remote = selected == 1
					s.mu.Unlock()
					if selected == 0 {
						remote.CancelDirectoryChoice()
					}
					updateDirectoryWindow(win)
				}
			}
			if browser.Remote && remote != nil && remoteEnabled {
				ui.Column(c).FillWidth().Grow(1).MinWidth(0).MinHeight(0).Clip().Padding(24).Children(func() {
					remoteui.DirectoryPicker(c, remoteui.DirectoryPickerModel{Store: remote, Ready: sessions.Snapshot().ClientReady, Enabled: remoteEnabled,
						OnCancel: func() { s.closeDirectoryPicker(win) },
						OnChoice: func(choice remotestore.DirectoryChoice) {
							if win == nil || win.IsDestroyed() {
								return
							}
							win.Update(func() {
								s.mu.Lock()
								valid := s.dirPickOpen && s.dirPickBrowser.Remote && s.dirPickRequest == request && s.dirPickHostGeneration == host
								s.mu.Unlock()
								if valid && sessions.ComposerHostGeneration() == host && remote.DirectoryChoiceCurrent(choice) && directorypicker.SelectablePath(choice.LocalMirror) {
									s.closeDirectoryPicker(win)
									if onPick != nil {
										onPick(mode, choice.LocalMirror)
									}
								}
							})
						}})
				})
				return
			}
			var onNativeMode func(bool)
			var onNativePick func()
			if runtime.GOOS == "windows" && cfgStore != nil {
				onNativeMode = func(enabled bool) { s.setDirectoryPickerNativeMode(sessions, win, enabled) }
				onNativePick = func() { s.nativeDirectoryPicker(sessions, win, onPick) }
			}
			ui.Column(c).FillWidth().Grow(1).MinWidth(0).MinHeight(0).Children(func() {
				dirpick.View(c, dirpick.Model{
					State: &browser, ThemeID: themeID, SystemDark: systemDark, Loading: loading, Err: errorText,
					OnNativePicker: onNativePick, OnNativeMode: onNativeMode,
					OnNavigate: func(path string) { s.scanDirectoryPicker(sessions, win, path, "navigate") },
					OnSelect: func(entry api.DirectoryListingCrumbsItem, right bool) {
						s.selectDirectoryPicker(sessions, win, entry, right)
					},
					OnRetry: func() {
						if browser.UseNative {
							s.nativeDirectoryPicker(sessions, win, onPick)
						} else if browser.RetrySelection != nil {
							s.selectDirectoryPicker(sessions, win, *browser.RetrySelection, false)
						} else {
							s.scanDirectoryPicker(sessions, win, attemptedPath, "navigate")
						}
					},
					OnBeginPath: func() { s.beginDirectoryPath(sessions, win) },
					OnEditPath:  func(draft string) { s.editDirectoryPath(sessions, win, draft) },
					OnSubmitPath: func(path string) {
						s.mu.Lock()
						s.dirPickBrowser.PreviewSuspended = true
						s.mu.Unlock()
						s.scanDirectoryPicker(sessions, win, path, "navigate")
					},
					OnCancelPath: func() { s.cancelDirectoryPath(sessions, win) },
					OnDismissError: func() {
						s.mu.Lock()
						if s.dirPickRequest == request && s.dirPickErr == errorText {
							s.dirPickErr = ""
						}
						s.mu.Unlock()
						updateDirectoryWindow(win)
					},
					OnDismissCreateError: func() {
						s.mu.Lock()
						if s.dirPickRequest == request {
							s.dirPickBrowser.CreateError = ""
						}
						s.mu.Unlock()
						updateDirectoryWindow(win)
					},
					OnCreate: func(parent, name string) { s.createDirectoryPicker(sessions, win, parent, name) },
					OnRetryCreate: func() {
						s.mu.Lock()
						b := s.dirPickBrowser.Clone()
						s.mu.Unlock()
						if b.FolderDraft != nil {
							s.createDirectoryPicker(sessions, win, b.TargetPath(), *b.FolderDraft)
						}
					},
					OnPick: func(path string) {
						if !directorypicker.SelectablePath(path) || sessions.ComposerHostGeneration() != host {
							return
						}
						s.mu.Lock()
						valid := s.dirPickOpen && !s.dirPickLoading && s.dirPickRequest == request && s.dirPickBrowser.PathDraft == nil && s.dirPickBrowser.FolderDraft == nil
						s.mu.Unlock()
						if valid {
							s.closeDirectoryPicker(win)
							if onPick != nil {
								onPick(mode, path)
							}
						}
					},
					OnClose: func() {
						if browser.FolderDraft == nil && !browser.NativeBusy {
							s.closeDirectoryPicker(win)
						}
					},
				})
			})
		})
	})
	if !open {
		if hadFolder || browser.NativeBusy {
			open = true
		} else {
			s.closeDirectoryPicker(win)
			return
		}
	}
	s.mergeDirectoryPickerUI(request, before, browser)
}

// The system picker remains a local desktop choice; its result is checked by
// this Host before entering the shared registration/adoption callback.
func (s *mainState) nativeDirectoryPicker(sessions *session.Store, win *mygo.Window, onPick func(string, string)) {
	if win == nil || win.IsDestroyed() {
		return
	}
	s.mu.Lock()
	busy := s.dirPickBrowser.NativeBusy || s.dirPickBrowser.FolderDraft != nil
	s.mu.Unlock()
	if busy {
		return
	}
	op := s.beginDirectoryScan(sessions, win)
	if op == nil {
		return
	}
	s.mu.Lock()
	initial, mode := s.dirPickBrowser.TargetPath(), s.dirPickMode
	s.mu.Unlock()
	if !directorypicker.SelectablePath(initial) {
		initial = ""
	}
	if !op.commit(func(b *dirpick.BrowserState) { b.NativeBusy = true; b.FailureToast = ""; s.dirPickErr = "" }) {
		op.finish()
		return
	}
	go func() {
		queued := false
		defer func() {
			if !queued {
				op.commit(func(b *dirpick.BrowserState) { b.NativeBusy = false })
				op.finish()
			}
		}()
		result, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: directoryPickerText("选择工作目录", "Select Workspace Directory"), Directory: true, DefaultPath: initial})
		if err != nil {
			op.commit(func(_ *dirpick.BrowserState) { s.dirPickErr = directoryFailureText(err) })
			return
		}
		if len(result) == 0 || !op.current() {
			return
		}
		listing, err := op.list(result[0])
		if err != nil {
			op.commit(func(_ *dirpick.BrowserState) { s.dirPickErr = directoryFailureText(err) })
			return
		}
		if !directorypicker.SelectablePath(listing.Path) || win == nil || win.IsDestroyed() {
			return
		}
		queued = true
		win.Update(func() {
			defer op.finish()
			if !win.IsDestroyed() && op.current() {
				s.closeDirectoryPicker(win)
				if onPick != nil {
					onPick(mode, listing.Path)
				}
			}
		})
	}()
}

func directoryPickerText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleZH {
		return zh
	}
	return en
}

func (s *mainState) mergeDirectoryPickerUI(request uint64, before, edited dirpick.BrowserState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirPickOpen && s.dirPickRequest == request {
		s.dirPickBrowser.MergeUI(before, edited)
	}
}

func directoryFailureText(err error) string {
	var remote *dsh.Error
	if errors.As(err, &remote) && remote.Message != "" {
		return remote.Message
	}
	return err.Error()
}

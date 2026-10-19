package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/dirpick"
	"whaleisle/internal/ui/theme"
)

func (s *mainState) openDirectoryPicker(sessions *session.Store, win *mygo.Window, mode string) {
	if sessions == nil {
		return
	}
	if mode == "" {
		mode = "workspace"
	}
	host := sessions.Snapshot().ComposerHostGeneration
	s.mu.Lock()
	s.dirPickOpen, s.dirPickMode, s.dirPickHostGeneration = true, mode, host
	s.dirPickListing = api.DirectoryListing{}
	s.dirPickBrowser = dirpick.BrowserState{}
	s.dirPickNewName = ""
	s.mu.Unlock()
	s.scanDirectoryPicker(sessions, win, "", "navigate")
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
	s.mu.Unlock()
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
	host := sessions.Snapshot().ComposerHostGeneration
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
	if op.ctx.Err() != nil || op.sessions.Snapshot().ComposerHostGeneration != op.host {
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
		if sessions.Snapshot().ComposerHostGeneration != host {
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
	if sessions == nil || sessions.Snapshot().ComposerHostGeneration != host {
		s.closeDirectoryPicker(win)
		return
	}
	s.mu.Lock()
	browser := s.dirPickBrowser.Clone()
	before := browser.Clone()
	hadFolder := browser.FolderDraft != nil
	loading, errorText, mode, request, attemptedPath := s.dirPickLoading, s.dirPickErr, s.dirPickMode, s.dirPickRequest, s.dirPickPath
	s.mu.Unlock()
	w, h := c.Size()
	colors := theme.Resolve(themeID).Active(systemDark)
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Key("directory-picker-dialog").Width(min(float32(680), max(float32(0), w-48))).Height(min(float32(500), max(float32(0), h-48))).MinWidth(0).MinHeight(0).Padding(0).Radius(28).Background(chrome.Color(colors.Field)).Children(func() {
			dirpick.View(c, dirpick.Model{
				State: &browser, ThemeID: themeID, SystemDark: systemDark, Loading: loading, Err: errorText,
				OnNavigate: func(path string) { s.scanDirectoryPicker(sessions, win, path, "navigate") },
				OnSelect: func(entry api.DirectoryListingCrumbsItem, right bool) {
					s.selectDirectoryPicker(sessions, win, entry, right)
				},
				OnRetry: func() {
					if browser.RetrySelection != nil {
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
					if !directorypicker.SelectablePath(path) || sessions.Snapshot().ComposerHostGeneration != host {
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
					if browser.FolderDraft == nil {
						s.closeDirectoryPicker(win)
					}
				},
			})
		})
	})
	if !open {
		if hadFolder {
			open = true
		} else {
			s.closeDirectoryPicker(win)
			return
		}
	}
	s.mergeDirectoryPickerUI(request, before, browser)
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

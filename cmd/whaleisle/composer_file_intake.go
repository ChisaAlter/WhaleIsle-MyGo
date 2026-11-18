package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"whaleisle/internal/errcopy"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
)

// Native file metadata and uploads share a small worker budget across drafts.
// A slow UNC file cannot fan out one blocked OS request per dropped path.
var composerFileWorkers = make(chan struct{}, 2)

func withComposerFileWorker(ctx context.Context, work func() error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case composerFileWorkers <- struct{}{}:
	}
	defer func() { <-composerFileWorkers }()
	if err := ctx.Err(); err != nil {
		return err
	}
	return work()
}

func (s *mainState) addComposerFiles(win *mygo.Window, sessions *session.Store, id string, d *composer.DraftState, paths []string) {
	s.beginComposerFileIntake(win, sessions, id, d, paths, false)
}

func (s *mainState) addTransferredComposerFiles(win *mygo.Window, sessions *session.Store, id string, d *composer.DraftState, paths []string) {
	s.beginComposerFileIntake(win, sessions, id, d, paths, true)
}

func (s *mainState) bindComposerFileIntake(win *mygo.Window, sessions *session.Store, id string, d *composer.DraftState) {
	if d == nil || d.FileIntake() == nil {
		return
	}
	op := d.FileIntake()
	if op.Busy || op.Ready {
		return
	}
	op.Retry = func() {
		if d.FileIntake() != op {
			return
		}
		paths, regularOnly := append([]string(nil), op.Paths...), op.RegularOnly
		d.CancelFileIntake()
		s.beginComposerFileIntake(win, sessions, id, d, paths, regularOnly)
	}
}

func (s *mainState) beginComposerFileIntake(win *mygo.Window, sessions *session.Store, id string, d *composer.DraftState, paths []string, regularOnly bool) {
	if d == nil || d.Sending || d.Preparing > 0 || len(paths) == 0 {
		return
	}
	if len(paths) > 128 {
		d.Error = "一次最多添加128个文件 / At most 128 files per intake"
		return
	}
	cwd, local := s.nativeComposerFileScope(sessions, id, d)
	// Existing non-window callers can collect unproven paths without touching
	// the file system. The actual UI always uses the owned-window path below.
	if win == nil && sessions == nil && !local {
		for _, path := range paths {
			if strings.TrimSpace(path) != "" {
				d.Attachments = append(d.Attachments, session.ComposerAttachment{Path: path, Name: filepath.Base(path)})
			}
		}
		return
	}
	if win == nil {
		return
	}
	var owner *session.ComposerTarget
	var limits *session.ComposerImageLimits
	if sessions != nil {
		var err error
		owner, err = sessions.CaptureComposerTarget(id)
		if err != nil && local {
			d.Error = errcopy.Sentence(err)
			return
		}
		limits = sessions.ComposerImageLimits(id)
	}
	parent := s.lifecycleContext()
	ctx, cancel := context.WithCancel(parent)
	op := d.BeginFileIntake(paths, regularOnly, cancel)
	if op == nil {
		return
	}
	existing := append([]session.ComposerAttachment(nil), d.Attachments...)
	input := append([]string(nil), paths...)
	current := func() bool {
		s.mu.Lock()
		valid := !s.quitting && s.composerDrafts[id] == d && s.sessions == sessions
		s.mu.Unlock()
		if !valid || owner != nil && !owner.Current() {
			return false
		}
		if local {
			for _, item := range sessions.ListedItems() {
				if item.SessionID == id {
					return session.HostPathKey(item.Cwd) == session.HostPathKey(cwd)
				}
			}
			return false
		}
		return true
	}
	go func() {
		var plan composerFilePlan
		err := withComposerFileWorker(ctx, func() error {
			files, err := inspectComposerFiles(ctx, input, regularOnly)
			if err != nil {
				return err
			}
			if err = validateComposerImageMetadata(ctx, existing, files, limits); err != nil {
				return err
			}
			plan, err = planNativeComposerFiles(cwd, files, local)
			return err
		})
		if err == nil {
			err = ctx.Err()
		}
		win.Update(func() {
			if d.FileIntake() != op {
				return
			}
			if !current() {
				err = errors.New(i18n.T(i18n.KeyCmdComposerHostChanged))
			}
			op.Apply = func() {
				if d.FileIntake() != op {
					return
				}
				if !current() {
					d.CompleteFileIntake(op, errors.New(i18n.T(i18n.KeyCmdComposerHostChanged)))
					s.bindComposerFileIntake(win, sessions, id, d)
					return
				}
				indices, applied := d.ApplyFileIntake(op, plan.references, plan.uploads)
				if applied && owner != nil && owner.SessionID != "" {
					for _, index := range indices {
						prepareComposerAttachmentContext(parent, owner, d, index, false, func(fn func()) { win.Update(fn) })
					}
				}
			}
			ready := d.CompleteFileIntake(op, err)
			s.bindComposerFileIntake(win, sessions, id, d)
			// Upload-only batches do not replace text or selection.
			if err == nil && (ready || len(plan.references) == 0) {
				op.Apply()
			}
		})
	}()
}

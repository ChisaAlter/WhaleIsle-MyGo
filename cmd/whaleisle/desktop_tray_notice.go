package main

import (
	"errors"
	"os"
	"path/filepath"
)

type desktopCloseAction uint8

const (
	desktopCloseQuit desktopCloseAction = iota
	desktopCloseHide
	desktopCloseKeep
)

// A failed tray must never strand a hidden main window or implicitly stop Host.
func desktopCloseDecision(quitting, closeToTray, trayReady bool) desktopCloseAction {
	if quitting || !closeToTray {
		return desktopCloseQuit
	}
	if !trayReady {
		return desktopCloseKeep
	}
	return desktopCloseHide
}

// Shared with Electron: a passive, once-per-user-data-directory hint. It never
// gates hiding or invokes Host lifecycle actions, even if storage/notify fails.
type trayHideNotice struct {
	markerPath   string
	acknowledged bool
}

func (n *trayHideNotice) close(hide func(), notify func() error) error {
	hide()
	if n.acknowledged {
		return nil
	}
	n.acknowledged = true
	if _, err := os.Stat(n.markerPath); err == nil {
		return nil
	}
	err := os.MkdirAll(filepath.Dir(n.markerPath), 0700)
	if err == nil {
		var f *os.File
		f, err = os.OpenFile(n.markerPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		if err == nil {
			err = f.Close()
		}
	}
	return errors.Join(err, notify())
}

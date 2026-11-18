//go:build windows

package main

import (
	"context"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func watchShortcutDocumentChanges(ctx context.Context, path string, onChange func()) {
	dir := filepath.Dir(path)
	notify, err := windows.FindFirstChangeNotification(
		dir,
		false,
		windows.FILE_NOTIFY_CHANGE_FILE_NAME|windows.FILE_NOTIFY_CHANGE_LAST_WRITE|windows.FILE_NOTIFY_CHANGE_SIZE,
	)
	if err != nil {
		watchShortcutDocumentStat(ctx, path, onChange)
		return
	}
	defer func() { _ = windows.FindCloseChangeNotification(notify) }()
	stopEvt, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		watchShortcutDocumentStat(ctx, path, onChange)
		return
	}
	defer func() { _ = windows.CloseHandle(stopEvt) }()
	go func() {
		<-ctx.Done()
		_ = windows.SetEvent(stopEvt)
	}()
	handles := []windows.Handle{stopEvt, notify}
	for {
		ev, err := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)
		if err != nil || ev == windows.WAIT_OBJECT_0 || ctx.Err() != nil {
			return
		}
		if ev == windows.WAIT_OBJECT_0+1 {
			onChange()
			if windows.FindNextChangeNotification(notify) != nil {
				return
			}
		}
	}
}

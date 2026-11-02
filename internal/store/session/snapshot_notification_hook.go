package session

import "sync/atomic"

// Tests can pause a published write immediately before its notification.
// Production leaves the hook nil, so it does not alter notification behavior.
var snapshotNotificationHook atomic.Pointer[func(string)]

func beforeSnapshotNotification(point string) {
	if hook := snapshotNotificationHook.Load(); hook != nil {
		(*hook)(point)
	}
}

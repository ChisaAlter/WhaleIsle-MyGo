//go:build !windows

package main

import "context"

func watchShortcutDocumentChanges(ctx context.Context, path string, onChange func()) {
	watchShortcutDocumentStat(ctx, path, onChange)
}

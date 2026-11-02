package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// Capture the same Host and selection epoch as the frame's other file readers.
func remoteWorkbookReader(sessions *session.Store, expected *dsh.Client, sid string) func(context.Context, string, workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
	if sessions == nil || expected == nil {
		return nil
	}
	reader, valid, err := sessions.MediaPreviewSource(sid)
	if err != nil || reader != expected {
		return nil
	}
	return func(ctx context.Context, path string, request workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
		if ext := strings.ToLower(filepath.Ext(path)); ext != ".xlsx" && ext != ".xls" && ext != ".csv" && ext != ".tsv" {
			return nil, errors.New(i18n.T(i18n.KeyCmdWorkbookReaderFormats))
		}
		data, err := mediapreview.ReadSnapshot(ctx, reader, sid, path, 16<<20, valid)
		if err != nil {
			return nil, err
		}
		return workspacefs.ReadNamedWorkbookBytesPage(ctx, path, data, request)
	}
}

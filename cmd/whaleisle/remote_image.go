package main

import (
	"context"
	"errors"

	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
)

// Capture the frame's Host and selected-session epoch. Even an unavailable Host
// returns a reader, so Files cannot mistake a Host path for a local file.
func remoteImageReader(sessions *session.Store, expected *dsh.Client, sid string) func(context.Context, string, int64) ([]byte, error) {
	unavailable := func(context.Context, string, int64) ([]byte, error) {
		return nil, errors.New("Image preview Host or session is unavailable; reload after reconnecting")
	}
	if sessions == nil || expected == nil {
		return unavailable
	}
	reader, valid, err := sessions.MediaPreviewSource(sid)
	if err != nil || reader != expected {
		return unavailable
	}
	return func(ctx context.Context, path string, limit int64) ([]byte, error) {
		return mediapreview.ReadSnapshot(ctx, reader, sid, path, int(limit), valid)
	}
}

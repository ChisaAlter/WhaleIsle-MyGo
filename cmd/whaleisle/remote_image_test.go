package main

import (
	"context"
	"testing"

	"whaleisle/internal/store/session"
)

func TestUnavailableImageHostStillOwnsReader(t *testing.T) {
	for _, store := range []*session.Store{nil, session.New()} {
		read := remoteImageReader(store, nil, "session-a")
		if read == nil {
			t.Fatal("missing Host permitted a local image fallback")
		}
		if data, err := read(context.Background(), "same.png", 16<<20); err == nil || data != nil {
			t.Fatal("unavailable Host did not fail closed")
		}
	}
}

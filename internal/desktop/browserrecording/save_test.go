package browserrecording

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordingStopArtifactUsesExclusiveOwnedPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "recordings")
	a := Artifact{MIME: "video/webm;codecs=vp8", Data: syntheticWebM()}
	first, err := a.Save(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.Save(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || filepath.Dir(first) != dir || filepath.Ext(first) != ".webm" {
		t.Fatalf("unsafe output paths: %q %q", first, second)
	}
	for _, path := range []string{first, second} {
		if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, a.Data) {
			t.Fatalf("output=%q err=%v", got, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	missing := filepath.Join(t.TempDir(), "cancelled")
	if path, err := a.Save(ctx, missing); path != "" || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel path=%q err=%v", path, err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("cancelled Save created a directory")
	}
	badDir := filepath.Join(t.TempDir(), "invalid")
	if path, err := (Artifact{MIME: "video/webm", Data: []byte("not video")}).Save(context.Background(), badDir); path != "" || err == nil {
		t.Fatalf("invalid path=%q err=%v", path, err)
	}
	if _, err := os.Stat(badDir); !os.IsNotExist(err) {
		t.Fatal("invalid artifact created output")
	}
}

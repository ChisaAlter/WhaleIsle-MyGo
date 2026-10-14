package components

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunActionUnknownAndNotInstalled(t *testing.T) {
	ud := t.TempDir()
	if err := RunAction(ud, "whalebridge", "start"); !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("start = %v", err)
	}
	if err := RunAction(ud, "whalebridge", "nope"); !errors.Is(err, ErrUnknownAction) {
		t.Fatalf("unknown = %v", err)
	}
}

func TestStopWhalebridgeMissingState(t *testing.T) {
	root := t.TempDir()
	if err := stopWhalebridge(root); err != nil {
		t.Fatal(err)
	}
}

func TestFormatActionError(t *testing.T) {
	msg := FormatActionError("whalebridge", "start", ErrNotInstalled)
	if msg != "whalebridge 尚未安装" {
		t.Fatal(msg)
	}
	_ = os.MkdirAll(filepath.Join(t.TempDir(), "x"), 0o755)
}

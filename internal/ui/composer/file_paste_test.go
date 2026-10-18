package composer

import (
	"errors"
	"github.com/egoist/mygo/transfer"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeFilePasteFixture(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "document #1.pdf")
	if err := os.WriteFile(path, []byte("synthetic attachment"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := transfer.FileData(path)
	if err != nil {
		t.Fatal(err)
	}
	d := NewDraftState("keep text")
	calls := 0
	m := Model{Draft: &d.Text, State: d, OnAttachFiles: func(paths []string) {
		calls++
		if len(paths) != 1 || paths[0] != path {
			t.Fatal(paths)
		}
	}}
	if !pasteNativeFiles(m, data.Files) || calls != 1 || d.Text != "keep text" {
		t.Fatal("native file representation not attached")
	}
	if pasteNativeFiles(m, func() ([]string, error) { return nil, transfer.ErrFormat }) {
		t.Fatal("ordinary text paste consumed")
	}
	if !pasteNativeFiles(m, func() ([]string, error) { return nil, errors.New("clipboard changed") }) || d.Error == "" {
		t.Fatal("read error fell through")
	}
	d.Error = ""
	intakeTransferredFiles(m, []string{path, root})
	if calls != 1 || d.Error == "" {
		t.Fatal("directory mixed with files was admitted")
	}
	d.Error = ""
	d.Sending = true
	pasteNativeFiles(m, data.Files)
	if calls != 1 || d.Error == "" {
		t.Fatal("sending draft admitted files")
	}
	text := transfer.New(transfer.NewItem(transfer.Bytes(transfer.Text, []byte(path))))
	if paths, _ := text.Files(); len(paths) != 0 {
		t.Fatal("text guessed as file")
	}
}

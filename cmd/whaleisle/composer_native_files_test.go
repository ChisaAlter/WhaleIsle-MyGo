package main

import (
	"github.com/egoist/mygo/ui"
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/ui/composer"
)

func TestComposerNativeFilesSelectionAndRouting(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a b.txt")
	os.WriteFile(path, []byte("hello"), 0600)
	image := filepath.Join(root, "image.png")
	os.WriteFile(image, []byte("\x89PNG\r\n\x1a\n"), 0600)
	d := composer.NewDraftState("😀 old tail")
	d.Client.SetSelectionUTF16(3, 6)
	uploads, err := insertNativeComposerFiles(d, root, []string{path, image}, true)
	if err != nil || d.Text != "😀 @\"a b.txt\" tail" || len(uploads) != 1 || uploads[0] != image {
		t.Fatalf("%q %v %v", d.Text, uploads, err)
	}
	if !d.Client.Undo() || d.Text != "😀 old tail" {
		t.Fatal("selection edit not undoable")
	}
	uploads, err = insertNativeComposerFiles(d, root, []string{path}, false)
	if err != nil || len(uploads) != 1 || d.Text != "😀 old tail" {
		t.Fatal("remote Host received local path")
	}
}
func TestComposerNativeFilesDirectoryAndInvalidBatch(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "my dir")
	os.Mkdir(dir, 0700)
	d := composer.NewDraftState("")
	if _, err := insertNativeComposerFiles(d, root, []string{dir}, true); err != nil || d.Text != "@\"my dir/\" " {
		t.Fatalf("directory %q %v", d.Text, err)
	}
	before := d.Text
	if _, err := insertNativeComposerFiles(d, root, []string{dir, filepath.Join(root, "missing")}, true); err == nil || d.Text != before {
		t.Fatal("invalid batch partly inserted")
	}
}

func TestComposerNativeFilesActualEntryUnprovenHost(t *testing.T) {
	s := &mainState{}
	d := composer.NewDraftState("keep")
	path := filepath.Join(t.TempDir(), "local.txt")
	os.WriteFile(path, []byte("local"), 0600)
	s.addComposerFiles(nil, nil, "remote-session", d, []string{path})
	if d.Text != "keep" || len(d.Attachments) != 1 || d.Attachments[0].Path != path {
		t.Fatal("actual entry leaked a local reference into unproven Host")
	}
	d.Sending = true
	s.addComposerFiles(nil, nil, "remote-session", d, []string{path})
	if len(d.Attachments) != 1 {
		t.Fatal("sending admitted attachment")
	}
}
func TestComposerNativeFilesQuotedPathAndOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	os.WriteFile(outside, []byte("text"), 0600)
	d := composer.NewDraftState("")
	if _, err := insertNativeComposerFiles(d, root, []string{outside}, true); err != nil || d.Text != "@"+filepath.ToSlash(outside)+" " {
		t.Fatalf("outside path %q %v", d.Text, err)
	}
	bad := filepath.Join(root, "bad\"name.txt")
	if err := os.WriteFile(bad, []byte("text"), 0600); err != nil {
		t.Skip("platform rejects quoted filenames")
	}
	before := d.Text
	if _, err := insertNativeComposerFiles(d, root, []string{bad}, true); err == nil || d.Text != before {
		t.Fatal("unsafe quote inserted")
	}
}

func TestComposerNativeFilesIMEKeepsDraft(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	os.WriteFile(path, []byte("text"), 0600)
	d := composer.NewDraftState("keep")
	d.Client.SetMarkedText(nil, "拼", ui.TextInputRange{Start: 1, End: 1})
	before := d.Text
	if _, err := insertNativeComposerFiles(d, root, []string{path}, true); err == nil || d.Text != before {
		t.Fatal("IME draft changed")
	}
	if _, marked := d.Client.MarkedRange(); !marked {
		t.Fatal("IME composition discarded")
	}
}

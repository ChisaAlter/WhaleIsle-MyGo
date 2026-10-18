package composer

import (
	"bytes"
	"github.com/egoist/mygo/ui"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
	"whaleisle/internal/store/session"
)

func TestAttachmentDisplaySnapshotAndStatus(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 4, 3)))
	path := filepath.Join(t.TempDir(), "photo.png")
	os.WriteFile(path, b.Bytes(), 0600)
	v := readAttachmentDisplay(session.ComposerAttachment{Path: path})
	if v.bitmap == nil || v.bytes != int64(b.Len()) || v.mediaType != "image/png" {
		t.Fatalf("bad image %+v", v)
	}
	w, h := v.bitmap.Size()
	if w != 4 || h != 3 {
		t.Fatal("preview changed original pixels")
	}
	d := NewDraftState("")
	d.Attachments = []session.ComposerAttachment{{Path: path}}
	if attachmentDisplayStatus(d, 0) != "待上传 / Pending" {
		t.Fatal("pending status")
	}
	a, _ := d.BeginAttachmentPreparation(0, false)
	if attachmentDisplayStatus(d, 0) != "上传中 / Uploading" {
		t.Fatal("upload status")
	}
	a.ImageData = "ready"
	d.CompleteAttachmentPreparation(0, a, a, nil)
	if attachmentDisplayStatus(d, 0) != "就绪 / Ready" {
		t.Fatal("ready status")
	}
}

func TestAttachmentDisplayRailRemovalAndPreview(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 4, 3)))
	path := filepath.Join(t.TempDir(), "photo.png")
	os.WriteFile(path, b.Bytes(), 0600)
	d := NewDraftState("")
	d.Attachments = []session.ComposerAttachment{{Path: path, Name: "photo.png"}}
	d.BeginAttachmentPreparation(0, false)
	tt := ui.NewTester(func(c *ui.Context) { paintAttachmentRail(c, Model{State: d}, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80)) }, 600, 500)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, ok := tt.Find("Preview attachment photo.png"); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("thumbnail did not load")
		}
		time.Sleep(time.Millisecond * 5)
		tt.Move(0, 0)
	}
	tt.Click("Preview attachment photo.png")
	if _, ok := tt.Find("关闭预览 / Close preview"); !ok {
		t.Fatal("preview missing")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Move(0, 0)
	if _, ok := tt.Find("关闭预览 / Close preview"); ok {
		t.Fatal("Escape did not dismiss preview")
	}
	_, ok := tt.Find("Remove attachment photo.png")
	if !ok {
		t.Fatal("remove missing")
	}
	tt.Click("Remove attachment photo.png")
	if len(d.Attachments) != 0 || d.Preparing != 0 {
		t.Fatal("uploading attachment not removable")
	}
}

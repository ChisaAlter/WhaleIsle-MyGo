package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

type mainComposerFileIterator struct {
	files []pluginui.ComposerFileContent
	onEnd func()
}

func (s *mainComposerFileIterator) Next() (pluginui.ComposerFileContent, error) {
	if len(s.files) == 0 {
		if s.onEnd != nil {
			s.onEnd()
			s.onEnd = nil
		}
		return pluginui.ComposerFileContent{}, io.EOF
	}
	file := s.files[0]
	s.files = s.files[1:]
	return file, nil
}

func mainComposerDirectUI(fn func() error) error { return fn() }

type mainComposerUploadTransport struct {
	name string
	raw  []byte
}

func (s *mainComposerUploadTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var req struct {
		RPCID, Method string
		Payload       struct {
			Args struct {
				AgentID string
				Request struct{ Data, Name string }
			}
		}
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(req.Payload.Args.Request.Data)
	if err != nil || req.Method != "fileUploads/upload" || req.Payload.Args.AgentID != "composer-owner" || req.Payload.Args.Request.Name != s.name || !bytes.Equal(raw, s.raw) {
		return nil, fmt.Errorf("unexpected native upload filename/content/owner")
	}
	encoded, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"receiptId": "actual-file-receipt", "file": map[string]any{"attachmentId": "actual-file", "name": s.name, "bytes": len(raw)}}}})
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(encoded))}, nil
}

func TestMainParityPluginFileStreamImportsActualBytesAndResolvesBoundFile(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	s, b := mainParityComposerFixture(t)
	const name = "actual File 用户.txt"
	const sourceType = "application/x-original-file"
	raw := bytes.Repeat([]byte("source File/drop bytes\n"), 30000) // larger than the socket limit
	s.dshClient.HTTP = &http.Client{Transport: &mainComposerUploadTransport{name, raw}}
	o, err := b.capture(s.nativePlugins.session.subject)
	if err != nil {
		t.Fatal(err)
	}
	o.draft.Text = "preserve user input"
	o.draft.Client.Sync()
	posted := make(chan func(), 256)
	source := &mainComposerFileIterator{files: []pluginui.ComposerFileContent{{Name: name, MediaType: sourceType, Size: int64(len(raw)), Content: io.NopCloser(bytes.NewReader(raw))}}}
	if err := b.importFiles(context.Background(), composerPluginBinding(o), source, mainComposerDirectUI, func(fn func()) { posted <- fn }); err != nil {
		t.Fatal(err)
	}
	if len(o.draft.Attachments) != 1 || o.draft.Text != "preserve user input" || o.draft.Preparing != 1 || o.draft.Attachments[0].TotalBytes != int64(len(raw)) {
		t.Fatal("stream import did not atomically adopt the original File")
	}
	deadline := time.After(3 * time.Second)
	for o.draft.Preparing > 0 {
		select {
		case fn := <-posted:
			fn()
		case <-deadline:
			t.Fatal("native streamed-file preparation did not finish")
		}
	}
	a := o.draft.Attachments[0]
	if a.PrepareError != "" || a.Name != name || a.SessionID != "composer-owner" || a.HostScope == "" || a.Upload.ReceiptId != "actual-file-receipt" || a.TotalBytes != int64(len(raw)) {
		t.Fatalf("native upload did not retain filename/bytes/receipt: %+v", a)
	}
	owner, err := b.capture(o.subject)
	if err != nil {
		t.Fatal(err)
	}
	binding := composerPluginBinding(owner)
	file, err := b.readFile(context.Background(), binding, a.PreparationID, mainComposerDirectUI)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Content.Close()
	resolved, err := io.ReadAll(file.Content)
	if err != nil || !bytes.Equal(resolved, raw) || file.Name != name || file.Size != int64(len(raw)) || file.MediaType != sourceType || pluginComposerSnapshot(owner).Attachments[0].MediaType != sourceType {
		t.Fatal("resolveDraftAttachments lost the original File bytes/name/size")
	}
	if _, err := b.readFile(context.Background(), binding, a.PreparationID+1, mainComposerDirectUI); err == nil {
		t.Fatal("reader opened a nonexistent preparation ID")
	}
	file2, err := b.readFile(context.Background(), binding, a.PreparationID, mainComposerDirectUI)
	if err != nil {
		t.Fatal(err)
	}
	defer file2.Content.Close()
	o.draft.Client.ReplaceText(nil, "replacement draft")
	if n, err := file2.Content.Read(make([]byte, 64*1024)); n != 0 || err == nil {
		t.Fatal("reader continued after native draft revision changed")
	}
}

func TestMainParityPluginFileStreamAdoptsWholeBatchUnderOneBinding(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	s, b := mainParityComposerFixture(t)
	o, err := b.capture(s.nativePlugins.session.subject)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString("R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7")
	source := &mainComposerFileIterator{}
	for i, name := range []string{"first.gif", "第二张.gif"} {
		source.files = append(source.files, pluginui.ComposerFileContent{Name: name, MediaType: []string{"image/x-original", ""}[i], Size: -1, Content: io.NopCloser(bytes.NewReader(raw))})
	}
	posted := make(chan func(), 4)
	if err := b.importFiles(context.Background(), composerPluginBinding(o), source, mainComposerDirectUI, func(fn func()) { posted <- fn }); err != nil {
		t.Fatal(err)
	}
	if len(o.draft.Attachments) != 2 || o.draft.Preparing != 2 || o.draft.Attachments[0].PreparationID == o.draft.Attachments[1].PreparationID || o.draft.Text != "" {
		t.Fatal("batch was partially adopted or converted to local @references")
	}
	for o.draft.Preparing > 0 {
		select {
		case fn := <-posted:
			fn()
		case <-time.After(3 * time.Second):
			t.Fatal("native image batch did not complete")
		}
	}
	for i, name := range []string{"first.gif", "第二张.gif"} {
		a := o.draft.Attachments[i]
		if a.Name != name || a.PrepareError != "" || a.TotalBytes != int64(len(raw)) || a.ImageData != base64.StdEncoding.EncodeToString(raw) {
			t.Fatalf("actual image file lost its name/size/bytes: %+v", a)
		}
		owner, err := b.capture(o.subject)
		if err != nil {
			t.Fatal(err)
		}
		file, err := b.readFile(context.Background(), composerPluginBinding(owner), a.PreparationID, mainComposerDirectUI)
		if err != nil {
			t.Fatal(err)
		}
		file.Content.Close()
		wantType := []string{"image/x-original", ""}[i]
		public := pluginComposerSnapshot(owner).Attachments[i]
		encoded, _ := json.Marshal(a)
		var restored session.ComposerAttachment
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatal(err)
		}
		if a.MediaType != "image/gif" || file.MediaType != wantType || public.MediaType != wantType || !public.Image || restored.SourceMediaType == nil || *restored.SourceMediaType != wantType {
			t.Fatal("original File.type/custom empty override lost across native preparation/reader/checkpoint")
		}
	}
}

type mainComposerBrokenReader struct{}

func (mainComposerBrokenReader) Read([]byte) (int, error) {
	return 0, errors.New("synthetic source stream failure")
}

func TestMainParityPluginFileStreamCleansUnadoptedCache(t *testing.T) {
	for _, mode := range []string{"stale-before", "stale-after", "source-error", "size-mismatch", "canceled", "bad-name"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv(paths.EnvUserData, t.TempDir())
			s, b := mainParityComposerFixture(t)
			o, err := b.capture(s.nativePlugins.session.subject)
			if err != nil {
				t.Fatal(err)
			}
			binding := composerPluginBinding(o)
			content := pluginui.ComposerFileContent{Name: "actual.txt", Size: 3, Content: io.NopCloser(strings.NewReader("abc"))}
			source := &mainComposerFileIterator{files: []pluginui.ComposerFileContent{content}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "stale-before":
				binding.Revision++
			case "stale-after":
				source.onEnd = func() { o.draft.Client.ReplaceText(nil, "edited during transfer") }
			case "source-error":
				source.files = append(source.files, pluginui.ComposerFileContent{Name: "broken.txt", Size: -1, Content: io.NopCloser(mainComposerBrokenReader{})})
			case "size-mismatch":
				source.files[0].Size++
			case "canceled":
				source.onEnd = cancel
			case "bad-name":
				source.files[0].Name = "../../private.txt"
			}
			if err := b.importFiles(ctx, binding, source, mainComposerDirectUI, func(func()) { t.Error("unadopted file posted preparation") }); err == nil {
				t.Fatal("invalid File import was accepted")
			}
			if len(o.draft.Attachments) != 0 || o.draft.Preparing != 0 {
				t.Fatal("failed import partially changed native attachments")
			}
			files, err := filepath.Glob(filepath.Join(paths.UserDataDir(), "cache", "attachments", "*"))
			if err != nil || len(files) != 0 {
				t.Fatalf("unadopted cache retained: %v, %v", files, err)
			}
		})
	}
}

func TestMainParityPluginFileErrorsDoNotExposeNativePath(t *testing.T) {
	b, o, _, _ := mainParityPluginRetryFixture(t)
	a := &o.draft.Attachments[0]
	a.PrepareError = (&os.PathError{Op: "open", Path: a.Path, Err: os.ErrNotExist}).Error()
	o, err := b.capture(o.subject)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(pluginComposerSnapshot(o))
	privatePath, _ := json.Marshal(a.Path)
	if strings.Contains(string(encoded), string(privatePath[1:len(privatePath)-1])) || strings.Contains(composerFileError(&os.PathError{Op: "open", Path: a.Path, Err: os.ErrNotExist}).Error(), a.Path) {
		t.Fatal("attachment error exposed private cache path")
	}
}

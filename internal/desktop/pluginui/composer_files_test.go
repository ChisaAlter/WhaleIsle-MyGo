package pluginui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

type composerFilesFixture struct {
	composerWireFixture
	files          []ComposerFileContent
	bytes          [][]byte
	imports, reads int
}

func (f *composerFilesFixture) ImportFiles(ctx context.Context, owner ComposerBinding, files ComposerFileIterator) error {
	f.imports++
	if f.draft.Owner != owner {
		return errors.New("stale native binding")
	}
	var batch []ComposerFileContent
	var data [][]byte
	for {
		file, err := files.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		b, err := io.ReadAll(file.Content)
		_ = file.Content.Close()
		if err != nil {
			return err
		}
		batch = append(batch, file)
		data = append(data, b)
	}
	if f.draft.Owner != owner {
		return errors.New("stale native binding")
	}
	f.files, f.bytes = batch, data
	f.draft.Owner.Revision++
	for i, file := range batch {
		f.draft.Attachments = append(f.draft.Attachments, ComposerFile{ID: uint64(i + 1), Name: file.Name, MediaType: file.MediaType, Size: int64(len(data[i]))})
	}
	return nil
}
func (f *composerFilesFixture) ReadFile(ctx context.Context, owner ComposerBinding, id uint64) (ComposerFileContent, error) {
	f.reads++
	if f.draft.Owner != owner || id == 0 || int(id) > len(f.files) {
		return ComposerFileContent{}, errors.New("foreign attachment")
	}
	file := f.files[id-1]
	file.Content = io.NopCloser(bytes.NewReader(f.bytes[id-1]))
	return file, nil
}
func multipartComposerRequest(t *testing.T, transfer composerTransferRequest, contents [][]byte, names []string, sizes []int64) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	owner, _ := w.CreateFormField("owner")
	_ = json.NewEncoder(owner).Encode(transfer)
	for i, b := range contents {
		m, _ := w.CreateFormField("metadata")
		_ = json.NewEncoder(m).Encode(map[string]any{"name": names[i], "mediaType": "text/plain", "size": sizes[i]})
		part, _ := w.CreateFormFile("file", names[i])
		_, _ = part.Write(b)
	}
	_ = w.Close()
	r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:45983"+composerFilesPath, &body)
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.Header.Set("Origin", "http://"+r.Host)
	return r
}
func TestComposerFileStreamKeepsOriginalBytesOutsideSocketBudget(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "s", Epoch: "1:2"}
	fixture := &composerFilesFixture{composerWireFixture: composerWireFixture{draft: ComposerDraft{Owner: ComposerBinding{Subject: subject, Binding: "captured", Revision: 7}}}}
	o := &Owner{selected: subject, composer: fixture, page: &bridgePage{id: 9}}
	transfer := composerTransferRequest{Page: 9, Owner: fixture.draft.Owner}
	large := bytes.Repeat([]byte("exact original browser File\n"), 28000)
	r := multipartComposerRequest(t, transfer, [][]byte{large, []byte("second file")}, []string{"原文件.txt", "second.txt"}, []int64{int64(len(large)), 11})
	w := httptest.NewRecorder()
	if !o.serve(w, r) || w.Code != http.StatusOK {
		t.Fatalf("stream import: %d %s", w.Code, w.Body.String())
	}
	if len(large) <= 512*1024 || !bytes.Equal(fixture.bytes[0], large) || fixture.files[0].Name != "原文件.txt" || fixture.files[0].MediaType != "text/plain" || len(fixture.files) != 2 {
		t.Fatal("original file identity/content lost")
	}
	var reply struct {
		Page  uint64
		Draft ComposerDraft
	}
	if err := json.Unmarshal(w.Body.Bytes(), &reply); err != nil || reply.Page != 9 || len(reply.Draft.Attachments) != 2 {
		t.Fatal(err, reply)
	}
	body, _ := json.Marshal(composerTransferRequest{Page: 9, Owner: reply.Draft.Owner, ID: 1})
	r = httptest.NewRequest(http.MethodPost, "http://127.0.0.1:45983"+composerFilePath, bytes.NewReader(body))
	r.Header.Set("Origin", "http://"+r.Host)
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	o.serve(w, r)
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), large) || w.Header().Get("Content-Type") != "text/plain" || w.Header().Get("Content-Disposition") == "" {
		t.Fatal("opaque-ID reader did not return exact approved bytes", w.Code)
	}
}
func TestComposerFileStreamRejectsExpiredOwnersAndIncompleteParts(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "s", Epoch: "1:2"}
	fixture := &composerFilesFixture{composerWireFixture: composerWireFixture{draft: ComposerDraft{Owner: ComposerBinding{Subject: subject, Binding: "captured", Revision: 7}}}}
	o := &Owner{selected: subject, composer: fixture, page: &bridgePage{id: 9}}
	for _, change := range []func(*composerTransferRequest, *http.Request){
		func(q *composerTransferRequest, r *http.Request) { q.Page = 8 },
		func(q *composerTransferRequest, r *http.Request) { q.Owner.Revision-- },
		func(q *composerTransferRequest, r *http.Request) { q.Owner.Subject.SessionID = "other" },
		func(q *composerTransferRequest, r *http.Request) { r.Header.Set("Origin", "http://foreign") },
	} {
		q := composerTransferRequest{Page: 9, Owner: fixture.draft.Owner}
		dummy := httptest.NewRequest("POST", "http://127.0.0.1", nil)
		change(&q, dummy)
		r := multipartComposerRequest(t, q, [][]byte{[]byte("bytes")}, []string{"file.txt"}, []int64{5})
		if dummy.Header.Get("Origin") != "" {
			r.Header.Set("Origin", dummy.Header.Get("Origin"))
		}
		w := httptest.NewRecorder()
		o.serve(w, r)
		if w.Code == 200 || fixture.imports != 0 {
			t.Fatal("expired/foreign import reached native consumer", w.Code)
		}
	}
	for _, size := range []int64{4, 6} {
		r := multipartComposerRequest(t, composerTransferRequest{Page: 9, Owner: fixture.draft.Owner}, [][]byte{[]byte("bytes")}, []string{"file.txt"}, []int64{size})
		w := httptest.NewRecorder()
		o.serve(w, r)
		if w.Code == 200 || len(fixture.bytes) != 0 {
			t.Fatal("truncated/mismatched file committed")
		}
	}
	if o.ComposerError() == "" {
		t.Fatal("transfer failure has no native composer feedback")
	}
	body, _ := json.Marshal(composerTransferRequest{Page: 9, Owner: fixture.draft.Owner, ID: 99})
	r := httptest.NewRequest("POST", "http://127.0.0.1:45983"+composerFilePath, bytes.NewReader(body))
	r.Header.Set("Origin", "http://"+r.Host)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	o.serve(w, r)
	if w.Code == 200 || fixture.reads != 0 {
		t.Fatal("unowned ID reached native reader")
	}
}

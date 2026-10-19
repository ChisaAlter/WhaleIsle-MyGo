package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

// Prepared fixtures only: the transport is an in-memory RoundTripper. No
// listener, Host, helper, renderer, model call, or external request is needed.
type visualReplyMemoryTransport func(*http.Request) (*http.Response, error)

func (f visualReplyMemoryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestVisualReplyRetryabilityMatchesPublishedPage(t *testing.T) {
	for _, reason := range []string{"ATTACHMENT_NOT_REFERENCED", "VISUAL_REPLY_TOO_LARGE", "VISUAL_REPLY_INVALID_UTF8", "ATTACHMENT_NOT_FOUND", "ATTACHMENT_CORRUPT", "INVALID_ATTACHMENT_REF"} {
		t.Run(reason, func(t *testing.T) {
			details, _ := json.Marshal(map[string]string{"reason": reason})
			err := fmt.Errorf("wrapped: %w", &dsh.Error{Code: "session/attachment-invalid", Details: details})
			if VisualReplyRetryable(err) {
				t.Fatal("permanent Host failure offers another read")
			}
		})
	}
	if VisualReplyRetryable(&dsh.Error{Code: "session/not-found"}) || VisualReplyRetryable(nil) {
		t.Fatal("missing session or success offers retry")
	}
	for _, err := range []error{
		context.DeadlineExceeded, context.Canceled, errors.New("isolated page closed"),
		&dsh.Error{Code: "session/attachment-invalid", Details: json.RawMessage(`{"reason":"HOST_BUSY"}`)},
		&dsh.Error{Code: "session/attachment-invalid", Details: json.RawMessage(`{"reason":12}`)},
		&dsh.Error{Code: "session/attachment-invalid", Details: json.RawMessage(`invalid`)},
		&dsh.Error{Code: "unauthorized"}, &dsh.Error{Code: "host/unavailable"},
	} {
		if !VisualReplyRetryable(err) {
			t.Fatalf("unknown/transient failure lost retry: %v", err)
		}
	}
}

func TestVisualReplyLocalContentFailuresArePermanent(t *testing.T) {
	ref := VisualReplyFile{AttachmentID: "page", Name: "page.html", Bytes: 3}
	changed := ref
	changed.AttachmentID = "other"
	large := strings.Repeat("x", MaxVisualReplyBytes+1)
	largeRef := ref
	largeRef.Bytes = int64(len(large))
	for _, tc := range []struct {
		name             string
		expected, actual VisualReplyFile
		html             string
	}{
		{"reference", ref, changed, "abc"}, {"bytes", ref, ref, "ab"},
		{"utf8", ref, ref, string([]byte{0xff, 0xff, 0xff})},
		{"oversize", largeRef, largeRef, large},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateVisualReplySnapshot(tc.expected, tc.actual, tc.html)
			var invalid *visualReplyContentError
			if !errors.As(err, &invalid) || VisualReplyRetryable(err) {
				t.Fatalf("invalid publication remains retryable: %v", err)
			}
		})
	}
	if err := validateVisualReplySnapshot(ref, ref, "abc"); err != nil {
		t.Fatal(err)
	}
}

func memoryVisualReplyStore(t *testing.T, failure *dsh.Error, duringRead func(*Store)) *Store {
	t.Helper()
	ref := VisualReplyFile{AttachmentID: "page", Name: "page.html", Bytes: 3}
	raw, _ := json.Marshal(map[string]any{"type": "tool/result", "surfaceOp": "append", "data": map[string]any{
		"message": map[string]any{"content": []any{map[string]any{"type": "file", "attachment": ref}}},
		"meta":    map[string]any{"visualReply": map[string]any{"version": 1, "title": "Fixture"}},
	}})
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://memory.invalid"}, HTTP: &http.Client{Transport: visualReplyMemoryTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/session/readVisualReply" {
			return nil, fmt.Errorf("unexpected fixture path: %s", r.URL.Path)
		}
		var request struct {
			RPCID string `json:"rpcId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return nil, err
		}
		result := map[string]any{"ok": true, "value": map[string]any{"attachment": ref, "html": "abc"}}
		if failure != nil {
			result = map[string]any{"ok": false, "error": map[string]any{"code": failure.Code, "message": failure.Message, "details": failure.Details}}
		}
		if duringRead != nil {
			duringRead(s)
		}
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": result})
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}}, nil)
	s.selectedID, s.messages = "A", []Message{{Raw: raw}}
	return s
}

func TestVisualReplySnapshotPreservesTypedFailureAndOwnerSemantics(t *testing.T) {
	for _, tc := range []struct {
		code, reason string
		retry        bool
	}{
		{"session/not-found", "", false}, {"session/attachment-invalid", "ATTACHMENT_CORRUPT", false},
		{"session/attachment-invalid", "HOST_BUSY", true}, {"host/unavailable", "", true},
	} {
		details, _ := json.Marshal(map[string]string{"reason": tc.reason})
		s := memoryVisualReplyStore(t, &dsh.Error{Code: tc.code, Details: details}, nil)
		_, _, err := s.ReadVisualReplySnapshot(context.Background(), "A", "page")
		var remote *dsh.Error
		if !errors.As(err, &remote) || remote.Code != tc.code || VisualReplyRetryable(err) != tc.retry {
			t.Fatalf("typed read classification: %v", err)
		}
	}
	s := memoryVisualReplyStore(t, nil, func(s *Store) { s.mu.Lock(); s.followGeneration++; s.mu.Unlock() })
	if _, _, err := s.ReadVisualReplySnapshot(context.Background(), "A", "page"); err == nil || !VisualReplyRetryable(err) {
		t.Fatalf("owner retirement became a permanent content failure: %v", err)
	}
	s = memoryVisualReplyStore(t, nil, nil)
	if _, _, err := s.ReadVisualReplySnapshot(context.Background(), "A", "absent"); err == nil || VisualReplyRetryable(err) {
		t.Fatalf("unpublished reference offers retry: %v", err)
	}
	s.mu.Lock()
	s.client = nil
	s.mu.Unlock()
	if _, _, err := s.ReadVisualReplySnapshot(context.Background(), "A", "page"); err == nil || !VisualReplyRetryable(err) {
		t.Fatalf("Host absence became permanent: %v", err)
	}
}

func TestVisualReplyPublicationInitialHeight(t *testing.T) {
	ref := VisualReplyFile{AttachmentID: "page", Name: "page.html", Bytes: 3}
	for _, tc := range []struct {
		name    string
		value   any
		present bool
		want    int
		valid   bool
	}{
		{"absent", nil, false, 480, true}, {"null", nil, true, 480, true},
		{"minimum", 80, true, 80, true}, {"maximum", 2000, true, 2000, true},
		{"fractional native pixel", 599.2, true, 600, true},
		{"below", 79, true, 0, false}, {"above", 2001, true, 0, false}, {"string", "480", true, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, ptc := range []bool{false, true} {
				presentation := map[string]any{"version": 1, "title": "Fixture"}
				if tc.present {
					presentation["height"] = tc.value
				}
				content := []any{map[string]any{"type": "file", "attachment": ref}}
				event := map[string]any{"type": "tool/result", "surfaceOp": "append", "data": map[string]any{"message": map[string]any{"content": content}, "meta": map[string]any{"visualReply": presentation}}}
				if ptc {
					event = map[string]any{"type": "tool/ptc-dispatch", "data": map[string]any{"name": "html_render", "arguments": presentation, "content": content}}
				}
				raw, _ := json.Marshal(event)
				publication, ok := PublishedVisualReply(raw)
				if ok != tc.valid || (ok && publication.Height != tc.want) {
					t.Fatalf("PTC=%v publication=%+v accepted=%v", ptc, publication, ok)
				}
			}
		})
	}
}

func TestVisualReplyPageSnapshotKeepsMetadataWithoutDuplicateRead(t *testing.T) {
	reads := 0
	s := memoryVisualReplyStore(t, nil, func(*Store) { reads++ })
	s.messages[0].Raw = json.RawMessage(strings.Replace(string(s.messages[0].Raw), `"title":"Fixture"`, `"title":"Fixture","height":912`, 1))
	page, valid, err := s.ReadVisualReplyPageSnapshot(context.Background(), "A", "page")
	if err != nil || valid == nil || !valid() || page.HTML != "abc" || page.Publication.Height != 912 || reads != 1 {
		t.Fatalf("page=%+v reads=%d error=%v", page, reads, err)
	}
	html, valid, err := s.ReadVisualReplySnapshot(context.Background(), "A", "page")
	if err != nil || valid == nil || !valid() || html != "abc" || reads != 2 {
		t.Fatalf("compatibility wrapper reads=%d error=%v", reads, err)
	}
}

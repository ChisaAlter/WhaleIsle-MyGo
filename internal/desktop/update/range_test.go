package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchRangePartialContent(t *testing.T) {
	payload := []byte("0123456789abcdef")
	var gotRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		start, end, ok := parseBytesRange(r.Header.Get("Range"), int64(len(payload)))
		if !ok {
			http.Error(w, "bad range", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[start : end+1])
	}))
	defer srv.Close()

	got, err := FetchRange(context.Background(), srv.URL, 4, 4, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "4567" {
		t.Fatalf("got %q", got)
	}
	if gotRange != "bytes=4-7" {
		t.Fatalf("Range header %q", gotRange)
	}
}

func TestFetchRangeUnsupportedFallsBackHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("full-body-ignores-range"))
	}))
	defer srv.Close()

	_, err := FetchRange(context.Background(), srv.URL, 0, 4, srv.Client())
	if !errors.Is(err, ErrRangeUnsupported) {
		t.Fatalf("want ErrRangeUnsupported, got %v", err)
	}
}

func TestFetchRangeHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()
	if _, err := FetchRange(context.Background(), srv.URL, 0, 1, srv.Client()); err == nil {
		t.Fatal("expected error")
	}
}

func parseBytesRange(h string, size int64) (start, end int64, ok bool) {
	var s, e int64
	if _, err := fmt.Sscanf(h, "bytes=%d-%d", &s, &e); err != nil {
		return 0, 0, false
	}
	if s < 0 || e < s || e >= size {
		return 0, 0, false
	}
	return s, e, true
}

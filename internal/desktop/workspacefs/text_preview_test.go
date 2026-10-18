package workspacefs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type textPrefixCounter struct{ n int }

func (r *textPrefixCounter) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	r.n += len(p)
	return len(p), nil
}

type textPrefixError struct{ err error }

func (r textPrefixError) Read([]byte) (int, error) { return 0, r.err }

func TestTextPrefixBoundaries(t *testing.T) {
	for _, n := range []int{0, 512*1024 + 1, TextPreviewLimit, TextPreviewLimit + 1} {
		body := strings.Repeat("a", n)
		got, err := readTextPrefix(strings.NewReader(body), int64(n))
		if err != nil || len(got.Text) != min(n, TextPreviewLimit) || got.Truncated != (n > TextPreviewLimit) || got.Size != int64(n) {
			t.Fatalf("n=%d: bytes=%d truncated=%v size=%d err=%v", n, len(got.Text), got.Truncated, got.Size, err)
		}
	}
	r := &textPrefixCounter{}
	got, err := readTextPrefix(r, 0) // file grew after Stat
	if err != nil || !got.Truncated || len(got.Text) != TextPreviewLimit || r.n != TextPreviewLimit+1 {
		t.Fatalf("unbounded or unmarked read: read=%d prefix=%d truncated=%v err=%v", r.n, len(got.Text), got.Truncated, err)
	}
	if got, err := readTextPrefix(strings.NewReader("short"), TextPreviewLimit+1); err != nil || !got.Truncated {
		t.Fatalf("concurrent shrink must remain conservatively read-only: %+v %v", got, err)
	}
	errRead := errors.New("synthetic read failure")
	if _, err := readTextPrefix(textPrefixError{errRead}, 0); !errors.Is(err, errRead) {
		t.Fatalf("lost read failure: %v", err)
	}
}

func TestTextPrefixEncoding(t *testing.T) {
	for _, runeText := range []string{"é", "界", "🙂"} {
		for split := 1; split < len(runeText); split++ {
			prefix := strings.Repeat("a", TextPreviewLimit-split)
			body := prefix + runeText + "tail"
			got, err := readTextPrefix(strings.NewReader(body), int64(len(body)))
			if err != nil || !got.Truncated || got.Text != prefix {
				t.Fatalf("rune=%q split=%d bytes=%d err=%v", runeText, split, len(got.Text), err)
			}
		}
	}
	for _, body := range [][]byte{{'a', 0, 'b'}, {'a', 0xff}, {'a', 0xe7, 0x95}, append([]byte{0xff}, bytes.Repeat([]byte{'a'}, TextPreviewLimit)...)} {
		if _, err := readTextPrefix(bytes.NewReader(body), int64(len(body))); !errors.Is(err, ErrBinary) {
			t.Fatalf("invalid UTF-8 or NUL accepted: length=%d err=%v", len(body), err)
		}
	}
}

func TestTextPrefixStrictSaveBoundary(t *testing.T) {
	root := t.TempDir() // Synthetic files only; no user workspace or Host.
	path := filepath.Join(root, "generated.txt")
	body := strings.Repeat("x", TextPreviewLimit+64)
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	prefix, err := ReadFileTextPreview(path)
	if err != nil || !prefix.Truncated || len(prefix.Text) != TextPreviewLimit || prefix.Size != int64(len(body)) {
		t.Fatalf("preview: bytes=%d truncated=%v size=%d err=%v", len(prefix.Text), prefix.Truncated, prefix.Size, err)
	}
	if _, err := ReadFileText(path, TextPreviewLimit); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("strict reader accepted partial baseline: %v", err)
	}
	calls := 0
	write := func(_, _ string) error { calls++; return nil }
	if err := SaveTextChecked(context.Background(), root, "generated.txt", prefix.Text, prefix.Text, write); !errors.Is(err, ErrTooLarge) || calls != 0 {
		t.Fatalf("partial preview reached writer: calls=%d err=%v", calls, err)
	}
	complete := strings.Repeat("c", 768*1024)
	if err := os.WriteFile(path, []byte(complete), 0600); err != nil {
		t.Fatal(err)
	}
	if err := SaveTextChecked(context.Background(), root, "generated.txt", complete, "reviewed draft", write); err != nil || calls != 1 {
		t.Fatalf("complete 512 KiB–1 MiB baseline rejected: calls=%d err=%v", calls, err)
	}
	if err := SaveTextChecked(context.Background(), root, "generated.txt", "old baseline", "draft", write); err == nil || calls != 1 {
		t.Fatalf("conflict protection weakened: calls=%d err=%v", calls, err)
	}
	if _, err := ReadFileTextPreview(root); !errors.Is(err, ErrNotFile) {
		t.Fatalf("directory accepted: %v", err)
	}
	if _, err := ReadFileTextPreview(filepath.Join(root, "missing.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing error lost: %v", err)
	}
	// Confirm the fake writer did not replace the generated file with a prefix.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	actual, err := io.ReadAll(f)
	if err != nil || string(actual) != complete {
		t.Fatalf("synthetic file changed: bytes=%d err=%v", len(actual), err)
	}
}

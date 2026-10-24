package workspacefs

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProjectFileSearchSubsequence(t *testing.T) {
	root := gitIsolatedRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "src", "client"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"FilePreview.tsx", "FileTree.tsx", "other.txt"} {
		if err := os.WriteFile(filepath.Join(root, "src", "client", name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{"FP", " @./f p ", "src/fprev", "FilePreview.tsx"} {
		got, err := SearchRelPaths(context.Background(), root, query, nil)
		if err != nil || !reflect.DeepEqual(got, []string{"src/client/FilePreview.tsx"}) {
			t.Errorf("query %q: got %v, err %v", query, got, err)
		}
	}
	got, err := SearchRelPaths(context.Background(), root, "pf", nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("reversed subsequence matched: %v, %v", got, err)
	}
	// A query reduced to empty shows all files, including nested files, not dirs.
	got, err = SearchRelPaths(context.Background(), root, "@./", nil)
	if err != nil || len(got) != 3 {
		t.Fatalf("prefix-only search: %v, %v", got, err)
	}
}

func TestProjectFileSearchRemoteSubsequence(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unmounted")
	remote := func(_ context.Context, rel string) ([]Entry, error) {
		if rel == "" {
			return []Entry{{Name: "目录", IsDir: true}}, nil
		}
		return []Entry{{Name: "文件预览.go"}, {Name: "Other.go"}}, nil
	}
	got, err := SearchRelPaths(context.Background(), root, "@ 文览", remote)
	if err != nil || !reflect.DeepEqual(got, []string{"目录/文件预览.go"}) {
		t.Fatalf("remote query: %v, %v", got, err)
	}
}

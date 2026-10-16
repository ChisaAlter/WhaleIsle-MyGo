package workspacefs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// TextPage retains the source's cursor: bytes locally, one-based lines on Host.
// Separator is explicit because workspaceFiles/read omits page-ending newlines.
type TextPage struct {
	Text, Version, Separator string
	Next                     int64
	EOF                      bool
}

func ReadFileTextPage(path string, offset int64) (TextPage, error) {
	if offset < 0 {
		return TextPage{}, errors.New("negative text offset")
	}
	f, err := os.Open(path)
	if err != nil {
		return TextPage{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return TextPage{}, err
	}
	if !st.Mode().IsRegular() {
		return TextPage{}, ErrNotFile
	}
	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return TextPage{}, err
	}
	r, err := readTextPrefix(f, st.Size()-offset)
	if err != nil {
		return TextPage{}, err
	}
	return TextPage{Text: r.Text, Next: offset + int64(len(r.Text)), EOF: !r.Truncated, Version: fmt.Sprintf("%d:%d", st.Size(), st.ModTime().UnixNano())}, nil
}

func ReadRemoteTextPage(ctx context.Context, client *dsh.Client, sessionID, path string, offset int64) (TextPage, error) {
	if client == nil {
		return TextPage{}, ErrNotFile
	}
	if strings.TrimSpace(sessionID) == "" {
		return TextPage{}, errors.New("Host text paging requires a session scope")
	}
	if offset == 0 {
		offset = 1
	}
	start, limit := float64(offset), float64(2000)
	var out api.WorkspaceFileText
	if err := client.Call(ctx, "workspaceFiles/read", map[string]any{"workspaceFileScopeId": sessionID, "path": path, "range": api.WorkspaceFileRange{Offset: &start, Limit: &limit}}, &out); err != nil {
		return TextPage{}, err
	}
	if out.Offset != start || (!out.Eof && out.Lines <= 0) {
		return TextPage{}, errors.New("text page did not advance")
	}
	separator := ""
	if offset > 1 {
		separator = "\n"
	}
	return TextPage{Text: out.Text, Version: out.Version, Separator: separator, Next: int64(out.Offset + out.Lines), EOF: out.Eof}, nil
}

package workspacefs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadWorkbookPage reads XLS or XLSX through the existing read-only page model.
// The caller must authorize filePath with JoinRoot before invoking this reader.
func ReadWorkbookPage(ctx context.Context, filePath string, request XLSXPageRequest) (*XLSXPage, error) {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".xlsx":
		return ReadXLSXPage(ctx, filePath, request)
	case ".xls":
		return readXLSFilePage(ctx, filePath, request)
	default:
		return nil, errors.New("Native workbook reader supports XLSX and XLS files")
	}
}

// ReadWorkbookBytesPage is shared by remote Host and child-session snapshots.
// OLE compound workbooks use the XLS reader; XLSX retains its existing reader.
func ReadWorkbookBytesPage(ctx context.Context, data []byte, request XLSXPageRequest) (*XLSXPage, error) {
	if !bytes.HasPrefix(data, []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}) {
		return ReadXLSXBytesPage(ctx, data, request)
	}
	if int64(len(data)) > xlsxMaxFileBytes {
		return nil, errors.New("XLS exceeds the 16 MiB native reader limit")
	}
	version := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	return readXLSPage(ctx, bytes.NewReader(data), version, request)
}

func readXLSFilePage(ctx context.Context, filePath string, request XLSXPageRequest) (*XLSXPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("XLS preview requires a regular file")
	}
	if st.Size() > xlsxMaxFileBytes {
		return nil, errors.New("XLS exceeds the 16 MiB native reader limit; use Open external")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	st, err = file.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > xlsxMaxFileBytes {
		return nil, errors.New("XLS changed or exceeds the 16 MiB native reader limit")
	}
	version := fmt.Sprintf("%d:%d", st.Size(), st.ModTime().UnixNano())
	page, err := readXLSPage(ctx, io.NewSectionReader(file, 0, st.Size()), version, request)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	current, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(st, current) || after.Size() != st.Size() || !after.ModTime().Equal(st.ModTime()) {
		return nil, errors.New("Workbook changed during reading; choose Reload")
	}
	return page, nil
}

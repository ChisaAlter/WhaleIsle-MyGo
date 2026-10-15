package files

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"whaleisle/internal/desktop/workspacefs"
)

// Workbook previews are always read-only. They never enter a UTF-8 Host reader
// or use their rendered page as a file-save baseline.
func isTablePreviewPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xlsx", ".xls":
		return true
	}
	return false
}
func readTablePreview(ctx context.Context, fullPath string, request workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.ToLower(filepath.Ext(fullPath)) != ".xlsx" {
		return nil, errors.New("Legacy XLS is not supported by the native table reader. Use Open external for this workbook; binary data is never sent to the text reader.")
	}
	return workspacefs.ReadXLSXPage(ctx, fullPath, request)
}

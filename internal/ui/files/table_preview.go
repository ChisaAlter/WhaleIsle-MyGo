package files

import (
	"context"
	"path/filepath"
	"strings"
	"whaleisle/internal/desktop/workspacefs"
)

// Workbook previews are always read-only. They never enter a UTF-8 Host reader
// or use their rendered page as a file-save baseline.
func isTablePreviewPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xlsx", ".xls", ".csv", ".tsv":
		return true
	}
	return false
}
func readTablePreview(ctx context.Context, fullPath string, request workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return workspacefs.ReadWorkbookPage(ctx, fullPath, request)
}

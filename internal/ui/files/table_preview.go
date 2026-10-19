package files

import (
	"context"
	"path/filepath"
	"strings"
	"whaleisle/internal/desktop/workspacefs"
)

// Binary workbook previews are always read-only. CSV/TSV stay in the desktop
// source editor, whose complete text is the original file-save baseline.
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
	return workspacefs.ReadWorkbookPage(ctx, fullPath, request)
}

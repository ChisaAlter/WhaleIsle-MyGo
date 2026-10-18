package composer

import (
	"path/filepath"
	"strings"

	"whaleisle/internal/desktop/workspacefs"
)

// AppendDroppedFiles appends @relative/path tokens (or markdown images for
// image files) for absolute paths under root. Paths outside the workspace are
// skipped. Existing draft text gets a leading space before the first mention
// when non-empty.
func AppendDroppedFiles(draft *string, root string, absPaths []string) {
	if draft == nil || root == "" || len(absPaths) == 0 {
		return
	}
	var mentions []string
	for _, abs := range absPaths {
		abs = strings.TrimSpace(abs)
		if abs == "" {
			continue
		}
		rel, err := workspacefs.RelPath(root, abs)
		if err != nil || rel == "" {
			continue
		}
		rel = filepath.ToSlash(rel)
		if isImageRel(rel) {
			mentions = append(mentions, "![]("+rel+")")
		} else {
			mentions = append(mentions, "@"+rel)
		}
	}
	if len(mentions) == 0 {
		return
	}
	block := strings.Join(mentions, " ")
	cur := strings.TrimRight(*draft, " \t")
	if cur == "" {
		*draft = block
		return
	}
	*draft = cur + " " + block
}

func isImageRel(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg":
		return true
	default:
		return false
	}
}

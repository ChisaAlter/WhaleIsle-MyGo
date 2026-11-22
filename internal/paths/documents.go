package paths

import (
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/product"
)

// documentsRoot resolves the folder the first-run workspace lives under. A
// managed user data root (WHALEISLE_USER_DATA — isolated runs, tests and
// portable installs) keeps its Documents inside itself so a run never creates
// folders in the user's real Documents; otherwise it is the platform's
// Documents folder (src/main/config.js defaultWorkspace).
func documentsRoot() string {
	if root := strings.TrimSpace(os.Getenv(EnvUserData)); root != "" {
		return filepath.Join(root, "Documents")
	}
	dir, err := osDocumentsDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		return filepath.Join(UserDataDir(), "Documents")
	}
	return dir
}

// DefaultWorkspace is the first-run project folder when config.workspace is
// empty: <Documents>\<product>, mirroring the reference shell's
// defaultWorkspace, which also recreates the folder when it vanished.
func DefaultWorkspace() string {
	return filepath.Join(documentsRoot(), product.EnglishName)
}

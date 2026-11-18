package credentials

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireLockReleasesLockWhenMetadataWriteFails(t *testing.T) {
	dir := t.TempDir()
	original := lockMetadataWrite
	lockMetadataWrite = func(*os.File, []byte) (int, error) { return 0, errors.New("disk full") }
	defer func() { lockMetadataWrite = original }()

	if _, err := acquireCredentialsLock(dir); err == nil {
		t.Fatal("expected acquisition to fail when the metadata write fails")
	}
	if _, statErr := os.Stat(filepath.Join(dir, lockFileName)); !os.IsNotExist(statErr) {
		t.Fatalf("lock file survived a failed metadata write: %v", statErr)
	}
	lockMetadataWrite = original
	release, err := acquireCredentialsLock(dir)
	if err != nil {
		t.Fatalf("lock was not acquirable after a failed attempt: %v", err)
	}
	release()
}

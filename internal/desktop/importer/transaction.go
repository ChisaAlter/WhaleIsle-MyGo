package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// This control directory is outside every selectable payload destination.
// Recovery never discovers or executes journals carried by imported data.
const transactionDirectory = ".mygo-import"
const transactionJournal = "journal.json"

var ErrRecoveryBlocked = errors.New("import recovery is blocked; original and staged data were preserved")
var errImportConflict = errors.New("import destination already exists")

type directoryTransaction struct {
	Version int    `json:"version"`
	Target  string `json:"target"`
	Phase   string `json:"phase"`
	Before  string `json:"before"`
	After   string `json:"after,omitempty"`
}

// RecoverInterrupted reconciles only our reserved, journaled directory swap.
// Call before Host startup and import. An active writer in another process,
// unreadable journal, unsafe path or unexpected tree keeps admission closed.
func RecoverInterrupted(home string) (bool, error) {
	if err := directoryPath(home); err != nil {
		return false, err
	}
	if err := checkLegacyImportJournal(home); err != nil {
		return false, err
	}
	control := filepath.Join(home, transactionDirectory)
	_, controlErr := os.Lstat(control)
	_, lockErr := os.Lstat(filepath.Join(home, ".mygo-import.lock"))
	if os.IsNotExist(controlErr) && os.IsNotExist(lockErr) {
		return false, nil
	}
	if controlErr != nil && !os.IsNotExist(controlErr) {
		return false, controlErr
	}
	if lockErr != nil && !os.IsNotExist(lockErr) {
		return false, lockErr
	}
	release, err := acquireImportLock(home)
	if err != nil {
		return false, err
	}
	defer release()
	err = recoverDirectoryTransaction(home)
	return controlErr == nil && err == nil, err
}

func beginImport(home string) (func(), error) {
	if err := directoryPath(home); err != nil {
		return nil, err
	}
	if err := checkLegacyImportJournal(home); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return nil, err
	}
	release, err := acquireImportLock(home)
	if err != nil {
		return nil, err
	}
	if err = recoverDirectoryTransaction(home); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

// directoryPath rejects links, non-directory ancestors and inspection failures.
// No path from a journal may traverse a symlink/junction to another data tree.
func directoryPath(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for p := absolute; ; p = filepath.Dir(p) {
		info, e := os.Lstat(p)
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if e == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return fmt.Errorf("unsafe import directory: %s", p)
		}
		if p == filepath.Dir(p) {
			return nil
		}
	}
}

func transactionTarget(home, rel string) (string, error) {
	if rel == "" || strings.ContainsAny(rel, "\\:\x00") || filepath.IsAbs(rel) {
		return "", ErrRecoveryBlocked
	}
	parts := strings.Split(rel, "/")
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return "", ErrRecoveryBlocked
		}
	}
	valid := len(parts) == 1 && parts[0] == "attachments"
	valid = valid || len(parts) >= 2 && parts[0] == "sessions"
	valid = valid || len(parts) == 2 && (parts[0] == "skills" || parts[0] == ".agent-presets")
	if !valid {
		return "", ErrRecoveryBlocked
	}
	target := filepath.Join(home, filepath.FromSlash(rel))
	if err := directoryPath(target); err != nil {
		return "", err
	}
	return target, nil
}

type importReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r importReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// Fingerprints distinguish a recorded tree from a newer external edit. They
// include relative names, file sizes, bytes and empty directories, never links.
func treeFingerprint(ctx context.Context, root string) (string, error) {
	if err := directoryPath(root); err != nil {
		return "", err
	}
	if _, err := os.Lstat(root); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			_, err = fmt.Fprintf(h, "d:%s\x00", filepath.ToSlash(rel))
			return err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsupported import entry: %s", path)
		}
		if _, err = fmt.Fprintf(h, "f:%s\x00%d\x00", filepath.ToSlash(rel), info.Size()); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		opened, err := f.Stat()
		if err != nil || !os.SameFile(info, opened) {
			f.Close()
			return fmt.Errorf("import entry changed: %s", path)
		}
		_, err = io.Copy(h, importReader{ctx: ctx, reader: f})
		closeErr := f.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyTree only writes private staging. Source and staged links are refused.
// Attachments overlay the old tree; session/skill/preset overwrites replace it.
func copyTree(ctx context.Context, src, dst string, overwrite bool) error {
	if err := directoryPath(src); err != nil {
		return err
	}
	if err := directoryPath(dst); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if err := directoryPath(target); err != nil {
				return err
			}
			return os.MkdirAll(target, 0700)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsupported import entry: %s", path)
		}
		if err := privatePath(target); err != nil {
			return err
		}
		if current, err := os.Lstat(target); err == nil {
			if !current.Mode().IsRegular() {
				return fmt.Errorf("unsupported staged destination: %s", target)
			}
			if !overwrite {
				return nil
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		opened, err := in.Stat()
		if err != nil || !os.SameFile(info, opened) {
			in.Close()
			return fmt.Errorf("import entry changed: %s", path)
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			in.Close()
			return err
		}
		_, err = io.Copy(out, importReader{ctx: ctx, reader: in})
		if err == nil {
			err = out.Sync()
		}
		inErr, outErr := in.Close(), out.Close()
		if err != nil {
			return err
		}
		if inErr != nil {
			return inErr
		}
		return outErr
	})
}

func saveDirectoryTransaction(control string, journal directoryTransaction) error {
	path := filepath.Join(control, transactionJournal)
	before, err := readImportText(path)
	if err != nil {
		return err
	}
	data, err := json.Marshal(journal)
	if err != nil {
		return err
	}
	return writeMergedFile(path, before, data)
}

// publishDirectory is called only while the full import owns the process lock.
func publishDirectory(ctx context.Context, home, src, dst string, overwrite, merge bool) error {
	rel, err := filepath.Rel(home, dst)
	if err != nil {
		return err
	}
	dst, err = transactionTarget(home, filepath.ToSlash(rel))
	if err != nil {
		return err
	}
	if err = ValidateRoots(src, dst); err != nil {
		return err
	}
	before, err := treeFingerprint(ctx, dst)
	if err != nil {
		return err
	}
	if before != "" && !overwrite && !merge {
		return errImportConflict
	}
	control := filepath.Join(home, transactionDirectory)
	if err = os.Mkdir(control, 0700); err != nil {
		return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
	}
	journal := directoryTransaction{Version: 1, Target: filepath.ToSlash(rel), Phase: "copying", Before: before}
	if err = saveDirectoryTransaction(control, journal); err != nil {
		_ = os.Remove(control)
		return err
	}
	stage, backup := filepath.Join(control, "new"), filepath.Join(control, "old")
	abort := func(cause error) error {
		// Reconciliation uses an uncancelled context so cancellation can restore
		// the old tree. Any unresolved failure remains on disk and blocks startup.
		if recoveryErr := recoverDirectoryTransaction(home); recoveryErr != nil {
			return fmt.Errorf("%w: %v; %v", ErrRecoveryBlocked, cause, recoveryErr)
		}
		return cause
	}
	if merge && before != "" {
		if err = copyTree(ctx, dst, stage, true); err != nil {
			return abort(err)
		}
	}
	if err = copyTree(ctx, src, stage, overwrite); err != nil {
		return abort(err)
	}
	journal.After, err = treeFingerprint(ctx, stage)
	if err != nil {
		return abort(err)
	}
	current, err := treeFingerprint(ctx, dst)
	if err != nil {
		return abort(err)
	}
	if current != before {
		return abort(fmt.Errorf("destination changed; rescan before importing"))
	}
	if err = ctx.Err(); err != nil {
		return abort(err)
	}
	journal.Phase = "ready"
	if err = saveDirectoryTransaction(control, journal); err != nil {
		return abort(err)
	}
	if err = directoryPath(filepath.Dir(dst)); err != nil {
		return abort(err)
	}
	if err = os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return abort(err)
	}
	if before != "" {
		if err = os.Rename(dst, backup); err != nil {
			return abort(err)
		}
	}
	if err = os.Rename(stage, dst); err != nil {
		return abort(err)
	}
	return recoverDirectoryTransaction(home)
}

func validFingerprint(s string, optional bool) bool {
	if s == "" {
		return optional
	}
	decoded, err := hex.DecodeString(s)
	return err == nil && len(decoded) == sha256.Size
}

func recoverDirectoryTransaction(home string) error {
	control := filepath.Join(home, transactionDirectory)
	if err := directoryPath(control); err != nil {
		return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
	}
	entries, err := os.ReadDir(control)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
	}
	if len(entries) == 0 {
		return os.Remove(control)
	}
	// Atomic journal temporary files are ours, but unknown content is never
	// recursively removed. Payload files cannot become recovery instructions.
	var temporary []string
	for _, entry := range entries {
		switch entry.Name() {
		case transactionJournal, "new", "old":
		default:
			if strings.HasPrefix(entry.Name(), ".import-section-") && entry.Type().IsRegular() {
				temporary = append(temporary, filepath.Join(control, entry.Name()))
			} else {
				return fmt.Errorf("%w: unrecognized recovery artifact %s", ErrRecoveryBlocked, filepath.Join(control, entry.Name()))
			}
		}
	}
	raw, err := readRecoveryFile(filepath.Join(control, transactionJournal), 64<<10)
	if err != nil || len(raw) == 0 || len(raw) > 64<<10 {
		return fmt.Errorf("%w: cannot read %s", ErrRecoveryBlocked, control)
	}
	var journal directoryTransaction
	if err = json.Unmarshal(raw, &journal); err != nil || journal.Version != 1 || !validFingerprint(journal.Before, true) || (journal.Phase != "copying" && journal.Phase != "ready" && journal.Phase != "restored" && journal.Phase != "committed") || !validFingerprint(journal.After, journal.Phase == "copying" || journal.Phase == "restored") {
		return fmt.Errorf("%w: invalid journal at %s", ErrRecoveryBlocked, control)
	}
	dst, err := transactionTarget(home, journal.Target)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
	}
	stage, backup := filepath.Join(control, "new"), filepath.Join(control, "old")
	ctx := context.Background()
	live, err := treeFingerprint(ctx, dst)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
	}
	// Once cleanup was journaled, its trees may be partly removed by a
	// prior crash. Require the complete recorded live tree before continuing.
	if journal.Phase == "restored" || journal.Phase == "committed" {
		expected := journal.Before
		if journal.Phase == "committed" {
			expected = journal.After
		}
		if live != expected {
			return fmt.Errorf("%w: live data changed during cleanup", ErrRecoveryBlocked)
		}
		for _, path := range []string{stage, backup} {
			if _, e := treeFingerprint(ctx, path); e != nil {
				return fmt.Errorf("%w: %v", ErrRecoveryBlocked, e)
			}
		}
		return cleanupDirectoryTransaction(control, temporary)
	}
	old, err := treeFingerprint(ctx, backup)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
	}
	resolvedPhase := "restored"
	if journal.Phase == "copying" {
		if live != journal.Before || old != "" {
			return fmt.Errorf("%w: destination changed during staging", ErrRecoveryBlocked)
		}
		// A cancelled/failed private copy may be incomplete. Validate its paths
		// without requiring it to equal the source before removing that staging.
		if _, err = treeFingerprint(ctx, stage); err != nil {
			return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
		}
	} else {
		staged, e := treeFingerprint(ctx, stage)
		if e != nil {
			return fmt.Errorf("%w: %v", ErrRecoveryBlocked, e)
		}
		if staged != "" && staged != journal.After || old != "" && old != journal.Before {
			return fmt.Errorf("%w: recovery data changed", ErrRecoveryBlocked)
		}
		switch {
		case live == journal.After && staged == "":
			// Publication completed; old is a verified backup of the replaced tree.
			resolvedPhase = "committed"
		case live == journal.Before && old == "":
			// No live rename happened, or rollback already restored the old tree.
		case live == "" && old != "" && old == journal.Before:
			if err = os.Rename(backup, dst); err != nil {
				return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
			}
		default:
			return fmt.Errorf("%w: unresolved directory swap at %s", ErrRecoveryBlocked, control)
		}
	}
	journal.Phase = resolvedPhase
	if err = saveDirectoryTransaction(control, journal); err != nil {
		return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
	}
	return cleanupDirectoryTransaction(control, temporary)
}

func cleanupDirectoryTransaction(control string, temporary []string) error {
	var err error
	for _, path := range []string{filepath.Join(control, "new"), filepath.Join(control, "old")} {
		if err = os.RemoveAll(path); err != nil {
			return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
		}
	}
	for _, path := range temporary {
		if err = os.Remove(path); err != nil {
			return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
		}
	}
	if err = os.Remove(filepath.Join(control, transactionJournal)); err != nil {
		return fmt.Errorf("%w: %v", ErrRecoveryBlocked, err)
	}
	return os.Remove(control)
}

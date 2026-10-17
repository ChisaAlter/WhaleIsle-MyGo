package installctrl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var installSnapshotEntries = []string{
	"package.json",
	"pnpm-lock.yaml",
	"pnpm-workspace.yaml",
	"node_modules",
}

// InstallSnapshot is a pre-add copy of the web profile install surface.
type InstallSnapshot struct {
	Dir     string
	Entries []snapshotEntry
}

type snapshotEntry struct {
	Name    string
	File    string
	Existed bool
}

func installPathStat(file string) (os.FileInfo, error) {
	return os.Lstat(file)
}

// CaptureInstallSnapshot copies package.json / lock / workspace / node_modules
// into a temp dir under the profile (Electron marketplace-install parity).
func CaptureInstallSnapshot(profileDir string) (*InstallSnapshot, error) {
	profileDir = strings.TrimSpace(profileDir)
	if profileDir == "" {
		return nil, fmt.Errorf("profile 路径未配置")
	}
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return nil, err
	}
	entries := make([]snapshotEntry, 0, len(installSnapshotEntries))
	for _, name := range installSnapshotEntries {
		file := filepath.Join(profileDir, name)
		st, err := installPathStat(file)
		existed := err == nil
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("无法安全备份 %s：%w", name, err)
		}
		if existed {
			if isSymlink(st) {
				// Symlinks/junctions at the top level are backed up as links
				// via copyPath; reject unexpected types for files.
			} else if name == "node_modules" {
				if !st.IsDir() {
					return nil, fmt.Errorf("无法安全备份 %s：需要普通文件或目录", name)
				}
			} else if st.IsDir() {
				return nil, fmt.Errorf("无法安全备份 %s：需要普通文件或目录", name)
			}
		}
		entries = append(entries, snapshotEntry{Name: name, File: file, Existed: existed})
	}
	dir, err := os.MkdirTemp(profileDir, ".install-rollback-")
	if err != nil {
		return nil, err
	}
	snap := &InstallSnapshot{Dir: dir, Entries: entries}
	for _, entry := range entries {
		if !entry.Existed {
			continue
		}
		dst := filepath.Join(dir, entry.Name)
		if err := copyPath(entry.File, dst); err != nil {
			_ = os.RemoveAll(dir)
			return nil, fmt.Errorf("无法创建安装回滚点：%w", err)
		}
	}
	return snap, nil
}

// DiscardInstallSnapshot removes the rollback directory; cleanup errors append to result.Log.
func DiscardInstallSnapshot(snap *InstallSnapshot, result Result) Result {
	if snap == nil || snap.Dir == "" {
		return result
	}
	if err := os.RemoveAll(snap.Dir); err != nil {
		msg := fmt.Sprintf("安装快照清理失败，保留于 %s：%v", snap.Dir, err)
		if result.Log != "" {
			result.Log += "\n" + msg
		} else {
			result.Log = msg
		}
	}
	return result
}

// RollbackInstall restores the snapshot and returns an enriched failure Result.
func RollbackInstall(snap *InstallSnapshot, failure Result) Result {
	if snap == nil {
		failure.OK = false
		if failure.Error == "" {
			failure.Error = "安装失败"
		}
		return failure
	}
	var errors []string
	for _, entry := range snap.Entries {
		if st, err := installPathStat(entry.File); err == nil && st != nil {
			failed := filepath.Join(snap.Dir, "failed-"+entry.Name)
			_ = os.RemoveAll(failed)
			if err := os.Rename(entry.File, failed); err != nil {
				errors = append(errors, entry.Name+": "+err.Error())
				continue
			}
		} else if err != nil && !os.IsNotExist(err) {
			errors = append(errors, entry.Name+": "+err.Error())
			continue
		}
		if entry.Existed {
			src := filepath.Join(snap.Dir, entry.Name)
			if err := os.Rename(src, entry.File); err != nil {
				errors = append(errors, entry.Name+": "+err.Error())
			}
		}
	}
	failure.OK = false
	if len(errors) > 0 {
		rollbackError := strings.Join(errors, "；") + "；回滚备份保留于 " + snap.Dir
		failure.RolledBack = false
		failure.RollbackError = rollbackError
		failure.NeedsAllowBuilds = false
		failure.AllowBuilds = []string{}
		base := failure.Error
		if base == "" {
			base = "安装失败"
		}
		failure.Error = base + "；自动回滚失败：" + rollbackError
		return failure
	}
	failure.RolledBack = true
	base := failure.Error
	if base == "" {
		base = "安装失败"
	}
	failure.Error = base + "；已恢复安装前状态"
	return DiscardInstallSnapshot(snap, failure)
}

func isSymlink(st os.FileInfo) bool {
	return st.Mode()&os.ModeSymlink != 0
}

func copyPath(src, dst string) error {
	st, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if isSymlink(st) {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if runtime.GOOS == "windows" {
			// Prefer junction recreation for directory links (needs no elevation
			// when using the same Create API Electron uses via ensureDirectoryLink).
			if targetSt, err := os.Stat(src); err == nil && targetSt.IsDir() {
				return createWindowsJunction(dst, target)
			}
		}
		return os.Symlink(target, dst)
	}
	if st.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst, st.Mode())
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from := filepath.Join(src, entry.Name())
		to := filepath.Join(dst, entry.Name())
		if err := copyPath(from, to); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

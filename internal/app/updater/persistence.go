package updater

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"whaleisle/internal/desktop/update"
)

// NativeUpdateRecord is the persistent JSON journal for paused/staged native updates.
type NativeUpdateRecord struct {
	Version                              int
	Route, Phase, StagedTag, StagedTitle string
	AutoDownload                         bool
	RequiredRuntime                      bool
	ExplicitVersion, InstallRequested    bool
	Received, Total                      int64
	Active                               update.NativeRelease
	Staged                               update.ApplyResult
}

func nativeReleaseStagingDir(userDataDir string, r update.NativeRelease) string {
	return filepath.Join(userDataDir, "updates", "native-partials", url.PathEscape(r.PackageKey()))
}

// EnsureLoadedAsync triggers asynchronous loading of persisted update state
// and invalidates the target window once complete.
func (c *Center) EnsureLoadedAsync(win WindowRef) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.releases.CenterLoaded || c.releases.CenterLoading {
		c.mu.Unlock()
		return
	}
	c.releases.CenterLoading = true
	c.mu.Unlock()
	ctx := c.lifecycleContext()
	go func() {
		if ctx.Err() == nil {
			c.EnsureLoaded()
		}
		c.mu.Lock()
		c.releases.CenterLoading = false
		c.mu.Unlock()
		if ctx.Err() != nil || c.bridge == nil {
			return
		}
		target := win
		if target == nil {
			mainWin, _ := c.bridge.PrimaryWindows()
			target = mainWin
		}
		if target != nil && c.bridge.WindowAlive(target) {
			c.bridge.InvalidateWindow(target)
		}
	}()
}

// EnsureLoaded restores persisted update-center state from disk once per Center.
// Lock order: diskMu -> mu (never reversed; disk I/O runs with mu unlocked).
func (c *Center) EnsureLoaded() {
	if c == nil {
		return
	}
	c.diskMu.Lock()
	defer c.diskMu.Unlock()

	autoPref := true
	hasPref := false
	if c.bridge != nil {
		if v, ok := c.bridge.AutoDownloadPref(); ok {
			autoPref = v
			hasPref = true
		}
	}

	c.mu.Lock()
	if c.releases.CenterLoaded {
		c.mu.Unlock()
		return
	}
	c.releases.CenterLoaded = true
	c.releases.AutoDownload = autoPref
	c.mu.Unlock()

	userDataDir := c.userDataDir()
	buildVer := c.env().BuildVersion
	name := filepath.Join(userDataDir, "updates", "update-center.json")
	raw, err := os.ReadFile(name)
	if os.IsNotExist(err) {
		return
	}
	var record NativeUpdateRecord
	if err == nil {
		err = json.Unmarshal(raw, &record)
		if err == nil && record.Version != 1 {
			err = fmt.Errorf("unsupported update-center version")
		}
	}
	if err != nil {
		preserved := name + ".invalid-" + time.Now().UTC().Format("20060102T150405")
		if os.Rename(name, preserved) != nil {
			preserved = name
		}
		c.mu.Lock()
		c.releases.Err = lifecycleText("更新恢复状态无法读取，原文件保留于 ", "Could not read update recovery state; original retained at ") + preserved
		c.releases.Phase = "error"
		c.mu.Unlock()
		return
	}
	// Migrate older journal autoDownload preference only when desktop config has no explicit value.
	if c.bridge != nil && !hasPref {
		if err := c.bridge.InitAutoDownloadPref(record.AutoDownload); err != nil {
			c.mu.Lock()
			c.releases.Err = lifecycleText("无法迁移自动下载偏好：", "Could not migrate automatic download preference: ") + err.Error()
			c.mu.Unlock()
		}
	}
	var refreshedAutoPref bool
	var refreshedHasPref bool
	if c.bridge != nil {
		refreshedAutoPref, refreshedHasPref = c.bridge.AutoDownloadPref()
	}

	c.mu.Lock()
	r := &c.releases
	if refreshedHasPref {
		r.AutoDownload = refreshedAutoPref
	} else {
		r.AutoDownload = record.AutoDownload
	}
	retireCompleted := record.Active.Tag != "" && update.CompareVersions(record.Active.Version, buildVer) <= 0 && (!record.ExplicitVersion || record.InstallRequested)
	if retireCompleted {
		r.Status = lifecycleText("已完成的更新记录已退役。", "The completed update record was retired.")
		nextRecord := NativeUpdateRecord{Version: 1, AutoDownload: r.AutoDownload}
		c.mu.Unlock()
		if err := writeNativeUpdateRecord(userDataDir, nextRecord); err != nil {
			c.mu.Lock()
			c.releases.Err = lifecycleText("保存更新状态失败：", "Could not save update state: ") + err.Error()
			c.mu.Unlock()
		}
		return
	}
	if record.Active.Tag != "" && record.Active.Validate(runtime.GOOS, runtime.GOARCH, false) == nil {
		r.Active = record.Active
		r.Active.DownloadRoute = record.Route
		r.Route = record.Route
		r.Rows = []update.NativeRelease{r.Active}
		r.CatalogRoute = record.Route
		r.Initialized = true
		r.RequireRuntime = record.RequiredRuntime
		r.ExplicitVersion, r.InstallRequested = record.ExplicitVersion, record.InstallRequested
		r.Selected = r.Active.PackageKey()
		r.Staging = nativeReleaseStagingDir(userDataDir, r.Active)
		r.Received, r.Total = record.Received, record.Total
		if (record.Phase == "ready" || record.Phase == "installing") && record.Staged.Path != "" {
			base := filepath.Clean(r.Staging) + string(filepath.Separator)
			if rel, err := filepath.Rel(base, record.Staged.Path); err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				if st, err := os.Stat(record.Staged.Path); err == nil && st.Mode().IsRegular() && st.Size() == record.Active.Size && record.Staged.Size == record.Active.Size && strings.EqualFold(record.Staged.SHA256, record.Active.SHA256) {
					r.Staged = record.Staged
					r.StagedTag = record.Active.Tag
					r.StagedTitle = record.StagedTitle
					r.Phase = "ready"
					r.Status = lifecycleText("已恢复安装包记录，安装前将再次校验。", "Installer record restored; it will be verified again before installation.")
					c.mu.Unlock()
					return
				}
			}
		}
		r.Paused = true
		r.Phase = "paused"
		r.Status = lifecycleText("上次下载已暂停，继续将复用已下载内容。", "Previous download paused; resume reuses downloaded content.")
	}
	c.mu.Unlock()
}

// Persist writes the current native update state atomically to update-center.json.
// Lock order: diskMu -> mu.
func (c *Center) Persist() {
	if c == nil {
		return
	}
	c.diskMu.Lock()
	defer c.diskMu.Unlock()

	userDataDir := c.userDataDir()
	c.mu.Lock()
	r := c.releases
	route := r.Route
	if r.Active.DownloadRoute != "" {
		route = r.Active.DownloadRoute
	}
	record := NativeUpdateRecord{
		Version:          1,
		Route:            route,
		Phase:            r.Phase,
		AutoDownload:     r.AutoDownload,
		RequiredRuntime:  r.RequireRuntime,
		ExplicitVersion:  r.ExplicitVersion,
		InstallRequested: r.InstallRequested,
		Active:           r.Active,
		Received:         r.Received,
		Total:            r.Total,
		Staged:           r.Staged,
		StagedTag:        r.StagedTag,
		StagedTitle:      r.StagedTitle,
	}
	c.mu.Unlock()
	if err := writeNativeUpdateRecord(userDataDir, record); err != nil {
		c.mu.Lock()
		c.releases.Err = lifecycleText("保存更新状态失败：", "Could not save update state: ") + err.Error()
		c.mu.Unlock()
	}
}

// ReadRecord inspects the persisted update-center.json under diskMu for safe concurrent reading.
func (c *Center) ReadRecord() (NativeUpdateRecord, error) {
	if c == nil {
		return NativeUpdateRecord{}, os.ErrInvalid
	}
	c.diskMu.Lock()
	defer c.diskMu.Unlock()
	raw, err := os.ReadFile(filepath.Join(c.userDataDir(), "updates", "update-center.json"))
	if err != nil {
		return NativeUpdateRecord{}, err
	}
	var record NativeUpdateRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return NativeUpdateRecord{}, err
	}
	return record, nil
}

func writeNativeUpdateRecord(userDataDir string, record NativeUpdateRecord) error {
	base := filepath.Join(userDataDir, "updates")
	if err := os.MkdirAll(base, 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(base, "update-center-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(base, "update-center.json"))
}

// DiscardNativePartial removes a partial download directory only if strictly
// contained beneath <userDataDir>/updates/native-partials.
func DiscardNativePartial(userDataDir, dir string) error {
	base, err := filepath.Abs(filepath.Join(userDataDir, "updates", "native-partials"))
	if err != nil {
		return err
	}
	target, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to discard a path outside native update partials")
	}
	return os.RemoveAll(target)
}

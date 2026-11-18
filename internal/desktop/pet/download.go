package pet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type RenderAssetsState int

const (
	RenderAssetsMissing RenderAssetsState = iota
	RenderAssetsDownloading
	RenderAssetsReady
	RenderAssetsError
)

type RenderAssetsSnapshot struct {
	State           RenderAssetsState
	DownloadedBytes int64
	TotalBytes      int64
	Error           string
	ActiveRoot      string
}

type renderDownloadState struct {
	mu            sync.Mutex
	downloading   bool
	errText       string
	downloaded    atomic.Int64
	total         int64
	cancel        context.CancelFunc
	reloadPending bool

	cachedRoot  string
	cachedSig   string
	cachedReady bool
	cachedValid bool
}

type downloadFileSpec struct {
	Path      string
	Size      int64
	SHA256    string
	AltSize   int64
	AltSHA256 string
	URL       string
}

var (
	renderAssetDownloadURLRewrite func(string) string
	testDownloadSpecs             []downloadFileSpec
	testVerifyRoot                func(string) error
)

func activeDownloadSpecs() ([]downloadFileSpec, int64, string, []byte, error) {
	manifestRaw, err := assets.ReadFile("assets/render-manifest.json")
	if err != nil {
		return nil, 0, "", nil, err
	}
	var manifest struct {
		Commit string `json:"commit"`
		Files  []struct {
			Path   string `json:"path"`
			Size   int64  `json:"size"`
			SHA256 string `json:"sha256"`
			URL    string `json:"url"`
		} `json:"files"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return nil, 0, "", nil, err
	}
	if len(testDownloadSpecs) > 0 {
		var total int64
		for _, spec := range testDownloadSpecs {
			total += spec.Size
		}
		commit := manifest.Commit
		if commit == "" {
			commit = "test"
		}
		return append([]downloadFileSpec(nil), testDownloadSpecs...), total, commit, manifestRaw, nil
	}

	specs := make([]downloadFileSpec, 0, len(manifest.Files)+1)
	var total int64
	for _, f := range manifest.Files {
		clean := path.Clean(strings.TrimSpace(f.Path))
		if clean == "." || clean == "" || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "..") {
			return nil, 0, "", nil, fmt.Errorf("invalid asset path %q", f.Path)
		}
		if f.Size <= 0 || len(f.SHA256) != 64 || !strings.HasPrefix(f.URL, "https://raw.githubusercontent.com/") {
			return nil, 0, "", nil, fmt.Errorf("invalid manifest entry for %q", f.Path)
		}
		specs = append(specs, downloadFileSpec{
			Path:   clean,
			Size:   f.Size,
			SHA256: f.SHA256,
			URL:    f.URL,
		})
		total += f.Size
	}

	provRaw, err := assets.ReadFile("assets/anime4k-provenance.json")
	if err != nil {
		return nil, 0, "", nil, err
	}
	var prov struct {
		Repository string `json:"repository"`
		Commit     string `json:"commit"`
		Files      []struct {
			Path       string `json:"path"`
			SourcePath string `json:"sourcePath"`
			Size       int64  `json:"size"`
			SHA256     string `json:"sha256"`
			LFSize     int64  `json:"lfSize"`
			LFSHA256   string `json:"lfSha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(provRaw, &prov); err != nil {
		return nil, 0, "", nil, err
	}
	for _, f := range prov.Files {
		if f.Path != "anime4k.js" {
			continue
		}
		src := strings.TrimPrefix(strings.TrimSpace(f.SourcePath), "reference/electron/")
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", prov.Repository, prov.Commit, src)
		if f.Size <= 0 || len(f.SHA256) != 64 || prov.Repository == "" || prov.Commit == "" || src == "" {
			return nil, 0, "", nil, errors.New("invalid anime4k provenance entry")
		}
		dlSize, dlSHA := f.Size, f.SHA256
		var altSize int64
		var altSHA string
		if f.LFSize > 0 && len(f.LFSHA256) == 64 {
			dlSize, dlSHA = f.LFSize, f.LFSHA256
			altSize, altSHA = f.Size, f.SHA256
		}
		specs = append(specs, downloadFileSpec{
			Path:      "anime4k.js",
			Size:      dlSize,
			SHA256:    dlSHA,
			AltSize:   altSize,
			AltSHA256: altSHA,
			URL:       rawURL,
		})
		total += dlSize
		break
	}
	return specs, total, manifest.Commit, manifestRaw, nil
}

func defaultDownloadTotalBytes() int64 {
	_, total, _, _, err := activeDownloadSpecs()
	if err != nil {
		return 0
	}
	return total
}

func computeRenderRootSignature(root string) (string, bool) {
	if root == "" {
		return "", false
	}
	st, err := os.Lstat(root)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return "", false
	}
	if testVerifyRoot != nil {
		specs, _, _, _, err := activeDownloadSpecs()
		if err != nil {
			return "", false
		}
		var b strings.Builder
		b.WriteString(root)
		b.WriteByte('|')
		for _, spec := range specs {
			fst, ferr := os.Stat(filepath.Join(root, filepath.FromSlash(spec.Path)))
			if ferr != nil || !fst.Mode().IsRegular() || fst.Size() != spec.Size {
				return "", false
			}
			b.WriteString(spec.Path)
			b.WriteByte(':')
			b.WriteString(strconv.FormatInt(fst.Size(), 10))
			b.WriteByte(':')
			b.WriteString(strconv.FormatInt(fst.ModTime().UnixNano(), 10))
			b.WriteByte(';')
		}
		return b.String(), true
	}

	manifestRaw, err := assets.ReadFile("assets/render-manifest.json")
	if err != nil {
		return "", false
	}
	var manifest struct {
		Files []renderAsset `json:"files"`
	}
	if json.Unmarshal(manifestRaw, &manifest) != nil || len(manifest.Files) == 0 {
		return "", false
	}
	var b strings.Builder
	b.WriteString(root)
	b.WriteByte('|')
	for _, f := range manifest.Files {
		fst, ferr := os.Stat(filepath.Join(root, filepath.FromSlash(f.Path)))
		if ferr != nil || !fst.Mode().IsRegular() || fst.Size() != f.Size {
			return "", false
		}
		b.WriteString(f.Path)
		b.WriteByte(':')
		b.WriteString(strconv.FormatInt(fst.Size(), 10))
		b.WriteByte(':')
		b.WriteString(strconv.FormatInt(fst.ModTime().UnixNano(), 10))
		b.WriteByte(';')
	}
	return b.String(), true
}

func verifyRenderRootForDownload(root string) error {
	if testVerifyRoot != nil {
		return testVerifyRoot(root)
	}
	_, err := loadRenderAssets(root)
	return err
}

// RenderAssetsStatus returns an O(1) cached readiness and progress snapshot for
// the desktop pet render pack (THA4 + Anime4K) so UI frames never re-hash 38 MiB.
func (p *Companion) RenderAssetsStatus() RenderAssetsSnapshot {
	defaultTotal := defaultDownloadTotalBytes()
	if p == nil {
		return RenderAssetsSnapshot{State: RenderAssetsMissing, TotalBytes: defaultTotal}
	}
	activeRoot := p.resolveActiveRenderRoot()

	p.renderDownload.mu.Lock()
	downloading := p.renderDownload.downloading
	errText := p.renderDownload.errText
	total := p.renderDownload.total
	if total <= 0 {
		total = defaultTotal
	}
	downloaded := p.renderDownload.downloaded.Load()
	cachedRoot := p.renderDownload.cachedRoot
	cachedSig := p.renderDownload.cachedSig
	cachedReady := p.renderDownload.cachedReady
	cachedValid := p.renderDownload.cachedValid
	p.renderDownload.mu.Unlock()

	if downloading {
		return RenderAssetsSnapshot{
			State:           RenderAssetsDownloading,
			DownloadedBytes: downloaded,
			TotalBytes:      total,
			ActiveRoot:      p.userRenderRoot(),
		}
	}

	sig, completeStats := computeRenderRootSignature(activeRoot)
	if !completeStats {
		p.renderDownload.mu.Lock()
		p.renderDownload.cachedRoot = activeRoot
		p.renderDownload.cachedSig = ""
		p.renderDownload.cachedReady = false
		p.renderDownload.cachedValid = false
		p.renderDownload.mu.Unlock()
		if errText != "" {
			return RenderAssetsSnapshot{
				State:      RenderAssetsError,
				TotalBytes: total,
				Error:      errText,
				ActiveRoot: activeRoot,
			}
		}
		return RenderAssetsSnapshot{
			State:      RenderAssetsMissing,
			TotalBytes: total,
			ActiveRoot: activeRoot,
		}
	}

	ready := cachedReady
	if !cachedValid || cachedRoot != activeRoot || cachedSig != sig {
		ready = verifyRenderRootForDownload(activeRoot) == nil
		p.renderDownload.mu.Lock()
		p.renderDownload.cachedRoot = activeRoot
		p.renderDownload.cachedSig = sig
		p.renderDownload.cachedReady = ready
		p.renderDownload.cachedValid = true
		p.renderDownload.mu.Unlock()
	}

	if ready {
		return RenderAssetsSnapshot{
			State:           RenderAssetsReady,
			DownloadedBytes: total,
			TotalBytes:      total,
			ActiveRoot:      activeRoot,
		}
	}
	if errText != "" {
		return RenderAssetsSnapshot{
			State:      RenderAssetsError,
			TotalBytes: total,
			Error:      errText,
			ActiveRoot: activeRoot,
		}
	}
	return RenderAssetsSnapshot{
		State:      RenderAssetsMissing,
		TotalBytes: total,
		ActiveRoot: activeRoot,
	}
}

// PollDownloadedRenderAssets must be called on the UI thread; when an in-app
// download finishes, it clears stale fallback errors and reloads an open pet window.
func (p *Companion) PollDownloadedRenderAssets() bool {
	if p == nil {
		return false
	}
	p.renderDownload.mu.Lock()
	pending := p.renderDownload.reloadPending
	if pending {
		p.renderDownload.reloadPending = false
		p.renderDownload.cachedValid = false
	}
	p.renderDownload.mu.Unlock()
	if !pending {
		return false
	}
	p.Error = ""
	if p.Store != nil && p.Store.DesktopPet().Enabled && p.ready {
		p.close()
		p.sync()
	}
	return true
}

func (p *Companion) CancelDownloadRenderAssets() {
	if p == nil {
		return
	}
	p.renderDownload.mu.Lock()
	cancel := p.renderDownload.cancel
	p.renderDownload.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// StartDownloadRenderAssets begins an asynchronous, cancellable, SHA-256-verified
// download of the pinned THA4 + Anime4K assets into <UserDataDir>/pet-render-assets.
func (p *Companion) StartDownloadRenderAssets(wake func()) {
	if p == nil {
		return
	}
	notify := func() {
		if wake != nil {
			wake()
		}
	}

	destRoot := p.userRenderRoot()
	specs, totalBytes, commit, manifestRaw, specErr := activeDownloadSpecs()

	p.renderDownload.mu.Lock()
	if p.renderDownload.downloading {
		p.renderDownload.mu.Unlock()
		return
	}
	if specErr != nil || destRoot == "" {
		if specErr != nil {
			p.renderDownload.errText = specErr.Error()
		} else {
			p.renderDownload.errText = "render asset destination unavailable"
		}
		p.renderDownload.mu.Unlock()
		notify()
		return
	}
	if st, err := os.Lstat(destRoot); err == nil && (st.Mode()&os.ModeSymlink != 0 || !st.IsDir()) {
		p.renderDownload.errText = "render asset destination is not a regular directory"
		p.renderDownload.mu.Unlock()
		notify()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	p.renderDownload.downloading = true
	p.renderDownload.errText = ""
	p.renderDownload.total = totalBytes
	p.renderDownload.downloaded.Store(0)
	p.renderDownload.cancel = cancel
	p.renderDownload.reloadPending = false
	p.renderDownload.cachedValid = false
	p.renderDownload.mu.Unlock()
	notify()

	go func() {
		defer cancel()
		err := p.runDownloadRenderAssets(ctx, destRoot, specs, commit, manifestRaw, notify)
		p.renderDownload.mu.Lock()
		p.renderDownload.downloading = false
		p.renderDownload.cancel = nil
		p.renderDownload.cachedValid = false
		if err != nil {
			if errors.Is(err, context.Canceled) {
				p.renderDownload.errText = ""
			} else {
				p.renderDownload.errText = err.Error()
			}
		} else {
			p.renderDownload.errText = ""
			p.renderDownload.downloaded.Store(totalBytes)
			p.renderDownload.reloadPending = true
		}
		p.renderDownload.mu.Unlock()
		notify()
	}()
}

func (p *Companion) runDownloadRenderAssets(
	ctx context.Context,
	destRoot string,
	specs []downloadFileSpec,
	commit string,
	manifestRaw []byte,
	notify func(),
) error {
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return err
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       30 * time.Second,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}

	var completed int64
	lastNotify := time.Now()
	for _, spec := range specs {
		if err := ctx.Err(); err != nil {
			return err
		}
		dest := filepath.Join(destRoot, filepath.FromSlash(spec.Path))
		if st, err := os.Lstat(dest); err == nil {
			if st.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("destination file is a symlink: %s", spec.Path)
			}
			altOK := spec.AltSize > 0 && len(spec.AltSHA256) == 64 && st.Mode().IsRegular() && st.Size() == spec.AltSize && verifyFileSHA256(dest, spec.AltSize, spec.AltSHA256)
			if (st.Mode().IsRegular() && st.Size() == spec.Size && verifyFileSHA256(dest, spec.Size, spec.SHA256)) || altOK {
				completed += spec.Size
				p.renderDownload.downloaded.Store(completed)
				notify()
				lastNotify = time.Now()
				continue
			}
		}

		downloadURL := spec.URL
		if renderAssetDownloadURLRewrite != nil {
			downloadURL = renderAssetDownloadURLRewrite(downloadURL)
		} else if !strings.HasPrefix(downloadURL, "https://raw.githubusercontent.com/") {
			return fmt.Errorf("untrusted render asset URL for %s", spec.Path)
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := downloadVerifiedAsset(ctx, client, downloadURL, dest, spec, func(written int64) {
			p.renderDownload.downloaded.Store(completed + written)
			if time.Since(lastNotify) >= 80*time.Millisecond {
				lastNotify = time.Now()
				notify()
			}
		}); err != nil {
			return err
		}
		completed += spec.Size
		p.renderDownload.downloaded.Store(completed)
		lastNotify = time.Now()
		notify()
	}

	if err := verifyRenderRootForDownload(destRoot); err != nil {
		return err
	}
	if err := writeRegularNoSymlink(filepath.Join(destRoot, "provenance.json"), manifestRaw); err != nil {
		return err
	}
	if err := writeRegularNoSymlink(filepath.Join(destRoot, ".pet-render-staged"), []byte(commit+"\n")); err != nil {
		return err
	}
	return nil
}

func verifyFileSHA256(path string, wantSize int64, wantSHA string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() != wantSize {
		return false
	}
	h := sha256.New()
	written, err := io.Copy(h, io.LimitReader(f, wantSize+1))
	if err != nil || written != wantSize {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == wantSHA
}

func downloadVerifiedAsset(
	ctx context.Context,
	client *http.Client,
	rawURL, dest string,
	spec downloadFileSpec,
	onProgress func(written int64),
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected HTTP %d", spec.Path, resp.StatusCode)
	}

	partial := dest + ".partial"
	_ = os.Remove(partial)
	out, err := os.OpenFile(partial, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	cleanupPartial := true
	defer func() {
		if out != nil {
			_ = out.Close()
		}
		if cleanupPartial {
			_ = os.Remove(partial)
		}
	}()

	h := sha256.New()
	reader := io.LimitReader(resp.Body, spec.Size+1)
	buf := make([]byte, 32*1024)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		nr, rerr := reader.Read(buf)
		if nr > 0 {
			written += int64(nr)
			if written > spec.Size {
				return fmt.Errorf("%s exceeded pinned size %d", spec.Path, spec.Size)
			}
			if _, werr := out.Write(buf[:nr]); werr != nil {
				return werr
			}
			_, _ = h.Write(buf[:nr])
			if onProgress != nil {
				onProgress(written)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	if err := out.Close(); err != nil {
		out = nil
		return err
	}
	out = nil

	if written != spec.Size {
		return fmt.Errorf("%s size mismatch: got %d, want %d", spec.Path, written, spec.Size)
	}
	if sum := hex.EncodeToString(h.Sum(nil)); sum != spec.SHA256 {
		return fmt.Errorf("%s SHA-256 mismatch", spec.Path)
	}
	if st, err := os.Lstat(dest); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("destination file is a symlink: %s", spec.Path)
	}
	if err := os.Rename(partial, dest); err != nil {
		return err
	}
	cleanupPartial = false
	return nil
}

func writeRegularNoSymlink(dest string, data []byte) error {
	if st, err := os.Lstat(dest); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("destination file is a symlink: %s", filepath.Base(dest))
	}
	tmp := dest + ".tmp"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

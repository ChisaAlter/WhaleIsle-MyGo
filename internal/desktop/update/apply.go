package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ApplyResult is the outcome of downloading (and optionally launching) an update.
type ApplyResult struct {
	Path      string
	Message   string
	Started   bool
	SHA256    string
	Size      int64
	Signature SignatureStatus
	Publisher string
	// UsedRange is true when at least one block was fetched via HTTP Range.
	UsedRange     bool
	CopiedBlocks  int
	FetchedBlocks int
}

// StageOptions configures DownloadAndStage verification.
type StageOptions struct {
	ReleaseURL string
	DestDir    string
	Start      bool
	HTTPClient *http.Client
	// ExpectSHA256 is optional hex digest; when set, download must match.
	ExpectSHA256 string
	// ExpectSize is optional Content-Length / feed size; when >0 must match bytes written.
	ExpectSize int64
	// RequireSignature enables Authenticode verification after download (Windows).
	// Fail closed when the probe reports a non-valid signature (or cannot run).
	// On non-Windows, Authenticode is skipped (documented no-op success).
	RequireSignature bool
	// ExpectedPublisher, when non-empty, requires the signer subject to contain
	// this substring (case-insensitive). Implies a signature probe on Windows.
	ExpectedPublisher string
}

// IncrementalOptions configures ApplyIncremental (blockmap-assisted staging).
type IncrementalOptions struct {
	CurrentVersion string
	Feed           Feed
	DestDir        string
	Start          bool
	HTTPClient     *http.Client
	// OldPath is a local previous installer. When it exists as a regular file,
	// ApplyIncremental tries ApplyBlockPatch (Range for changed blocks). Missing
	// or empty OldPath uses the full-file download path.
	OldPath string
	// RequireSignature / ExpectedPublisher are passed through to staging.
	RequireSignature  bool
	ExpectedPublisher string
}

// DownloadAndStage fetches ReleaseURL into destDir (or os.TempDir) and returns
// the local path. It does not replace the running binary; on Windows it can
// Start the installer when start is true.
func DownloadAndStage(ctx context.Context, releaseURL, destDir string, start bool, client *http.Client) (ApplyResult, error) {
	return DownloadAndStageOpts(ctx, StageOptions{
		ReleaseURL: releaseURL,
		DestDir:    destDir,
		Start:      start,
		HTTPClient: client,
	})
}

// DownloadAndStageOpts is DownloadAndStage with hash/size verification.
func DownloadAndStageOpts(ctx context.Context, opt StageOptions) (ApplyResult, error) {
	if ctx.Err() != nil {
		return ApplyResult{}, ctx.Err()
	}
	releaseURL := strings.TrimSpace(opt.ReleaseURL)
	if releaseURL == "" {
		return ApplyResult{}, fmt.Errorf("empty release URL")
	}
	client := opt.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	destDir := opt.DestDir
	if destDir == "" {
		destDir = filepath.Join(os.TempDir(), "whaleisle-updates")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return ApplyResult{}, err
	}
	name := filepath.Base(strings.Split(releaseURL, "?")[0])
	if name == "" || name == "." || name == "/" {
		name = "WhaleIsle-update.exe"
	}
	dest := filepath.Join(destDir, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
	if err != nil {
		return ApplyResult{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return ApplyResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ApplyResult{}, fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return ApplyResult{}, err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, 512<<20))
	if err != nil {
		_ = f.Close()
		_ = os.Remove(dest)
		return ApplyResult{}, err
	}
	if err := f.Close(); err != nil {
		return ApplyResult{}, err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if opt.ExpectSize > 0 && n != opt.ExpectSize {
		_ = os.Remove(dest)
		return ApplyResult{}, fmt.Errorf("size mismatch: got %d want %d", n, opt.ExpectSize)
	}
	expect := strings.ToLower(strings.TrimSpace(opt.ExpectSHA256))
	if expect != "" && expect != sum {
		_ = os.Remove(dest)
		return ApplyResult{}, fmt.Errorf("sha256 mismatch")
	}

	sig, err := CheckSignature(ctx, dest, VerifyOptions{
		RequireSignature:  opt.RequireSignature,
		ExpectedPublisher: opt.ExpectedPublisher,
	})
	if err != nil {
		_ = os.Remove(dest)
		return ApplyResult{}, err
	}

	out := ApplyResult{
		Path:      dest,
		Message:   "staged " + dest,
		SHA256:    sum,
		Size:      n,
		Signature: sig.Status,
		Publisher: sig.Publisher,
	}
	if expect != "" {
		out.Message += "; sha256 ok"
	}
	if sig.Status == SignatureValid {
		out.Message += "; authenticode ok"
	}
	if opt.Start && runtime.GOOS == "windows" && strings.HasSuffix(strings.ToLower(dest), ".exe") {
		if err := exec.Command(dest).Start(); err != nil {
			out.Message += "; start failed: " + err.Error()
			return out, err
		}
		out.Started = true
		out.Message += "; installer started"
	}
	return out, nil
}

// ApplyIncremental stages an update when CanIncremental reports eligibility.
//
//  1. Require CanIncremental(current, feed); otherwise return an error directing
//     callers to DownloadAndStage / DownloadAndStageOpts.
//  2. Resolve a BlockMapInfo (DownloadBlockMap from Feed.BlockMapURL, or inline
//     Feed.BlockMap). If neither yields a valid map, return an error suggesting
//     the full-file path.
//  3. When OldPath is a readable regular file, ApplyBlockPatch copies matching
//     blocks from that file and Range-fetches the rest (full GET if Range is
//     unsupported). Otherwise download the full Feed.URL, then VerifyBlockMap.
//     Verification failure is fail-closed (staged file removed).
func ApplyIncremental(ctx context.Context, opt IncrementalOptions) (ApplyResult, error) {
	if ctx.Err() != nil {
		return ApplyResult{}, ctx.Err()
	}
	if !CanIncremental(opt.CurrentVersion, opt.Feed) {
		return ApplyResult{}, fmt.Errorf("incremental apply unsupported: use DownloadAndStage / DownloadAndStageOpts")
	}
	client := opt.HTTPClient
	info, err := resolveBlockMap(ctx, opt.Feed, client)
	if err != nil {
		return ApplyResult{}, err
	}
	releaseURL := strings.TrimSpace(opt.Feed.URL)
	if releaseURL == "" {
		return ApplyResult{}, fmt.Errorf("empty release URL")
	}
	if old := strings.TrimSpace(opt.OldPath); old != "" {
		if st, err := os.Stat(old); err == nil && st.Mode().IsRegular() {
			return ApplyBlockPatchOpts(ctx, PatchOptions{
				OldPath:           old,
				Feed:              opt.Feed,
				BlockMap:          info,
				DestDir:           opt.DestDir,
				Start:             opt.Start,
				HTTPClient:        client,
				RequireSignature:  opt.RequireSignature,
				ExpectedPublisher: opt.ExpectedPublisher,
			})
		}
	}
	res, err := DownloadAndStageOpts(ctx, StageOptions{
		ReleaseURL:        releaseURL,
		DestDir:           opt.DestDir,
		Start:             opt.Start,
		HTTPClient:        client,
		ExpectSHA256:      opt.Feed.SHA256,
		ExpectSize:        opt.Feed.Size,
		RequireSignature:  opt.RequireSignature,
		ExpectedPublisher: opt.ExpectedPublisher,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	if err := VerifyBlockMap(res.Path, info); err != nil {
		_ = os.Remove(res.Path)
		return ApplyResult{}, fmt.Errorf("blockmap verify failed: %w", err)
	}
	res.Message += "; blockmap verified (full download)"
	return res, nil
}

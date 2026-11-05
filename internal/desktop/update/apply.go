package update

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

// StagePhase names work actually entered by the staging operation. Byte
// progress is independent: receiving all bytes does not imply verification.
type StagePhase string

const (
	StageDownloading StagePhase = "downloading"
	StageVerifying   StagePhase = "verifying"
	StageReady       StagePhase = "ready"
)

func reportStagePhase(ctx context.Context, notify func(StagePhase), phase StagePhase) {
	if notify != nil && ctx.Err() == nil {
		notify(phase)
	}
}

// StageOptions configures DownloadAndStage verification.
type StageOptions struct {
	// Resume retains an identity-bound partial file on interruption. Existing
	// callers remain one-shot; native update-center explicitly opts in.
	Resume     bool
	ReleaseURL string
	DestDir    string
	Start      bool
	HTTPClient *http.Client
	// ExpectSHA256 is optional hex digest; when set, download must match.
	ExpectSHA256 string
	// ExpectSize is optional Content-Length / feed size; when >0 must match bytes written.
	ExpectSize int64
	// MaxBytes bounds downloads; zero keeps the existing 512 MiB limit.
	MaxBytes int64
	// OnProgress is called during download with received and expected bytes.
	OnProgress func(received, total int64)
	// OnPhase reports download and verification work on the staging goroutine.
	// NativeStageOptions reports ready only after its complete pipeline succeeds.
	OnPhase func(StagePhase)
	// RequireSignature enables platform signature verification after download.
	// Fail closed when the probe reports a non-valid signature (or cannot run).
	// Windows uses Authenticode; Darwin uses codesign; other platforms skip the probe.
	RequireSignature bool
	// ExpectedPublisher, when non-empty, requires the signer subject to contain
	// this substring (case-insensitive). Implies a signature probe on Windows or Darwin.
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
	// OnProgress is called during the download (full or patch) with received
	// and expected bytes, so an in-app download row can report progress.
	OnProgress func(received, total int64)
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
	maxBytes := opt.MaxBytes
	if maxBytes == 0 {
		maxBytes = 512 << 20
	}
	if maxBytes < 1 || maxBytes > 8<<30 || opt.ExpectSize > maxBytes {
		return ApplyResult{}, fmt.Errorf("installer exceeds download size limit")
	}
	client := opt.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute, Transport: updateTransport()}
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
	work := dest
	if opt.Resume {
		work += ".part"
	}
	reportStagePhase(ctx, opt.OnPhase, StageDownloading)
	n, sum, err := receiveUpdatePayload(ctx, opt, client, work, maxBytes)
	if err != nil {
		return ApplyResult{}, err
	}
	reportStagePhase(ctx, opt.OnPhase, StageVerifying)
	if opt.ExpectSize > 0 && n != opt.ExpectSize {
		_ = os.Remove(work)
		return ApplyResult{}, fmt.Errorf("size mismatch: got %d want %d", n, opt.ExpectSize)
	}
	expect := strings.ToLower(strings.TrimSpace(opt.ExpectSHA256))
	if expect != "" && expect != sum {
		_ = os.Remove(work)
		return ApplyResult{}, fmt.Errorf("sha256 mismatch")
	}

	sig, err := CheckSignature(ctx, work, VerifyOptions{
		RequireSignature:  opt.RequireSignature,
		ExpectedPublisher: opt.ExpectedPublisher,
	})
	if err != nil {
		if !opt.Resume || ctx.Err() == nil {
			_ = os.Remove(work)
		}
		return ApplyResult{}, err
	}

	if err := ctx.Err(); err != nil {
		if !opt.Resume {
			_ = os.Remove(work)
		}
		return ApplyResult{}, err
	}
	if opt.OnProgress != nil {
		opt.OnProgress(n, opt.ExpectSize)
	}
	if err := ctx.Err(); err != nil {
		if !opt.Resume {
			_ = os.Remove(work)
		}
		return ApplyResult{}, err
	}
	if opt.Resume {
		if err := os.Rename(work, dest); err != nil {
			return ApplyResult{}, err
		}
		_ = os.Remove(work + ".json")
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
		label := "signature"
		if runtime.GOOS == "windows" {
			label = "authenticode"
		} else if runtime.GOOS == "darwin" {
			label = "codesign"
		}
		out.Message += "; " + label + " ok"
	}
	if opt.Start && CanStartStaged(out) {
		return StartStaged(out)
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
				OnProgress:        opt.OnProgress,
			})
		}
	}
	res, err := DownloadAndStageOpts(ctx, StageOptions{
		ReleaseURL:        releaseURL,
		DestDir:           opt.DestDir,
		Start:             false,
		HTTPClient:        client,
		ExpectSHA256:      opt.Feed.SHA256,
		ExpectSize:        opt.Feed.Size,
		RequireSignature:  opt.RequireSignature,
		ExpectedPublisher: opt.ExpectedPublisher,
		OnProgress:        opt.OnProgress,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	if err := VerifyBlockMap(res.Path, info); err != nil {
		_ = os.Remove(res.Path)
		return ApplyResult{}, fmt.Errorf("blockmap verify failed: %w", err)
	}
	res.Message += "; blockmap verified (full download)"
	if opt.Start && CanStartStaged(res) {
		return StartStaged(res)
	}
	return res, nil
}

type stageProgressWriter struct {
	notify   func(int64, int64)
	total    int64
	received int64
	last     time.Time
}

func (p *stageProgressWriter) Write(b []byte) (int, error) {
	p.received += int64(len(b))
	if p.notify != nil && (p.last.IsZero() || time.Since(p.last) >= 100*time.Millisecond || p.received == p.total) {
		p.last = time.Now()
		p.notify(p.received, p.total)
	}
	return len(b), nil
}

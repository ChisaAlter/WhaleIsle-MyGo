package market

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// InstallSpaceReserve is the free space kept back after an install, matching
// reference/electron/src/main/install-space.js:6 (256 MiB).
const InstallSpaceReserve int64 = 256 << 20

// ErrSpaceUnsupported reports a volume whose free space cannot be queried. The
// native preflight reports this to the caller instead of treating it as zero.
var ErrSpaceUnsupported = errors.New("disk space cannot be checked on this volume")

// FormatGiB mirrors the native formatBytes: two decimals and a "GiB" suffix.
func FormatGiB(bytes int64) string {
	return fmt.Sprintf("%.2f GiB", float64(bytes)/(1024*1024*1024))
}

// freeBytesAt is the platform query; tests replace it through freeBytes.
func freeBytesAt(dir string) (int64, error) { return platformFreeBytes(dir) }

// freeBytes is a seam so the preflight logic is testable without real volumes.
var freeBytes = freeBytesAt

// CheckInstallSpace is the native install-space.js checkInstallSpace preflight:
// it estimates the destination volume (walking up from a directory that does not
// exist yet), requires size plus the 256 MiB reserve, and reports an unsupported
// filesystem instead of pretending the volume is full or empty. It is only a
// preflight: writers must still handle ENOSPC.
func CheckInstallSpace(dir string, size int64) (available int64, err error) {
	if size < 0 {
		return 0, errors.New("invalid installation size")
	}
	probe := dir
	if abs, absErr := filepath.Abs(dir); absErr == nil {
		probe = abs
	}
	for {
		avail, statErr := freeBytes(probe)
		switch {
		case statErr == nil:
			required := size + InstallSpaceReserve
			if avail < required {
				return avail, fmt.Errorf(
					"磁盘空间不足：%s 所在磁盘可用 %s，本步骤预计需要 %s（含预留空间）。请释放该磁盘空间后重试。",
					dir, FormatGiB(avail), FormatGiB(required))
			}
			return avail, nil
		case errors.Is(statErr, os.ErrNotExist):
			parent := filepath.Dir(probe)
			if parent == probe {
				return 0, ErrSpaceUnsupported
			}
			probe = parent
		default:
			return 0, ErrSpaceUnsupported
		}
	}
}

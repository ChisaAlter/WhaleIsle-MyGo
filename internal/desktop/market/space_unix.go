//go:build !windows

package market

import "syscall"

// platformFreeBytes reports the free bytes available on dir's volume
// (statfs.bavail * bsize, as the native preflight does).
func platformFreeBytes(dir string) (int64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return int64(st.Bavail) * int64(st.Bsize), nil
}

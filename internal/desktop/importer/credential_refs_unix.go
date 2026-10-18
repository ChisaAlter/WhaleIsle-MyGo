//go:build !windows

package importer

import (
	"os"
	"syscall"
)

func credentialFileHasSingleLink(file *os.File, info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink == 1
}

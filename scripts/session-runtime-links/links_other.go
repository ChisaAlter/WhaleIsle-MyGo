//go:build !windows

package main

import (
	"os"
	"path/filepath"
)

func resolveDirectoryLink(link string) (string, error) { return filepath.EvalSymlinks(link) }
func runtimeReparseTag(filename string) (uint32, bool, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return 0, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 0xA0000003, true, nil
	}
	return 0, false, nil
}

func createDirectoryLink(target, link string) error { return os.Symlink(target, link) }
func removeDirectoryLink(link string) error         { return os.Remove(link) }

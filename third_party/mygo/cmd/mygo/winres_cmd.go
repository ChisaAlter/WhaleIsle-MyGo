package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// runWinres writes the app's Windows resources (icon, manifest, version) as a
// .syso file for a plain "go build", which links .syso files of the main
// package into the executable. Packaging scripts that drive the Go toolchain
// themselves can produce the same resources "mygo build" embeds.
func runWinres(args []string) error {
	flags := newFlags("winres", "[dir]", "Write the app's Windows resources (.syso) into a main package for go build.")
	pkg := flags.String("pkg", "", "main package directory that receives the .syso (default: dir)")
	arch := flags.String("arch", runtime.GOARCH, "target architecture: amd64, arm64 or 386")
	version := flags.String("version", "", "override the version from mygo.json")
	out := flags.String("out", "", "write the .syso here instead of <pkg>/mygo_windows_<arch>.syso")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 1 {
		return fmt.Errorf("winres accepts one app directory")
	}
	dir := "."
	if flags.NArg() == 1 {
		dir = flags.Arg(0)
	}
	c, err := loadConfig(dir)
	if err != nil {
		return err
	}
	if *version != "" {
		c.Version = *version
	}
	syso, err := winresSyso(c, *arch)
	if err != nil {
		return err
	}
	target := *out
	if target == "" {
		p := *pkg
		if p == "" {
			p = dir
		}
		target = filepath.Join(p, "mygo_windows_"+*arch+".syso")
	}
	if err := os.WriteFile(target, syso, 0o644); err != nil {
		return err
	}
	fmt.Println("Wrote", target)
	return nil
}

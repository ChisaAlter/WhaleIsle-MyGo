// Command gen-api turns dsh typert remote descriptor files
// (lib/typert.remote-client.js) into Go descriptor tables and best-effort
// wire types under internal/dsh/api. It only uses the standard library.
//
// Usage:
//
//	go run ./tools/gen-api -in <pkg-root> [-in <pkg-root> ...] [-out internal/dsh/api] [-pkg api]
//
// Each -in root must contain lib/typert.remote-client.js (repeatable or
// comma-separated). Outputs: one dsh_<pkg>_gen.go per input package plus a
// registry_gen.go aggregating all endpoints.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const genVersion = "0.1.0"

// multiFlag accepts repeated flags and comma-separated values.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			*m = append(*m, part)
		}
	}
	return nil
}

func main() {
	var in multiFlag
	out := flag.String("out", filepath.Join("internal", "dsh", "api"), "output directory for generated Go files")
	pkgName := flag.String("pkg", "api", "package name of generated files")
	flag.Var(&in, "in", "package root containing lib/typert.remote-client.js (repeatable, comma-separated)")
	flag.Parse()
	if err := run(in, *out, *pkgName); err != nil {
		fmt.Fprintln(os.Stderr, "gen-api:", err)
		os.Exit(1)
	}
}

func run(roots []string, outDir, pkgName string) error {
	if len(roots) == 0 {
		return errors.New("no input packages; pass -in <pkg-root> at least once")
	}
	var pkgs []*genPackage
	seen := map[string]string{} // goName -> source package
	for _, root := range roots {
		srcPath := filepath.Join(root, "lib", "typert.remote-client.js")
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", srcPath, err)
		}
		pf, err := parseTypertFile(string(data))
		if err != nil {
			return fmt.Errorf("parse %s: %w", srcPath, err)
		}
		if pf.file.Package == "" {
			return fmt.Errorf("%s: TYPERT_REMOTE.package is missing", srcPath)
		}
		goName, prefix := goPackageNames(pf.file.Package, filepath.Base(root))
		if prev, dup := seen[goName]; dup {
			if prev == pf.file.Package {
				continue // same package built twice; keep the first copy
			}
			return fmt.Errorf("input packages %q and %q both map to Go name %q; pass only one build of each package", prev, pf.file.Package, goName)
		}
		seen[goName] = pf.file.Package
		short := pf.file.Package
		if i := strings.LastIndex(short, "/"); i >= 0 {
			short = short[i+1:]
		}
		pkgs = append(pkgs, &genPackage{
			root:      root,
			srcPath:   srcPath,
			parsed:    pf,
			tf:        pf.file,
			goName:    goName,
			prefix:    prefix,
			shortName: short,
		})
	}

	g := newGoTypeer()
	for _, p := range pkgs {
		rendered, err := renderPackageFile(p, g, pkgName)
		if err != nil {
			return fmt.Errorf("render %s: %w", p.tf.Package, err)
		}
		p.rendered = rendered
	}
	registry, err := renderRegistry(pkgs, pkgName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", outDir, err)
	}
	for _, p := range pkgs {
		path := filepath.Join(outDir, p.goName+"_gen.go")
		if err := os.WriteFile(path, p.rendered, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Printf("wrote %s (%d endpoints)\n", path, len(p.tf.Descriptors))
	}
	regPath := filepath.Join(outDir, "registry_gen.go")
	if err := os.WriteFile(regPath, registry, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", regPath, err)
	}
	total := 0
	for _, p := range pkgs {
		total += len(p.tf.Descriptors)
	}
	fmt.Printf("wrote %s (%d endpoints)\n", regPath, total)
	return nil
}

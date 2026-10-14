// Command gen-events turns dsh docs/persistence-schema.json into Go session
// event wire types under internal/dsh/events. It only uses the standard library.
//
// Usage:
//
//	go run ./tools/gen-events -in <persistence-schema.json> [-out internal/dsh/events] [-pkg events]
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const genVersion = "0.1.0"

func main() {
	in := flag.String("in", filepath.Join("runtime", "vendor", "deepseek-harness", "docs", "persistence-schema.json"),
		"path to persistence-schema.json")
	out := flag.String("out", filepath.Join("internal", "dsh", "events"), "output directory for generated Go files")
	pkgName := flag.String("pkg", "events", "package name of generated files")
	flag.Parse()
	if err := run(*in, *out, *pkgName); err != nil {
		fmt.Fprintln(os.Stderr, "gen-events:", err)
		os.Exit(1)
	}
}

func run(inPath, outDir, pkgName string) error {
	if inPath == "" {
		return errors.New("no input; pass -in <persistence-schema.json>")
	}
	data, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inPath, err)
	}
	inv, err := parseInventory(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", inPath, err)
	}
	rendered, err := renderEventsPackage(inv, inPath, pkgName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", outDir, err)
	}
	outPath := filepath.Join(outDir, "events_gen.go")
	if err := os.WriteFile(outPath, rendered, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	nEvent := 0
	for _, r := range inv.Roots {
		if r.Kind == "event" {
			nEvent++
		}
	}
	fmt.Printf("wrote %s (%d roots, %d events)\n", outPath, len(inv.Roots), nEvent)
	return nil
}

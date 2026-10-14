// stage-terminal copies a locally supplied pinned library into build output.
// It intentionally has no fetch or library-loading path.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"whaleisle/internal/desktop/terminalruntime"
)

func main() {
	source := flag.String("source", os.Getenv("WHALEISLE_TERMINAL_LIBRARY"), "local pinned native library (no downloads)")
	out := flag.String("out", "", "directory containing whaleisle.exe")
	target := flag.String("target", runtime.GOOS+"-"+runtime.GOARCH, "packaged GOOS-GOARCH")
	root := flag.String("root", ".", "project source root")
	flag.Parse()
	if *out == "" {
		fail(fmt.Errorf("-out is required"))
	}
	spec, err := terminalruntime.Specification(*target)
	if err != nil {
		fail(err)
	}
	binding, err := os.ReadFile(filepath.Join(*root, "third_party", "mygo", "plugins", "terminal", "mygo-plugin.json"))
	if err != nil {
		fail(err)
	}
	if err := terminalruntime.CheckBinding(binding, *target); err != nil {
		fail(err)
	}
	path, err := terminalruntime.Resolve(spec, *source, []string{filepath.Join(*root, ".toolchain", "natives", *target), filepath.Join(*root, "resources", *target), terminalruntime.CacheDirectory(spec)})
	if err != nil {
		fail(fmt.Errorf("%w\nSupply WHALEISLE_TERMINAL_LIBRARY from %s (SHA256 %s)", err, spec.URL, spec.SHA256))
	}
	if err := terminalruntime.Stage(path, *out, spec); err != nil {
		fail(err)
	}
	fmt.Printf("Staged verified %s for %s, license and provenance; no code executed.\n", spec.Name, *target)
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

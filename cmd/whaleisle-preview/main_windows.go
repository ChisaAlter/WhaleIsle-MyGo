//go:build windows && (amd64 || arm64)

// whaleisle-preview is a dedicated helper, never a general file/URL opener.
// Build separately; the application does not launch it by default.
package main

import (
	"fmt"
	"os"

	"whaleisle/internal/desktop/previewprocess"
)

func main() {
	if err := previewprocess.RunHelper(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

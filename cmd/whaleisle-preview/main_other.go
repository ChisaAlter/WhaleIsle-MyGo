//go:build (!windows && !linux) || (!amd64 && !arm64)

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "interactive AppContainer preview requires Windows amd64 or arm64")
	os.Exit(1)
}

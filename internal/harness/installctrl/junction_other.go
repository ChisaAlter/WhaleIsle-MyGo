//go:build !windows

package installctrl

import "os"

func createWindowsJunction(link, target string) error {
	return os.Symlink(target, link)
}

//go:build !windows

package credentials

// Electron does not participate in .credentials.lock. Its POSIX singleton is
// a socket/cookie protocol, not a file lock we can safely claim for a save.
func sharedProfileWriteSupport() error { return ErrSharedProfileWriteUnavailable }

func acquireProfileWrite(string) (func(), error) {
	return nil, sharedProfileWriteSupport()
}

// Private stores have no Electron owner; the ordinary credentials lock and
// baseline check retain the existing contract for those callers.
func acquirePrivateProfileWrite(string) (func(), error) { return func() {}, nil }

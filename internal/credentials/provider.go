package credentials

// The OS face is injectable so storage fixtures never touch a real Keychain.
// supported describes compiled support, not whether a user's key is unlocked.
type credentialProvider struct {
	version   string
	backend   string
	supported bool
	key       func(dir string, create bool) ([]byte, error)
}

var credentialOS = platformCredentialProvider()

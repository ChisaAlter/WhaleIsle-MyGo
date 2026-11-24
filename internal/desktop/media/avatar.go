package media

import (
	"net/url"
	"strings"
)

// avatarPixels covers a 32-DIP avatar at two device pixels per DIP; every
// account avatar view shares this one decode.
const avatarPixels = 64

// avatarKey maps an account avatar value to its request: a public http(s)
// URL without credentials, or a data:image URI. Anything else (including a
// local filesystem path) is refused: profile fields never name local files.
func avatarKey(raw string) (Key, Source, bool) {
	raw = strings.TrimSpace(raw)
	var key Key
	var src Source
	switch {
	case strings.HasPrefix(raw, "data:image/"):
		key, src = DataURL(raw)
	case strings.HasPrefix(raw, "https://"), strings.HasPrefix(raw, "http://"):
		u, err := url.Parse(raw)
		if err != nil || u.User != nil || u.Host == "" {
			return Key{}, Source{}, false
		}
		key, src = Remote(raw)
	default:
		return Key{}, Source{}, false
	}
	key = key.Sized(avatarPixels, avatarPixels)
	key.Cover = true
	return key, src, true
}

// Avatar requests the shared account avatar picture for raw, or returns nil
// when raw is not an acceptable avatar source. It never blocks.
func (s *Service) Avatar(raw string) *Handle {
	key, src, ok := avatarKey(raw)
	if !ok {
		return nil
	}
	return s.Request(nil, key, src)
}

// RetryAvatar forgets a failed avatar so the next Avatar call fetches and
// decodes it again (an explicit user refresh, not a repaint loop).
func (s *Service) RetryAvatar(raw string) {
	if key, _, ok := avatarKey(raw); ok {
		s.Forget(key)
	}
}

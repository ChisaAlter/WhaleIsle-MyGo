package platform

// BrowserPartitionValid accepts only opaque lowercase SHA-256 identifiers, not
// filesystem paths, origins, credentials or runtime-selected directory names.
func BrowserPartitionValid(key string) bool {
	if len(key) != 64 {
		return false
	}
	for _, c := range key {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

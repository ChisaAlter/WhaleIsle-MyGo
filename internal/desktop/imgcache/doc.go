// Package imgcache downloads http(s) image URLs into userData/cache/images
// for chat inline rendering. Files are named sha256(URL) with no extension.
// Fetches enforce Content-Type image/* and a default 8 MiB size cap.
package imgcache

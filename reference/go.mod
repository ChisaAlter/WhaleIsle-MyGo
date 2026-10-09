// Separate module so `go build ./...` and `go test ./...` in the app skip
// the Go sources inside the reference snapshot (vendor/whalebridge).
module whaleisle-reference

go 1.27

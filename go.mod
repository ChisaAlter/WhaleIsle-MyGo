module whaleisle

go 1.27.2

tool github.com/egoist/mygo/cmd/mygo

require (
	github.com/alecthomas/chroma/v2 v2.27.0
	github.com/coder/websocket v1.8.15
	github.com/egoist/mygo v0.3.5
	github.com/sergi/go-diff v1.4.0
	github.com/yuin/goldmark v1.7.8
	golang.org/x/crypto v0.45.0
)

require (
	github.com/dlclark/regexp2/v2 v2.2.1 // indirect
	github.com/ebitengine/purego v0.11.1 // indirect
	github.com/go-text/typesetting v0.3.5 // indirect
	github.com/go-text/typesetting-utils v0.0.0-20260922124329-7d4869a3934e // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

replace github.com/egoist/mygo => ./third_party/mygo

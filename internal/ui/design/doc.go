// Package design holds WhaleIsle's design tokens: the coss ui system
// (docs/design/coss) resolved for the native toolkit.
//
// tokens_gen.go is generated from docs/design/coss/tokens.json by
// `go generate ./internal/ui/design`; edit the spec or the generator, never
// the generated file. The hand-written files add the typed values the
// generator cannot know (the accent override, the focus ring recipes, the
// opacity and spacing rules) and the resolver that turns the window's
// scheme and the Host accent into a Tokens value.
//
// Colors, radii, spacing, type, shadows, opacity and motion used by
// internal/ui/coss come from here and nowhere else; the ratchet in
// internal/ui/designguard keeps literals from creeping back into the rest
// of internal/ui.
package design

//go:generate go run ./gen -in ../../../docs/design/coss/tokens.json -out tokens_gen.go

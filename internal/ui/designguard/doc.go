// Package designguard holds the ratchet that keeps hard-coded colors and
// radius/spacing literals from growing in internal/ui outside the design
// system (internal/ui/design and internal/ui/coss). The baseline in
// testdata/baseline.json records today's counts per file; a count may only
// shrink. Regenerate it after removing literals with
//
//	DESIGNGUARD_UPDATE=1 go test ./internal/ui/designguard
package designguard

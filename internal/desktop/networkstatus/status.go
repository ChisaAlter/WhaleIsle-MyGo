// Package networkstatus observes local OS connectivity without network probes.
package networkstatus

type Availability uint8

const (
	Unknown Availability = iota
	Offline
	Online
)

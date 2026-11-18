package main

// localizedError marks an error whose message is already written for a person:
// a sentence from the catalogue that names what happened and what to do next.
// The shell shows it as it stands, while a raw cause stays in the log.
type localizedError struct {
	message string
}

func (e *localizedError) Error() string { return e.message }

// localized wraps a catalogue sentence so the shell shows it verbatim instead
// of appending a raw cause to it.
func localized(message string) error { return &localizedError{message: message} }

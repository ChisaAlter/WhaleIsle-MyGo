package main

// windowNonActivatable reports whether the main window is created with
// WS_EX_NOACTIVATE, which stops the shell from putting it in front. A launch with
// nobody behind it — `--background` or an isolated desktop run — must not be
// raisable, not even by a program that calls SetForegroundWindow from outside. A
// direct launch must be: the person double-clicked the app and expects it there.
// The two ways a person asks for a background window, a click and an app-initiated
// reveal, clear the style first.
func windowNonActivatable(background, isolated bool) bool {
	return background || isolated
}

package terminal

import "errors"

// HostActions use the initiating session's existing composer and navigation.
// Selection never becomes terminal input and never automatically sends a chat.
type HostActions struct {
	CanAddToChat bool
	AddToChat    func(string) error
	OpenTarget   func(selection, cwd string) error
}

func ApplyHostSelection(selection string, valid func() bool, action func(string) error) error {
	if valid == nil || !valid() {
		return errors.New("Terminal selection is no longer available")
	}
	if selection == "" {
		return errors.New("Select terminal text first")
	}
	if action == nil {
		return errors.New("Terminal action is unavailable")
	}
	return action(selection)
}

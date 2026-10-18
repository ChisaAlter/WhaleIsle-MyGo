// Package taskguard protects native lifecycle actions using known local sessions.
// It does not inspect Host jobs/schedules or acquire a Host admission lock.
package taskguard

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
)

// ConfirmQuit shows a native question dialog when running > 0. It returns
// true when quit should proceed (no running sessions, or the user chose quit).
func ConfirmQuit(parent *mygo.Window, running int, title, messageFmt, quitLabel, cancelLabel string) bool {
	approved, err := ConfirmAction(parent, running, title, messageFmt, quitLabel, cancelLabel)
	if err != nil {
		log.Println("taskguard confirm:", err)
	}
	return approved
}

// ConfirmAction returns dialog failures to the caller so they remain visible.
// Cancel is both the keyboard default and Escape action.
func ConfirmAction(parent *mygo.Window, running int, title, messageFmt, actionLabel, cancelLabel string) (bool, error) {
	return ConfirmRunning(running, func() (bool, error) {
		msg := messageFmt
		if messageFmt != "" {
			msg = fmt.Sprintf(messageFmt, running)
		}
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent: parent, Type: mygo.MessageQuestion, Title: title, Message: msg,
			Buttons: []string{cancelLabel, actionLabel}, DefaultButton: 0, CancelButton: 0,
		})
		return res.Button == 1, err
	})
}

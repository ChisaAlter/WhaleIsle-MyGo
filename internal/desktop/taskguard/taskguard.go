// Package taskguard mirrors Electron quit protection when harness sessions are
// still running (M11 stub).
package taskguard

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
)

// ConfirmQuit shows a native question dialog when running > 0. It returns
// true when quit should proceed (no running sessions, or the user chose quit).
func ConfirmQuit(parent *mygo.Window, running int, title, messageFmt, quitLabel, cancelLabel string) bool {
	if running <= 0 {
		return true
	}
	msg := messageFmt
	if messageFmt != "" {
		msg = fmt.Sprintf(messageFmt, running)
	}
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:        parent,
		Type:          mygo.MessageQuestion,
		Title:         title,
		Message:       msg,
		Buttons:       []string{cancelLabel, quitLabel},
		DefaultButton: 0,
		CancelButton:  0,
	})
	if err != nil {
		log.Println("taskguard confirm:", err)
		return false
	}
	return res.Button == 1
}

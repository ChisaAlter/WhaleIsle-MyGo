package main

import (
	"fmt"
	"strings"
	"whaleisle/internal/ui/composer"
)

// submitProjectRecoveryDraft stages the verified request in the original Lead.
func submitProjectRecoveryDraft(d *composer.DraftState, text string, submit func()) error {
	if d == nil || submit == nil || strings.TrimSpace(text) == "" {
		return fmt.Errorf("recovery request is unavailable")
	}
	if d.Text != "" || len(d.Attachments) > 0 || d.Sending || d.Preparing > 0 || d.EditSeq > 0 || d.EditSessionID != "" {
		return fmt.Errorf("Lead has a draft or pending send; finish that draft first")
	}
	d.Text = text
	d.Client.Sync()
	submit()
	return nil
}

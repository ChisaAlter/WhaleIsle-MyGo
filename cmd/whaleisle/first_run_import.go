package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/egoist/mygo"

	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

// This file is the startup side of the data location: the warning for a
// recorded directory that is not there, and the one-time offer to copy the
// original Whale Isle profile into MyGo's own directory.
//
// Both run inside WhenReady before anything writes into the data directory.
// They are the only dialogs before config.json is read, so they cannot use the
// stored locale; WhenReady first switches the catalog to the system's own UI
// language, and the stored preference takes over once the config is open.

// resolvePendingDataLocation reports a recorded location that was missing at
// startup, before the fallback creates anything in the default directory. The
// two answers are "quit and fix the path" and "continue with the default";
// dismissing the dialog counts as the first, so no click ever continues by
// accident.
func (s *mainState) resolvePendingDataLocation() bool {
	s.mu.Lock()
	pending, unreadable := s.pendingDataLocation, s.pendingDataLocationUnreadable
	s.pendingDataLocation = ""
	s.pendingDataLocationUnreadable = false
	s.mu.Unlock()
	if pending == "" {
		return true
	}
	fallback := paths.UserDataDir()
	message := fmt.Sprintf(i18n.T(i18n.KeyDataLocationFallbackBody), pending, fallback)
	if unreadable {
		message = fmt.Sprintf(i18n.T(i18n.KeyDataLocationFallbackCorrupt), fallback)
	}
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageWarning,
		Title:   i18n.T(i18n.KeyDataLocationFallbackTitle),
		Message: message,
		Buttons: []string{i18n.T(i18n.KeyDataLocationFallbackQuit), i18n.T(i18n.KeyDataLocationFallbackContinue)},
		// Enter and dismissal both pick the safe answer: fix the record
		// rather than quietly building a second profile.
		DefaultButton: 0,
		CancelButton:  0,
	})
	if err != nil || answer.Button != 1 {
		log.Printf("data location: %q is not usable; quitting to let the user fix the record", pending)
		mygo.App.Quit()
		return false
	}
	log.Printf("data location: %q is not usable; continuing with %q for this launch", pending, fallback)
	return true
}

// claimDeferredProfile takes the desktop profile lock when startup postponed
// it to let the data-location warning go first. It is claimed here, before the
// first write, so two instances still cannot share a directory.
func (s *mainState) claimDeferredProfile() bool {
	s.mu.Lock()
	deferred := s.profileClaimDeferred
	s.profileClaimDeferred = false
	s.mu.Unlock()
	if !deferred {
		return true
	}
	release, err := claimDesktopProfile()
	if err != nil {
		// The operating-system cause stays in the log; the dialog carries the
		// product sentence and what to do about it.
		log.Printf("desktop profile: %v", err)
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Type:    mygo.MessageError,
			Title:   i18n.T(i18n.KeyDataLocationFallbackTitle),
			Message: i18n.T(i18n.KeyDataLocationProfileLocked),
		})
		mygo.App.Quit()
		return false
	}
	s.mu.Lock()
	s.releaseProfile = release
	s.mu.Unlock()
	return true
}

// offerFirstRunImport asks once whether the original Whale Isle profile should
// be copied into MyGo's own directory. Copying never moves or deletes source
// data, and each way out records its own outcome in the destination: a
// completed run, a deliberate 跳过, and a window closed without an answer are
// different markers. A failed import stays partial and is offered again on the
// next launch. Settings offers the same import whether or not an answer was
// recorded.
func (s *mainState) offerFirstRunImport() {
	if isolatedDesktopRun() || slices.Contains(os.Args, "--background") {
		return
	}
	dest := paths.UserDataDir()
	plan, offered := importer.DetectFirstRun(dest, importer.ElectronUserDataDir())
	if !offered {
		return
	}
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{
		Type:          mygo.MessageQuestion,
		Title:         i18n.T(i18n.KeyFirstRunImportTitle),
		Message:       fmt.Sprintf(i18n.T(i18n.KeyFirstRunImportBody), plan.Source, plan.Dest),
		Buttons:       []string{i18n.T(i18n.KeyFirstRunImportAccept), i18n.T(i18n.KeyFirstRunImportSkip)},
		DefaultButton: 0,
		// The offer must not pull the focus away from what the person is doing:
		// a launch that steals a click reads as the app misbehaving.
		NoForeground: true,
		// No cancel button: closing the window (or pressing Escape) must not
		// land on 跳过, which made a dismissal indistinguishable from a
		// decision. Windows' task dialog reports a dismissal as this
		// index instead of the skip button.
		CancelButton: -1,
	})
	if err != nil {
		// A dialog that could not be shown is not an answer: leave the offer
		// standing for the next launch instead of recording an outcome.
		log.Println("first-run import: asking:", err)
		return
	}
	if answer.Button < 0 {
		if err := importer.DismissFirstRun(dest); err != nil {
			log.Println("first-run import: recording the dismissal:", err)
		}
		s.invalidateLegacyImport()
		return
	}
	if answer.Button != 0 {
		if err := importer.SkipFirstRun(dest); err != nil {
			log.Println("first-run import: recording the skip:", err)
		}
		s.invalidateLegacyImport()
		return
	}
	result, err := importer.RunFirstRun(context.Background(), plan)
	for _, line := range result.Journal {
		log.Println("first-run import:", line)
	}
	if err != nil {
		log.Println("first-run import:", err)
		s.invalidateLegacyImport()
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Type:    mygo.MessageWarning,
			Title:   i18n.T(i18n.KeyFirstRunImportFailedTitle),
			Message: i18n.T(i18n.KeyFirstRunImportFailedBody),
		})
		return
	}
	s.invalidateLegacyImport()
	s.setDataLocationOutcome("", fmt.Sprintf(i18n.T(i18n.KeyDataLocationImportSummary), result.Copied, result.Skipped, result.Failed))
}

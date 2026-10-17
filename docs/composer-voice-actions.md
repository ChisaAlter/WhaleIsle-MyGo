# Composer voice cancellation and setup

While the draft has focus, plain non-repeating Escape cancels requesting,
recording or transcribing before it can arm or fire model-response stop. IME,
suggestions, the model chooser and approval/question ownership keep priority.
The existing native dialog focus trap continues to route modal Escape separately.
Cancel uses the existing voice controller generation/cancellation path and does
not send a prompt or discard the composer draft. Existing click cancellation stays.

The voice button opens the existing settings view when no start callback, no
connected Host, no selected speech provider, or failed voice feedback is present.
The settings path does not invoke recording. Existing preparation checks remain
in startComposerVoice / PrepareSpeechRecording for stale readiness. A selected
provider whose readiness changes between frames still reports its existing error;
the next render exposes Voice settings. The current settings tab is retained,
with tooltip guidance to speech input/plugin preparation and microphone permissions.

Verification: targeted native draft Escape cases passed for requesting, recording,
transcribing, IME/modal/suggestion/model chooser and modified/repeated keys. The
existing StopSequence fixture also passed. ui.Tester verified unavailable voice
buttons invoke only the settings callback. Windows amd64 frame compilation passed.
No real microphone, transcription service or desktop runtime was accessed.

Scope: Escape cancellation here is focused-composer routing. Global Escape while
focus is elsewhere is not added, so it cannot bypass another surface's ownership.

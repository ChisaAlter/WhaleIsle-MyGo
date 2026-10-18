# Native composer microphone dictation

The supplied Electron source really mounts
`packages/experimental/client-ui-voice-input` into `conversation.input.activity`.
Its `VoiceInput.tsx` records only on an explicit user action, freezes the selected
speech provider/language, supports stop/cancel, and inserts the transcript into
an existing draft without submitting it. `audio.ts` creates canonical mono,
16 kHz PCM16 WAV for `speech/transcribe`. Previously native MyGo only exposed an
audio-file picker using that RPC; it had no microphone capture lifecycle.

## User flow

The composer now has **录音 / Record**, **停止并转写 / Stop**, and **取消录音 /
Discard** alongside the existing audio-file picker and provider switcher. Record
uses the system-selected Windows microphone after checking the existing Host's
speech catalog. It does not enable a plugin, choose a different provider,
call `speech/prepare`, download a model, or create a separate credential. The
selected provider must already report ready, standby or waking, as in the
original. Otherwise the real missing-setup status is shown before microphone
acquisition. Existing plugin settings own provider installation/preparation.

The Record tooltip discloses capture and transfer to the currently selected
speech provider. Stop, or reaching the bounded recording limit,
finishes capture and asks the existing authenticated Host to transcribe. The
normal Host/provider implementation owns any configured local/cloud inference.
No chat prompt/send method is called by this flow. The original provider and
language remain frozen for that recording; switching them is disabled while
recording/transcribing. Transcription failure/empty result preserves the draft.

Insertion uses the original native UTF-16 selection with undo support. If draft
text, edit target or active IME state changed, the transcript is retained as a
pending result and only the explicit Insert button writes it at the current
selection. Its text can be inspected in the status tooltip. Discard drops it.
A changed session, Host, draft owner, closed/hidden window or switched main
panel cancels and discards late results. Window blur cancels active acquisition/
recording, while a transcription already submitted can finish if its owning
conversation stays valid. No raw audio survives as a file or saved preference.

## Capture and bounds

`internal/desktop/voicecapture` uses Windows' system winmm.dll, loaded only from
the system directory. Importing the package never opens a microphone; Start is
reached only through the explicit native Record action. Desktop microphone
privacy settings and driver support are respected. No settings are modified or
permission denial bypassed. Windows may require the user to enable desktop
microphone access; unsupported devices/formats fail with the draft intact.
Other platforms return an explicit unsupported result.

A single pinned PCM buffer avoids callback/requeue lifetime hazards. The effective
limit is the smaller of the Host catalog, 120 seconds, 8 MiB complete WAV, and the
PCM sample count supported by that byte budget. Stop/reset returns the partial
buffer, then unprepares/closes, clears native audio and unpins. Cancel and context
cancellation release/discard; a full/time-limited buffer stops hardware and only
retains bounded PCM until completion. Results are consumed once. Audio is
validated against the original canonical WAV contract before an RPC, and the
recording RPC client refuses redirects instead of forwarding audio/Host cookies.

If a faulty native driver repeatedly refuses reset/unprepare/close, its one
bounded allocation stays pinned to avoid use-after-free; new recordings are
blocked until restart. Hardware cleanup timing still depends on native calls.
This implementation does not add browser echo/noise processing or new audio
assets. It uses the native device/driver path, not a recording WebView.

## Implementation / verification

- `voicecapture`: bounded native microphone owner and in-memory WAV encoding.
- `voiceinput`: single-operation requesting/recording/transcribing/feedback
  lifecycle, cancellation, deadline and stale-grant/result protection.
- `session/speech_recording.go`: frozen authenticated speech plan, readiness,
  limits, WAV validation and existing provider/language RPC.
- `composer/speech_insertion.go`: original selection capture and safe pending
  insertion, using the existing draft client instead of a second draft store.
- `cmd/whaleisle/composer_voice.go`: actual session/Host/window ownership; minimal
  main callback wiring plus frame/composer controls. Existing audio-file import
  remains a separate available action.

One temporary fixture used only synthetic WAV, fake Capture and a loopback mock
Host. It covered duplicate start/stop, successful UTF-16 insertion, edited-draft
conflict/explicit insertion, errors, unprepared provider refusal without capture
or preparation, cancellation, stale results/grants, WAV rejection and detached
Host plans. No real microphone, account, remote speech provider or model ran.
Windows compile is not proof of microphone privacy, device format support,
actual audio quality, native input handling or live provider compatibility.

Native API references: [waveInOpen](https://learn.microsoft.com/en-us/windows/win32/api/mmeapi/nf-mmeapi-waveinopen),
[waveInReset](https://learn.microsoft.com/en-us/windows/win32/api/mmeapi/nf-mmeapi-waveinreset),
[waveInUnprepareHeader](https://learn.microsoft.com/en-us/windows/win32/api/mmeapi/nf-mmeapi-waveinunprepareheader),
[waveInClose](https://learn.microsoft.com/en-us/windows/win32/api/mmeapi/nf-mmeapi-waveinclose),
[WAVEHDR](https://learn.microsoft.com/en-us/windows/win32/api/mmeapi/ns-mmeapi-wavehdr),
[Go runtime.Pinner](https://pkg.go.dev/runtime#Pinner).

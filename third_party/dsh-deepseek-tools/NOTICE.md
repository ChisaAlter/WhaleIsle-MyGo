# dsh-deepseek-tools production source

Source: `C:\AI\Deepseek-Harness-Desktop\vendor\dsh-deepseek-tools` (the 本体 desktop
repository, `ChisaAlter/Deepseek-Harness-Desktop`), at commit
`334a3db3de60ad1a1d1443e9a3993580ce75fb03`. The desktop added the package in
`67c982980f0` ("DeepSeek speech and document AI tools").
Version: 0.1.0 (`package.json`).

License: MIT, declared by `package.json`. The package directory itself carries no
LICENSE file, so the repository-root LICENSE of the same checkout is included
unchanged as `LICENSE` — MIT, Copyright (c) 2026 Chisa — and this note records
where it came from. Keep that pairing when moving this tree.

Only the shipped sources are included: `lib/api.js`, `lib/desktop-confirm.js`,
`lib/index.js`, `client/client.js`, `package.json`, `README.md`. Both files under
upstream `tests/` are omitted: they drive the upstream Host and its test runtime,
not this desktop.

WhaleIsle adaptations: none. Every file matches the upstream checkout; the only
difference is line endings, which this repository normalizes to LF
(`.gitattributes: * text=auto`) while upstream keeps CRLF. Fix behaviour here only
by re-copying from upstream and recording the change in this note, never by
editing in place.

Host contract this package needs, recorded because the desktop has to supply it:

- Sibling module `vendor/dsh-shared/lib/internal-secret.mjs`: `lib/desktop-confirm.js`
  imports it relatively (`../../dsh-shared/...`). It is not part of this tree and
  WhaleIsle's runtime stage does not carry it yet, so the package cannot load
  until the stage supplies that sibling.
- Desktop control plane `POST /desktop/confirm` on the loopback install channel
  (`DSH_DESKTOP_INSTALL_URL` + Bearer token): every TTS/ASR/OCR call asks the
  desktop to confirm the upload first and fails closed when the channel is
  absent. See `lib/desktop-confirm.js`.
- The Host composition row (`dsh-deepseek-tools`, gated on
  `DSHD_DEEPSEEK_TOOLS_ENABLED`) is owned by the Host composition lane, not by
  this tree.

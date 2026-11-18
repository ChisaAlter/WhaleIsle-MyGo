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

WhaleIsle adaptations: one, in `lib/desktop-confirm.js`. Upstream reads the
control token from the in-process secret store through a sibling package
(`../../dsh-shared/lib/internal-secret.mjs`); this copy reads
`process.env.DSH_DESKTOP_INSTALL_TOKEN`, the way the desktop plugins WhaleIsle
already stages do, so the package has no sibling dependency and needs no second
copy of `dsh-shared`. Everything else matches the upstream checkout apart from
line endings, which this repository normalizes to LF (`.gitattributes: * text=auto`)
while upstream keeps CRLF. Fix behaviour here only by re-copying from upstream and
recording the change in this note, never by editing in place.

Host contract this package needs, recorded because the desktop has to supply it:

- Desktop control plane `POST /desktop/confirm` on the loopback install channel
  (`DSH_DESKTOP_INSTALL_URL` + Bearer token): every TTS/ASR/OCR call asks the
  desktop to confirm the upload first and fails closed when the channel is
  absent. See `lib/desktop-confirm.js`.
- The mount row is written by `internal/harness/overlay.go` and gated on
  `DSHD_DEEPSEEK_TOOLS_ENABLED`, which `internal/harness/launch.go` sets for a
  desktop start.

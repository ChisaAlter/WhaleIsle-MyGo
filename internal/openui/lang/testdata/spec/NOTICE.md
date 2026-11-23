# OpenUI Lang conformance fixtures (vendored)

- Upstream: https://github.com/thesysdev/openui
- Path: `spec/fixtures/1.0/` (copied unmodified into `fixtures/1.0/`)
- Commit: `06ecd41c` ("spec: OpenUI 1.0 specification (#1277)", 2026-10-14)
- License: MIT, Copyright (c) 2011-2024 Thesys Inc. (see `LICENSE` beside this file)

Why this commit: the Whale UI viewer bundled in
`internal/desktop/openui/viewer/v2` derives its language runtime from
`@openuidev/lang-core` 1.0.0, tag commit
`f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20` (see the viewer's
`THIRD_PARTY_NOTICES.txt`). The 1.0 specification and its 59 fixtures were
published two days later at `06ecd41c`; between the two commits
`packages/lang-core/src` changed only by adding the conformance test runner, so
these fixtures describe exactly the semantics of the bundled lang-core 1.0.0.
The fixture files are byte-identical on upstream `main` (spec 1.1 changed only
`language.md`).

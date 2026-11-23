# coss component gallery

Compares `internal/ui/coss` with coss ui itself, specimen by specimen
(component × variant × size × state, light and dark).

```sh
node scripts/coss-gallery/run.mjs                 # everything → tmp/coss-gallery/report.md
node scripts/coss-gallery/run.mjs --only button   # one sheet
node scripts/coss-gallery/run.mjs --keep          # keep the work dir (native/, ref/, side/, zoom/)
```

- **Native**: `TestGalleryRender` (internal/ui/coss/gallery_*_test.go) renders
  every specimen with MyGo's headless Tester at scale 1, the app's font stack
  (`theme.TextFamily("")`) and the Host accents (#4176e6 light, #7aaaff dark).
  Hover, pressed and focus are forced through the package's test hook; the
  interaction tests in `interaction_test.go` drive the same components with
  real pointer and key events.
- **Reference**: `ref.mjs` renders each specimen's coss markup — class strings
  resolved from `docs/design/coss/index.json` (`{{cva:…}}`, `{{part:…}}`) and
  merged with tailwind-merge as coss's `cn()` does — compiled by
  tailwindcss 4.1.17 against coss's pinned `globals.css`, in headless Chromium
  (playwright-core with an installed chrome-headless-shell), at the same
  scale, font and accent (`--primary/--primary-foreground/--ring` overridden as
  the app's tokens do). Pseudo-classes are forced over CDP.
- **Diff**: `TestGalleryDiff` writes `report.md`: per specimen the share of
  pixels whose channel delta exceeds 32, and native − reference deltas of box
  width/height (layout), text inset (padding), text vertical center, corner
  radius (estimated from pixels on both sides) and the fill just inside the
  left edge. Set `COSS_GALLERY_ZOOM=<id substring>` with `--keep` to write 4×
  crops.

Specimens without markup (composites such as chat messages, states, the
composer and the command palette) render natively only; their parts are
compared on their own sheets.

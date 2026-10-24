# Round 170 — D-MAIN structure report, fixture recipe and criterion test

Follow-up to round167 (`round167-acceptance-pipeline-pilot.md`), which proved the capture mechanics
but measured 99.99 % mismatch because the two windows were not on the same fixture.

Deliverables this round: `evidence/accept-D-MAIN/fixture.md` (complete recipe, both sides),
`scripts/acceptance-structure.py` (new: name+role matching, 1-DIP tolerance, missing/extra, per-region
pixel ratios → `structure.json`), the structure report itself, and this verdict. No product code, no
contract edit, no commit, no passN exe; the installed app's files were never written.

## 1. Fixture — defined, one step blocked

`fixture.md` pins the row: client 1440×860 DIP @150 % (2160×1290 px; 920 DIP does not fit the working
area with the taskbar), both windows anchored at (0,0), dpi 144, zh-CN, terminal closed, one empty
session, **light theme**.

- **MyGo side is reproducible**: throwaway profile with `autoStartDesktop`, `goPreviewNoticeAck`,
  `locale=zh`, temp workspace, `welcome-shown`, and a ~45 s settle for the harness handshake. It
  renders light today (and ignores `theme: deepseek.dark` / `dark` — a real defect owned by task-52;
  it happens to match the light target, and it blocks any *dark* row).
- **Native side is half reproducible**: the empty session is one UIA click (`新建会话` at DIP
  (14,70) 252×38), but the theme switch could not be automated:
  `Ctrl+,` opened settings only on the first attempt; `账号菜单` → `设置` never produced the menu item
  in UIA (0 matches in 8 retries); `Alt+F` + arrow walk left the tree at 83 nodes with no settings
  window; and the appearance buttons (`浅色` (1052,265), `深色` (1335,265), `跟随系统` (1619,265),
  physical) expose no Invoke/Selection pattern, so only a real mouse click works **after** the modal
  is open. The 3-click human recipe is recorded in `fixture.md`; the coordinate table makes a future
  automation trivial once the modal can be opened reliably.

Conclusion: the matched pair is **not yet captured**. The numbers below are therefore from the
unmatched pair (round167 capture, same 1440×860 fixture size), which is exactly what this report is
for — separating *fixture* error from *renderer* delta.

## 2. Numbers (unmatched pair, 1440×860 DIP, native 0.3.10 vs MyGo `e96c0543`)

Pixel (threshold 0, `diff.json`): 2 786 097 / 2 786 400 = **99.99 %**, channel max Δ 240, masked 0.

Per region (`structure.json`):

| region | pixels | mismatch | ratio |
| --- | --- | --- | --- |
| sidebar | 550 830 | 550 664 | 99.97 % |
| header | 155 970 | 155 966 | 100.00 % |
| hero | 883 830 | 883 830 | 100.00 % |
| composer | 571 890 | 571 890 | 100.00 % |
| footer | 21 930 | 21 916 | 99.94 % |

The theme difference alone (native dark `rgb(30,38,47)` sidebar vs MyGo light `rgb(146,186,191)`; hero
`rgb(17,36,45)` vs `rgb(237,246,241)`) accounts for essentially every pixel, so these ratios say
nothing about layout.

Structural parity (UIA, name+role key, 1-DIP tolerance):

| metric | value |
| --- | --- |
| named native controls / MyGo controls | 64 / 44 |
| matched by (type, name) | 15 |
| within 1 DIP | **4** |
| outside 1 DIP | 11 |
| native-only (missing on MyGo) | 48 |
| MyGo-only (extra) | 29 |

The 11 outside-tolerance matches, and what they actually mean:

| control | Δ (DIP) | cause |
| --- | --- | --- |
| 标准模式 | x +472.0, y +14.0 | hero block centres in a column with no session list — fixture |
| 账户信息 | x +12.7, y −28.7 | signed-in account row vs signed-out seat — state |
| 账号菜单 | y +8.7, h −8 | native trigger 44 DIP (signed in) vs merged seat 36 DIP — state, and the correct spec for each |
| 收起侧边栏 / 多选会话 / 视图选项 / 切换终端抽屉 / 最小化 / 最大化 / 关闭 | y +4.7…+6.7 | everything below the native session tree is shifted; the header buttons differ by the window-control band, which is a *desktop-injection* difference (Electron draws the native caption plate; MyGo draws its own) |
| 探索未至之鲸 (Text) | y +24.0, w +14 | hero vertical seat inside a column without the reserved caption |

The `missing` / `extra` lists also expose a **second, independent gap**: the two apps expose
different accessible names for the same control. Electron: `Button 新建会话`, `Text 探索未至之鲸`,
`Button 选择工作区`; MyGo: `Button 新会话` (+ `Text` children), `Text WHALE ISLE`, `Button 选择工作区`
with a different role. Name+role is therefore a fragile key for structural parity — a stable
per-control id (or a maintained name map with an explicit alias table) is needed before the criterion
can be automated. This is a finding for the contract's `controls.json`, not a MyGo defect by itself.

## 3. Proposed criterion from round167, tested against these numbers

| criterion | result | why |
| --- | --- | --- |
| masked mismatch = 0 | **not testable** | no mask was needed in this state (no caret/spinner/toast) and the frame is 100 % different for theme reasons — masking cannot rescue an unmatched fixture |
| structural parity: every native control present, rect ≤1 DIP | **fail on the unmatched pair** (4/15 within tolerance, 48 missing, 29 extra) — the failures are explained by fixture, state and accessible-name drift, not by layout of shared chrome | needs (a) the matched fixture, (b) a stable control key |
| per-region mismatch ≤0.1 % after theme+data match | **not testable yet** | ratio is 99.94–100 % before the match; the fixture is the precondition |
| text-AA counted separately | **possible today** | the pipeline can dilate text boxes by 1 DIP and count only outside them; that measurement is only meaningful once the fixture matches |

Practical refinement to propose (for review, contract untouched):

1. Fixture precondition is a hard gate — a row may only be scored when `fixture.md` is applied and
   the pipeline records `theme`, `locale`, `session`, `workspace count` and `signed-in state` in
   `accept-log.txt`. Fail the row as *not measurable* otherwise (rather than "99.99 % mismatch").
2. Structural parity keys on **a stable id where UIA exposes one** (Electron `AutomationId`/DOM id vs
   MyGo element keys), with a maintained alias table for names; unmatched names must be reported as
   `alias-missing`, not as `missing control`.
3. Pixel scoring stays per region with the mask registry, `≤0.1 %` after the match, and text-AA
   counted outside a 1-DIP dilation of text boxes. The native caption band and the MyGo caption band
   are injection differences and need their own registered mask or an agreed-excluded band.

## 4. L-HOME (launcher) proposal

L-HOME cannot be captured as defined today, for a structural reason: the native launcher is its own
window (**1060×660**, separate BrowserWindow) while MyGo's launcher is a panel inside the 1440×920
main window (phase-1 state, task-12). "Same client size" therefore cannot hold across the two.

Proposal (needs a Lead fixture decision, no code change):

- **Option A (recommended):** capture L-HOME as *window-to-window* with each app at its own native
  launcher size — native 1060×660 window, MyGo main window resized to 1060×660 showing the launcher
  panel — and record the fixture as "launcher, 1060×660, panel-vs-window" so the row is scored on the
  panel's own geometry, not on the window shell. The pipeline already resizes both sides, so this is a
  parameter change plus a note.
- **Option B:** open the native launcher through its own UI and capture only the launcher window, then
  capture MyGo's launcher panel cropped to the same 1060×660 region. Higher fidelity, but the native
  launcher window must be opened first; a second instance currently only focuses the existing main
  window, and the launcher is only reachable on a cold start (no workspace/launcher gate) — i.e. it
  needs its own launch recipe and its own profile, not the running instance.
- **Option C:** declare L-HOME out of scope until MyGo's launcher becomes a separate 1060×660 window
  (task-12 phase 2), and score only the launcher's inner panel meanwhile.

Until the Lead picks one, L-HOME stays unmeasured.

## 5. Pipeline changes this round

- `scripts/acceptance-structure.py` (new): name+role matching, 1-DIP tolerance, missing/extra lists,
  per-region pixel ratios, `structure.json` — the structural half of the criterion, computable without
  a matched fixture.
- `evidence/accept-D-MAIN/fixture.md`: the recipe, the exact appearance-button coordinates, and the
  blocked automation step written down so the next round can finish it in minutes.
- `structure.json` copied next to the round167 artifacts.
- Kept from round167 and still correct: Electron lazy-UIA warm-up, Electron child-HWND occlusion
  handling, 1-px crop record, BOM-free JSON.

## 6. State

Native app: untouched on disk, no theme change applied (still `跟随系统`, dark), still on the empty
`新会话` created in round167 — its previous session must be re-selected by the user (the row is not
reachable in the UIA tree from the collapsed workspace group). MyGo pilot process and throwaway
profiles were stopped and deleted. No commit, no passN exe.

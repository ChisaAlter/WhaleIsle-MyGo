# PET-21: automatic desktop-pet rest

## Source contract

This closes the real automatic-rest gap in the existing pet audit. It is an
application desktop-pet feature, unrelated to ChatGPT Pets. Latest checked
upstream: `ChisaAlter/WhaleIsle@e9f584be6d3b491ce777bb6823f30bc2bbc6f195`.

- `src/renderer/pet-live2d.js:3494–3515`: idle greater than 225 seconds enters
  doze, greater than 240 seconds enters sleep; interaction/actions/panels suppress
  the transition. MyGo previously had only a sleepy chatter category.
- `src/main/pet-stats.js:16–25,79–94,107–132`: care remains in
  `live2dPet.stats`, with satiety −3/hour, mood tending toward 62 by 8/hour,
  capped 72-hour elapsed drift and no affection decay. This already exists in
  MyGo and is not replaced or made into a timer-driven write.
- Sleep is temporary page state. Explicit waking commits the existing −7 mood
  care action; contact and successful pat/file/token meals wake for free.

## Native behavior

A window-owned one-second timer updates only local rest state. After 225 seconds
without interaction, the existing rest status displays `打盹中 / Dozing`; after
240 seconds, it enters the same real sleeping state as the existing Sleep action.
The existing renderer status poll pauses/resumes sprite/THA4 work. There is no
new renderer, sleep pose, generated message, model call, Host RPC or storage field.

Controls, pointer/focus contact and native drag samples reset the idle period.
Control contact does not undo sleep; touching the character still wakes for free.
An open compact chat, active care/picker/chat/look/growth action, active selected
session task, drag or recent native contact suppresses the nap. A hidden native window stops
the timer and showing it restarts a fresh idle period.
Suppression resets the idle period, so closing a busy panel never causes an
immediate overdue nap. Successful care wake also restarts the full idle period.

Close, hide, disable and page recreation stop/invalidate the timer. Host owner/generation
changes stop/invalidate queued old-window work before a fresh idle period starts.
Local care still works offline. Page recreation starts awake; Host switching
retains current sleep and stored care. A failed explicit wake save retains sleep.
No timer persists hunger, grants points, scans logs or rewrites preferences.

## Verification and remaining gaps

A single synthetic Go fixture invocation (`-count=1`) passed exact thresholds,
contact cancellation, fresh-page state, blocking/backwards-clock recovery,
explicit-wake save failure/idempotence, free care wake and timer cancellation
state. No real Host, model, Bot, user log or filesystem meal was exercised.
Windows compilation is reserved for the integration owner. Native GUI timing,
hover delivery and sleep pose are not claimed verified.

PET-19 automatic food-bowl scan/pre-feed refresh remains a separate unimplemented
module. The existing manual refresh and settled-food ledger are unchanged.

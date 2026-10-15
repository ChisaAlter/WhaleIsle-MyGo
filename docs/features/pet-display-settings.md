# Desktop pet display and interaction settings

The native desktop pet now persists scale (0.6–1.6, 0.1 steps), character opacity
(0.3–1.0, 0.05 steps), position locking, Shift-to-drag, click sound and the pet-chat
entry switch. The values follow the original WhaleIsle 90465b67 pet settings.

Save applies these settings to the current window. Changing only these options
keeps the current renderer and temporary sleep state alive. A skin, rendering mode
or enabled-state change still recreates the relevant window. Only the character
fades; task/action controls remain readable. Locking and the Shift requirement
change the native draggable region. Disabling pet chat blocks the native action
as well as disabling its link. The existing main conversation remains available.

Settings live under `mygoDesktopPet.behavior` in the existing config file. Missing
or malformed fields default independently, and unrelated configuration survives
atomic writes. The configuration type also reserves the original behavior fields
for the separate runtime-behavior module; this change does not claim those
fields are already exposed or wired.

Source tests cover configuration normalization and persistence. This delivery
has not executed the pet, a Host, a model, screenshots, or the test suite.

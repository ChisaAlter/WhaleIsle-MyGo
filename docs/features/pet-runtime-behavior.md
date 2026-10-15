# Desktop pet runtime behavior

The original WhaleIsle 90465b67 local behavior is now connected to the native
pet's saved settings:

- Self-talk uses the packaged original global/personality dialogue pools,
  shuffle bags, care thresholds, time-of-day categories and activity intervals.
  There is no background model request. Busy interactions, sleep and disabled
  self-talk suppress ambient bubbles.
- Activity selects both talk and wandering intervals. Wandering uses actual
  native window positions with original speed/reach/easing ranges. Work-area
  bounds prevent off-screen movement; contact and busy/sleep states interrupt it.
- Power saving activates after five idle minutes and suppresses autonomous
  talk/motion while reducing the THA4 inference budget from 50 ms to 110 ms.
  The native sprite fallback halves its frame cadence from 120 ms to 240 ms.
- Manual drag anchors are saved separately from transient wandering positions.
  The old native window-state anchor is read once if no explicit anchor exists.
  Automatic movement never overwrites that manual anchor.
- Position lock and Shift-to-drag constrain manual dragging. As in the original,
  locking manual dragging does not silently turn off wandering; its switch is
  independent.

The personality selection affects local speech in this module. Synchronizing an
explicit saved personality to Whale assistant settings is a separate module.
The main persistent conversation navigation is retained.

The module contains pure motion test source. No tests, real app, Host, model,
screenshot or GPU execution was performed during authoring.

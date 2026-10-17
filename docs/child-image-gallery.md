# CH77 child image gallery

Explicitly loaded child resource images now open a modal by clicking their
thumbnail or Enlarge button. Previous/Next traverse only successfully decoded,
already-loaded local paths in the current resource snapshot (stable reference
order). Fit uses the available viewport; 1:1 uses natural bitmap dimensions in
UI logical pixels and permits scrolling. Close, backdrop, and Escape use the
existing DialogBase modal/focus handling.

The retained viewer key includes Host history binding, complete child address,
and navigation token. Changing ownership hides the old modal. Removing the
selected resource from the snapshot closes it. No remote prefetch, global parent
attachment resolver, arbitrary URL authorization, or new image fetching exists.
The established ReadAgentResource child authorization and caching are unchanged.

Focused UI test: TestChildImageGalleryScopedNavigation passed using two synthetic
PNG files and an excluded remote URL. It checks modal opening, previous/next,
Fit and 1:1 controls, child scope isolation, resource revocation and Escape.
The agents package compiled in the same targeted invocation. Native desktop
acceptance and physical pixel/DPI behavior were not exercised. Automatic tool
image fetching remains explicit-read rather than an automatic original gallery.

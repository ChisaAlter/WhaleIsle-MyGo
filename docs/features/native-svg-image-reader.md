# Native SVG image reader

The Files SVG default is now decoded by resvg 0.46.0 in a no-cgo wazero
instance. The complete accepted static image is rasterized from authorized SVG
bytes, not captured from a screenshot or privileged page. Text/tspan/textPath,
styles, gradients, clips, masks, embedded raster images and static filters go through resvg rather than the toolkit's
shape-only SVG subset. The existing native image view supplies fit-width,
presets, +/- from fit, pointer zoom, pinch synthesis and drag-pan. Dirty text
drafts still take precedence over image display.

The native-only validator retains the 1 MiB, structure, geometry and local-reference
bounds, adding bounded base64 PNG/JPEG/GIF/WebP resources and static filter nodes.
A resource-free structural projection reuses the original validation limits; the
renderer receives the original image bytes. The browser-document validator is
unchanged. No document JavaScript, external image, file href, network URL, nested
SVG data image or arbitrary WASM is accepted. Each renderer instance has no WASI filesystem preopens,
network access, environment or application bridge. The only fonts passed are
bundled Go font bytes and a fixed platform system-font list, with a combined
64 MiB limit. Missing typefaces may use fallback fonts; exact browser typography
is not claimed. SVG decoding remains subject to the 32-million-pixel bound,
a 512 MiB WASM linear-memory limit and a cancellable 10-second operation.

The implementation is a pinned MIT wrapper and checksum-verified embedded WASM
from github.com/emulatorchen/pptx-glimpse-go v0.0.2, with wazero v1.12.0.
resvg 0.46.0 is dual MIT/Apache-2.0. Wrapper licenses, exact upstream source/
artifact provenance, source build recipe, Cargo lock and license texts for
all 60 locked crates are retained under the renderer's upstream directory.
No optional CGo backend or GPL wrapper is included.

New fixtures cover the embedded artifact hash, actual nonempty text rasterization,
gradient content, rejected active/external SVG, and native routing. Integration
owns execution of focused fixtures and any platform/runtime acceptance; no GUI
acceptance or comprehensive SVG/browser-equivalence claim follows from source
integration alone. Still AVIF uses its separate native decoder; explicit isolated compatibility
remains available and does not gain document script authority.

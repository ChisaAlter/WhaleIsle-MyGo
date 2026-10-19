# Beam blur and coverage continuation

Fixed reference: `e353e8e13fa0e7f9cb6bc16368e2fc00d6b3be49`, read with
`git show`, including `vendor/deepseek-harness/packages/client/ui-conversation/src/client/ComposerBeam.module.css`
and `ComposerBeam.tsx`. The original worktree is not the reference.

The CSS bloom pseudo-element is a 1.5px ring at the card's radius22, filtered
with `blur(8px)` before its parent radius26, inset-minus4 overflow clip. Inner
light also has independent `inset 0 0 18px 3px` and `42px 10px` shadows above
the radial backgrounds. Its registered angle, palette stops, hue, opacity and
breathing transitions remain on the existing native timing path.

The previous native code repeated three box convolutions inside the final
four-pixel output viewport. That both approximated the Gaussian and discarded
intermediate halo contributions that later box passes should bring back. Its
nearest-edge erfc for inset shadows also omitted other edges and the two
dimensional rounded hole. A single center-distance ramp approximated coverage.

The native Beam now uses a normalized, separable Gaussian at device scale,
with a finite three-sigma kernel, transparent source outside its support and
the outer clip applied after filtering. Sparse samples and first differences
times the Gaussian CDF evaluate the same discrete convolution; solid shadow
intervals need only their boundary transitions. Inset shadows blur the actual
spread-inset rounded hole and take its complement. Boundary pixels use 4x4
area samples; interior/exterior pixels take constant coverage. Inner/ring/bloom
geometry, source layer order and filter/color order remain distinct.

Geometry, inset shadows and each animated RGBA frame run in one finite Beam
job off the UI thread. Only one job and one buffered result can exist per
raster. Static geometry/kernel storage is reused. Native paint exchanges two
RGBA buffers, retaining its completed image while the next job runs; obsolete
size/palette results cannot replace that image. No persistent worker, native
window, SDK API or effects framework was added. A detached Beam leaves only
its already-running finite calculation, whose result send cannot block.

Regression preparation covers an independent full two-dimensional Gaussian
sum at the viewport edge, solid/CDF and sparse paths, kernel reuse, the opposite
edge contribution in a narrow inset hole, bloom-before-clip and rounded area
coverage. Root's final integrated Windows ordinary run at `aa159b6a` passed,
including these fixtures; see the [current integration checkpoint](source-depth-integration-20261019.md#current-windows-checkpoint)
and [fixed-commit receipt](../../evidence/final-svg-closed-regression-20261019/root/receipt.json).
The matching main/helper build also passed; the main application was not
launched. The owner implementation phase itself performed no test or build.

This does not claim Chromium pixel equivalence: finite three-sigma sampling,
4x4 coverage, the existing 2048 conic samples and the existing native RGBA upload
remain rasterization boundaries. One completed image may be retained during
an in-flight calculation. Real motion, timing, resize, resource release and CPU
cost still require the authorized Windows validation phase.

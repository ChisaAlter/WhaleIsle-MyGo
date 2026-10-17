# WT04: macOS Browser inspection

The Browser toolbar now offers “允许 Safari 检查” on macOS. An explicit click enables the public WKWebView `inspectable` property only on the current verified, binding-free Browser page. The default remains disabled. Plugin pages, transient account pages and document previews do not acquire this capability. Closing the Browser tab destroys the inspectable view.

The user then selects the page in Safari's Develop menu under the current Mac/app. The UI supplies those steps and does not claim an inspector window opened. Safari developer features must already be enabled by the user; this implementation does not alter Safari settings or launch an external app automatically.

Public `inspectable` requires macOS 13.3+. This application's persistent Browser profile already requires macOS 14+, so the effective Browser minimum remains 14. There is no public WKWebView equivalent here for programmatically opening/docking Electron DevTools. No private `_inspector` selector or private developer-extras setting was added to this Browser path. WT04 remains partial relative to Electron's automatic detached Inspector behavior.

References:
- https://developer.apple.com/documentation/webkit/wkwebview/isinspectable
- https://developer.apple.com/documentation/safari-developer-tools/enabling-inspecting-content-in-your-apps
- https://webkit.org/blog/13966/webkit-features-in-safari-16-4/

Native macOS runtime acceptance still required; no macOS application was run in this workspace.

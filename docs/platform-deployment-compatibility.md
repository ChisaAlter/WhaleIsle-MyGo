# PR02 deployment resource and popup compatibility

Original src/main/platform-view.js:133–155 injects deployment/client headers only
for the exact account origin. Lines160–174 open clean HTTPS window.open/blank
links in the system browser, while same-tab off-origin navigation and redirects
remain denied. It does not add a self-only resource CSP.

MyGo now preserves upstream Content-Security-Policy headers without adding a
second self-only policy. This lets the trusted deployment specify CDN, frame and
payment resource sources itself. No domain allowlist was guessed and no generic
network proxy was added. The gateway still targets only its captured account
origin and refuses cross-origin redirect responses. Referrer-Policy remains
no-referrer; deployment headers/cookies are still injected only server-side on
the pinned upstream request.

Platform's private transient browser now opts into a popup-only external callback.
Clean HTTPS destinations open through the existing OS OpenExternal route with
no WebView profile, request headers or cookie transfer. Local account popup URLs
are translated back to the real account origin. Invalid schemes, credentials in
URLs, lease entry URLs, and URLs containing the known token/lease secret are
rejected. Ordinary transient previews and HTML helpers receive no such callback
and keep their existing popup/isolation policies. Same-tab external navigation
continues to be blocked, matching the original rather than making payment
redirects universally navigable.

The trusted-page preload also removes Authorization, deployment/client headers
from explicitly external fetch/XHR calls. Fetch omits credentials and referrers
for those destinations. Matched account fetches still rewrite to the pinned
lease. This is transport hygiene for the trusted deployment, not a sandbox for
hostile account-origin JavaScript: the platform API intentionally exposes its
account token to trusted page code, as in the original.

One targeted invocation passed platform fixtures (0.188s) and compiled webview
and cmd/whaleisle. Fixtures used a synthetic TLS account and Node JavaScript:
- upstream CDN/payment CSP is retained exactly once;
- cross-origin redirect stays rejected;
- clean HTTPS popup is allowed; userinfo/scheme/token/lease leakage is rejected;
- loopback account popup deproxies without browser context transfer;
- external fetch/XHR strips sensitive headers; account fetch preserves auth.

No actual account, payment provider or browser launch was used. Actual deployment
CSP/absolute-origin assumptions, CDN CORS, popup activation and provider flow
still need Windows/macOS acceptance. Trusted upstream CSP is passed through
rather than rewritten, so a deployment policy tied to an absolute account origin
may require deployment-side embedded-page support for the loopback transport.

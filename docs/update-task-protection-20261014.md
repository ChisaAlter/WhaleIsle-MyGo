# Protected update installer handoff

Confirmed unprotected entry: native `showUpdateCheck` used `Start:true` for both full and incremental downloads. It now stages and verifies first, then offers an explicit cancel-default “Install and quit” confirmation, saves the existing resume snapshot before locking, and uses the shared Host task coordinator with operation `update`.

Only the admitted commit launches the installer. A changed staged file (hash/size mismatch), launcher error, cancelled prompt or Host protection failure keeps the existing Host/window; the coordinator releases any acquired lock on failure. A successful launch uses `finalizeQuit` for existing phone runtime, Host, session, terminal and window cleanup, then quits without reentering the lock or task confirmation. Import admission and supervisor identity are rechecked before launch.

Both incremental full-download fallbacks now perform blockmap verification before optional installer launch. `StartStaged` rechecks the verified bytes after a potentially long confirmation. Existing feed sources and signature policy are unchanged. No installer or real Host was launched/stopped during implementation.

Other checked paths: install-control restart callback and runtime settings reload already route through protected `restartUnlessImport`. The browser `OnReload` is WebView navigation, not Host replacement. Component install is currently an explicit unsupported stub; it is not presented as an implemented Host update.

Focused verification: one invocation `go test ./internal/desktop/taskcontrol ./internal/desktop/update -run '^TestUpdateProtection' -count=1` passed both packages. It covers 11 coordinator branches with fake-token HTTP mock, plus staged-byte change, launch failure, duplicate-start rejection and successful fake launch. Windows compile belongs to the integration owner. No full suite or live installer test is claimed.

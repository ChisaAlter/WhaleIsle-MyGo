# Round113 process-level ToolCallTree device scale

PASSED. Four independent installed Chrome child launches used process `--force-device-scale-factor`, with no CDP Emulation calls. [Full measured JSON](tool-tree-pass257-scale-2026-10-23T23-43-12-114Z-69c98e19.json) and [concise output](tool-tree-pass257-scale-2026-10-23T23-43-12-114Z-69c98e19.txt).

|Requested scale|Actual DPR|Computed border CSS px|Child X delta|22+8+observed border|Native border math|Root H|Seat H|Child H|Cleanup|
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
|1|1|1|31|31|1|110|112|54|true|
|1.5|1.5|0.666667|30.666667938232422|30.666667|0.6666666666666666|110|112|54|true|
|2|2|0.5|30.5|30.5|0.5|110|112|54|true|
|3|3|0.333333|30.333335876464844|30.333333|0.3333333333333333|110|112|54|true|

Exact installed CSS and mappings, source offsets and untouched before/after SHA256 values are retained in JSON. The own cards are synthetic24px blocks with root-child-grandchild and sibling wrappers; the tree CSS is unmodified. Root plain-block trailing2px margin collapses outside the root box: root110px, enclosing flex-seat112px, child54px. All positions are CSS px. Expected indentation uses the observed computed border, independently from the native formula comparison.

Each run verifies spawned browser PID via SystemInfo, exact unique profile, CDP port and startup scale flag via Browser.getBrowserCommandLine, then closes only that browser; process exit, all observed owned child exits, released loopback port and removed profile are recorded. No app server, remote page request, original application/profile/port3080 access, lock-screen/foreground changes, dependency installation, build, broad tests, source edit or commit occurred.

This is Chrome process-level startup scale evidence, not physical-monitor input or Electron/full-app acceptance. It is separate from the earlier geometry acceptance, whose raw result remains 8/9 cases and 135/136 assertions because its root-height oracle incorrectly expected106/108 instead of the measured110/112. Each scale ran once in the same invocation; no passing rerun.

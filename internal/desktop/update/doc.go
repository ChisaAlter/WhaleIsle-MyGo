// Package update implements the MyGo release updater for M11 (replacing
// electron-updater). Set WHALEISLE_UPDATE_FEED to a JSON document:
//
//	{
//	  "version":"1.2.3",
//	  "url":"https://…/installer.exe",
//	  "notes":"optional",
//	  "sha256":"…",
//	  "size":123,
//	  "blockMapURL":"https://…/installer.blockmap.json",
//	  "blockMap":{"fileSize":123,"blocks":[{"offset":0,"size":4096,"sha256":"…"}]}
//	}
//
// blockMapURL and blockMap are optional. The blockmap stub schema is a JSON
// object with blocks[] of {offset,size,sha256} (fileSize optional). See
// BlockMapInfo, ParseBlockMap, DownloadBlockMap, CanIncremental, VerifyBlockMap,
// FetchRange, ApplyBlockPatch, and ApplyIncremental.
//
// Without a feed URL, CheckForUpdates returns StatusNotConfigured (no network).
//
// DownloadAndStageOpts verifies optional sha256/size, then optional Authenticode
// on Windows (PowerShell Get-AuthenticodeSignature) via RequireSignature /
// ExpectedPublisher. On non-Windows, Authenticode probes return SignatureSkipped
// (documented no-op).
//
// Incremental/blockmap: CanIncremental is a heuristic. ApplyIncremental
// resolves the blockmap then, when IncrementalOptions.OldPath exists, runs
// ApplyBlockPatch (copy unchanged blocks by offset+sha256; HTTP Range for the
// rest; full GET if Range is unsupported). Otherwise it full-downloads then
// VerifyBlockMap (fail closed). When CanIncremental is false or no usable
// blockmap exists, callers should use DownloadAndStage / DownloadAndStageOpts.
package update

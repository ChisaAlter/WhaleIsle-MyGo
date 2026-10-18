# PR03: native update review and installation consent

The startup update notice uses the existing prepared native Versions catalog. It shows current/available versions, release notes, package variants, full/delta methods and published download sizes. Unavailable recipes are omitted. Notes exceeding 3000 Unicode characters disclose truncation and point to the Versions page.

The notice's affirmative action now opens Versions and selects the release. It does not download or install. The existing Versions page remains responsible for full/delta selection, downloading and verification; no updater was reimplemented.

After verification, the visible install action explains that it launches the verified installer, stops Host and quits the desktop app. Clicking that button is the installation confirmation, so the following duplicate generic confirmation is skipped only for this explicit action. Other callers retain their confirmation. Active-task/unknown-task warnings, resume saving, lifecycle admission, platform checks and last-moment staged-byte verification remain intact. macOS still opens the verified disk image for user installation.

Verification: synthetic catalog-summary fixture and Windows amd64 command package compilation. No real update was downloaded, installed or launched. No desktop runtime execution was performed.

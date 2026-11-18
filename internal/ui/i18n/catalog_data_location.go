package i18n

// Keys for the user data location: the Settings → 通用 row, the reasons a
// chosen directory is refused, the one-time legacy import offer, and the two
// sentences the shell needs when the stored locale is not readable yet.
//
// Every sentence names what happened and what to do next. Raw causes (OS
// errors, probe file names, transport errors) belong in the log, never here.
const (
	KeyDataLocationTitle              = "data_location.title"
	KeyDataLocationHint               = "data_location.hint"
	KeyDataLocationHintImport         = "data_location.hint.import"
	KeyDataLocationChange             = "data_location.change"
	KeyDataLocationOpen               = "data_location.open"
	KeyDataLocationReset              = "data_location.reset"
	KeyDataLocationCurrent            = "data_location.current"
	KeyDataLocationUnavailable        = "data_location.unavailable"
	KeyDataLocationRecordBroken       = "data_location.record_broken"
	KeyDataLocationApplying           = "data_location.applying"
	KeyDataLocationBusy               = "data_location.busy"
	KeyDataLocationRefusedEmpty       = "data_location.refused.empty"
	KeyDataLocationRefusedRelative    = "data_location.refused.relative"
	KeyDataLocationRefusedInvalid     = "data_location.refused.invalid"
	KeyDataLocationRefusedNotDir      = "data_location.refused.not_directory"
	KeyDataLocationRefusedNotWritable = "data_location.refused.not_writable"
	KeyDataLocationRefusedNested      = "data_location.refused.nested"
	KeyDataLocationRefusedInstall     = "data_location.refused.install"
	KeyDataLocationRefusedRemote      = "data_location.refused.remote"
	KeyDataLocationRefusedRecord      = "data_location.refused.record"
	KeyDataLocationRefusedGeneric     = "data_location.refused.generic"
	KeyDataLocationWriteFailed        = "data_location.write_failed"
	KeyDataLocationOpenFailed         = "data_location.open_failed"
	KeyDataLocationReadOnly           = "data_location.read_only"
	KeyDataLocationConfirmBody        = "data_location.confirm.body"
	KeyDataLocationConfirmAction      = "data_location.confirm.action"
	KeyDataLocationImport             = "data_location.import"
	KeyDataLocationImportPrevious     = "data_location.import.previous"
	KeyDataLocationImportRunning      = "data_location.import.running"
	KeyDataLocationImportSummary      = "data_location.import.summary"
	KeyDataLocationImportFailed       = "data_location.import.failed"
	KeyFirstRunImportTitle            = "first_run.import.title"
	KeyFirstRunImportBody             = "first_run.import.body"
	KeyFirstRunImportAccept           = "first_run.import.accept"
	KeyFirstRunImportSkip             = "first_run.import.skip"
	KeyFirstRunImportFailedTitle      = "first_run.import.failed_title"
	KeyFirstRunImportFailedBody       = "first_run.import.failed_body"
	KeyDataLocationFallbackTitle      = "data_location.fallback.title"
	KeyDataLocationFallbackBody       = "data_location.fallback.body"
	KeyDataLocationFallbackCorrupt    = "data_location.fallback.corrupt"
	KeyDataLocationFallbackQuit       = "data_location.fallback.quit"
	KeyDataLocationFallbackContinue   = "data_location.fallback.continue"
	KeyDataLocationProfileLocked      = "data_location.profile_locked"
	KeyConfigLoadFailedBody           = "config.load_failed.body"
	KeyProtectedActionFailed          = "protected_action.failed"
)

func init() {
	Register(KeyDataLocationTitle, "数据位置", "Data location")
	Register(KeyDataLocationHint,
		"重启后生效，不会删除原目录；新位置从空开始。",
		"Takes effect after a restart. The original directory is not deleted, and the new location starts empty.")
	Register(KeyDataLocationHintImport,
		"下面的导入按钮可以把原来的数据复制进来，复制不会移动或删除原数据。",
		"The import buttons below copy the existing data in; copying moves and deletes nothing.")
	Register(KeyDataLocationChange, "更改…", "Change…")
	Register(KeyDataLocationOpen, "打开目录", "Open folder")
	Register(KeyDataLocationReset, "恢复默认", "Restore default")
	Register(KeyDataLocationCurrent, "当前：", "Current: ")
	Register(KeyDataLocationUnavailable, "已记录的位置当前不可用：%s", "The recorded location is unavailable: %s")
	Register(KeyDataLocationRecordBroken,
		"位置记录无法读取：%s；已改用默认位置。",
		"The location record %s cannot be read; the default is in use.")
	Register(KeyDataLocationApplying,
		"数据位置已记录，正在重启应用以使用新位置。",
		"Data location recorded. Restarting to use it.")
	Register(KeyDataLocationBusy,
		"另一项受保护操作正在进行，请稍后重试。",
		"Another protected operation is running. Try again in a moment.")
	Register(KeyDataLocationRefusedEmpty, "请选择一个文件夹。", "Choose a folder.")
	Register(KeyDataLocationRefusedRelative, "请选择绝对路径：%s", "Choose an absolute path: %s")
	Register(KeyDataLocationRefusedInvalid,
		"无法解析该路径：%s。请重新选择。",
		"That path could not be resolved: %s. Choose it again.")
	Register(KeyDataLocationRefusedNotDir,
		"该路径已存在但不是文件夹：%s。请另选一个文件夹。",
		"That path already exists and is not a folder: %s. Choose another folder.")
	Register(KeyDataLocationRefusedNotWritable,
		"该位置无法写入：%s。请换一个文件夹或检查权限。",
		"That location is not writable: %s. Choose another folder or check permissions.")
	Register(KeyDataLocationRefusedNested,
		"新位置不能包含当前数据目录，也不能位于其中：%s 与 %s 重叠。请另选一个文件夹。",
		"The new location cannot contain the current data directory or sit inside it: %s overlaps %s. Choose another folder.")
	Register(KeyDataLocationRefusedInstall,
		"数据位置不能位于安装目录内：%s 与 %s 重叠，卸载或更新会动到用户数据。请另选一个文件夹。",
		"The data location cannot overlap the installation directory: %s overlaps %s, and an update or uninstall would touch user data there. Choose another folder.")
	Register(KeyDataLocationRefusedRemote,
		"数据位置不能放在网络驱动器上：%s。请选择本机文件夹。",
		"The data location cannot be on a network drive: %s. Choose a folder on this computer.")
	Register(KeyDataLocationRefusedRecord,
		"该位置会包含数据位置记录：%s。记录必须留在数据目录之外，请另选一个文件夹。",
		"That location would contain the data-location record: %s. The record has to stay outside the data directory; choose another folder.")
	Register(KeyDataLocationRefusedGeneric,
		"该位置不可用，请换一个文件夹后重试。",
		"That location cannot be used. Choose another folder and try again.")
	Register(KeyDataLocationWriteFailed,
		"无法保存数据位置，请检查目标文件夹的权限后重试。",
		"Could not save the data location. Check the target folder's permissions and try again.")
	Register(KeyDataLocationOpenFailed,
		"无法打开该文件夹，请检查路径是否仍然可用。",
		"Could not open that folder; check that the path still exists.")
	Register(KeyDataLocationReadOnly,
		"本次运行由隔离或便携模式指定了数据目录，不能更改或恢复默认。",
		"This run has its data directory set by isolated or portable mode; it cannot be changed or restored here.")
	Register(KeyDataLocationConfirmBody,
		"将从当前位置：\n%s\n切换到：\n%s\n\n新位置从空开始，原目录保留且不会被删除；重启后生效。",
		"Moves the data location from:\n%s\nto:\n%s\n\nThe new directory starts empty, the old one is kept and never deleted, and the change takes effect after a restart.")
	Register(KeyDataLocationConfirmAction, "更改并重启", "Change and restart")
	Register(KeyDataLocationImport, "从原版鲸屿导入…", "Import from Whale Isle…")
	Register(KeyDataLocationImportPrevious, "从上一个位置导入…", "Import from the previous location…")
	Register(KeyDataLocationImportRunning, "正在导入数据…", "Importing data…")
	Register(KeyDataLocationImportSummary,
		"已导入 %d 项，跳过 %d 项，失败 %d 项。原数据仍保留在原位置。",
		"Imported %d items, skipped %d, failed %d. The original data stays where it was.")
	Register(KeyDataLocationImportFailed,
		"导入未完成，请稍后重试；已复制的部分会保留。",
		"The import did not finish. Try again in a moment; anything already copied is kept.")
	Register(KeyFirstRunImportTitle, "从原版鲸屿导入数据", "Import data from Whale Isle")
	Register(KeyFirstRunImportBody,
		"检测到可导入的数据：%s\n\n将复制到 MyGo 自己的数据目录：%s\n\n复制不会移动或删除原数据，可以稍后在「设置 → 通用 → 数据位置」再次导入。",
		"Data that can be imported was found at: %s\n\nIt will be copied into MyGo's own data directory: %s\n\nCopying moves and deletes nothing, and you can import again later from Settings → General → Data location.")
	Register(KeyFirstRunImportAccept, "导入", "Import")
	Register(KeyFirstRunImportSkip, "跳过", "Skip")
	Register(KeyFirstRunImportFailedTitle, "导入未完成", "Import did not complete")
	Register(KeyFirstRunImportFailedBody,
		"已复制的部分保留，下次启动会继续导入；也可以稍后在「设置 → 通用 → 数据位置」再试。",
		"Copied items are kept; the next launch continues the import, and Settings → General → Data location can run it again.")
	Register(KeyDataLocationFallbackTitle, "已记录的数据位置不可用", "Recorded data location unavailable")
	Register(KeyDataLocationFallbackBody,
		"已记录的数据位置无法使用：\n%s\n\n本次启动改用默认位置：\n%s\n\n现在继续会在默认位置新建数据，原路径的数据不会被移动或删除；修好路径后重新启动即可回到原位置。",
		"The recorded data location cannot be used:\n%s\n\nThis launch falls back to the default:\n%s\n\nContinuing creates new data in the default location. Nothing at the recorded path is moved or deleted, and fixing the path returns the next launch to it.")
	Register(KeyDataLocationFallbackCorrupt,
		"位置记录无法读取，本次启动改用默认位置：\n%s\n\n原始记录未被删除，修好它可以回到原位置；也可以在用默认位置继续后到设置里重新选择。",
		"The location record cannot be read, so this launch falls back to the default:\n%s\n\nThe record itself is left in place; fixing it returns the next launch to it, and the location can also be chosen again in Settings.")
	Register(KeyDataLocationFallbackQuit, "退出并修复路径", "Quit and fix the path")
	Register(KeyDataLocationFallbackContinue, "以默认位置继续", "Continue with the default")
	Register(KeyDataLocationProfileLocked,
		"无法取得数据目录的独占访问。请先关闭其他鲸屿 MyGo 或鲸屿桌面实例，再重新打开。",
		"Exclusive access to the data directory is unavailable. Close the other Whale Isle MyGo or Whale Isle desktop instance and open this one again.")
	Register(KeyConfigLoadFailedBody,
		"配置文件 %s 无法读取，原文件已保留。请备份并修复该文件后重新启动应用。",
		"The configuration file %s cannot be read. The original file is preserved; back it up, repair it, then start the app again.")
	Register(KeyProtectedActionFailed,
		"操作没有完成，请稍后重试。",
		"The action did not finish. Try again in a moment.")
}

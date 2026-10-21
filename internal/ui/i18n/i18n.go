// Package i18n is a minimal locale catalog for shell chrome strings.
//
// Catalogs cover zh-CN (default) and en. Config stores "zh" or "en";
// Normalize maps those to catalog ids. Missing keys fall back to English,
// then to the key itself.
package i18n

import (
	"strings"
	"sync"
)

// Catalog locale ids.
const (
	LocaleZH      = "zh-CN"
	LocaleEN      = "en"
	DefaultLocale = LocaleZH
)

// Shell chrome keys seeded for M1.
const (
	KeyChatManualCompaction               = "chat.manual_compaction"
	KeyChatCompactionCounts               = "chat.compaction_counts"
	KeyChatCompactionSummaryMissing       = "chat.compaction_summary_missing"
	KeyCommandTitle                       = "command.title"
	KeyCommandRunning                     = "command.running"
	KeyCommandDone                        = "command.done"
	KeyCommandFailed                      = "command.failed"
	KeyCommandCatalogLoading              = "command.catalog_loading"
	KeyCommandCatalogRetry                = "command.catalog_retry"
	KeyAppName                            = "app.name"
	KeyTrayOpen                           = "tray.open"
	KeyTrayQuit                           = "tray.quit"
	KeyQuitConfirmTitle                   = "quit_confirm.title"
	KeyQuitConfirmMessage                 = "quit_confirm.message"
	KeyQuitConfirmQuit                    = "quit_confirm.quit"
	KeyQuitConfirmCancel                  = "quit_confirm.cancel"
	KeyTitlebarClose                      = "titlebar.close"
	KeyTitlebarMinimize                   = "titlebar.minimize"
	KeyTitlebarMaximize                   = "titlebar.maximize"
	KeyStatusPlaceholder                  = "status.placeholder"
	KeyBootTitle                          = "boot.title"
	KeyBootStarting                       = "boot.starting"
	KeyBootReady                          = "boot.ready"
	KeyBootFailed                         = "boot.failed"
	KeyBootStopped                        = "boot.stopped"
	KeyBootRetry                          = "boot.retry"
	KeyBootExport                         = "boot.export"
	KeyBootWaiting                        = "boot.waiting"
	KeyBootPlugins                        = "boot.plugins"
	KeyBootPluginsDetail                  = "boot.plugins_detail"
	KeyBootPluginsIndeterminate           = "boot.plugins_indeterminate"
	KeySidebarSessions                    = "sidebar.sessions"
	KeySidebarRefresh                     = "sidebar.refresh"
	KeySidebarNew                         = "sidebar.new"
	KeySidebarLoading                     = "sidebar.loading"
	KeySidebarEmpty                       = "sidebar.empty"
	KeySidebarRename                      = "sidebar.rename"
	KeySidebarRenameCancel                = "sidebar.rename_cancel"
	KeySidebarSearch                      = "sidebar.search"
	KeySidebarSearchLoading               = "sidebar.search_loading"
	KeySidebarArchive                     = "sidebar.archive"
	KeySidebarUnarchive                   = "sidebar.unarchive"
	KeySidebarArchivedSection             = "sidebar.archived_section"
	KeySidebarBrowseWorkspace             = "sidebar.browse_workspace"
	KeySidebarRevealWorkspace             = "sidebar.reveal_workspace"
	KeySidebarRenameWorkspace             = "sidebar.rename_workspace"
	KeySidebarRenameWorkspaceSave         = "sidebar.rename_workspace_save"
	KeySidebarDeleteWorkspace             = "sidebar.delete_workspace"
	KeySidebarMoveUp                      = "sidebar.move_up"
	KeySidebarDelete                      = "sidebar.delete"
	KeySidebarCopyID                      = "sidebar.copy_id"
	KeySidebarFork                        = "sidebar.fork"
	KeyWorkspaceRegistered                = "workspace.registered"
	KeySidebarDeleteConfirm               = "sidebar.delete_confirm"
	KeySidebarPin                         = "sidebar.pin"
	KeySidebarUnpin                       = "sidebar.unpin"
	KeySidebarPinnedPrefix                = "sidebar.pinned_prefix"
	KeySidebarBrand                       = "sidebar.brand"
	KeySidebarMarket                      = "sidebar.market"
	KeySidebarSchedule                    = "sidebar.schedule"
	KeySidebarAssistant                   = "sidebar.assistant"
	KeySidebarWorkspaceSection            = "sidebar.workspace_section"
	KeySidebarNewSessionFallback          = "sidebar.new_session_fallback"
	KeyChatEmpty                          = "chat.empty"
	KeyChatHeroSlogan                     = "chat.hero_slogan"
	KeyChatHeroPreview                    = "chat.hero_preview"
	KeyChatHeroChooseWorkspace            = "chat.hero_choose_workspace"
	KeyChatThinking                       = "chat.thinking"
	KeyChatFeedbackUp                     = "chat.feedback_up"
	KeyChatFeedbackDown                   = "chat.feedback_down"
	KeyChatFeedbackSelectedMark           = "chat.feedback_selected_mark"
	KeyChatFeedbackCommentPlaceholder     = "chat.feedback_comment_placeholder"
	KeyChatFeedbackSend                   = "chat.feedback_send"
	KeyChatConnConnecting                 = "chat.conn_connecting"
	KeyChatConnReconnecting               = "chat.conn_reconnecting"
	KeyChatUsageJobsToday                 = "chat.usage_jobs_today"
	KeyChatUsageBalance                   = "chat.usage_balance"
	KeyComposerModel                      = "composer.model"
	KeyComposerPermission                 = "composer.permission"
	KeyComposerPermissionDefault          = "composer.permission_default"
	KeyComposerAgentPreset                = "composer.agent_preset"
	KeyComposerSend                       = "composer.send"
	KeyComposerStop                       = "composer.stop"
	KeyComposerModeQueue                  = "composer.mode_queue"
	KeyComposerModeSteer                  = "composer.mode_steer"
	KeyComposerPlaceholder                = "composer.placeholder"
	KeyComposerPlaceholderHero            = "composer.placeholder_hero"
	KeyComposerNoSession                  = "composer.no_session"
	KeyComposerWaiting                    = "composer.waiting"
	KeyComposerDropHint                   = "composer.drop_hint"
	KeyComposerSkills                     = "composer.skills"
	KeyComposerSkillUserOnly              = "composer.skill_user_only"
	KeyComposerSkillsFilter               = "composer.skills_filter"
	KeyComposerSkillsHubSearch            = "composer.skills_hub_search"
	KeyComposerSkillsHubInstall           = "composer.skills_hub_install"
	KeyComposerSkillsDelete               = "composer.skills_delete"
	KeyComposerSkillsCreate               = "composer.skills_create"
	KeyComposerSkillsCreateName           = "composer.skills_create_name"
	KeyComposerSkillsToggleModel          = "composer.skills_toggle_model"
	KeyComposerSkillsEmpty                = "composer.skills_empty"
	KeyComposerPickDir                    = "composer.pick_dir"
	KeyComposerPresetDefault              = "composer.preset_default"
	KeyComposerPickDirShort               = "composer.pick_dir_short"
	KeyComposerPickImage                  = "composer.pick_image"
	KeyComposerSpeech                     = "composer.speech"
	KeyComposerSpeechCycle                = "composer.speech_cycle"
	KeyComposerDirDialogTitle             = "composer.dir_dialog_title"
	KeyComposerDirOutside                 = "composer.dir_outside"
	KeyComposerDirNoWorkspace             = "composer.dir_no_workspace"
	KeyComposerChipEdit                   = "composer.chip_edit"
	KeyComposerChipEditPlaceholder        = "composer.chip_edit_placeholder"
	KeyComposerChipEditApply              = "composer.chip_edit_apply"
	KeyComposerChipEditCancel             = "composer.chip_edit_cancel"
	KeyFilesTitle                         = "files.title"
	KeyFilesNoWorkspace                   = "files.no_workspace"
	KeyFilesPickFile                      = "files.pick_file"
	KeyFilesTooLarge                      = "files.too_large"
	KeyFilesBinary                        = "files.binary"
	KeyFilesShowDiff                      = "files.show_diff"
	KeyFilesEdit                          = "files.edit"
	KeyFilesSave                          = "files.save"
	KeyFilesCancelEdit                    = "files.cancel_edit"
	KeyFilesSaved                         = "files.saved"
	KeyFilesModeSplit                     = "files.mode_split"
	KeyFilesModeEdit                      = "files.mode_edit"
	KeyFilesModeHighlight                 = "files.mode_highlight"
	KeyFilesModeNext                      = "files.mode_next"
	KeyGitPRView                          = "git.pr_view"
	KeyGitPRCreate                        = "git.pr_create"
	KeyGitPRPanel                         = "git.pr_panel"
	KeyGitPRRefresh                       = "git.pr_refresh"
	KeyGitPRClose                         = "git.pr_close"
	KeyGitPRCheckout                      = "git.pr_checkout"
	KeyGitPRCheckoutOK                    = "git.pr_checkout_ok"
	KeyGitPRNoGH                          = "git.pr_no_gh"
	KeyGitPRNone                          = "git.pr_none"
	KeyRightPaneFiles                     = "frame.right_pane_files"
	KeyRightPaneDiff                      = "frame.right_pane_diff"
	KeyRightPaneAgents                    = "frame.right_pane_agents"
	KeyFrameCaptionAgents                 = "frame.caption_agents"
	KeyFrameCaptionProjectDocs            = "frame.caption_project_docs"
	KeyDiffTitle                          = "diff.title"
	KeyDiffRefresh                        = "diff.refresh"
	KeyDiffHint                           = "diff.hint"
	KeyDiffEmpty                          = "diff.empty"
	KeyDiffNoWorkspace                    = "diff.no_workspace"
	KeyDiffNotRepo                        = "diff.not_repo"
	KeyInteractionAllow                   = "interaction.allow"
	KeyInteractionAllowAlways             = "interaction.allow_always"
	KeyInteractionDeny                    = "interaction.deny"
	KeyInteractionCallID                  = "interaction.call_id"
	KeyInteractionCopyArgs                = "interaction.copy_args"
	KeyInteractionContinue                = "interaction.continue"
	KeyInteractionManual                  = "interaction.manual"
	KeyInteractionAutoAllowOn             = "interaction.auto_allow_on"
	KeyInteractionAutoAllowed             = "interaction.auto_allowed"
	KeyBootExportSaved                    = "boot.export_saved"
	KeySettingsOpen                       = "settings.open"
	KeyDesktopTools                       = "desktop.tools"
	KeySettingsTitle                      = "settings.title"
	KeySettingsBack                       = "settings.back"
	KeySettingsNamespaces                 = "settings.namespaces"
	KeySettingsEmpty                      = "settings.empty"
	KeySettingsRefresh                    = "settings.refresh"
	KeySettingsLoading                    = "settings.loading"
	KeySettingsSave                       = "settings.save"
	KeySettingsReplace                    = "settings.replace"
	KeySettingsReplaceHint                = "settings.replace_hint"
	KeySettingsReplaceConfirmTitle        = "settings.replace_confirm_title"
	KeySettingsReplaceConfirmMessage      = "settings.replace_confirm_message"
	KeySettingsReplaceCancel              = "settings.replace_cancel"
	KeySettingsReadOnly                   = "settings.read_only"
	KeySettingsWritable                   = "settings.writable"
	KeySettingsRevision                   = "settings.revision"
	KeySettingsFilter                     = "settings.filter"
	KeySettingsFilterEmpty                = "settings.filter_empty"
	KeySettingsRevealSecrets              = "settings.reveal_secrets"
	KeySettingsPickNS                     = "settings.pick_namespace"
	KeySettingsDesktopNav                 = "settings.desktop.nav"
	KeySettingsDesktopTitle               = "settings.desktop.title"
	KeySettingsDesktopHint                = "settings.desktop.hint"
	KeySettingsDesktopWorkspace           = "settings.desktop.workspace"
	KeySettingsDesktopBrowseWorkspace     = "settings.desktop.browse_workspace"
	KeySettingsDesktopPort                = "settings.desktop.port"
	KeySettingsDesktopTheme               = "settings.desktop.theme"
	KeySettingsDesktopLocale              = "settings.desktop.locale"
	KeySettingsDesktopCloseToTray         = "settings.desktop.close_to_tray"
	KeySettingsDesktopOpenAtLogin         = "settings.desktop.open_at_login"
	KeySettingsDesktopSaved               = "settings.desktop.saved"
	KeySettingsDesktopDetectElectron      = "settings.desktop.detect_electron"
	KeySettingsDesktopCheckUpdates        = "settings.desktop.check_updates"
	KeySettingsDesktopDetectElectronTitle = "settings.desktop.detect_electron.title"
	KeySettingsDesktopCheckUpdatesTitle   = "settings.desktop.check_updates.title"
	KeySettingsCredNav                    = "settings.cred.nav"
	KeySettingsCredTitle                  = "settings.cred.title"
	KeySettingsCredHint                   = "settings.cred.hint"
	KeySettingsCredShow                   = "settings.cred.show"
	KeySettingsCredAPIKey                 = "settings.cred.api_key"
	KeySettingsCredClearAPIKey            = "settings.cred.clear_api_key"
	KeySettingsCredClearSecrets           = "settings.cred.clear_secrets"
	KeySettingsCredBaseURL                = "settings.cred.base_url"
	KeySettingsCredGithub                 = "settings.cred.github"
	KeySettingsCredRemote                 = "settings.cred.remote"
	KeySettingsCredRelay                  = "settings.cred.relay"
	KeySettingsCredSaved                  = "settings.cred.saved"
	KeySettingsDesktopWallpaper           = "settings.desktop.wallpaper"
	KeySettingsDesktopWallpaperNone       = "settings.desktop.wallpaper_none"
	KeyLauncherOpen                       = "launcher.open"
	KeyLauncherTitle                      = "launcher.title"
	KeyLauncherClose                      = "launcher.close"
	KeyLauncherHint                       = "launcher.hint"
	KeyLauncherVersion                    = "launcher.version"
	KeyLauncherVendor                     = "launcher.vendor"
	KeyLauncherOn                         = "launcher.on"
	KeyLauncherOff                        = "launcher.off"
	KeyLauncherWorkspace                  = "launcher.workspace"
	KeyLauncherWorkspaceHint              = "launcher.workspace_hint"
	KeyLauncherNoWorkspace                = "launcher.no_workspace"
	KeyLauncherStart                      = "launcher.start"
	KeyLauncherScanImport                 = "launcher.scan_import"
	KeyLauncherCheckUpdate                = "launcher.check_update"
	KeyLauncherOpenSettings               = "launcher.open_settings"
	KeyLauncherImportTitle                = "launcher.import_title"
	KeyLauncherUpdateTitle                = "launcher.update_title"
	KeyLauncherPluginsTitle               = "launcher.plugins_title"
	KeyLauncherPluginsNone                = "launcher.plugins_none"
	KeyLauncherPluginRecoveryTitle        = "launcher.plugin_recovery_title"
	KeyLauncherSkipUserPlugins            = "launcher.skip_user_plugins"
	KeyLauncherSkipUserPluginsOn          = "launcher.skip_user_plugins_on"
	KeyLauncherSkipUserPluginsOff         = "launcher.skip_user_plugins_off"
	KeyLauncherEnableSkipUserPlugins      = "launcher.enable_skip_user_plugins"
	KeyLauncherClearPluginRecovery        = "launcher.clear_plugin_recovery"
	KeyLauncherRestartHarness             = "launcher.restart_harness"
	KeyLauncherBrandRole                  = "launcher.brand_role"
	KeyLauncherTabHome                    = "launcher.tab.home"
	KeyLauncherTabComponents              = "launcher.tab.components"
	KeyLauncherTabVersions                = "launcher.tab.versions"
	KeyLauncherTabImport                  = "launcher.tab.import"
	KeyLauncherTabPlugins                 = "launcher.tab.plugins"
	KeyLauncherTabSettings                = "launcher.tab.settings"
	KeyLauncherHomeLede                   = "launcher.home.lede"
	KeyLauncherGotoVersions               = "launcher.goto_versions"
	KeyLauncherDesktopTitle               = "launcher.desktop.title"
	KeyLauncherDesktopDesc                = "launcher.desktop.desc"
	KeyLauncherDesktopRunning             = "launcher.desktop.running"
	KeyLauncherDesktopStopped             = "launcher.desktop.stopped"
	KeyLauncherStartDesktop               = "launcher.start_desktop"
	KeyLauncherOpenDesktop                = "launcher.open_desktop"
	KeyLauncherCloseDesktop               = "launcher.close_desktop"
	KeyLauncherRunningNote                = "launcher.running_note"
	KeyLauncherMyComponents               = "launcher.my_components"
	KeyLauncherMyComponentsLede           = "launcher.my_components_lede"
	KeyLauncherSettingsOnlyHint           = "launcher.settings_only_hint"
	KeyLauncherOptRouteTitle              = "launcher.opt.route_title"
	KeyLauncherOptRouteDesc               = "launcher.opt.route_desc"
	KeyLauncherOptQuitTitle               = "launcher.opt.quit_title"
	KeyLauncherOptQuitDesc                = "launcher.opt.quit_desc"
	KeyLauncherOptTrayTitle               = "launcher.opt.tray_title"
	KeyLauncherOptTrayDesc                = "launcher.opt.tray_desc"
	KeyLauncherOptAutoTitle               = "launcher.opt.auto_title"
	KeyLauncherOptAutoDesc                = "launcher.opt.auto_desc"
	KeyLauncherOptAskTitle                = "launcher.opt.ask_title"
	KeyLauncherOptAskDesc                 = "launcher.opt.ask_desc"
	KeyLauncherRouteGitHub                = "launcher.route.github"
	KeyLauncherRouteCNB                   = "launcher.route.cnb"
	KeyLauncherRouteGitHubDetail          = "launcher.route.github_detail"
	KeyLauncherRouteCNBDetail             = "launcher.route.cnb_detail"
	KeyLauncherRetryStart                 = "launcher.retry_start"
	KeyLauncherStarting                   = "launcher.starting"
	KeyLauncherBadgeInstalled             = "launcher.badge.installed"
	KeyLauncherBadgeMissing               = "launcher.badge.missing"
	KeyLauncherDiagTitle                  = "launcher.diag.title"
	KeyLauncherRouteLine                  = "launcher.route_line"
	KeyLauncherRouteLabel                 = "launcher.route_label"
	KeyLauncherRouteCurrentPrefix         = "launcher.route_current_prefix"
	KeyLauncherComponentsLede             = "launcher.components.lede"
	KeyLauncherComponentsEmpty            = "launcher.components.empty"
	KeyLauncherLocalVersion               = "launcher.local_version"
	KeyLauncherImportLede                 = "launcher.import.lede"
	KeyLauncherImportEmpty                = "launcher.import.empty"
	KeyLauncherImportPickDir              = "launcher.import.pick_dir"
	KeyLauncherImportAddSkillDir          = "launcher.import.add_skill_dir"
	KeyLauncherImportRescan               = "launcher.import.rescan"
	KeyLauncherImportRescanFirst          = "launcher.import.rescan_first"
	KeyLauncherImportByDir                = "launcher.import.by_dir"
	KeyLauncherImportSelectAll            = "launcher.import.select_all"
	KeyLauncherImportSelectNone           = "launcher.import.select_none"
	KeyLauncherImportCatSessions          = "launcher.import.cat.sessions"
	KeyLauncherImportCatSkills            = "launcher.import.cat.skills"
	KeyLauncherImportCatPlugins           = "launcher.import.cat.plugins"
	KeyLauncherImportCatMCP               = "launcher.import.cat.mcp"
	KeyLauncherImportCatSettings          = "launcher.import.cat.settings"
	KeyLauncherImportCatPresets           = "launcher.import.cat.presets"
	KeyLauncherImportSummary              = "launcher.import.summary"
	KeyLauncherImportSummarySkillDir      = "launcher.import.summary_skill_dir"
	KeyLauncherImportPickSourceTitle      = "launcher.import.pick_source_title"
	KeyLauncherImportAddSkillTitle        = "launcher.import.add_skill_title"
	KeyLauncherImportOverwrite            = "launcher.import.overwrite"
	KeyLauncherImportOverwriteDesc        = "launcher.import.overwrite_desc"
	KeyLauncherImportAttachments          = "launcher.import.attachments"
	KeyLauncherImportAttachmentsDesc      = "launcher.import.attachments_desc"
	KeyLauncherImportSelected             = "launcher.import.selected"
	KeyLauncherImportPluginsNote          = "launcher.import.plugins_note"
	KeyLauncherImportChecklist            = "launcher.import.checklist"
	KeyLauncherUpdateAvailable            = "launcher.update.available"
	KeyLauncherUpdateNow                  = "launcher.update.now"
	KeyLauncherViewVersions               = "launcher.view_versions"
	KeyLauncherPluginFailTitle            = "launcher.plugin_fail_title"
	KeyLauncherPluginFailDisable          = "launcher.plugin_fail_disable"
	KeyLauncherPluginFailOpen             = "launcher.plugin_fail_open"
	KeyLauncherInstallDesktopTitle        = "launcher.install_desktop_title"
	KeyLauncherInstallDesktopDesc         = "launcher.install_desktop_desc"
	KeyLauncherInstallDesktopCTA          = "launcher.install_desktop_cta"
	KeyLauncherInstallDesktopVerify       = "launcher.install_desktop_verify"
	KeyLauncherManageComponents           = "launcher.manage_components"
	KeyLauncherCompStart                  = "launcher.comp.start"
	KeyLauncherCompStartNamed             = "launcher.comp.start_named"
	KeyLauncherCompRetry                  = "launcher.comp.retry"
	KeyLauncherCompInstall                = "launcher.comp.install"
	KeyLauncherCompStop                   = "launcher.comp.stop"
	KeyLauncherCompManage                 = "launcher.comp.manage"
	KeyLauncherCompOpenSettings           = "launcher.comp.open_settings"
	KeyLauncherCompOfficial               = "launcher.comp.official"
	KeyLauncherCompUpdatable              = "launcher.comp.updatable"
	KeyLauncherCompUpdateAvail            = "launcher.comp.update_avail"
	KeyLauncherCompKindService            = "launcher.comp.kind.service"
	KeyLauncherCompKindTool               = "launcher.comp.kind.tool"
	KeyLauncherCompStateRunning           = "launcher.comp.state.running"
	KeyLauncherCompStateStopped           = "launcher.comp.state.stopped"
	KeyLauncherCompStateAvailable         = "launcher.comp.state.available"
	KeyLauncherCompStateError             = "launcher.comp.state.error"
	KeyLauncherVerDev                     = "launcher.ver.dev"
	KeyLauncherVerNeedInstall             = "launcher.ver.need_install"
	KeyLauncherVerNeedInstallDesc         = "launcher.ver.need_install_desc"
	KeyLauncherCompCheckUpdate            = "launcher.comp.check_update"
	KeyLauncherCompRollback               = "launcher.comp.rollback"
	KeyLauncherVerBackHome                = "launcher.ver.back_home"
	KeyLauncherVerInstallPath             = "launcher.ver.install_path"
	KeyLauncherVerUpdateTo                = "launcher.ver.update_to"
	KeyLauncherVerAllList                 = "launcher.ver.all_list"
	KeyLauncherVerRefresh                 = "launcher.ver.refresh"
	KeyLauncherVerCurrent                 = "launcher.ver.current"
	KeyLauncherVerLatest                  = "launcher.ver.latest"
	KeyLauncherVerInstalled               = "launcher.ver.installed"
	KeyLauncherVerUpdateToV               = "launcher.ver.update_to_v"
	KeyLauncherVerEmpty                   = "launcher.ver.empty"
	KeyLauncherImportEmptyItems           = "launcher.import.empty_items"
	KeyLauncherImportEmptySessions        = "launcher.import.empty_sessions"
	KeyLauncherImportEmptySkills          = "launcher.import.empty_skills"
	KeyLauncherImportEmptyPlugins         = "launcher.import.empty_plugins"
	KeyLauncherImportEmptyMCP             = "launcher.import.empty_mcp"
	KeyLauncherImportEmptySettings        = "launcher.import.empty_settings"
	KeyLauncherImportEmptyPresets         = "launcher.import.empty_presets"
	KeyLauncherImportConflict             = "launcher.import.conflict"
	KeyLauncherImportUnsupported          = "launcher.import.unsupported"
	KeyLauncherImportStart                = "launcher.import.start"
	KeyLauncherImportLog                  = "launcher.import.log"
	KeyLauncherImportCopy                 = "launcher.import.copy"
	KeyLauncherPluginsNeedRuntime         = "launcher.plugins.need_runtime"
	KeyLauncherPluginsNeedRuntimeDesc     = "launcher.plugins.need_runtime_desc"
	KeyLauncherPluginsEmpty               = "launcher.plugins.empty"
	KeyLauncherPluginsUnknownHint         = "launcher.plugins.unknown_hint"
	KeyLauncherSettingsLede               = "launcher.settings.lede"
	KeyTrajSearch                         = "traj.search"
	KeyTrajDuration                       = "traj.duration"
	KeyTrajTurns                          = "traj.turns"
	KeyTrajCalls                          = "traj.calls"
	KeyTrajTimeline                       = "traj.timeline"
	KeyTrajColInput                       = "traj.col.input"
	KeyTrajColOutput                      = "traj.col.output"
	KeyTrajColThink                       = "traj.col.think"
	KeyTrajColTime                        = "traj.col.time"
	KeyTrajColModel                       = "traj.col.model"
	KeyTrajColTools                       = "traj.col.tools"
	KeyTrajLoadEarlier                    = "traj.load_earlier"
	KeyTrajLoading                        = "traj.loading"
	KeyTrajEmpty                          = "traj.empty"
	KeyTrajDetailTitle                    = "traj.detail_title"
	KeyTrajDetailHint                     = "traj.detail_hint"
	KeyTrajTabOverview                    = "traj.tab.overview"
	KeyTrajTabArgs                        = "traj.tab.args"
	KeyTrajTabResult                      = "traj.tab.result"
	KeyTrajTabTiming                      = "traj.tab.timing"
	KeyTrajNoArgs                         = "traj.no_args"
	KeyTrajNoResult                       = "traj.no_result"
	KeyTrajTimingLine                     = "traj.timing_line"
	KeyTrajNoTiming                       = "traj.no_timing"
	KeyTrajCopy                           = "traj.copy"
	KeyTrajToolResult                     = "traj.tool_result"
	KeyTrajToolCallOnly                   = "traj.tool_call_only"
	KeyTrajCompacted                      = "traj.compacted"
	KeyTrajDeliverables                   = "traj.deliverables"
	KeyTrajRoleUser                       = "traj.role.user"
	KeyTrajRoleAssistant                  = "traj.role.assistant"
	KeyTrajRoleSystem                     = "traj.role.system"
	KeyTrajInProgress                     = "traj.in_progress"
	KeyTrajKindTool                       = "traj.kind.tool"
	KeyTrajKindThink                      = "traj.kind.think"
	KeyTrajKindMessage                    = "traj.kind.message"
	KeyTrajTurnN                          = "traj.turn_n"
	KeySettingsSectionAccount             = "settings.section.account"
	KeySettingsSectionGeneral             = "settings.section.general"
	KeySettingsSectionAppearance          = "settings.section.appearance"
	KeySettingsSectionInterface           = "settings.section.interface"
	KeySettingsSectionModels              = "settings.section.models"
	KeySettingsSectionPlugins             = "settings.section.plugins"
	KeySettingsSectionSkills              = "settings.section.skills"
	KeySettingsSectionMarket              = "settings.section.market"
	KeySettingsSectionRemote              = "settings.section.remote"
	KeySettingsSectionMCP                 = "settings.section.mcp"
	KeySettingsSectionAgentPresets        = "settings.section.agent_presets"
	KeySettingsSectionPet                 = "settings.section.pet"
	KeySettingsSectionAbout               = "settings.section.about"
	KeySettingsSectionUsageStats          = "settings.section.usage_stats"
	KeySettingsSectionRemoteWorkspace     = "settings.section.remote_workspace"
	KeySettingsSectionOther               = "settings.section.other"
	KeyUsageOpen                          = "usage.open"
	KeyUsageTitle                         = "usage.title"
	KeyUsageBack                          = "usage.back"
	KeyUsageRefresh                       = "usage.refresh"
	KeyUsageLoading                       = "usage.loading"
	KeyUsagePluginNote                    = "usage.plugin_note"
	KeyUsageAccount                       = "usage.account"
	KeyUsageBalance                       = "usage.balance"
	KeyUsageRecent                        = "usage.recent"
	KeyUsageRecentEmpty                   = "usage.recent_empty"
	KeyUsageNoData                        = "usage.no_data"
	KeyUsageWebLink                       = "usage.web_link"
	KeyUsageChartTitle                    = "usage.chart_title"
	KeyUsageChartNote                     = "usage.chart_note"
	KeyUsageHeatmapTitle                  = "usage.heatmap_title"
	KeyUsageHeatmapNote                   = "usage.heatmap_note"
	KeyUsageOpenWeb                       = "usage.open_web"
	KeyUsageOpenTopUp                     = "usage.open_topup"
	KeyUsageCopyURL                       = "usage.copy_url"
	KeyUsageCopyTopUp                     = "usage.copy_topup"
	KeyUsageCopyBalance                   = "usage.copy_balance"
	KeyUsageKillJob                       = "usage.kill_job"
	KeyUsageFollowJob                     = "usage.follow_job"
	KeyUsageExpiry                        = "usage.expiry"
	KeySettingsAccountBonuses             = "settings.account.bonuses"
	KeySettingsAccountBonusesEmpty        = "settings.account.bonuses_empty"
	KeySettingsAccountBonusAck            = "settings.account.bonus_ack"
	KeySettingsAccountRunningTasks        = "settings.account.running_tasks"
	KeySettingsAboutAnalytics             = "settings.about.analytics"
	KeySettingsAboutAnalyticsOn           = "settings.about.analytics_on"
	KeySettingsAboutAnalyticsOff          = "settings.about.analytics_off"
	KeyDirPickTitle                       = "dirpick.title"
	KeyDirPickClose                       = "dirpick.close"
	KeyDirPickEmpty                       = "dirpick.empty"
	KeyDirPickSelect                      = "dirpick.select"
	KeyDirPickSelectHere                  = "dirpick.select_here"
	KeyDirPickNewName                     = "dirpick.new_name"
	KeyDirPickCreate                      = "dirpick.create"
	KeyTerminalTitle                      = "terminal.title"
	KeyTerminalShow                       = "terminal.show"
	KeyTerminalHide                       = "terminal.hide"
	KeyTerminalUnavailable                = "terminal.unavailable"
	KeyTerminalRestart                    = "terminal.restart"
	KeyTerminalCopyCwd                    = "terminal.copy_cwd"
	KeyGitFetch                           = "git.fetch"
	KeyGitPull                            = "git.pull"
	KeyGitPush                            = "git.push"
	KeyGitCommit                          = "git.commit"
	KeyGitCommitOK                        = "git.commit_ok"
	KeyGitFetchOK                         = "git.fetch_ok"
	KeyGitPullOK                          = "git.pull_ok"
	KeyGitPushOK                          = "git.push_ok"
	KeyGitActionFailed                    = "git.action_failed"
	KeyGitCommitPlaceholder               = "git.commit_placeholder"
	KeyGitCommitConfirmTitle              = "git.commit_confirm_title"
	KeyGitCommitConfirmMessage            = "git.commit_confirm_message"
	KeyGitCommitCancel                    = "git.commit_cancel"
	KeyGitCommitNotDirty                  = "git.commit_not_dirty"
	KeyWebViewOpen                        = "webview.open"
	KeyWebViewTitle                       = "webview.title"
	KeyWebViewBack                        = "webview.back"
	KeyWebViewNotOpen                     = "webview.not_open"
	KeyWebViewCopyURL                     = "webview.copy_url"
	KeyWebViewOpenBrowser                 = "webview.open_browser"
	KeyWebViewOpenHost                    = "webview.open_host"
	KeyWebViewCloseHost                   = "webview.close_host"
	KeyWebViewOrigin                      = "webview.origin"
	KeyBrowserPreviewOpen                 = "browserpreview.open"
	KeyBrowserPreviewTitle                = "browserpreview.title"
	KeyBrowserPreviewBack                 = "browserpreview.back"
	KeyBrowserPreviewURLPlaceholder       = "browserpreview.url_placeholder"
	KeyBrowserPreviewGo                   = "browserpreview.go"
	KeyBrowserPreviewOpenExternal         = "browserpreview.open_external"
	KeyBrowserPreviewCopyURL              = "browserpreview.copy_url"
	KeyBrowserPreviewReload               = "browserpreview.reload"
	KeyBrowserPreviewCloseHost            = "browserpreview.close_host"
	KeyBrowserPreviewHint                 = "browserpreview.hint"
	KeyBrowserPreviewInvalidURL           = "browserpreview.invalid_url"
	KeyChatCitations                      = "chat.citations"
	KeyRemoteOpen                         = "remote.open"
	KeyRemoteTitle                        = "remote.title"
	KeyRemoteBack                         = "remote.back"
	KeyRemoteWorkspaceToggle              = "remote.workspace_toggle"
	KeyRemoteToggleOn                     = "remote.toggle_on"
	KeyRemoteToggleOff                    = "remote.toggle_off"
	KeyRemoteHarnessOverlay               = "remote.harness_overlay"
	KeyRemotePairingEnabled               = "remote.pairing_enabled"
	KeyRemotePhoneGit                     = "remote.phone_git"
	KeyRemoteMode                         = "remote.mode"
	KeyRemotePort                         = "remote.port"
	KeyRemoteMux                          = "remote.mux"
	KeyRemoteCredentials                  = "remote.credentials"
	KeyRemoteOn                           = "remote.on"
	KeyRemoteOff                          = "remote.off"
	KeyRemoteMuxWaiting                   = "remote.mux_waiting"
	KeyRemoteMuxConnected                 = "remote.mux_connected"
	KeyRemoteMuxDisconnected              = "remote.mux_disconnected"
	KeyRemoteCredPresent                  = "remote.cred_present"
	KeyRemoteCredAbsent                   = "remote.cred_absent"
	KeyRemoteWorkspaceOff                 = "remote.workspace_off"
	KeyRemoteSummaryDisabled              = "remote.summary_disabled"
	KeyRemoteSummaryEnabled               = "remote.summary_enabled"
	KeyRemoteSSHTitle                     = "remote.ssh_title"
	KeyRemoteSSHStatus                    = "remote.ssh_status"
	KeyRemoteSSHSummaryReady              = "remote.ssh_summary_ready"
	KeyRemoteSSHSummaryIncomplete         = "remote.ssh_summary_incomplete"
	KeyRemoteSSHSummaryDisabled           = "remote.ssh_summary_disabled"
	KeyRemoteSSHSummaryInactive           = "remote.ssh_summary_inactive"
	KeyRemoteSSHStatusOff                 = "remote.ssh_status_off"
	KeyRemoteSSHReady                     = "remote.ssh_ready"
	KeyRemoteSSHIncomplete                = "remote.ssh_incomplete"
	KeyRemoteSSHDisabled                  = "remote.ssh_disabled"
	KeyRemoteSSHMissing                   = "remote.ssh_missing"
	KeyRemoteSSHBind                      = "remote.ssh_bind"
	KeyRemoteSSHLanTLS                    = "remote.ssh_lan_tls"
	KeyRemoteSSHRelayToken                = "remote.ssh_relay_token"
	KeyRemoteSSHPairingOff                = "remote.ssh_pairing_off"
	KeyRemoteSSHProbe                     = "remote.ssh_probe"
	KeyRemoteSSHProbeBtn                  = "remote.ssh_probe_btn"
	KeyRemoteSSHTunnelTitle               = "remote.ssh_tunnel_title"
	KeyRemoteSSHTunnelNote                = "remote.ssh_tunnel_note"
	KeyRemoteSSHTunnelState               = "remote.ssh_tunnel_state"
	KeyRemoteSSHFieldHost                 = "remote.ssh_field_host"
	KeyRemoteSSHFieldUser                 = "remote.ssh_field_user"
	KeyRemoteSSHFieldPort                 = "remote.ssh_field_port"
	KeyRemoteSSHFieldIdentity             = "remote.ssh_field_identity"
	KeyRemoteSSHFieldForward              = "remote.ssh_field_forward"
	KeyRemoteSSHTunnelSave                = "remote.ssh_tunnel_save"
	KeyRemoteSSHTunnelConnect             = "remote.ssh_tunnel_connect"
	KeyRemoteSSHTunnelDisconnect          = "remote.ssh_tunnel_disconnect"
	KeyRemoteSSHHostUnknownTitle          = "remote.ssh_host_unknown_title"
	KeyRemoteSSHHostUnknownMessage        = "remote.ssh_host_unknown_message"
	KeyRemoteSSHHostAccept                = "remote.ssh_host_accept"
	KeyRemoteSSHHostDecline               = "remote.ssh_host_decline"
	KeyRemoteSSHHostDeclined              = "remote.ssh_host_declined"
	KeyRemoteSSHHostChangedTitle          = "remote.ssh_host_changed_title"
	KeyRemoteSSHHostChangedMessage        = "remote.ssh_host_changed_message"
	KeyRemoteSSHHostChangedStatus         = "remote.ssh_host_changed_status"
	KeyRemoteSSHConnected                 = "remote.ssh_connected"
	KeyChatLoadEarlier                    = "chat.load_earlier"
	KeyChatLoadEarlierBusy                = "chat.load_earlier_busy"
	KeyChatEdit                           = "chat.edit"
	KeyChatEditHint                       = "chat.edit_hint"
	KeyChatEditRunning                    = "chat.edit_running"
	KeyChatEditUnsupported                = "chat.edit_unsupported"
	KeyChatEditStale                      = "chat.edit_stale"
	KeyChatCopy                           = "chat.copy"
	KeyChatQuote                          = "chat.quote"
	KeyChatReconnect                      = "chat.reconnect"
	KeyChatDeliverables                   = "chat.deliverables"
	KeyChatCompaction                     = "chat.compaction"
	KeyChatImageClose                     = "chat.image_close"
	KeyPlanBoardTitle                     = "planboard.title"
	KeyPlanBoardExpand                    = "planboard.expand"
	KeyPlanBoardCollapse                  = "planboard.collapse"
	KeyPlanBoardEmpty                     = "planboard.empty"
	KeyPlanBoardPlan                      = "planboard.plan"
	KeyPlanBoardPlanOn                    = "planboard.plan_on"
	KeyPlanBoardPlanOff                   = "planboard.plan_off"
	KeyPlanBoardGoal                      = "planboard.goal"
	KeyPlanBoardNoGoal                    = "planboard.no_goal"
	KeyPlanBoardQueue                     = "planboard.queue"
	KeyPlanBoardQueueEmpty                = "planboard.queue_empty"
	KeyPlanBoardQueueEdit                 = "planboard.queue_edit"
	KeyPlanBoardQueueSteer                = "planboard.queue_steer"
	KeyPlanBoardQueueRemove               = "planboard.queue_remove"
	KeyPlanBoardTodos                     = "planboard.todos"
	KeyPlanBoardTodosEmpty                = "planboard.todos_empty"
	KeyPlanBoardTodoDone                  = "planboard.todo_done"
	KeyPlanBoardTodoActive                = "planboard.todo_active"
	KeyPlanBoardTodoPending               = "planboard.todo_pending"
	KeyPlanBoardGoalCreate                = "planboard.goal_create"
	KeyPlanBoardGoalEdit                  = "planboard.goal_edit"
	KeyPlanBoardGoalComplete              = "planboard.goal_complete"
	KeyPlanBoardGoalClear                 = "planboard.goal_clear"
	KeyPlanBoardGoalPause                 = "planboard.goal_pause"
	KeyPlanBoardGoalResume                = "planboard.goal_resume"
	KeyPlanBoardGoalDraftPlaceholder      = "planboard.goal_draft_placeholder"
	KeyTodoPanelTitle                     = "todopanel.title"
	KeyChatRunFailed                      = "chat.run_failed"
	KeyChatTurnDone                       = "chat.turn_done"
	KeyChatTurnDoneSec                    = "chat.turn_done_sec"
	KeyChatTurnDoneMin                    = "chat.turn_done_min"
	KeyChatTurnDoneMinSec                 = "chat.turn_done_min_sec"
	KeyUpdateDownload                     = "update.download"
	KeyUpdateDownloadTitle                = "update.download_title"
	KeySettingsAccountNav                 = "settings.account.nav"
	KeySettingsAccountTitle               = "settings.account.title"
	KeySettingsAccountHint                = "settings.account.hint"
	KeySettingsAccountStub                = "settings.account.stub"
	KeySettingsAccountCredHint            = "settings.account.cred_hint"
	KeySettingsAccountStatus              = "settings.account.status"
	KeySettingsAccountEmail               = "settings.account.email"
	KeySettingsAccountName                = "settings.account.name"
	KeySettingsAccountPlan                = "settings.account.plan"
	KeySettingsAccountUsageURL            = "settings.account.usage_url"
	KeySettingsAccountTopUpURL            = "settings.account.topup_url"
	KeySettingsAccountBalance             = "settings.account.balance"
	KeySettingsAccountAttempt             = "settings.account.attempt"
	KeySettingsAccountAuthorize           = "settings.account.authorize"
	KeySettingsAccountSignIn              = "settings.account.sign_in"
	KeySettingsAccountSignOut             = "settings.account.sign_out"
	KeySettingsAccountCancelSignIn        = "settings.account.cancel_sign_in"
	KeySettingsAboutNav                   = "settings.about.nav"
	KeySettingsAboutTitle                 = "settings.about.title"
	KeySettingsAboutHint                  = "settings.about.hint"
	KeySettingsAboutApp                   = "settings.about.app"
	KeySettingsAboutVersion               = "settings.about.version"
	KeySettingsAboutID                    = "settings.about.id"
	KeySettingsAboutUserData              = "settings.about.userdata"
	KeySettingsAboutVendor                = "settings.about.vendor"
	KeySettingsAboutRevealData            = "settings.about.reveal_data"
	KeySettingsAboutRevealLogs            = "settings.about.reveal_logs"
	KeySettingsAboutRevealVendor          = "settings.about.reveal_vendor"
	KeySettingsAboutCopyVersion           = "settings.about.copy_version"
	KeySettingsAboutCopyData              = "settings.about.copy_data"
	KeySettingsAppearanceNav              = "settings.appearance.nav"
	KeySettingsAppearanceTitle            = "settings.appearance.title"
	KeySettingsAppearanceHint             = "settings.appearance.hint"
	KeySettingsAppearanceClearWallpaper   = "settings.appearance.clear_wallpaper"
	KeySettingsAppearanceBrowseWallpaper  = "settings.appearance.browse_wallpaper"
	KeySettingsAppearanceWallpaperDir     = "settings.appearance.wallpaper_dir"
	KeySettingsInterfaceNav               = "settings.interface.nav"
	KeySettingsInterfaceTitle             = "settings.interface.title"
	KeySettingsInterfaceHint              = "settings.interface.hint"
	KeySettingsInterfaceAutoStart         = "settings.interface.auto_start"
	KeySettingsInterfaceAutoStartDesc     = "settings.interface.auto_start_desc"
	KeyChatPaneTabChat                    = "chat.pane_tab.chat"
	KeyChatPaneTabTrajectory              = "chat.pane_tab.trajectory"
	KeyChatBandTitle                      = "chat.band_title"
	KeyChatControlNote                    = "chat.control_note"
	KeySettingsInterfaceViewTabs          = "settings.interface.view_tabs"
	KeySettingsInterfaceViewTabsDesc      = "settings.interface.view_tabs_desc"
	KeySettingsInterfaceAutoAllow         = "settings.interface.auto_allow"
	KeySettingsInterfaceAutoAllowDesc     = "settings.interface.auto_allow_desc"
	KeySettingsModelsNav                  = "settings.models.nav"
	KeySettingsModelsTitle                = "settings.models.title"
	KeySettingsModelsHint                 = "settings.models.hint"
	KeySettingsModelsDiscover             = "settings.models.discover"
	KeySettingsModelsPicker               = "settings.models.picker"
	KeySettingsModelsEmpty                = "settings.models.empty"
	KeySettingsModelsSession              = "settings.models.session"
	KeySettingsModelsNoSession            = "settings.models.no_session"
	KeySettingsMCPNav                     = "settings.mcp.nav"
	KeySettingsMCPTitle                   = "settings.mcp.title"
	KeySettingsMCPHint                    = "settings.mcp.hint"
	KeySettingsMCPEmpty                   = "settings.mcp.empty"
	KeySettingsMCPStubHint                = "settings.mcp.stub_hint"
	KeySettingsMCPSchemaHint              = "settings.mcp.schema_hint"
	KeySettingsMCPEnabled                 = "settings.mcp.enabled"
	KeySettingsMCPDisabled                = "settings.mcp.disabled"
	KeySettingsMCPEnable                  = "settings.mcp.enable"
	KeySettingsMCPDisable                 = "settings.mcp.disable"
	KeySettingsMCPRetry                   = "settings.mcp.retry"
	KeySettingsMCPAuthorize               = "settings.mcp.authorize"
	KeySettingsMCPTest                    = "settings.mcp.test"
	KeySettingsMCPDelete                  = "settings.mcp.delete"
	KeySettingsPluginsInspect             = "settings.plugins.inspect"
	KeySettingsPluginsNav                 = "settings.plugins.nav"
	KeySettingsPluginsTitle               = "settings.plugins.title"
	KeySettingsPluginsHint                = "settings.plugins.hint"
	KeySettingsPluginsEmpty               = "settings.plugins.empty"
	KeySettingsPluginsNoRPC               = "settings.plugins.no_rpc"
	KeySettingsPluginsMarketHint          = "settings.plugins.market_hint"
	KeySettingsPluginsOpenMarket          = "settings.plugins.open_market"
	KeySettingsPluginsDisable             = "settings.plugins.disable"
	KeySettingsPluginsEnable              = "settings.plugins.enable"
	KeySettingsSkillsNav                  = "settings.skills.nav"
	KeySettingsSkillsTitle                = "settings.skills.title"
	KeySettingsSkillsHint                 = "settings.skills.hint"
	KeySettingsSkillsEmpty                = "settings.skills.empty"
	KeySettingsSkillsSession              = "settings.skills.session"
	KeySettingsSkillsNoSession            = "settings.skills.no_session"
	KeySettingsSkillsModelInvocable       = "settings.skills.model_invocable"
	KeySettingsSkillsUserOnly             = "settings.skills.user_only"
	KeySettingsSkillsInsert               = "settings.skills.insert"
	KeySettingsSkillsInspect              = "settings.skills.inspect"
	KeySettingsSkillsMakeUserOnly         = "settings.skills.make_user_only"
	KeySettingsSkillsMakeModel            = "settings.skills.make_model"
	KeySettingsSkillsHubTitle             = "settings.skills.hub_title"
	KeySettingsSkillsHubHint              = "settings.skills.hub_hint"
	KeySettingsSkillsHubPlaceholder       = "settings.skills.hub_placeholder"
	KeySettingsSkillsHubSearch            = "settings.skills.hub_search"
	KeySettingsSkillsHubInstall           = "settings.skills.hub_install"
	KeySettingsSkillsEditTitle            = "settings.skills.edit_title"
	KeySettingsSkillsEditDesc             = "settings.skills.edit_desc"
	KeySettingsSkillsEditContent          = "settings.skills.edit_content"
	KeySettingsSkillsSaveUpdate           = "settings.skills.save_update"
	KeySettingsAgentPresetsNav            = "settings.agent_presets.nav"
	KeySettingsAgentPresetsTitle          = "settings.agent_presets.title"
	KeySettingsAgentPresetsHint           = "settings.agent_presets.hint"
	KeySettingsAgentPresetsPicker         = "settings.agent_presets.picker"
	KeySettingsAgentPresetsEmpty          = "settings.agent_presets.empty"
	KeySettingsAgentPresetsSession        = "settings.agent_presets.session"
	KeySettingsAgentPresetsNoSession      = "settings.agent_presets.no_session"
	KeySettingsAgentPresetsRead           = "settings.agent_presets.read"
	KeyFilesOpenWebView                   = "files.open_webview"
	KeyFilesOpenExternal                  = "files.open_external"
	KeyFilesOpenWith                      = "files.open_with"
	KeyFilesReveal                        = "files.reveal"
	KeyFilesPDFHint                       = "files.pdf_hint"
	KeyFilesXLSXHint                      = "files.xlsx_hint"
	KeyFilesOfficeHint                    = "files.office_hint"
	KeyFilesRenderPDF                     = "files.render_pdf"
	KeySettingsMCPUpsert                  = "settings.mcp.upsert"
	KeySettingsMCPUpsertHint              = "settings.mcp.upsert_hint"
	KeySettingsMCPUpsertSave              = "settings.mcp.upsert_save"
	KeyScheduleOpen                       = "schedule.open"
	KeyScheduleTitle                      = "schedule.title"
	KeyScheduleBack                       = "schedule.back"
	KeyScheduleRefresh                    = "schedule.refresh"
	KeyScheduleLoading                    = "schedule.loading"
	KeyScheduleNote                       = "schedule.note"
	KeyScheduleEmpty                      = "schedule.empty"
	KeyScheduleDelete                     = "schedule.delete"
	KeyScheduleHistory                    = "schedule.history"
	KeyScheduleCopyHistory                = "schedule.copy_history"
	KeyScheduleEditTitle                  = "schedule.edit_title"
	KeyScheduleSaveTitle                  = "schedule.save_title"
	KeyScheduleEditPrompt                 = "schedule.edit_prompt"
	KeyScheduleSavePrompt                 = "schedule.save_prompt"
	KeyAgentsTitle                        = "agents.title"
	KeyAgentsClose                        = "agents.close"
	KeyAgentsRootLabel                    = "agents.root_label"
	KeyAgentsRootFallback                 = "agents.root_fallback"
	KeyAgentsPickSession                  = "agents.pick_session"
	KeyAgentsChildren                     = "agents.children"
	KeyAgentsEmpty                        = "agents.empty"
	KeyAgentsStop                         = "agents.stop"
	KeyAgentsInterrupt                    = "agents.interrupt"
	KeyAgentsInterruptAll                 = "agents.interrupt_all"
	KeyAgentsPromptPlaceholder            = "agents.prompt_placeholder"
	KeyAgentsPromptSend                   = "agents.prompt_send"
	KeyFilesFilter                        = "files.filter"
	KeyFilesFilterEmpty                   = "files.filter_empty"
	KeyFilesRefreshHint                   = "files.refresh_hint"
	KeyRemoteMobileTitle                  = "remote.mobile_title"
	KeyRemoteMobileNote                   = "remote.mobile_note"
	KeyRemoteMobileAppBase                = "remote.mobile_app_base"
	KeyRemoteMobileDevices                = "remote.mobile_devices"
	KeyRemoteMobileNeedToken              = "remote.mobile_need_token"
	KeyRemotePhoneTitle                   = "remote.phone_title"
	KeyRemotePhoneHowTo                   = "remote.phone_howto"
	KeyRemotePhoneStatus                  = "remote.phone_status"
	KeyRemotePhoneReady                   = "remote.phone_ready"
	KeyRemotePhoneIncomplete              = "remote.phone_incomplete"
	KeyRemotePhoneDisabled                = "remote.phone_disabled"
	KeyRemotePhoneMissing                 = "remote.phone_missing"
	KeyRemotePhoneRelayToken              = "remote.phone_relay_token"
	KeyRemotePhoneLanBind                 = "remote.phone_lan_bind"
	KeyRemotePhoneLanTLS                  = "remote.phone_lan_tls"
	KeyRemotePhoneConnectURL              = "remote.phone_connect_url"
	KeyRemotePhoneQRPlaceholder           = "remote.phone_qr_placeholder"
	KeyRemotePhoneURLUnavailable          = "remote.phone_url_unavailable"
	KeyRemotePhoneCopyURL                 = "remote.phone_copy_url"
	KeyRemotePhoneOpenURL                 = "remote.phone_open_url"
	KeyRemotePhoneRefresh                 = "remote.phone_refresh"
	KeyRemotePhoneDeviceList              = "remote.phone_device_list"
	KeyRemotePhoneCopied                  = "remote.phone_copied"
	KeyRemotePhoneOpened                  = "remote.phone_opened"
	KeyRemotePhoneOpenFailed              = "remote.phone_open_failed"
	KeyRemotePhoneNoURL                   = "remote.phone_no_url"
	KeyRemotePhoneRefreshed               = "remote.phone_refreshed"
	KeyRemotePhoneCopyFailed              = "remote.phone_copy_failed"
	KeyIMOpen                             = "im.open"
	KeyIMOpenWeb                          = "im.open_web"
	KeyIMChannels                         = "im.channels"
	KeyIMCompose                          = "im.compose"
	KeyIMSend                             = "im.send"
	KeyIMTitle                            = "im.title"
	KeyIMBack                             = "im.back"
	KeyIMRefresh                          = "im.refresh"
	KeyIMLoading                          = "im.loading"
	KeyIMStubNote                         = "im.stub_note"
	KeyIMHarnessOverlay                   = "im.harness_overlay"
	KeyIMPluginDisabled                   = "im.plugin_disabled"
	KeyIMMux                              = "im.mux"
	KeyIMGeneratedRPC                     = "im.generated_rpc"
	KeyIMOn                               = "im.on"
	KeyIMOff                              = "im.off"
	KeyIMMuxWaiting                       = "im.mux_waiting"
	KeyIMMuxConnected                     = "im.mux_connected"
	KeyIMMuxDisconnected                  = "im.mux_disconnected"
	KeyIMRPCYes                           = "im.rpc_yes"
	KeyIMRPCNo                            = "im.rpc_no"
	KeyIMRPCPartial                       = "im.rpc_partial"
	KeyIMNoRPC                            = "im.no_rpc"
	KeyIMConnectionPlaceholder            = "im.connection_placeholder"
	KeyIMListVia                          = "im.list_via"
	KeyIMListEmpty                        = "im.list_empty"
	KeyIMMethods                          = "im.methods"
	KeyIMPluginOff                        = "im.plugin_off"
	KeyIMLocalMode                        = "im.local_mode"
	KeyIMHistory                          = "im.history"
	KeyIMHistoryEmpty                     = "im.history_empty"
	KeyIMNoChannel                        = "im.no_channel"
	KeyIMYou                              = "im.you"
	KeyIMPeer                             = "im.peer"
	KeyIMBridge                           = "im.bridge"
	KeyIMBridgeRPC                        = "im.bridge_rpc"
	KeyIMBridgeCordis                     = "im.bridge_cordis"
	KeyIMBridgeInventory                  = "im.bridge_inventory"
	KeyIMBridgeLocal                      = "im.bridge_local"
	KeyIMBridgeIdle                       = "im.bridge_idle"
	KeyIMPluginDiag                       = "im.plugin_diag"
	KeyIMHistoryVia                       = "im.history_via"
	KeyIMHistorySynced                    = "im.history_synced"
	KeyMarketOpen                         = "market.open"
	KeyMarketTitle                        = "market.title"
	KeyMarketBack                         = "market.back"
	KeyMarketRefresh                      = "market.refresh"
	KeyMarketLoading                      = "market.loading"
	KeyMarketHarnessOverlay               = "market.harness_overlay"
	KeyMarketPluginDisabled               = "market.plugin_disabled"
	KeyMarketMux                          = "market.mux"
	KeyMarketGeneratedRPC                 = "market.generated_rpc"
	KeyMarketInstallToolchain             = "market.install_toolchain"
	KeyMarketOn                           = "market.on"
	KeyMarketOff                          = "market.off"
	KeyMarketMuxWaiting                   = "market.mux_waiting"
	KeyMarketMuxConnected                 = "market.mux_connected"
	KeyMarketMuxDisconnected              = "market.mux_disconnected"
	KeyMarketRPCYes                       = "market.rpc_yes"
	KeyMarketRPCNo                        = "market.rpc_no"
	KeyMarketRPCPartial                   = "market.rpc_partial"
	KeyMarketNoRPC                        = "market.no_rpc"
	KeyMarketListVia                      = "market.list_via"
	KeyMarketListEmpty                    = "market.list_empty"
	KeyMarketMethods                      = "market.methods"
	KeyMarketPluginOff                    = "market.plugin_off"
	KeyMarketInstallHint                  = "market.install_hint"
	KeyMarketInstallAdd                   = "market.install_add"
	KeyMarketInstallSpecLabel             = "market.install_spec_label"
	KeyMarketInstallPlaceholder           = "market.install_placeholder"
	KeyMarketCopyLog                      = "market.copy_log"
	KeyMarketInstallButton                = "market.install_button"
	KeyMarketRemove                       = "market.remove"
	KeyMarketAllowBuildsButton            = "market.allow_builds_button"
	KeyMarketAllowBuilds                  = "market.allow_builds"
	KeyMarketInstallNotReady              = "market.install_not_ready"
	KeyMarketInstallOK                    = "market.install_ok"
	KeyMarketInstallRunning               = "market.install_running"
	KeyMarketInstallLog                   = "market.install_log"
	KeyMarketOpenHarness                  = "market.open_harness"
	KeyMarketRegistry                     = "market.registry"
	KeyMarketExemptions                   = "market.exemptions"
	KeyMarketCancelInstall                = "market.cancel_install"
	KeyMarketCancelOK                     = "market.cancel_ok"
	KeyMarketIncompatible                 = "market.incompatible"
	KeyMarketAcceptExemption              = "market.accept_exemption"
	KeyAssistantOpen                      = "assistant.open"
	KeyAssistantTitle                     = "assistant.title"
	KeyAssistantBack                      = "assistant.back"
	KeyAssistantStubNote                  = "assistant.stub_note"
	KeyAssistantWhaleTitle                = "assistant.whale_title"
	KeyAssistantDshbotTitle               = "assistant.dshbot_title"
	KeyAssistantConfigEnabled             = "assistant.config_enabled"
	KeyAssistantVendorPresent             = "assistant.vendor_present"
	KeyAssistantOn                        = "assistant.on"
	KeyAssistantOff                       = "assistant.off"
	KeyAssistantToggleOn                  = "assistant.toggle_on"
	KeyAssistantToggleOff                 = "assistant.toggle_off"
	KeyAssistantOpenWhaleWeb              = "assistant.open_whale_web"
	KeyAssistantOpenDshbotWeb             = "assistant.open_dshbot_web"
	KeyAssistantRestartNote               = "assistant.restart_note"
	KeyAssistantStatusTitle               = "assistant.status_title"
	KeyAssistantLastWebURL                = "assistant.last_web_url"
	KeyAssistantLastAction                = "assistant.last_action"
	KeyAssistantComposeTitle              = "assistant.compose_title"
	KeyAssistantComposeHint               = "assistant.compose_hint"
	KeyAssistantComposePlaceholder        = "assistant.compose_placeholder"
	KeyAssistantSend                      = "assistant.send"
	KeyAssistantNoSession                 = "assistant.no_session"
	KeyAssistantNoRPC                     = "assistant.no_rpc"
	KeyAssistantPromptSent                = "assistant.prompt_sent"
	KeyAssistantPluginRPCTitle            = "assistant.plugin_rpc_title"
	KeyAssistantPluginRPCHint             = "assistant.plugin_rpc_hint"
	KeyAssistantWhaleEnsure               = "assistant.whale_ensure"
	KeyAssistantWhaleCatalog              = "assistant.whale_catalog"
	KeyAssistantWhalePetChat              = "assistant.whale_pet_chat"
	KeyAssistantWhalePetState             = "assistant.whale_pet_state"
	KeyAssistantWhalePetLook              = "assistant.whale_pet_look"
	KeyAssistantDshbotDescribe            = "assistant.dshbot_describe"
	KeyAssistantDshbotCatalog             = "assistant.dshbot_catalog"
	KeyAssistantDshbotActivity            = "assistant.dshbot_activity"
	KeyAssistantDshbotRosterTitle         = "assistant.dshbot_roster_title"
	KeyAssistantDshbotRosterRefresh       = "assistant.dshbot_roster_refresh"
	KeyAssistantDshbotRosterEmpty         = "assistant.dshbot_roster_empty"
	KeyAssistantDshbotRosterUpdated       = "assistant.dshbot_roster_updated"
	KeyAssistantDshbotRosterBot           = "assistant.dshbot_roster_bot"
	KeyAssistantDshbotRosterRoom          = "assistant.dshbot_roster_room"
	KeyAssistantDshbotRosterOpen          = "assistant.dshbot_roster_open"
	KeyAssistantModeTitle                 = "assistant.mode_title"
	KeyAssistantModeHint                  = "assistant.mode_hint"
	KeyAssistantModeLabel                 = "assistant.mode_label"
	KeyAssistantModeWhale                 = "assistant.mode_whale"
	KeyAssistantModeDshbot                = "assistant.mode_dshbot"
	KeyAssistantOpenModeWeb               = "assistant.open_mode_web"
	KeyAssistantOpenSession               = "assistant.open_session"
	KeyAssistantHistoryTitle              = "assistant.history_title"
	KeyAssistantHistoryEmpty              = "assistant.history_empty"
	KeyAssistantBusy                      = "assistant.busy"
	KeyAssistantLastErr                   = "assistant.last_err"
	KeyAssistantSessionID                 = "assistant.session_id"
	KeyAssistantWhaleRPCSent              = "assistant.whale_rpc_sent"
	KeyAssistantWhaleRPCErr               = "assistant.whale_rpc_err"
	KeyAssistantDshbotRouted              = "assistant.dshbot_routed"
	KeyAssistantDshbotProbeOK             = "assistant.dshbot_probe_ok"
	KeyAssistantDshbotProbeErr            = "assistant.dshbot_probe_err"
	KeyAssistantEnsuring                  = "assistant.ensuring"
	KeyAssistantChatting                  = "assistant.chatting"
)

var (
	mu       sync.RWMutex
	current  = DefaultLocale
	catalogs = map[string]map[string]string{
		LocaleZH: zhCN,
		LocaleEN: en,
	}
)

var zhCN = map[string]string{
	KeyChatManualCompaction:           "手动压缩",
	KeyChatCompactionCounts:           "已压缩 %d 条历史记录（约 %d tokens）",
	KeyChatCompactionSummaryMissing:   "摘要未包含在已加载的历史中",
	KeyCommandTitle:                   "命令",
	KeyCommandRunning:                 "正在执行…",
	KeyCommandDone:                    "已完成",
	KeyCommandFailed:                  "执行失败",
	KeyCommandCatalogLoading:          "正在加载命令…",
	KeyCommandCatalogRetry:            "重试加载命令",
	KeyAppName:                        "MyGo鲸屿",
	KeyTrayOpen:                       "打开主窗口",
	KeyTrayQuit:                       "退出",
	KeyQuitConfirmTitle:               "仍有任务在运行",
	KeyQuitConfirmMessage:             "当前有 %d 个会话仍在运行。确定要退出吗？",
	KeyQuitConfirmQuit:                "退出",
	KeyQuitConfirmCancel:              "取消",
	KeyTitlebarClose:                  "关闭",
	KeyTitlebarMinimize:               "最小化",
	KeyTitlebarMaximize:               "最大化",
	KeyStatusPlaceholder:              "骨架：窗口/托盘就绪",
	KeyBootTitle:                      "正在启动 Harness",
	KeyBootStarting:                   "正在启动本地服务…",
	KeyBootReady:                      "本地服务已就绪",
	KeyBootFailed:                     "启动失败",
	KeyBootStopped:                    "未启动",
	KeyBootRetry:                      "重试",
	KeyBootExport:                     "导出日志",
	KeyBootWaiting:                    "等待输出…",
	KeyBootPlugins:                    "正在加载插件 %d/%d",
	KeyBootPluginsDetail:              "%s",
	KeyBootPluginsIndeterminate:       "正在加载插件…",
	KeySidebarSessions:                "会话",
	KeySidebarRefresh:                 "刷新",
	KeySidebarNew:                     "新建",
	KeySidebarLoading:                 "正在加载会话…",
	KeySidebarEmpty:                   "暂无会话",
	KeySidebarRename:                  "重命名",
	KeySidebarRenameCancel:            "取消",
	KeySidebarSearch:                  "搜索会话…",
	KeySidebarSearchLoading:           "搜索中…",
	KeySidebarArchive:                 "归档",
	KeySidebarUnarchive:               "取消归档",
	KeySidebarArchivedSection:         "已归档",
	KeySidebarBrowseWorkspace:         "选择工作区…",
	KeySidebarRevealWorkspace:         "在资源管理器中打开",
	KeySidebarRenameWorkspace:         "重命名工作区…",
	KeySidebarRenameWorkspaceSave:     "保存工作区名",
	KeySidebarDeleteWorkspace:         "删除工作区",
	KeySidebarMoveUp:                  "置顶排序",
	KeySidebarDelete:                  "删除",
	KeySidebarCopyID:                  "复制 ID",
	KeySidebarFork:                    "分叉",
	KeyWorkspaceRegistered:            "工作区已注册: %s",
	KeySidebarDeleteConfirm:           "确定删除此会话？此操作不可撤销。",
	KeySidebarPin:                     "置顶",
	KeySidebarUnpin:                   "取消置顶",
	KeySidebarPinnedPrefix:            "📌 ",
	KeySidebarBrand:                   "MyGo鲸屿",
	KeySidebarMarket:                  "插件",
	KeySidebarSchedule:                "自动化任务",
	KeySidebarAssistant:               "鲸鱼娘",
	KeySidebarWorkspaceSection:        "工作区",
	KeySidebarNewSessionFallback:      "新会话",
	KeyChatEmpty:                      "选择会话后开始对话",
	KeyChatHeroSlogan:                 "探索未至之境",
	KeyChatHeroPreview:                "预览版",
	KeyChatHeroChooseWorkspace:        "选择工作区",
	KeyChatThinking:                   "思考",
	KeyChatConnConnecting:             "连接中…",
	KeyChatConnReconnecting:           "已断开，重连中…",
	KeyChatUsageJobsToday:             "今日任务",
	KeyChatUsageBalance:               "余额",
	KeyChatFeedbackUp:                 "👍",
	KeyChatFeedbackDown:               "👎",
	KeyChatFeedbackSelectedMark:       " ✓",
	KeyChatFeedbackCommentPlaceholder: "可选：不喜欢的原因…",
	KeyChatFeedbackSend:               "发送反馈",
	KeyComposerModel:                  "模型",
	KeyComposerPermission:             "权限",
	KeyComposerPermissionDefault:      "工作区内修改",
	KeyComposerAgentPreset:            "Agent 预设",
	KeyComposerSend:                   "发送",
	KeyComposerStop:                   "停止",
	KeyComposerModeQueue:              "排队",
	KeyComposerModeSteer:              "插话",
	// Match dsh ui-conversation locales.js placeholder.default / placeholder.hero.
	KeyComposerPlaceholder:                "发消息或创建任务, / 调用指令, @ 文件或对话",
	KeyComposerPlaceholderHero:            "描述你想要构建的内容, / 调用指令, @ 文件或对话",
	KeyComposerNoSession:                  "未选择会话",
	KeyComposerWaiting:                    "等待服务…",
	KeyComposerDropHint:                   "松开以附加 @工作区路径",
	KeyComposerSkills:                     "技能",
	KeyComposerSkillUserOnly:              "仅用户",
	KeyComposerSkillsFilter:               "过滤技能…",
	KeyComposerSkillsHubSearch:            "搜索 Hub",
	KeyComposerSkillsHubInstall:           "安装",
	KeyComposerSkillsDelete:               "删除",
	KeyComposerSkillsCreate:               "创建技能",
	KeyComposerSkillsCreateName:           "新技能名…",
	KeyComposerSkillsToggleModel:          "模型可调用",
	KeyComposerSkillsEmpty:                "无匹配技能",
	KeyComposerPickDir:                    "选择目录",
	KeyComposerPresetDefault:              "标准模式",
	KeyComposerPickDirShort:               "目录",
	KeyComposerPickImage:                  "图片",
	KeyComposerSpeech:                     "语音文件",
	KeyComposerSpeechCycle:                "STT▾",
	KeyComposerDirDialogTitle:             "选择工作区目录",
	KeyComposerDirOutside:                 "所选目录不在工作区内",
	KeyComposerDirNoWorkspace:             "未配置工作区",
	KeyComposerChipEdit:                   "编辑标签",
	KeyComposerChipEditPlaceholder:        "@路径 或 /技能",
	KeyComposerChipEditApply:              "应用",
	KeyComposerChipEditCancel:             "取消",
	KeyFilesTitle:                         "工作区文件",
	KeyFilesFilter:                        "筛选文件…",
	KeyFilesFilterEmpty:                   "无匹配文件",
	KeyFilesRefreshHint:                   "树随磁盘实时列出",
	KeyFilesNoWorkspace:                   "未配置工作区",
	KeyFilesPickFile:                      "选择文件以预览",
	KeyFilesTooLarge:                      "文件过大，无法预览",
	KeyFilesBinary:                        "二进制文件，无法预览",
	KeyFilesShowDiff:                      "Diff",
	KeyFilesEdit:                          "编辑",
	KeyFilesSave:                          "保存",
	KeyFilesCancelEdit:                    "取消",
	KeyFilesSaved:                         "已保存",
	KeyFilesModeSplit:                     "分屏预览",
	KeyFilesModeEdit:                      "仅编辑",
	KeyFilesModeHighlight:                 "仅高亮",
	KeyFilesModeNext:                      "切换",
	KeyUpdateDownload:                     "下载并安装",
	KeyUpdateDownloadTitle:                "更新安装",
	KeySettingsAccountNav:                 "账号与登录",
	KeySettingsAccountTitle:               "账号与登录",
	KeySettingsAccountHint:                "Harness 账号状态（account/getState、getProfile、getBalance）。",
	KeySettingsAccountStub:                "未连接 Harness 时显示占位；连接后点刷新拉取账号摘要。",
	KeySettingsAccountCredHint:            "API Key 等密钥请到左侧「凭据」页编辑。",
	KeySettingsAccountStatus:              "状态",
	KeySettingsAccountEmail:               "邮箱",
	KeySettingsAccountName:                "名称",
	KeySettingsAccountPlan:                "套餐",
	KeySettingsAccountUsageURL:            "用量链接",
	KeySettingsAccountTopUpURL:            "充值链接",
	KeySettingsAccountBalance:             "余额摘要",
	KeySettingsAccountAttempt:             "登录阶段",
	KeySettingsAccountAuthorize:           "授权链接",
	KeySettingsAccountSignIn:              "登录",
	KeySettingsAccountSignOut:             "退出登录",
	KeySettingsAccountCancelSignIn:        "取消登录",
	KeySettingsAboutNav:                   "关于",
	KeySettingsAboutTitle:                 "关于 MyGo鲸屿",
	KeySettingsAboutHint:                  "原生 MyGo 壳版本与数据路径。",
	KeySettingsAboutApp:                   "应用",
	KeySettingsAboutVersion:               "版本",
	KeySettingsAboutID:                    "identifier",
	KeySettingsAboutUserData:              "userData",
	KeySettingsAboutVendor:                "vendor",
	KeySettingsAboutRevealData:            "打开数据目录",
	KeySettingsAboutRevealLogs:            "打开日志目录",
	KeySettingsAboutRevealVendor:          "打开 Vendor 目录",
	KeySettingsAboutCopyVersion:           "复制版本号",
	KeySettingsAboutCopyData:              "复制路径",
	KeySettingsAppearanceNav:              "外观",
	KeySettingsAppearanceTitle:            "外观",
	KeySettingsAppearanceHint:             "主题与壁纸（写入 config.json）。",
	KeySettingsAppearanceClearWallpaper:   "清除壁纸",
	KeySettingsAppearanceBrowseWallpaper:  "浏览壁纸…",
	KeySettingsAppearanceWallpaperDir:     "wallpapers/",
	KeySettingsInterfaceNav:               "界面",
	KeySettingsInterfaceTitle:             "界面",
	KeySettingsInterfaceHint:              "语言与窗口行为（写入 config.json）。主题与壁纸在「外观」；工作区与端口在「桌面」。",
	KeySettingsInterfaceAutoStart:         "启动时自动打开桌面",
	KeySettingsInterfaceAutoStartDesc:     "配置工作区后，应用启动时自动启动 harness 桌面（与启动器开关相同）。",
	KeyChatPaneTabChat:                    "对话",
	KeyChatPaneTabTrajectory:              "轨迹",
	KeyChatBandTitle:                      "快捷操作",
	KeyChatControlNote:                    "会话控制",
	KeySettingsInterfaceViewTabs:          "对话与轨迹页签",
	KeySettingsInterfaceViewTabsDesc:      "在会话顶栏显示「对话」「轨迹」切换。关闭后仍停留在当前视图。",
	KeySettingsInterfaceAutoAllow:         "工具自动允许",
	KeySettingsInterfaceAutoAllowDesc:     "自动批准工具审批请求；关闭后需在卡片上手动允许 / 始终允许 / 拒绝。",
	KeySettingsModelsNav:                  "模型",
	KeySettingsModelsTitle:                "模型",
	KeySettingsModelsHint:                 "当前会话的模型目录（session/modelCatalog）。",
	KeySettingsModelsDiscover:             "发现提供商",
	KeySettingsModelsPicker:               "默认模型",
	KeySettingsModelsEmpty:                "暂无模型（需 Harness Ready 且已拉目录）。",
	KeySettingsModelsSession:              "会话",
	KeySettingsModelsNoSession:            "未选中会话时仍可刷新目录；应用需先选中会话。",
	KeySettingsMCPNav:                     "MCP",
	KeySettingsMCPTitle:                   "MCP 服务器",
	KeySettingsMCPHint:                    "宿主 MCP 服务器列表（mcpServers/list）。",
	KeySettingsMCPEmpty:                   "暂无 MCP 服务器。",
	KeySettingsMCPStubHint:                "Harness 未连接或 host MCP 插件未就绪；连接后点刷新。",
	KeySettingsMCPSchemaHint:              "仍可通过左侧 schema 表单命名空间配置相关项。",
	KeySettingsMCPEnabled:                 "已启用",
	KeySettingsMCPDisabled:                "已禁用",
	KeySettingsMCPEnable:                  "启用",
	KeySettingsMCPDisable:                 "禁用",
	KeySettingsMCPRetry:                   "重试",
	KeySettingsMCPAuthorize:               "授权",
	KeySettingsMCPTest:                    "测试",
	KeySettingsMCPDelete:                  "删除",
	KeySettingsMCPUpsert:                  "添加 / 更新服务器",
	KeySettingsMCPUpsertHint:              "粘贴 mcpServers/upsert 的 spec JSON（stdio / http 等）。",
	KeySettingsMCPUpsertSave:              "保存到 Host",
	KeySettingsPluginsNav:                 "插件",
	KeySettingsPluginsTitle:               "插件",
	KeySettingsPluginsHint:                "已安装插件（pluginManager/listPlugins）。",
	KeySettingsPluginsEmpty:               "暂无已安装插件。",
	KeySettingsPluginsNoRPC:               "codegen 未含 pluginManager/listPlugins 时无法列出插件。",
	KeySettingsPluginsMarketHint:          "完整浏览与安装请打开插件市场面板。",
	KeySettingsPluginsOpenMarket:          "打开市场",
	KeySettingsPluginsDisable:             "禁用",
	KeySettingsPluginsEnable:              "启用",
	KeySettingsPluginsInspect:             "检查",
	KeySettingsSkillsNav:                  "技能",
	KeySettingsSkillsTitle:                "技能",
	KeySettingsSkillsHint:                 "当前会话技能列表（skills/list）。",
	KeySettingsSkillsEmpty:                "暂无技能（需已选会话并刷新）。",
	KeySettingsSkillsSession:              "会话",
	KeySettingsSkillsNoSession:            "未选中会话时无法拉取技能列表。",
	KeySettingsSkillsModelInvocable:       "模型可调用",
	KeySettingsSkillsUserOnly:             "仅用户",
	KeySettingsSkillsInsert:               "插入到输入框",
	KeySettingsSkillsInspect:              "查看",
	KeySettingsSkillsMakeUserOnly:         "改为仅用户",
	KeySettingsSkillsMakeModel:            "允许模型调用",
	KeySettingsSkillsHubTitle:             "技能中心",
	KeySettingsSkillsHubHint:              "搜索 skillInventory/searchHub，安装到用户技能根。",
	KeySettingsSkillsHubPlaceholder:       "搜索技能…",
	KeySettingsSkillsHubSearch:            "搜索",
	KeySettingsSkillsHubInstall:           "安装",
	KeySettingsSkillsEditTitle:            "编辑技能",
	KeySettingsSkillsEditDesc:             "描述",
	KeySettingsSkillsEditContent:          "内容（Markdown）",
	KeySettingsSkillsSaveUpdate:           "保存更新",
	KeySettingsAgentPresetsNav:            "Agent 预设",
	KeySettingsAgentPresetsTitle:          "Agent 预设",
	KeySettingsAgentPresetsHint:           "选择后续新任务使用的默认 Agent 预设。",
	KeySettingsAgentPresetsPicker:         "默认预设",
	KeySettingsAgentPresetsEmpty:          "暂无预设（需 Harness Ready 且已拉列表）。",
	KeySettingsAgentPresetsSession:        "会话",
	KeySettingsAgentPresetsNoSession:      "未选中会话时仍可刷新列表；应用需先选中会话。",
	KeySettingsAgentPresetsRead:           "查看文档",
	KeyFilesOpenWebView:                   "WebView2 打开",
	KeyFilesOpenExternal:                  "系统打开",
	KeyFilesOpenWith:                      "打开方式",
	KeyFilesReveal:                        "在资源管理器中显示",
	KeyFilesPDFHint:                       "PDF：请用 WebView2 或系统默认应用打开。",
	KeyFilesXLSXHint:                      "电子表格：原生预览失败时请用系统应用打开。",
	KeyFilesOfficeHint:                    "Office 文档：请用系统默认应用打开。",
	KeyFilesRenderPDF:                     "转为 PDF 预览",
	KeyGitPRView:                          "查看 PR",
	KeyGitPRCreate:                        "草稿 PR",
	KeyGitPRPanel:                         "PR",
	KeyGitPRRefresh:                       "刷新 PR",
	KeyGitPRClose:                         "关闭",
	KeyGitPRCheckout:                      "检出",
	KeyGitPRCheckoutOK:                    "已检出 PR",
	KeyGitPRNoGH:                          "未找到 GitHub CLI (gh)",
	KeyGitPRNone:                          "当前分支无 PR",
	KeyRightPaneFiles:                     "文件",
	KeyRightPaneDiff:                      "Diff",
	KeyRightPaneAgents:                    "智能体",
	KeyFrameCaptionAgents:                 "◎  智能体团队",
	KeyFrameCaptionProjectDocs:            "▣  项目资料",
	KeyDiffTitle:                          "工作区 Diff",
	KeyDiffRefresh:                        "刷新",
	KeyDiffHint:                           "点「刷新」运行 git diff",
	KeyDiffEmpty:                          "没有未提交的差异",
	KeyDiffNoWorkspace:                    "未配置工作区",
	KeyDiffNotRepo:                        "不是 Git 仓库",
	KeyInteractionAllow:                   "允许",
	KeyInteractionAllowAlways:             "始终允许",
	KeyInteractionDeny:                    "拒绝",
	KeyInteractionCallID:                  "调用 ID",
	KeyInteractionCopyArgs:                "复制参数",
	KeyInteractionContinue:                "继续",
	KeyInteractionManual:                  "改为手动审批",
	KeyInteractionAutoAllowOn:             "工具审批：自动允许",
	KeyInteractionAutoAllowed:             "已自动允许",
	KeyBootExportSaved:                    "日志已保存",
	KeySettingsOpen:                       "设置",
	KeyDesktopTools:                       "工具",
	KeySettingsTitle:                      "Harness 设置",
	KeySettingsBack:                       "返回",
	KeySettingsNamespaces:                 "命名空间",
	KeySettingsEmpty:                      "暂无设置项",
	KeySettingsRefresh:                    "刷新",
	KeySettingsLoading:                    "加载中…",
	KeySettingsSave:                       "保存",
	KeySettingsReplace:                    "替换文档",
	KeySettingsReplaceHint:                "用当前表单完整替换该命名空间文档（非局部补丁）。",
	KeySettingsReplaceConfirmTitle:        "确认替换文档",
	KeySettingsReplaceConfirmMessage:      "将用当前草稿完整替换所选命名空间的设置文档，继续？",
	KeySettingsReplaceCancel:              "取消",
	KeySettingsReadOnly:                   "当前配置只读",
	KeySettingsWritable:                   "可写",
	KeySettingsRevision:                   "修订",
	KeySettingsFilter:                     "过滤命名空间…",
	KeySettingsFilterEmpty:                "无匹配命名空间",
	KeySettingsRevealSecrets:              "显示密钥",
	KeySettingsPickNS:                     "选择左侧命名空间",
	KeySettingsDesktopNav:                 "桌面",
	KeySettingsDesktopTitle:               "桌面与本地配置",
	KeySettingsDesktopHint:                "写入本机 config.json（与 Electron 版共用数据目录）。修改端口后需重启 Harness。",
	KeySettingsDesktopWorkspace:           "工作区路径",
	KeySettingsDesktopBrowseWorkspace:     "浏览…",
	KeySettingsDesktopPort:                "Harness 端口",
	KeySettingsDesktopTheme:               "主题",
	KeySettingsDesktopLocale:              "界面语言",
	KeySettingsDesktopCloseToTray:         "关闭时隐藏到托盘",
	KeySettingsDesktopOpenAtLogin:         "登录时启动",
	KeySettingsDesktopSaved:               "本地配置已保存",
	KeySettingsDesktopDetectElectron:      "检测 Electron 数据",
	KeySettingsDesktopCheckUpdates:        "检查更新",
	KeySettingsCredNav:                    "凭据",
	KeySettingsCredTitle:                  "凭据 (credentials.json)",
	KeySettingsCredHint:                   "与 Electron 共用数据目录；保存写入 DPAPI 信封（若可用）。勾选「显示密钥」后可编辑敏感字段。",
	KeySettingsCredShow:                   "显示密钥",
	KeySettingsCredAPIKey:                 "API Key",
	KeySettingsCredClearAPIKey:            "清除 API Key",
	KeySettingsCredClearSecrets:           "清除全部密钥",
	KeySettingsCredBaseURL:                "Base URL",
	KeySettingsCredGithub:                 "GitHub Token",
	KeySettingsCredRemote:                 "Remote Token",
	KeySettingsCredRelay:                  "Remote Relay Token",
	KeySettingsCredSaved:                  "凭据已保存",
	KeySettingsDesktopWallpaper:           "壁纸",
	KeySettingsDesktopWallpaperNone:       "（无）",
	KeyLauncherOpen:                       "启动器",
	KeyLauncherTitle:                      "MyGo鲸屿 启动器",
	KeyLauncherClose:                      "关闭",
	KeyLauncherHint:                       "更新询问、版本、Electron 数据探测与插件排查；配置工作区后可启动桌面。",
	KeyLauncherVersion:                    "版本",
	KeyLauncherVendor:                     "vendored harness",
	KeyLauncherOn:                         "就绪",
	KeyLauncherOff:                        "缺失",
	KeyLauncherWorkspace:                  "工作区",
	KeyLauncherWorkspaceHint:              "本地工作区绝对路径",
	KeyLauncherNoWorkspace:                "尚未配置工作区",
	KeyLauncherStart:                      "启动桌面",
	KeyLauncherScanImport:                 "检测 Electron 数据",
	KeyLauncherCheckUpdate:                "检查更新",
	KeyLauncherOpenSettings:               "打开设置",
	KeyLauncherImportTitle:                "导入探测",
	KeyLauncherUpdateTitle:                "更新",
	KeyLauncherPluginsTitle:               "disabledPlugins",
	KeyLauncherPluginsNone:                "无禁用插件条目",
	KeyLauncherPluginRecoveryTitle:        "插件排查",
	KeyLauncherSkipUserPlugins:            "skip-user-plugins",
	KeyLauncherSkipUserPluginsOn:          "已启用（recovery）",
	KeyLauncherSkipUserPluginsOff:         "未启用",
	KeyLauncherEnableSkipUserPlugins:      "启用 skip-user-plugins",
	KeyLauncherClearPluginRecovery:        "清除 pluginRecovery",
	KeyLauncherRestartHarness:             "重启 Harness",
	KeyLauncherBrandRole:                  "启动器",
	KeyLauncherTabHome:                    "首页",
	KeyLauncherTabComponents:              "组件",
	KeyLauncherTabVersions:                "版本",
	KeyLauncherTabImport:                  "导入",
	KeyLauncherTabPlugins:                 "插件排查",
	KeyLauncherTabSettings:                "设置",
	KeyLauncherHomeLede:                   "桌面端与组件，一处管理。",
	KeyLauncherGotoVersions:               "版本管理",
	KeyLauncherDesktopTitle:               "MyGo鲸屿桌面端",
	KeyLauncherDesktopDesc:                "在桌面端开始对话，管理模型与工作区。",
	KeyLauncherDesktopRunning:             "桌面端已就绪",
	KeyLauncherDesktopStopped:             "桌面端未运行",
	KeyLauncherStartDesktop:               "启动桌面端",
	KeyLauncherOpenDesktop:                "打开桌面端",
	KeyLauncherCloseDesktop:               "关闭桌面端",
	KeyLauncherRunningNote:                "关闭启动器窗口后，桌面端继续运行。可从托盘重新打开启动器。",
	KeyLauncherMyComponents:               "我的组件",
	KeyLauncherMyComponentsLede:           "按需开启，扩展 MyGo鲸屿的能力。",
	KeyLauncherSettingsOnlyHint:           "以下选项仅作用于启动器。完整应用设置请在桌面端中调整。",
	KeyLauncherOptRouteTitle:              "下载线路",
	KeyLauncherOptRouteDesc:               "桌面端安装与更新的下载来源。两条线路提供同一候选构建的相同文件；切换只影响之后发起的查询与下载，进行中的任务仍走原线路。",
	KeyLauncherOptQuitTitle:               "就绪后关闭启动器",
	KeyLauncherOptQuitDesc:                "桌面端就绪后自动关闭本窗口。若主窗口未打开，关闭启动器将退出应用。",
	KeyLauncherOptTrayTitle:               "关闭时最小化到托盘",
	KeyLauncherOptTrayDesc:                "开启后点关闭按钮窗口会收进托盘，组件继续运行；托盘菜单可重新打开或完全退出。",
	KeyLauncherOptAutoTitle:               "打开后自动启动桌面端",
	KeyLauncherOptAutoDesc:                "与桌面端「通用设置 → 启动时」同步。否：仅打开启动器；是：跳过启动器直接进桌面（启动器仍可从托盘右键打开）。存在更新确认、待导入或上次启动失败时不自动启动。",
	KeyLauncherOptAskTitle:                "发现新正式版时询问",
	KeyLauncherOptAskDesc:                 "关闭后仍可在「版本」中手动安装。草稿不会出现在 latest。",
	KeyLauncherRouteGitHub:                "GitHub",
	KeyLauncherRouteCNB:                   "CNB",
	KeyLauncherRouteGitHubDetail:          "国外线路 · GitHub Releases 直连",
	KeyLauncherRouteCNBDetail:             "国内线路 · CNB 镜像",
	KeyLauncherComponentsLede:             "按需安装独立工具与服务。鲸桥可统一管理模型，安装后自动连接桌面端的模型渠道。",
	KeyLauncherRetryStart:                 "重试启动",
	KeyLauncherStarting:                   "启动中…",
	KeyLauncherBadgeInstalled:             "已安装",
	KeyLauncherBadgeMissing:               "未就绪",
	KeyLauncherDiagTitle:                  "启动诊断",
	KeyLauncherRouteLine:                  "线路 GitHub",
	KeyLauncherRouteLabel:                 "线路",
	KeyLauncherRouteCurrentPrefix:         "当前线路",
	KeyLauncherComponentsEmpty:            "暂无已启用组件。",
	KeyLauncherLocalVersion:               "本机版本",
	KeyLauncherImportLede:                 "只读扫描官方 home。勾选后写入桌面 home，不修改官方目录；不复制项目工作区。将迁移会话、附件、技能、插件、MCP 与个性化设置。",
	KeyLauncherImportEmpty:                "尚未扫描。点击「重新扫描」探测 Electron 数据。",
	KeyLauncherImportPickDir:              "选择目录",
	KeyLauncherImportAddSkillDir:          "添加技能目录",
	KeyLauncherImportRescan:               "重新扫描",
	KeyLauncherImportRescanFirst:          "请先重新扫描再导入。",
	KeyLauncherImportByDir:                "按目录分组",
	KeyLauncherImportSelectAll:            "全选本栏",
	KeyLauncherImportSelectNone:           "全不选",
	KeyLauncherImportCatSessions:          "会话",
	KeyLauncherImportCatSkills:            "技能",
	KeyLauncherImportCatPlugins:           "插件",
	KeyLauncherImportCatMCP:               "MCP",
	KeyLauncherImportCatSettings:          "设置",
	KeyLauncherImportCatPresets:           "预设",
	KeyLauncherImportSummary:              "来源: %s\n目标: %s\n会话 %d · 技能 %d · 插件 %d · MCP %d · 设置 %d · 预设 %d",
	KeyLauncherImportSummarySkillDir:      "来源: %s\n目标: %s\n会话 %d · 技能 %d · 插件 %d · MCP %d · 设置 %d · 预设 %d\n已添加技能目录 %s（+%d）",
	KeyLauncherImportPickSourceTitle:      "选择导入来源",
	KeyLauncherImportAddSkillTitle:        "添加技能目录",
	KeyLauncherImportOverwrite:            "覆盖已有",
	KeyLauncherImportOverwriteDesc:        "写入时覆盖桌面 home 中已有同名项",
	KeyLauncherImportAttachments:          "导入附件",
	KeyLauncherImportAttachmentsDesc:      "一并迁移会话附件与媒体文件",
	KeyLauncherImportSelected:             "已选",
	KeyLauncherImportPluginsNote:          "勾选的插件将按原 GitHub/npm 版本规格重新安装。内置、本地和不支持的来源跳过；构建脚本需明确批准。",
	KeyLauncherImportChecklist:            "查看完整迁移清单",
	KeyLauncherUpdateAvailable:            "可更新至 v%s（当前 v%s）",
	KeyLauncherUpdateNow:                  "立即更新",
	KeyLauncherViewVersions:               "查看版本",
	KeyLauncherPluginFailTitle:            "插件加载失败",
	KeyLauncherPluginFailDisable:          "禁用并启动 MyGo鲸屿",
	KeyLauncherPluginFailOpen:             "查看插件排查",
	KeyLauncherInstallDesktopTitle:        "安装桌面端",
	KeyLauncherInstallDesktopDesc:         "未检测到本机安装的桌面端。选择下载线路后安装。",
	KeyLauncherInstallDesktopCTA:          "下载并安装桌面端",
	KeyLauncherInstallDesktopVerify:       "下载后校验文件完整性",
	KeyLauncherManageComponents:           "管理组件",
	KeyLauncherCompStart:                  "启动",
	KeyLauncherCompStartNamed:             "开启%s",
	KeyLauncherCompRetry:                  "重试启动",
	KeyLauncherCompInstall:                "安装",
	KeyLauncherCompStop:                   "停止",
	KeyLauncherCompManage:                 "管理",
	KeyLauncherCompOpenSettings:           "打开设置",
	KeyLauncherCompOfficial:               "官方组件",
	KeyLauncherCompUpdatable:              "可更新",
	KeyLauncherCompUpdateAvail:            "有更新 v%s",
	KeyLauncherCompKindService:            "服务",
	KeyLauncherCompKindTool:               "工具",
	KeyLauncherCompStateRunning:           "运行中",
	KeyLauncherCompStateStopped:           "已停止",
	KeyLauncherCompStateAvailable:         "可安装",
	KeyLauncherCompStateError:             "异常",
	KeyLauncherVerDev:                     "源码运行",
	KeyLauncherVerNeedInstall:             "需要先安装桌面端",
	KeyLauncherVerNeedInstallDesc:         "选择线路后可继续浏览版本列表并下载安装。",
	KeyLauncherCompCheckUpdate:            "检查并更新",
	KeyLauncherCompRollback:               "回滚至 v%s",
	KeyLauncherVerBackHome:                "回到首页安装",
	KeyLauncherVerInstallPath:             "安装位置  %s",
	KeyLauncherVerUpdateTo:                "更新到此版本",
	KeyLauncherVerAllList:                 "全部版本 · 点击版本行查看详情",
	KeyLauncherVerRefresh:                 "刷新",
	KeyLauncherVerCurrent:                 "当前",
	KeyLauncherVerLatest:                  "最新",
	KeyLauncherVerInstalled:               "已安装",
	KeyLauncherVerUpdateToV:               "更新至 v%s",
	KeyLauncherVerEmpty:                   "暂无已记录的版本产物。检查更新后将显示可用版本。",
	KeyLauncherImportEmptyItems:           "没有可列出的项。重新扫描后显示探测结果。",
	KeyLauncherImportEmptySessions:        "没有可列出的会话。重新扫描后显示探测结果。",
	KeyLauncherImportEmptySkills:          "没有可列出的技能。",
	KeyLauncherImportEmptyPlugins:         "没有可列出的插件。",
	KeyLauncherImportEmptyMCP:             "没有可列出的 MCP 配置。",
	KeyLauncherImportEmptySettings:        "没有可列出的设置项。",
	KeyLauncherImportEmptyPresets:         "没有可列出的预设。",
	KeyLauncherImportConflict:             "（冲突）",
	KeyLauncherImportUnsupported:          "（不支持）",
	KeyLauncherImportStart:                "开始导入",
	KeyLauncherImportLog:                  "导入日志",
	KeyLauncherImportCopy:                 "复制",
	KeyLauncherPluginsNeedRuntime:         "需要先安装桌面端运行时",
	KeyLauncherPluginsNeedRuntimeDesc:     "安装完成后可在此排查用户插件冲突。",
	KeyLauncherPluginsEmpty:               "未检测到插件，当前插件环境为空。",
	KeyLauncherPluginsUnknownHint:         "未能从日志确定具体插件。可逐项禁用下列插件后重新启动，以排查冲突。",
	KeyLauncherSettingsLede:               "",
	KeyTrajSearch:                         "搜索",
	KeyTrajDuration:                       "时长",
	KeyTrajTurns:                          "轮次",
	KeyTrajCalls:                          "调用",
	KeyTrajTimeline:                       "轨迹时间线",
	KeyTrajColInput:                       "输入",
	KeyTrajColOutput:                      "输出",
	KeyTrajColThink:                       "思考",
	KeyTrajColTime:                        "时间",
	KeyTrajColModel:                       "模型",
	KeyTrajColTools:                       "工具",
	KeyTrajLoadEarlier:                    "加载更早的历史",
	KeyTrajLoading:                        "正在加载轨迹…",
	KeyTrajEmpty:                          "无内容",
	KeyTrajDetailTitle:                    "事件详情",
	KeyTrajDetailHint:                     "选择一行查看概述 / 参数 / 结果 / 计时。",
	KeyTrajTabOverview:                    "概述",
	KeyTrajTabArgs:                        "参数",
	KeyTrajTabResult:                      "结果",
	KeyTrajTabTiming:                      "计时",
	KeyTrajNoArgs:                         "未捕获参数",
	KeyTrajNoResult:                       "未捕获结果",
	KeyTrajTimingLine:                     "时长 / 序号: %s",
	KeyTrajNoTiming:                       "未记录",
	KeyTrajCopy:                           "复制",
	KeyTrajToolResult:                     "工具结果",
	KeyTrajToolCallOnly:                   "（仅工具调用）",
	KeyTrajCompacted:                      "已压缩",
	KeyTrajDeliverables:                   "交付物",
	KeyTrajRoleUser:                       "用户",
	KeyTrajRoleAssistant:                  "助手",
	KeyTrajRoleSystem:                     "系统",
	KeyTrajInProgress:                     "进行中",
	KeyTrajKindTool:                       "工具",
	KeyTrajKindThink:                      "思考",
	KeyTrajKindMessage:                    "消息",
	KeyTrajTurnN:                          "第 %d 轮",
	KeySettingsDesktopDetectElectronTitle: "Electron 数据探测",
	KeySettingsDesktopCheckUpdatesTitle:   "检查更新",
	KeySettingsSectionAccount:             "账号",
	KeySettingsSectionGeneral:             "通用",
	KeySettingsSectionAppearance:          "外观",
	KeySettingsSectionInterface:           "界面",
	KeySettingsSectionModels:              "模型",
	KeySettingsSectionPlugins:             "插件",
	KeySettingsSectionSkills:              "技能",
	KeySettingsSectionMarket:              "市场",
	KeySettingsSectionRemote:              "远程",
	KeySettingsSectionMCP:                 "MCP",
	KeySettingsSectionAgentPresets:        "Agent 预设",
	KeySettingsSectionPet:                 "桌宠",
	KeySettingsSectionAbout:               "关于",
	KeySettingsSectionUsageStats:          "用量",
	KeySettingsSectionRemoteWorkspace:     "远程工作区",
	KeySettingsSectionOther:               "其他",
	KeyUsageOpen:                          "用量",
	KeyUsageTitle:                         "用量统计",
	KeyUsageBack:                          "返回",
	KeyUsageRefresh:                       "刷新",
	KeyUsageLoading:                       "加载中…",
	KeyUsagePluginNote:                    "原生面板已含热力格与柱状图（job/list）；更完整系列仍可由 harness 插件 dsh-usage-panel 提供。",
	KeyUsageAccount:                       "账号状态",
	KeyUsageBalance:                       "余额 (account/getBalance)",
	KeyUsageRecent:                        "最近活动",
	KeyUsageRecentEmpty:                   "暂无条目；选择会话后刷新可拉取 job/list。",
	KeyUsageNoData:                        "尚未加载",
	KeyUsageWebLink:                       "网页用量",
	KeyUsageChartTitle:                    "近 7 日任务数",
	KeyUsageChartNote:                     "原生柱状图（job/list StartedAt 近 7 日）。",
	KeyUsageHeatmapTitle:                  "近 28 日活动热力图",
	KeyUsageHeatmapNote:                   "原生热力格（job/list StartedAt）；插件 dsh-usage-panel 仍有更完整系列。",
	KeyUsageOpenWeb:                       "WebView2 打开用量页",
	KeyUsageOpenTopUp:                     "打开充值页",
	KeyUsageCopyURL:                       "复制用量链接",
	KeyUsageCopyTopUp:                     "复制充值链接",
	KeyUsageCopyBalance:                   "复制余额",
	KeyUsageKillJob:                       "终止",
	KeyUsageFollowJob:                     "跟踪",
	KeyUsageExpiry:                        "到期",
	KeySettingsAccountBonuses:             "未读奖励",
	KeySettingsAccountBonusesEmpty:        "暂无未读奖励",
	KeySettingsAccountBonusAck:            "已读",
	KeySettingsAccountRunningTasks:        "账号任务进行中",
	KeySettingsAboutAnalytics:             "产品分析",
	KeySettingsAboutAnalyticsOn:           "已启用",
	KeySettingsAboutAnalyticsOff:          "未启用",
	KeyDirPickTitle:                       "选择目录",
	KeyDirPickClose:                       "关闭",
	KeyDirPickEmpty:                       "此目录为空",
	KeyDirPickSelect:                      "选择",
	KeyDirPickSelectHere:                  "选择当前位置",
	KeyDirPickNewName:                     "新建文件夹名…",
	KeyDirPickCreate:                      "新建",
	KeyTerminalTitle:                      "终端",
	KeyTerminalShow:                       "显示终端",
	KeyTerminalHide:                       "隐藏终端",
	KeyTerminalUnavailable:                "终端不可用",
	KeyTerminalRestart:                    "重启",
	KeyTerminalCopyCwd:                    "复制路径",
	KeyGitFetch:                           "取",
	KeyGitPull:                            "拉",
	KeyGitPush:                            "推",
	KeyGitCommit:                          "提交",
	KeyGitCommitOK:                        "已提交",
	KeyGitFetchOK:                         "已获取",
	KeyGitPullOK:                          "已拉取",
	KeyGitPushOK:                          "已推送",
	KeyGitActionFailed:                    "Git 失败",
	KeyGitCommitPlaceholder:               "提交说明",
	KeyGitCommitConfirmTitle:              "确认提交",
	KeyGitCommitConfirmMessage:            "将执行 git add -A 并提交全部变更，继续？",
	KeyGitCommitCancel:                    "取消",
	KeyGitCommitNotDirty:                  "工作区无未提交变更",
	KeyWebViewOpen:                        "插件 Web",
	KeyWebViewTitle:                       "第三方插件界面",
	KeyWebViewBack:                        "返回",
	KeyWebViewNotOpen:                     "插件页面尚未打开。",
	KeyWebViewOpenBrowser:                 "在浏览器中打开 Harness",
	KeyWebViewOpenHost:                    "打开内嵌 WebView2",
	KeyWebViewCloseHost:                   "关闭内嵌 WebView2",
	KeyWebViewCopyURL:                     "复制链接",
	KeyWebViewOrigin:                      "Harness 地址",
	KeyBrowserPreviewOpen:                 "浏览器预览",
	KeyBrowserPreviewTitle:                "浏览器预览",
	KeyBrowserPreviewBack:                 "返回",
	KeyBrowserPreviewURLPlaceholder:       "输入 https://…",
	KeyBrowserPreviewGo:                   "打开",
	KeyBrowserPreviewOpenExternal:         "系统浏览器",
	KeyBrowserPreviewCopyURL:              "复制链接",
	KeyBrowserPreviewReload:               "刷新",
	KeyBrowserPreviewCloseHost:            "关闭 WebView",
	KeyBrowserPreviewHint:                 "输入任意 http(s) 地址并在内嵌 WebView2 中预览（与插件插槽 Web 面板独立）。",
	KeyBrowserPreviewInvalidURL:           "无效的 http(s) 地址",
	KeyChatCitations:                      "来源",
	KeyRemoteOpen:                         "远程",
	KeyRemoteTitle:                        "远程工作区",
	KeyRemoteBack:                         "返回",
	KeyRemoteWorkspaceToggle:              "远程工作区 (remoteWorkspaceEnabled)",
	KeyRemoteToggleOn:                     "开启",
	KeyRemoteToggleOff:                    "关闭",
	KeyRemoteHarnessOverlay:               "Harness dsh-remote 覆盖",
	KeyRemotePhoneGit:                     "手机 Git（仅当前会话工作区）",
	KeyRemotePairingEnabled:               "手机/LAN 远程 (remoteEnabled)",
	KeyRemoteMode:                         "remoteMode",
	KeyRemotePort:                         "remotePort",
	KeyRemoteMux:                          "remote.mux WebSocket",
	KeyRemoteCredentials:                  "credentials remoteToken",
	KeyRemoteOn:                           "开",
	KeyRemoteOff:                          "关",
	KeyRemoteMuxWaiting:                   "等待 Harness",
	KeyRemoteMuxConnected:                 "已连接",
	KeyRemoteMuxDisconnected:              "未连接",
	KeyRemoteCredPresent:                  "已配置",
	KeyRemoteCredAbsent:                   "未配置",
	KeyRemoteWorkspaceOff:                 "remoteWorkspaceEnabled 为关：启动时不会挂载 dsh-remote 覆盖。",
	KeyRemoteSummaryDisabled:              "远程工作区已关闭",
	KeyRemoteSummaryEnabled:               "远程工作区",
	KeyRemoteSSHTitle:                     "SSH 远程工作区",
	KeyRemoteSSHStatus:                    "SSH 就绪",
	KeyRemoteSSHSummaryReady:              "配置齐全，可以启动配对；可用下方 TCP 探测验证端口。",
	KeyRemoteSSHSummaryIncomplete:         "配置不完整，请补齐下列项后再启动配对。",
	KeyRemoteSSHSummaryDisabled:           "SSH 模式已选，但远程工作区或配对未启用。",
	KeyRemoteSSHSummaryInactive:           "当前连接方式不是 SSH。",
	KeyRemoteSSHStatusOff:                 "未启用",
	KeyRemoteSSHReady:                     "配置齐全（可启动配对）",
	KeyRemoteSSHIncomplete:                "配置不完整",
	KeyRemoteSSHDisabled:                  "工作区或配对未启用",
	KeyRemoteSSHMissing:                   "缺少",
	KeyRemoteSSHBind:                      "remoteBindAddress",
	KeyRemoteSSHLanTLS:                    "remoteLanTls",
	KeyRemoteSSHRelayToken:                "credentials remoteRelayToken",
	KeyRemoteSSHPairingOff:                "remoteEnabled 为关：SSH 模式仍不会启动配对守护进程。",
	KeyRemoteSSHProbe:                     "TCP 探测",
	KeyRemoteSSHProbeBtn:                  "探测配对端口",
	KeyChatLoadEarlier:                    "加载更早消息",
	KeyChatLoadEarlierBusy:                "加载中…",
	KeyChatEdit:                           "编辑",
	KeyChatEditHint:                       "正在重新编辑此消息",
	KeyChatEditRunning:                    "会话运行中，暂时无法编辑",
	KeyChatEditUnsupported:                "仅支持编辑纯文本消息",
	KeyChatEditStale:                      "当前消息已变化或会话正在运行，请取消编辑后重试",
	KeyChatCopy:                           "复制",
	KeyChatQuote:                          "引用",
	KeyChatReconnect:                      "重连",
	KeyChatDeliverables:                   "交付物",
	KeyChatCompaction:                     "上下文压缩",
	KeyChatImageClose:                     "关闭",
	KeyPlanBoardTitle:                     "计划面板",
	KeyPlanBoardExpand:                    "展开",
	KeyPlanBoardCollapse:                  "收起",
	KeyPlanBoardEmpty:                     "暂无计划 / 目标 / 队列 / 待办",
	KeyPlanBoardPlan:                      "计划",
	KeyPlanBoardPlanOn:                    "计划模式：开",
	KeyPlanBoardPlanOff:                   "计划模式：关",
	KeyPlanBoardGoal:                      "目标",
	KeyPlanBoardNoGoal:                    "无当前目标",
	KeyPlanBoardQueue:                     "队列",
	KeyPlanBoardQueueEmpty:                "队列为空",
	KeyPlanBoardQueueEdit:                 "编辑",
	KeyPlanBoardQueueSteer:                "插话",
	KeyPlanBoardQueueRemove:               "移除",
	KeyPlanBoardTodos:                     "待办",
	KeyPlanBoardTodosEmpty:                "无待办",
	KeyPlanBoardTodoDone:                  "已完成 %d",
	KeyPlanBoardTodoActive:                "进行中 %d",
	KeyPlanBoardTodoPending:               "待处理 %d",
	KeyPlanBoardGoalCreate:                "创建目标",
	KeyPlanBoardGoalEdit:                  "保存目标",
	KeyPlanBoardGoalComplete:              "完成",
	KeyPlanBoardGoalClear:                 "清除",
	KeyPlanBoardGoalPause:                 "暂停",
	KeyPlanBoardGoalResume:                "继续",
	KeyPlanBoardGoalDraftPlaceholder:      "目标描述…",
	KeyTodoPanelTitle:                     "任务",
	KeyChatRunFailed:                      "本轮运行失败",
	KeyChatTurnDone:                       "已完成",
	KeyChatTurnDoneSec:                    "已完成，用时 %d 秒",
	KeyChatTurnDoneMin:                    "已完成，用时 %d 分",
	KeyChatTurnDoneMinSec:                 "已完成，用时 %d 分 %d 秒",
	KeyScheduleOpen:                       "定时任务",
	KeyScheduleTitle:                      "定时任务",
	KeyScheduleBack:                       "返回",
	KeyScheduleRefresh:                    "刷新",
	KeyScheduleLoading:                    "加载中…",
	KeyScheduleNote:                       "数据来自 schedule/list 与 schedule/catalog（当前会话）。新建定时任务由对话/智能体写入；此处可改标题、查历史与删除（schedule/update|history|delete）。",
	KeyScheduleEmpty:                      "暂无定时任务；选择会话后刷新。",
	KeyScheduleDelete:                     "删除",
	KeyScheduleHistory:                    "历史",
	KeyScheduleCopyHistory:                "复制历史",
	KeyScheduleEditTitle:                  "标题",
	KeyScheduleSaveTitle:                  "保存标题",
	KeyScheduleEditPrompt:                 "提示词",
	KeyScheduleSavePrompt:                 "保存提示词",
	KeyAgentsTitle:                        "智能体团队",
	KeyAgentsClose:                        "关闭",
	KeyAgentsRootLabel:                    "根会话",
	KeyAgentsRootFallback:                 "当前会话",
	KeyAgentsPickSession:                  "选择一个会话后查看子代理。",
	KeyAgentsChildren:                     "子代理",
	KeyAgentsEmpty:                        "暂无子代理会话。",
	KeyAgentsStop:                         "停止",
	KeyAgentsInterrupt:                    "中断",
	KeyAgentsInterruptAll:                 "全部中断",
	KeyAgentsPromptPlaceholder:            "向子代理发送消息…",
	KeyAgentsPromptSend:                   "发送",
	KeyRemoteMobileTitle:                  "手机远程",
	KeyRemoteMobileNote:                   "显示远程配置与已登记设备；当前桌面尚未接入 offer v2 配对运行时。",
	KeyRemoteMobileAppBase:                "remoteAppBaseUrl",
	KeyRemoteMobileDevices:                "已配对设备数",
	KeyRemoteMobileNeedToken:              "已开 remoteEnabled 但未配置 remoteToken，手机无法配对。",
	KeyRemotePhoneTitle:                   "手机远程",
	KeyRemotePhoneHowTo:                   "下方入口仅打开已配置的手机网页，不代表服务在线或配对成功。配对 daemon 端口不能作为手机网页地址。",
	KeyRemotePhoneStatus:                  "手机远程就绪",
	KeyRemotePhoneReady:                   "配置项齐全（未验证配对服务）",
	KeyRemotePhoneIncomplete:              "配置不完整",
	KeyRemotePhoneDisabled:                "配对未启用",
	KeyRemotePhoneMissing:                 "缺少",
	KeyRemotePhoneRelayToken:              "credentials remoteRelayToken",
	KeyRemotePhoneLanBind:                 "remoteBindAddress (LAN)",
	KeyRemotePhoneLanTLS:                  "remoteLanTls",
	KeyRemotePhoneConnectURL:              "手机网页入口",
	KeyRemotePhoneQRPlaceholder:           "配对二维码暂不可用：尚未接入 offer v2 与中继连接状态，不能用普通网页地址代替配对码。",
	KeyRemotePhoneURLUnavailable:          "未配置有效手机网页入口。remoteAppBaseUrl 需为不含凭据、查询参数或片段的 HTTP(S) 网页地址。",
	KeyRemotePhoneCopyURL:                 "复制网页地址",
	KeyRemotePhoneOpenURL:                 "打开 URL",
	KeyRemotePhoneRefresh:                 "刷新状态",
	KeyRemotePhoneDeviceList:              "已配对设备",
	KeyRemotePhoneCopied:                  "连接 URL 已复制到剪贴板。",
	KeyRemotePhoneOpened:                  "已在 WebView / 浏览器打开连接 URL。",
	KeyRemotePhoneOpenFailed:              "打开连接 URL 失败",
	KeyRemotePhoneNoURL:                   "当前无可用连接 URL。",
	KeyRemotePhoneRefreshed:               "已从 credentials 刷新配对状态与设备列表。",
	KeyRemotePhoneCopyFailed:              "复制到剪贴板失败",
	KeyIMOpen:                             "消息",
	KeyIMOpenWeb:                          "打开 dsh-im Web",
	KeyIMChannels:                         "频道 / 会话",
	KeyIMCompose:                          "发送消息",
	KeyIMSend:                             "发送",
	KeyRemoteSSHTunnelTitle:               "SSH 会话",
	KeyRemoteSSHTunnelNote:                "公钥登录 + 可选本地转发（ssh-tunnel.json）。",
	KeyRemoteSSHTunnelState:               "状态",
	KeyRemoteSSHFieldHost:                 "主机",
	KeyRemoteSSHFieldUser:                 "用户",
	KeyRemoteSSHFieldPort:                 "端口",
	KeyRemoteSSHFieldIdentity:             "私钥",
	KeyRemoteSSHFieldForward:              "转发",
	KeyRemoteSSHTunnelSave:                "保存隧道配置",
	KeyRemoteSSHTunnelConnect:             "连接",
	KeyRemoteSSHTunnelDisconnect:          "断开",
	KeyRemoteSSHHostUnknownTitle:          "未知 SSH 主机密钥",
	KeyRemoteSSHHostUnknownMessage:        "主机 %s 的密钥尚未信任。\n类型: %s\n指纹: %s\n\n是否将此密钥加入 known_hosts 并继续连接？",
	KeyRemoteSSHHostAccept:                "信任并连接",
	KeyRemoteSSHHostDecline:               "拒绝",
	KeyRemoteSSHHostDeclined:              "已拒绝未知主机密钥；未连接",
	KeyRemoteSSHHostChangedTitle:          "SSH 主机密钥已变更",
	KeyRemoteSSHHostChangedMessage:        "主机 %s 的密钥与 known_hosts 中记录的不符（远程出示 %s %s）。为防止中间人攻击，已拒绝连接。请手动核验后更新 known_hosts。",
	KeyRemoteSSHHostChangedStatus:         "主机密钥已变更，已拒绝连接",
	KeyRemoteSSHConnected:                 "SSH 已连接",
	KeyIMTitle:                            "即时消息",
	KeyIMBack:                             "返回",
	KeyIMRefresh:                          "刷新",
	KeyIMLoading:                          "加载中…",
	KeyIMStubNote:                         "连接 Host 后显示 IM 会话。",
	KeyIMHarnessOverlay:                   "Harness dsh-im 覆盖",
	KeyIMPluginDisabled:                   "config disabledPlugins 含 dsh-im",
	KeyIMMux:                              "remote.mux WebSocket",
	KeyIMGeneratedRPC:                     "codegen im/* RPC",
	KeyIMOn:                               "是",
	KeyIMOff:                              "否",
	KeyIMMuxWaiting:                       "等待 Harness",
	KeyIMMuxConnected:                     "已连接",
	KeyIMMuxDisconnected:                  "未连接",
	KeyIMRPCYes:                           "已注册 list RPC",
	KeyIMRPCNo:                            "无 im/* 端点",
	KeyIMRPCPartial:                       "有 im/* 但无 list",
	KeyIMNoRPC:                            "internal/dsh/api 中未发现 im/* 方法（dsh-im 插件 RPC 未纳入 codegen）。",
	KeyIMConnectionPlaceholder:            "连接占位：Harness Ready 且 mux 就绪后，完整 IM UI 仍由 dsh-im 插件提供；codegen 接入 im/* 后本页将自动拉列表 RPC。",
	KeyIMListVia:                          "列表 RPC",
	KeyIMListEmpty:                        "暂无频道（本地或列表 RPC）。",
	KeyIMMethods:                          "已注册的 im/* 方法",
	KeyIMPluginOff:                        "dsh-im 已在 disabledPlugins 中禁用。",
	KeyIMLocalMode:                        "本地 / 诊断模式：无 im/* 响应；频道与历史仍写入 userData/im/，并显示 dsh-im 插件清单诊断。",
	KeyIMHistory:                          "消息历史",
	KeyIMHistoryEmpty:                     "此频道尚无消息。发送一条，或刷新以拉取服务端历史。",
	KeyIMNoChannel:                        "请选择一个频道。",
	KeyIMYou:                              "我",
	KeyIMPeer:                             "对方",
	KeyIMBridge:                           "同步桥",
	KeyIMBridgeRPC:                        "im/* RPC",
	KeyIMBridgeCordis:                     "Cordis connection.status",
	KeyIMBridgeInventory:                  "pluginInventory",
	KeyIMBridgeLocal:                      "仅本地",
	KeyIMBridgeIdle:                       "未刷新",
	KeyIMPluginDiag:                       "连接诊断",
	KeyIMHistoryVia:                       "历史 RPC",
	KeyIMHistorySynced:                    "已合并服务端历史到本地。",
	KeyMarketOpen:                         "插件市场",
	KeyMarketTitle:                        "插件市场",
	KeyMarketBack:                         "返回",
	KeyMarketRefresh:                      "刷新",
	KeyMarketLoading:                      "加载中…",
	KeyMarketHarnessOverlay:               "market 覆盖",
	KeyMarketPluginDisabled:               "市场插件已禁用",
	KeyMarketMux:                          "Mux 连接",
	KeyMarketGeneratedRPC:                 "pluginManager RPC",
	KeyMarketInstallToolchain:             "本机安装工具链",
	KeyMarketOn:                           "是",
	KeyMarketOff:                          "否",
	KeyMarketMuxWaiting:                   "等待 Harness",
	KeyMarketMuxConnected:                 "已连接",
	KeyMarketMuxDisconnected:              "未连接",
	KeyMarketRPCYes:                       "含 listPlugins",
	KeyMarketRPCNo:                        "无 listPlugins",
	KeyMarketRPCPartial:                   "有其他 pluginManager 方法",
	KeyMarketNoRPC:                        "codegen 未含 pluginManager/listPlugins 时无法列出插件。",
	KeyMarketListVia:                      "列表 RPC",
	KeyMarketListEmpty:                    "（暂无已安装插件）",
	KeyMarketMethods:                      "pluginManager 方法",
	KeyMarketPluginOff:                    "ui-settings-market 已在 disabledPlugins 中禁用。",
	KeyMarketInstallHint:                  "输入 github:owner/repo 或 github:owner/repo#ref，通过本机 dsh plugin add 安装（无需 DSH_DESKTOP_INSTALL_URL）。",
	KeyMarketInstallAdd:                   "从 GitHub 安装",
	KeyMarketInstallSpecLabel:             "安装规格",
	KeyMarketInstallPlaceholder:           "github:owner/repo",
	KeyMarketInstallButton:                "安装",
	KeyMarketRemove:                       "卸载",
	KeyMarketAllowBuildsButton:            "允许构建并重试",
	KeyMarketAllowBuilds:                  "需要 allowBuilds",
	KeyMarketInstallNotReady:              "需要 Node.js 22+ 与已构建的 vendored harness（setup-harness）。",
	KeyMarketInstallOK:                    "安装成功；正在重启 Harness…",
	KeyMarketInstallRunning:               "正在安装…",
	KeyMarketInstallLog:                   "安装日志",
	KeyMarketCopyLog:                      "复制日志",
	KeyMarketOpenHarness:                  "打开完整市场",
	KeyMarketRegistry:                     "注册源",
	KeyMarketExemptions:                   "版本豁免",
	KeyMarketCancelInstall:                "取消安装",
	KeyMarketCancelOK:                     "已请求取消安装",
	KeyMarketIncompatible:                 "版本不兼容",
	KeyMarketAcceptExemption:              "接受风险并豁免",
	KeyAssistantOpen:                      "助理",
	KeyAssistantTitle:                     "鲸鱼娘与 dshbot",
	KeyAssistantBack:                      "返回",
	KeyAssistantStubNote:                  "Whale 使用专属助手；Dshbot 先选择 Bot 或群聊，再解析其专属会话。完整设置与管理仍可打开插件页面。",
	KeyAssistantWhaleTitle:                "dsh-whale 助理",
	KeyAssistantDshbotTitle:               "dshbot",
	KeyAssistantConfigEnabled:             "config 已启用",
	KeyAssistantVendorPresent:             "runtime/vendor 存在",
	KeyAssistantOn:                        "是",
	KeyAssistantOff:                       "否",
	KeyAssistantToggleOn:                  "启用",
	KeyAssistantToggleOff:                 "关闭",
	KeyAssistantOpenWhaleWeb:              "高级：WebView2 打开鲸鱼娘",
	KeyAssistantOpenDshbotWeb:             "高级：WebView2 打开 dshbot",
	KeyAssistantRestartNote:               "切换开关后需重启 Harness 才会重新挂载 overlay。",
	KeyAssistantStatusTitle:               "状态",
	KeyAssistantLastWebURL:                "最近 WebView URL",
	KeyAssistantLastAction:                "最近原生操作",
	KeyAssistantComposeTitle:              "快捷提问",
	KeyAssistantComposeHint:               "Whale 发送给专属助手；Dshbot 发送给明确选择的 Bot 或群聊。失败时保留草稿。",
	KeyAssistantComposePlaceholder:        "输入原生快捷提问…",
	KeyAssistantSend:                      "发送",
	KeyAssistantNoSession:                 "无选中会话 — 请先在聊天侧栏创建或选择会话，或点「打开会话」。",
	KeyAssistantNoRPC:                     "会话 / 插件 RPC 未就绪 — 已打开当前模式 WebView 作为兜底。",
	KeyAssistantPromptSent:                "已发送到当前会话。",
	KeyAssistantPluginRPCTitle:            "插件控制面（HTTP）",
	KeyAssistantPluginRPCHint:             "直连 harness 上的 /dsh-whale/* 与 /dshbot/*（client-request 信封），不经过 typert gen-api。",
	KeyAssistantWhaleEnsure:               "确保助理会话",
	KeyAssistantWhaleCatalog:              "鲸鱼目录",
	KeyAssistantWhalePetChat:              "pet/chat（用上方草稿）",
	KeyAssistantWhalePetState:             "pet/state 历史",
	KeyAssistantWhalePetLook:              "pet/look 看图",
	KeyAssistantDshbotDescribe:            "dshbot describe",
	KeyAssistantDshbotCatalog:             "bot 目录",
	KeyAssistantDshbotActivity:            "bot 动态",
	KeyAssistantDshbotRosterTitle:         "Bot / 群聊列表",
	KeyAssistantDshbotRosterRefresh:       "刷新列表",
	KeyAssistantDshbotRosterEmpty:         "尚无缓存。点「刷新列表」从 dshbot/describe 拉取。",
	KeyAssistantDshbotRosterUpdated:       "更新时间",
	KeyAssistantDshbotRosterBot:           "Bot",
	KeyAssistantDshbotRosterRoom:          "群聊",
	KeyAssistantDshbotRosterOpen:          "打开",
	KeyAssistantModeLabel:                 "模式",
	KeyAssistantModeTitle:                 "助理模式",
	KeyAssistantModeHint:                  "切换 Whale / Dshbot：选择助手类型及对应插件页面。",
	KeyAssistantModeWhale:                 "Whale",
	KeyAssistantModeDshbot:                "Dshbot",
	KeyAssistantOpenModeWeb:               "打开当前模式 WebView",
	KeyAssistantOpenSession:               "打开会话",
	KeyAssistantBusy:                      "处理中…",
	KeyAssistantLastErr:                   "最近错误",
	KeyAssistantSessionID:                 "助理会话 ID",
	KeyAssistantHistoryTitle:              "原生历史",
	KeyAssistantHistoryEmpty:              "尚无本地原生提问记录。发送一条开始记录。",
	KeyAssistantWhaleRPCSent:              "已通过 dsh-whale/pet/chat 完成一轮。",
	KeyAssistantWhaleRPCErr:               "鲸鱼娘 RPC 失败",
	KeyAssistantDshbotRouted:              "已发送到所选 Bot 的专属会话。",
	KeyAssistantDshbotProbeOK:             "dshbot/describe 可用。",
	KeyAssistantDshbotProbeErr:            "dshbot/describe 不可用",
	KeyAssistantEnsuring:                  "正在 ensure 鲸鱼娘会话…",
	KeyAssistantChatting:                  "正在 pet/chat…",
}

var en = map[string]string{
	KeyChatManualCompaction:               "Manual compaction",
	KeyChatCompactionCounts:               "Compacted %d history items (~%d tokens)",
	KeyChatCompactionSummaryMissing:       "Summary not included in loaded history",
	KeyCommandTitle:                       "Command",
	KeyCommandRunning:                     "Running…",
	KeyCommandDone:                        "Completed",
	KeyCommandFailed:                      "Failed",
	KeyCommandCatalogLoading:              "Loading commands…",
	KeyCommandCatalogRetry:                "Retry command list",
	KeyAppName:                            "MyGo鲸屿",
	KeyTrayOpen:                           "Open main window",
	KeyTrayQuit:                           "Quit",
	KeyQuitConfirmTitle:                   "Tasks still running",
	KeyQuitConfirmMessage:                 "%d session(s) are still running. Quit anyway?",
	KeyQuitConfirmQuit:                    "Quit",
	KeyQuitConfirmCancel:                  "Cancel",
	KeyTitlebarClose:                      "Close",
	KeyTitlebarMinimize:                   "Minimize",
	KeyTitlebarMaximize:                   "Maximize",
	KeyStatusPlaceholder:                  "Skeleton: window and tray ready",
	KeyBootTitle:                          "Starting Harness",
	KeyBootStarting:                       "Starting local service…",
	KeyBootReady:                          "Local service ready",
	KeyBootFailed:                         "Start failed",
	KeyBootStopped:                        "Stopped",
	KeyBootRetry:                          "Retry",
	KeyBootExport:                         "Export log",
	KeyBootWaiting:                        "Waiting for output…",
	KeyBootPlugins:                        "Loading plugins %d/%d",
	KeyBootPluginsDetail:                  "%s",
	KeyBootPluginsIndeterminate:           "Loading plugins…",
	KeySidebarSessions:                    "Sessions",
	KeySidebarRefresh:                     "Refresh",
	KeySidebarNew:                         "New",
	KeySidebarLoading:                     "Loading sessions…",
	KeySidebarEmpty:                       "No sessions yet",
	KeySidebarRename:                      "Rename",
	KeySidebarRenameCancel:                "Cancel",
	KeySidebarSearch:                      "Search sessions…",
	KeySidebarSearchLoading:               "Searching…",
	KeySidebarArchive:                     "Archive",
	KeySidebarUnarchive:                   "Unarchive",
	KeySidebarArchivedSection:             "Archived",
	KeySidebarBrowseWorkspace:             "Choose workspace…",
	KeySidebarRevealWorkspace:             "Show in file manager",
	KeySidebarRenameWorkspace:             "Rename workspace…",
	KeySidebarRenameWorkspaceSave:         "Save workspace name",
	KeySidebarDeleteWorkspace:             "Delete workspace",
	KeySidebarMoveUp:                      "Move to top",
	KeySidebarDelete:                      "Delete",
	KeySidebarCopyID:                      "Copy ID",
	KeySidebarFork:                        "Fork",
	KeyWorkspaceRegistered:                "Workspace registered: %s",
	KeySidebarDeleteConfirm:               "Delete this session? This cannot be undone.",
	KeySidebarPin:                         "Pin",
	KeySidebarUnpin:                       "Unpin",
	KeySidebarPinnedPrefix:                "Pin · ",
	KeySidebarBrand:                       "MyGo鲸屿",
	KeySidebarMarket:                      "Plugins",
	KeySidebarSchedule:                    "Scheduled tasks",
	KeySidebarAssistant:                   "Whale assistant",
	KeySidebarWorkspaceSection:            "Workspace",
	KeySidebarNewSessionFallback:          "New session",
	KeyChatEmpty:                          "Select a session to chat",
	KeyChatHeroSlogan:                     "Into the Unknown",
	KeyChatHeroPreview:                    "Preview",
	KeyChatHeroChooseWorkspace:            "Choose workspace",
	KeyChatThinking:                       "Thinking",
	KeyChatConnConnecting:                 "Connecting…",
	KeyChatConnReconnecting:               "Disconnected — reconnecting…",
	KeyChatUsageJobsToday:                 "Jobs today",
	KeyChatUsageBalance:                   "Balance",
	KeyChatFeedbackUp:                     "👍",
	KeyChatFeedbackDown:                   "👎",
	KeyChatFeedbackSelectedMark:           " ✓",
	KeyChatFeedbackCommentPlaceholder:     "Optional: why this was unhelpful…",
	KeyChatFeedbackSend:                   "Send feedback",
	KeyComposerModel:                      "Model",
	KeyComposerPermission:                 "Permissions",
	KeyComposerPermissionDefault:          "Edit within workspace",
	KeyComposerAgentPreset:                "Agent preset",
	KeyComposerSend:                       "Send",
	KeyComposerStop:                       "Stop",
	KeyComposerModeQueue:                  "Queue",
	KeyComposerModeSteer:                  "Steer",
	KeyComposerPlaceholder:                "Message or run a task, / commands, @ files or sessions",
	KeyComposerPlaceholderHero:            "Describe what you want to build, / commands, @ files or sessions",
	KeyComposerNoSession:                  "No session selected",
	KeyComposerWaiting:                    "Waiting for service…",
	KeyComposerDropHint:                   "Drop to attach @workspace paths",
	KeyComposerSkills:                     "Skills",
	KeyComposerSkillUserOnly:              "User only",
	KeyComposerSkillsFilter:               "Filter skills…",
	KeyComposerSkillsHubSearch:            "Search Hub",
	KeyComposerSkillsHubInstall:           "Install",
	KeyComposerSkillsDelete:               "Delete",
	KeyComposerSkillsCreate:               "Create skill",
	KeyComposerSkillsCreateName:           "New skill name…",
	KeyComposerSkillsToggleModel:          "Model-invocable",
	KeyComposerSkillsEmpty:                "No matching skills",
	KeyComposerPickDir:                    "Choose folder",
	KeyComposerPresetDefault:              "Standard",
	KeyComposerPickDirShort:               "Folder",
	KeyComposerPickImage:                  "Image",
	KeyComposerSpeech:                     "Speech file",
	KeyComposerSpeechCycle:                "STT▾",
	KeyComposerDirDialogTitle:             "Choose workspace folder",
	KeyComposerDirOutside:                 "Selected folder is outside the workspace",
	KeyComposerDirNoWorkspace:             "No workspace configured",
	KeyComposerChipEdit:                   "Edit tag",
	KeyComposerChipEditPlaceholder:        "@path or /skill",
	KeyComposerChipEditApply:              "Apply",
	KeyComposerChipEditCancel:             "Cancel",
	KeyFilesTitle:                         "Workspace files",
	KeyFilesFilter:                        "Filter files…",
	KeyFilesFilterEmpty:                   "No matching files",
	KeyFilesRefreshHint:                   "Tree lists from disk live",
	KeyFilesNoWorkspace:                   "No workspace configured",
	KeyFilesPickFile:                      "Select a file to preview",
	KeyFilesTooLarge:                      "File too large to preview",
	KeyFilesBinary:                        "Binary file; cannot preview",
	KeyFilesShowDiff:                      "Diff",
	KeyFilesEdit:                          "Edit",
	KeyFilesSave:                          "Save",
	KeyFilesCancelEdit:                    "Cancel",
	KeyFilesSaved:                         "Saved",
	KeyFilesModeSplit:                     "Split preview",
	KeyFilesModeEdit:                      "Edit only",
	KeyFilesModeHighlight:                 "Highlight only",
	KeyFilesModeNext:                      "Next",
	KeyUpdateDownload:                     "Download & install",
	KeyUpdateDownloadTitle:                "Update install",
	KeySettingsAccountNav:                 "Account & login",
	KeySettingsAccountTitle:               "Account & login",
	KeySettingsAccountHint:                "Harness account state (account/getState, getProfile, getBalance).",
	KeySettingsAccountStub:                "Stub while Harness is disconnected; refresh after connect for a summary.",
	KeySettingsAccountCredHint:            "Edit API keys on the Credentials page in the left nav.",
	KeySettingsAccountStatus:              "Status",
	KeySettingsAccountEmail:               "Email",
	KeySettingsAccountName:                "Name",
	KeySettingsAccountPlan:                "Plan",
	KeySettingsAccountUsageURL:            "Usage URL",
	KeySettingsAccountTopUpURL:            "Top-up URL",
	KeySettingsAccountBalance:             "Balance summary",
	KeySettingsAccountAttempt:             "Sign-in phase",
	KeySettingsAccountAuthorize:           "Authorize URL",
	KeySettingsAccountSignIn:              "Sign in",
	KeySettingsAccountSignOut:             "Sign out",
	KeySettingsAccountCancelSignIn:        "Cancel sign-in",
	KeySettingsAboutNav:                   "About",
	KeySettingsAboutTitle:                 "About MyGo鲸屿",
	KeySettingsAboutHint:                  "Native MyGo shell version and data paths.",
	KeySettingsAboutApp:                   "App",
	KeySettingsAboutVersion:               "Version",
	KeySettingsAboutID:                    "identifier",
	KeySettingsAboutUserData:              "userData",
	KeySettingsAboutVendor:                "vendor",
	KeySettingsAboutRevealData:            "Open data folder",
	KeySettingsAboutRevealLogs:            "Open logs folder",
	KeySettingsAboutRevealVendor:          "Open vendor folder",
	KeySettingsAboutCopyVersion:           "Copy version",
	KeySettingsAboutCopyData:              "Copy path",
	KeySettingsAppearanceNav:              "Appearance",
	KeySettingsAppearanceTitle:            "Appearance",
	KeySettingsAppearanceHint:             "Theme and wallpaper (writes config.json).",
	KeySettingsAppearanceClearWallpaper:   "Clear wallpaper",
	KeySettingsAppearanceBrowseWallpaper:  "Browse wallpaper…",
	KeySettingsAppearanceWallpaperDir:     "wallpapers/",
	KeySettingsInterfaceNav:               "Interface",
	KeySettingsInterfaceTitle:             "Interface",
	KeySettingsInterfaceHint:              "Language and window behavior (writes config.json). Theme and wallpaper live on Appearance; workspace and port on Desktop.",
	KeySettingsInterfaceAutoStart:         "Auto-start desktop on launch",
	KeySettingsInterfaceAutoStartDesc:     "When a workspace is set, start the harness desktop at app launch (same as the launcher toggle).",
	KeyChatPaneTabChat:                    "Chat",
	KeyChatPaneTabTrajectory:              "Trajectory",
	KeyChatBandTitle:                      "Quick actions",
	KeyChatControlNote:                    "Session control",
	KeySettingsInterfaceViewTabs:          "Chat and Trajectory tabs",
	KeySettingsInterfaceViewTabsDesc:      "Show the Chat and Trajectory switcher in the session header. Turning this off keeps the current view.",
	KeySettingsInterfaceAutoAllow:         "Auto-allow tools",
	KeySettingsInterfaceAutoAllowDesc:     "Automatically approve tool approval requests; when off, use Allow / Always allow / Deny on the card.",
	KeySettingsModelsNav:                  "Models",
	KeySettingsModelsTitle:                "Models",
	KeySettingsModelsHint:                 "Session model catalog (session/modelCatalog).",
	KeySettingsModelsDiscover:             "Discover providers",
	KeySettingsModelsPicker:               "Default model",
	KeySettingsModelsEmpty:                "No models yet (needs Harness Ready + catalog refresh).",
	KeySettingsModelsSession:              "Session",
	KeySettingsModelsNoSession:            "Catalog can refresh without a session; apply needs a selected session.",
	KeySettingsMCPNav:                     "MCP",
	KeySettingsMCPTitle:                   "MCP servers",
	KeySettingsMCPHint:                    "Host MCP server list (mcpServers/list).",
	KeySettingsMCPEmpty:                   "No MCP servers yet.",
	KeySettingsMCPStubHint:                "Harness not connected or host MCP plugin not ready; refresh after connect.",
	KeySettingsMCPSchemaHint:              "Schema form namespaces on the left still work for related settings.",
	KeySettingsMCPEnabled:                 "Enabled",
	KeySettingsMCPDisabled:                "Disabled",
	KeySettingsMCPEnable:                  "Enable",
	KeySettingsMCPDisable:                 "Disable",
	KeySettingsMCPRetry:                   "Retry",
	KeySettingsMCPAuthorize:               "Authorize",
	KeySettingsMCPTest:                    "Test",
	KeySettingsMCPDelete:                  "Delete",
	KeySettingsMCPUpsert:                  "Add / update server",
	KeySettingsMCPUpsertHint:              "Paste mcpServers/upsert spec JSON (stdio / http, etc.).",
	KeySettingsMCPUpsertSave:              "Save to host",
	KeySettingsPluginsNav:                 "Plugins",
	KeySettingsPluginsTitle:               "Plugins",
	KeySettingsPluginsHint:                "Installed plugins (pluginManager/listPlugins).",
	KeySettingsPluginsEmpty:               "No installed plugins yet.",
	KeySettingsPluginsNoRPC:               "Cannot list plugins when codegen lacks pluginManager/listPlugins.",
	KeySettingsPluginsMarketHint:          "Open the market panel for browsing and installs.",
	KeySettingsPluginsOpenMarket:          "Open market",
	KeySettingsPluginsDisable:             "Disable",
	KeySettingsPluginsEnable:              "Enable",
	KeySettingsPluginsInspect:             "Inspect",
	KeySettingsSkillsNav:                  "Skills",
	KeySettingsSkillsTitle:                "Skills",
	KeySettingsSkillsHint:                 "Session skill list (skills/list).",
	KeySettingsSkillsEmpty:                "No skills yet (needs a selected session + refresh).",
	KeySettingsSkillsSession:              "Session",
	KeySettingsSkillsNoSession:            "Skill list requires a selected session.",
	KeySettingsSkillsModelInvocable:       "Model-invocable",
	KeySettingsSkillsUserOnly:             "User only",
	KeySettingsSkillsInsert:               "Insert into composer",
	KeySettingsSkillsInspect:              "Inspect",
	KeySettingsSkillsMakeUserOnly:         "Make user-only",
	KeySettingsSkillsMakeModel:            "Allow model invoke",
	KeySettingsSkillsHubTitle:             "Skill hub",
	KeySettingsSkillsHubHint:              "Search skillInventory/searchHub; install into the user skill root.",
	KeySettingsSkillsHubPlaceholder:       "Search skills…",
	KeySettingsSkillsHubSearch:            "Search",
	KeySettingsSkillsHubInstall:           "Install",
	KeySettingsSkillsEditTitle:            "Edit skill",
	KeySettingsSkillsEditDesc:             "Description",
	KeySettingsSkillsEditContent:          "Content (Markdown)",
	KeySettingsSkillsSaveUpdate:           "Save update",
	KeySettingsAgentPresetsNav:            "Agent presets",
	KeySettingsAgentPresetsTitle:          "Agent presets",
	KeySettingsAgentPresetsHint:           "Choose the default Agent preset for future tasks.",
	KeySettingsAgentPresetsPicker:         "Default preset",
	KeySettingsAgentPresetsEmpty:          "No presets yet (needs Harness Ready + list refresh).",
	KeySettingsAgentPresetsRead:           "View document",
	KeySettingsAgentPresetsSession:        "Session",
	KeySettingsAgentPresetsNoSession:      "List can refresh without a session; apply needs a selected session.",
	KeyFilesOpenWebView:                   "Open in WebView2",
	KeyFilesOpenExternal:                  "Open externally",
	KeyFilesOpenWith:                      "Open with",
	KeyFilesReveal:                        "Show in folder",
	KeyFilesPDFHint:                       "PDF: open in WebView2 or the system default app.",
	KeyFilesXLSXHint:                      "Spreadsheet: if native preview fails, open with the system app.",
	KeyFilesOfficeHint:                    "Office document: open with the system default app.",
	KeyFilesRenderPDF:                     "Render PDF preview",
	KeyGitPRView:                          "View PR",
	KeyGitPRCreate:                        "Draft PR",
	KeyGitPRPanel:                         "PR",
	KeyGitPRRefresh:                       "Refresh PRs",
	KeyGitPRClose:                         "Close",
	KeyGitPRCheckout:                      "Checkout",
	KeyGitPRCheckoutOK:                    "Checked out PR",
	KeyGitPRNoGH:                          "GitHub CLI (gh) not found",
	KeyGitPRNone:                          "No PR on this branch",
	KeyRightPaneFiles:                     "Files",
	KeyRightPaneDiff:                      "Diff",
	KeyRightPaneAgents:                    "Agents",
	KeyFrameCaptionAgents:                 "◎  Agent team",
	KeyFrameCaptionProjectDocs:            "▣  Project docs",
	KeyDiffTitle:                          "Workspace diff",
	KeyDiffRefresh:                        "Refresh",
	KeyDiffHint:                           "Press Refresh to run git diff",
	KeyDiffEmpty:                          "No uncommitted changes",
	KeyDiffNoWorkspace:                    "No workspace configured",
	KeyDiffNotRepo:                        "Not a Git repository",
	KeyInteractionAllow:                   "Allow",
	KeyInteractionAllowAlways:             "Always allow",
	KeyInteractionDeny:                    "Deny",
	KeyInteractionCallID:                  "Call ID",
	KeyInteractionCopyArgs:                "Copy args",
	KeyInteractionContinue:                "Continue",
	KeyInteractionManual:                  "Require approval",
	KeyInteractionAutoAllowOn:             "Tool approvals: auto-allow",
	KeyInteractionAutoAllowed:             "Auto-allowed",
	KeyBootExportSaved:                    "Log saved",
	KeySettingsOpen:                       "Settings",
	KeyDesktopTools:                       "Tools",
	KeySettingsTitle:                      "Harness settings",
	KeySettingsBack:                       "Back",
	KeySettingsNamespaces:                 "Namespaces",
	KeySettingsEmpty:                      "No settings namespaces",
	KeySettingsRefresh:                    "Refresh",
	KeySettingsLoading:                    "Loading…",
	KeySettingsSave:                       "Save",
	KeySettingsReplace:                    "Replace document",
	KeySettingsReplaceHint:                "Fully replace this namespace document with the current form (not a partial patch).",
	KeySettingsReplaceConfirmTitle:        "Confirm replace",
	KeySettingsReplaceConfirmMessage:      "This fully replaces the selected namespace settings document with the current drafts. Continue?",
	KeySettingsReplaceCancel:              "Cancel",
	KeySettingsReadOnly:                   "Settings are read-only",
	KeySettingsWritable:                   "Writable",
	KeySettingsRevision:                   "Revision",
	KeySettingsFilter:                     "Filter namespaces…",
	KeySettingsFilterEmpty:                "No matching namespaces",
	KeySettingsRevealSecrets:              "Show secrets",
	KeySettingsPickNS:                     "Select a namespace on the left",
	KeySettingsDesktopNav:                 "Desktop",
	KeySettingsDesktopTitle:               "Desktop & local config",
	KeySettingsDesktopHint:                "Writes config.json on disk (shared with the Electron data directory). Restart Harness after changing the port.",
	KeySettingsDesktopWorkspace:           "Workspace path",
	KeySettingsDesktopBrowseWorkspace:     "Browse…",
	KeySettingsDesktopPort:                "Harness port",
	KeySettingsDesktopTheme:               "Theme",
	KeySettingsDesktopLocale:              "UI language",
	KeySettingsDesktopCloseToTray:         "Close to tray",
	KeySettingsDesktopOpenAtLogin:         "Open at login",
	KeySettingsDesktopSaved:               "Local config saved",
	KeySettingsDesktopDetectElectron:      "Detect Electron data",
	KeySettingsDesktopCheckUpdates:        "Check for updates",
	KeySettingsCredNav:                    "Credentials",
	KeySettingsCredTitle:                  "Credentials (credentials.json)",
	KeySettingsCredHint:                   "Shared with the Electron data directory; save writes a DPAPI envelope when available. Check Show secrets to edit sensitive fields.",
	KeySettingsCredShow:                   "Show secrets",
	KeySettingsCredAPIKey:                 "API Key",
	KeySettingsCredClearAPIKey:            "Clear API Key",
	KeySettingsCredClearSecrets:           "Clear all secrets",
	KeySettingsCredBaseURL:                "Base URL",
	KeySettingsCredGithub:                 "GitHub Token",
	KeySettingsCredRemote:                 "Remote Token",
	KeySettingsCredRelay:                  "Remote Relay Token",
	KeySettingsCredSaved:                  "Credentials saved",
	KeySettingsDesktopWallpaper:           "Wallpaper",
	KeySettingsDesktopWallpaperNone:       "(none)",
	KeyLauncherOpen:                       "Launcher",
	KeyLauncherTitle:                      "MyGo鲸屿 launcher",
	KeyLauncherClose:                      "Close",
	KeyLauncherHint:                       "Update check, version, Electron data probe, and plugin triage. Set a workspace then start the desktop.",
	KeyLauncherVersion:                    "Version",
	KeyLauncherVendor:                     "vendored harness",
	KeyLauncherOn:                         "ready",
	KeyLauncherOff:                        "missing",
	KeyLauncherWorkspace:                  "Workspace",
	KeyLauncherWorkspaceHint:              "Absolute workspace path",
	KeyLauncherNoWorkspace:                "No workspace configured",
	KeyLauncherStart:                      "Start desktop",
	KeyLauncherScanImport:                 "Probe Electron data",
	KeyLauncherCheckUpdate:                "Check for updates",
	KeyLauncherOpenSettings:               "Open settings",
	KeyLauncherImportTitle:                "Import probe",
	KeyLauncherUpdateTitle:                "Updates",
	KeyLauncherPluginsTitle:               "disabledPlugins",
	KeyLauncherPluginsNone:                "No disabled plugin entries",
	KeyLauncherPluginRecoveryTitle:        "Plugin troubleshooting",
	KeyLauncherSkipUserPlugins:            "skip-user-plugins",
	KeyLauncherSkipUserPluginsOn:          "enabled (recovery)",
	KeyLauncherSkipUserPluginsOff:         "off",
	KeyLauncherEnableSkipUserPlugins:      "Enable skip-user-plugins",
	KeyLauncherClearPluginRecovery:        "Clear pluginRecovery",
	KeyLauncherRestartHarness:             "Restart harness",
	KeyLauncherBrandRole:                  "Launcher",
	KeyLauncherTabHome:                    "Home",
	KeyLauncherTabComponents:              "Components",
	KeyLauncherTabVersions:                "Versions",
	KeyLauncherTabImport:                  "Import",
	KeyLauncherTabPlugins:                 "Plugin triage",
	KeyLauncherTabSettings:                "Settings",
	KeyLauncherHomeLede:                   "Desktop and components, managed in one place.",
	KeyLauncherGotoVersions:               "Version management",
	KeyLauncherDesktopTitle:               "MyGo鲸屿 Desktop",
	KeyLauncherDesktopDesc:                "Chat on desktop, manage models and workspace.",
	KeyLauncherDesktopRunning:             "Desktop running",
	KeyLauncherDesktopStopped:             "Desktop not running",
	KeyLauncherStartDesktop:               "Start desktop",
	KeyLauncherOpenDesktop:                "Open desktop",
	KeyLauncherCloseDesktop:               "Stop desktop",
	KeyLauncherRunningNote:                "Closing the launcher keeps the desktop running. Reopen the launcher from the tray.",
	KeyLauncherMyComponents:               "My components",
	KeyLauncherMyComponentsLede:           "Enable components to extend MyGo鲸屿.",
	KeyLauncherSettingsOnlyHint:           "These options apply only to the launcher. Adjust full app settings on the desktop.",
	KeyLauncherOptRouteTitle:              "Download route",
	KeyLauncherOptRouteDesc:               "Source for desktop install and updates. Both routes serve the same candidate builds; switching only affects later queries and downloads.",
	KeyLauncherOptQuitTitle:               "Close launcher when ready",
	KeyLauncherOptQuitDesc:                "Automatically close this window once the desktop is ready. If the main window is not open, closing the launcher quits the app.",
	KeyLauncherOptTrayTitle:               "Minimize to tray on close",
	KeyLauncherOptTrayDesc:                "Close hides to the tray while components keep running; reopen or quit from the tray menu.",
	KeyLauncherOptAutoTitle:               "Auto-start desktop on open",
	KeyLauncherOptAutoDesc:                "Synced with desktop General → On startup. Off: launcher only. On: skip launcher into desktop (launcher still available from tray).",
	KeyLauncherOptAskTitle:                "Ask when a stable update is found",
	KeyLauncherOptAskDesc:                 "When off, you can still install manually under Versions. Drafts never appear in latest.",
	KeyLauncherRouteGitHub:                "GitHub",
	KeyLauncherRouteCNB:                   "CNB",
	KeyLauncherRouteGitHubDetail:          "International route · Direct GitHub Releases",
	KeyLauncherRouteCNBDetail:             "China route · CNB mirror",
	KeyLauncherRetryStart:                 "Retry start",
	KeyLauncherStarting:                   "Starting…",
	KeyLauncherBadgeInstalled:             "Installed",
	KeyLauncherBadgeMissing:               "Not ready",
	KeyLauncherDiagTitle:                  "Startup diagnosis",
	KeyLauncherRouteLine:                  "Route GitHub",
	KeyLauncherRouteLabel:                 "Route",
	KeyLauncherRouteCurrentPrefix:         "Current route",
	KeyLauncherComponentsLede:             "Install standalone tools and services as needed. Whalebridge manages models and connects to the desktop after install.",
	KeyLauncherComponentsEmpty:            "No components enabled yet.",
	KeyLauncherLocalVersion:               "Installed version",
	KeyLauncherImportLede:                 "Read-only scan of the official home. Checked items write into desktop home without modifying the official tree or copying project workspaces. Migrates sessions, attachments, skills, plugins, MCP, and personal settings.",
	KeyLauncherImportEmpty:                "Not scanned yet. Click Rescan to probe Electron data.",
	KeyLauncherImportPickDir:              "Choose folder",
	KeyLauncherImportAddSkillDir:          "Add skills folder",
	KeyLauncherImportRescan:               "Rescan",
	KeyLauncherImportRescanFirst:          "Rescan before importing.",
	KeyLauncherImportByDir:                "Grouped by folder",
	KeyLauncherImportSelectAll:            "Select all in column",
	KeyLauncherImportSelectNone:           "Select none",
	KeyLauncherImportCatSessions:          "Sessions",
	KeyLauncherImportCatSkills:            "Skills",
	KeyLauncherImportCatPlugins:           "Plugins",
	KeyLauncherImportCatMCP:               "MCP",
	KeyLauncherImportCatSettings:          "Settings",
	KeyLauncherImportCatPresets:           "Presets",
	KeyLauncherImportSummary:              "Source: %s\nDest: %s\nSessions %d · Skills %d · Plugins %d · MCP %d · Settings %d · Presets %d",
	KeyLauncherImportSummarySkillDir:      "Source: %s\nDest: %s\nSessions %d · Skills %d · Plugins %d · MCP %d · Settings %d · Presets %d\nAdded skills folder %s (+%d)",
	KeyLauncherImportPickSourceTitle:      "Choose import source",
	KeyLauncherImportAddSkillTitle:        "Add skills folder",
	KeyLauncherImportOverwrite:            "Overwrite existing",
	KeyLauncherImportOverwriteDesc:        "Overwrite same-named items already in desktop home",
	KeyLauncherImportAttachments:          "Import attachments",
	KeyLauncherImportAttachmentsDesc:      "Also migrate session attachments and media",
	KeyLauncherImportSelected:             "Selected",
	KeyLauncherImportPluginsNote:          "Selected plugins are reinstalled from their original GitHub/npm version specs. Built-in, local and unsupported sources are skipped; build scripts require explicit approval.",
	KeyLauncherImportChecklist:            "View full migration checklist",
	KeyLauncherUpdateAvailable:            "Update available: v%s (current v%s)",
	KeyLauncherUpdateNow:                  "Update now",
	KeyLauncherViewVersions:               "View versions",
	KeyLauncherPluginFailTitle:            "Plugin load failed",
	KeyLauncherPluginFailDisable:          "Disable plugins and start",
	KeyLauncherPluginFailOpen:             "Open plugin troubleshooting",
	KeyLauncherInstallDesktopTitle:        "Install desktop",
	KeyLauncherInstallDesktopDesc:         "No local desktop install detected. Pick a download route, then install.",
	KeyLauncherInstallDesktopCTA:          "Download and install desktop",
	KeyLauncherInstallDesktopVerify:       "Integrity checks run after download",
	KeyLauncherManageComponents:           "Manage components",
	KeyLauncherCompStart:                  "Start",
	KeyLauncherCompStartNamed:             "Start %s",
	KeyLauncherCompRetry:                  "Retry start",
	KeyLauncherCompInstall:                "Install",
	KeyLauncherCompStop:                   "Stop",
	KeyLauncherCompManage:                 "Manage",
	KeyLauncherCompOpenSettings:           "Open settings",
	KeyLauncherCompOfficial:               "Official",
	KeyLauncherCompUpdatable:              "Update available",
	KeyLauncherCompUpdateAvail:            "Update v%s",
	KeyLauncherCompKindService:            "Service",
	KeyLauncherCompKindTool:               "Tool",
	KeyLauncherCompStateRunning:           "Running",
	KeyLauncherCompStateStopped:           "Stopped",
	KeyLauncherCompStateAvailable:         "Installable",
	KeyLauncherCompStateError:             "Error",
	KeyLauncherVerDev:                     "Running from source",
	KeyLauncherVerNeedInstall:             "Install the desktop app first",
	KeyLauncherVerNeedInstallDesc:         "Pick a download route, then browse versions and install.",
	KeyLauncherCompCheckUpdate:            "Check and update",
	KeyLauncherCompRollback:               "Roll back to v%s",
	KeyLauncherVerBackHome:                "Back to home to install",
	KeyLauncherVerInstallPath:             "Install path  %s",
	KeyLauncherVerUpdateTo:                "Update to this version",
	KeyLauncherVerAllList:                 "All versions · click a row for details",
	KeyLauncherVerRefresh:                 "Refresh",
	KeyLauncherVerCurrent:                 "Current",
	KeyLauncherVerLatest:                  "Latest",
	KeyLauncherVerInstalled:               "Installed",
	KeyLauncherVerUpdateToV:               "Update to v%s",
	KeyLauncherVerEmpty:                   "No recorded version artifacts yet. Check for updates to list available builds.",
	KeyLauncherImportEmptyItems:           "Nothing to list. Rescan to show probe results.",
	KeyLauncherImportEmptySessions:        "No sessions to list. Rescan to show probe results.",
	KeyLauncherImportEmptySkills:          "No skills to list.",
	KeyLauncherImportEmptyPlugins:         "No plugins to list.",
	KeyLauncherImportEmptyMCP:             "No MCP configs to list.",
	KeyLauncherImportEmptySettings:        "No settings to list.",
	KeyLauncherImportEmptyPresets:         "No presets to list.",
	KeyLauncherImportConflict:             " (conflict)",
	KeyLauncherImportUnsupported:          " (unsupported)",
	KeyLauncherImportStart:                "Start import",
	KeyLauncherImportLog:                  "Import log",
	KeyLauncherImportCopy:                 "Copy",
	KeyLauncherPluginsNeedRuntime:         "Install the desktop runtime first",
	KeyLauncherPluginsNeedRuntimeDesc:     "After install, use this tab to troubleshoot user plugin conflicts.",
	KeyLauncherPluginsEmpty:               "No plugins detected; the plugin environment is empty.",
	KeyLauncherPluginsUnknownHint:         "Could not identify a specific plugin from logs. Disable entries below one by one and restart to isolate conflicts.",
	KeyLauncherSettingsLede:               "",
	KeyTrajSearch:                         "Search",
	KeyTrajDuration:                       "Duration",
	KeyTrajTurns:                          "Turns",
	KeyTrajCalls:                          "Calls",
	KeyTrajTimeline:                       "Trajectory timeline",
	KeyTrajColInput:                       "Input",
	KeyTrajColOutput:                      "Output",
	KeyTrajColThink:                       "Think",
	KeyTrajColTime:                        "Time",
	KeyTrajColModel:                       "Model",
	KeyTrajColTools:                       "Tools",
	KeyTrajLoadEarlier:                    "Load earlier history",
	KeyTrajLoading:                        "Loading trajectory…",
	KeyTrajEmpty:                          "No content",
	KeyTrajDetailTitle:                    "Event details",
	KeyTrajDetailHint:                     "Select a row to view overview / args / result / timing.",
	KeyTrajTabOverview:                    "Overview",
	KeyTrajTabArgs:                        "Args",
	KeyTrajTabResult:                      "Result",
	KeyTrajTabTiming:                      "Timing",
	KeyTrajNoArgs:                         "No args captured",
	KeyTrajNoResult:                       "No result captured",
	KeyTrajTimingLine:                     "Duration / seq: %s",
	KeyTrajNoTiming:                       "Not recorded",
	KeyTrajCopy:                           "Copy",
	KeyTrajToolResult:                     "Tool result",
	KeyTrajToolCallOnly:                   "(tool call only)",
	KeyTrajCompacted:                      "Compacted",
	KeyTrajDeliverables:                   "Deliverables",
	KeyTrajRoleUser:                       "User",
	KeyTrajRoleAssistant:                  "Assistant",
	KeyTrajRoleSystem:                     "System",
	KeyTrajInProgress:                     "In progress",
	KeyTrajKindTool:                       "Tool",
	KeyTrajKindThink:                      "Think",
	KeyTrajKindMessage:                    "Message",
	KeyTrajTurnN:                          "Turn %d",
	KeySettingsDesktopDetectElectronTitle: "Electron data probe",
	KeySettingsDesktopCheckUpdatesTitle:   "Check for updates",
	KeySettingsSectionAccount:             "Account",
	KeySettingsSectionGeneral:             "General",
	KeySettingsSectionAppearance:          "Appearance",
	KeySettingsSectionInterface:           "Interface",
	KeySettingsSectionModels:              "Models",
	KeySettingsSectionPlugins:             "Plugins",
	KeySettingsSectionSkills:              "Skills",
	KeySettingsSectionMarket:              "Market",
	KeySettingsSectionRemote:              "Remote",
	KeySettingsSectionMCP:                 "MCP",
	KeySettingsSectionAgentPresets:        "Agent presets",
	KeySettingsSectionPet:                 "Pet",
	KeySettingsSectionAbout:               "About",
	KeySettingsSectionUsageStats:          "Usage",
	KeySettingsSectionRemoteWorkspace:     "Remote workspace",
	KeySettingsSectionOther:               "Other",
	KeyUsageOpen:                          "Usage",
	KeyUsageTitle:                         "Usage",
	KeyUsageBack:                          "Back",
	KeyUsageRefresh:                       "Refresh",
	KeyUsageLoading:                       "Loading…",
	KeyUsagePluginNote:                    "Native panel includes heatmap cells and bars (job/list); richer series may still come from dsh-usage-panel.",
	KeyUsageAccount:                       "Account",
	KeyUsageBalance:                       "Balance (account/getBalance)",
	KeyUsageRecent:                        "Recent activity",
	KeyUsageRecentEmpty:                   "No rows yet; pick a session and refresh for job/list.",
	KeyUsageNoData:                        "Not loaded yet",
	KeyUsageWebLink:                       "Web usage",
	KeyUsageChartTitle:                    "Jobs · last 7 days",
	KeyUsageChartNote:                     "Native bars from job/list StartedAt (last 7 days).",
	KeyUsageHeatmapTitle:                  "Activity heatmap · last 28 days",
	KeyUsageHeatmapNote:                   "Native intensity cells from job/list StartedAt; dsh-usage-panel still has richer series.",
	KeyUsageOpenWeb:                       "Open usage page in WebView2",
	KeyUsageOpenTopUp:                     "Open top-up page",
	KeyUsageCopyURL:                       "Copy usage URL",
	KeyUsageCopyTopUp:                     "Copy top-up URL",
	KeyUsageCopyBalance:                   "Copy balance",
	KeyUsageKillJob:                       "Kill",
	KeyUsageFollowJob:                     "Follow",
	KeyUsageExpiry:                        "Expiry",
	KeySettingsAccountBonuses:             "Unnotified bonuses",
	KeySettingsAccountBonusesEmpty:        "No unnotified bonuses",
	KeySettingsAccountBonusAck:            "Dismiss",
	KeySettingsAccountRunningTasks:        "Account tasks running",
	KeySettingsAboutAnalytics:             "Product analytics",
	KeySettingsAboutAnalyticsOn:           "Enabled",
	KeySettingsAboutAnalyticsOff:          "Disabled",
	KeyDirPickTitle:                       "Choose folder",
	KeyDirPickClose:                       "Close",
	KeyDirPickEmpty:                       "This folder is empty",
	KeyDirPickSelect:                      "Select",
	KeyDirPickSelectHere:                  "Select current folder",
	KeyDirPickNewName:                     "New folder name…",
	KeyDirPickCreate:                      "Create",
	KeyTerminalTitle:                      "Terminal",
	KeyTerminalShow:                       "Show terminal",
	KeyTerminalHide:                       "Hide terminal",
	KeyTerminalUnavailable:                "Terminal unavailable",
	KeyTerminalRestart:                    "Restart",
	KeyTerminalCopyCwd:                    "Copy path",
	KeyGitFetch:                           "Fetch",
	KeyGitPull:                            "Pull",
	KeyGitPush:                            "Push",
	KeyGitCommit:                          "Commit",
	KeyGitCommitOK:                        "Committed",
	KeyGitFetchOK:                         "Fetched",
	KeyGitPullOK:                          "Pulled",
	KeyGitPushOK:                          "Pushed",
	KeyGitActionFailed:                    "Git failed",
	KeyGitCommitPlaceholder:               "Commit message",
	KeyGitCommitConfirmTitle:              "Confirm commit",
	KeyGitCommitConfirmMessage:            "This runs git add -A and commits all changes. Continue?",
	KeyGitCommitCancel:                    "Cancel",
	KeyGitCommitNotDirty:                  "No uncommitted changes",
	KeyWebViewOpen:                        "Plugin web",
	KeyWebViewTitle:                       "Third-party plugin UI",
	KeyWebViewBack:                        "Back",
	KeyWebViewNotOpen:                     "The plugin page is not open yet.",
	KeyWebViewOpenBrowser:                 "Open Harness in browser",
	KeyWebViewOpenHost:                    "Open embedded WebView2",
	KeyWebViewCloseHost:                   "Close embedded WebView2",
	KeyWebViewCopyURL:                     "Copy URL",
	KeyWebViewOrigin:                      "Harness origin",
	KeyBrowserPreviewOpen:                 "Browser preview",
	KeyBrowserPreviewTitle:                "Browser preview",
	KeyBrowserPreviewBack:                 "Back",
	KeyBrowserPreviewURLPlaceholder:       "Enter https://…",
	KeyBrowserPreviewGo:                   "Open",
	KeyBrowserPreviewOpenExternal:         "System browser",
	KeyBrowserPreviewCopyURL:              "Copy URL",
	KeyBrowserPreviewReload:               "Reload",
	KeyBrowserPreviewCloseHost:            "Close WebView",
	KeyBrowserPreviewHint:                 "Enter any http(s) URL to preview in the embedded WebView2 (separate from plugin slot web).",
	KeyBrowserPreviewInvalidURL:           "Invalid http(s) URL",
	KeyChatCitations:                      "Sources",
	KeyRemoteOpen:                         "Remote",
	KeyRemoteTitle:                        "Remote workspace",
	KeyRemoteBack:                         "Back",
	KeyRemoteWorkspaceToggle:              "Remote workspace (remoteWorkspaceEnabled)",
	KeyRemoteToggleOn:                     "Enable",
	KeyRemoteToggleOff:                    "Disable",
	KeyRemoteHarnessOverlay:               "Harness dsh-remote overlay",
	KeyRemotePhoneGit:                     "Phone Git (current session workspace only)",
	KeyRemotePairingEnabled:               "Phone/LAN remote (remoteEnabled)",
	KeyRemoteMode:                         "remoteMode",
	KeyRemotePort:                         "remotePort",
	KeyRemoteMux:                          "remote.mux WebSocket",
	KeyRemoteCredentials:                  "credentials remoteToken",
	KeyRemoteOn:                           "On",
	KeyRemoteOff:                          "Off",
	KeyRemoteMuxWaiting:                   "Waiting for Harness",
	KeyRemoteMuxConnected:                 "Connected",
	KeyRemoteMuxDisconnected:              "Disconnected",
	KeyRemoteCredPresent:                  "Configured",
	KeyRemoteCredAbsent:                   "Not configured",
	KeyRemoteWorkspaceOff:                 "remoteWorkspaceEnabled is off: the dsh-remote harness overlay is not mounted at startup.",
	KeyRemoteSummaryDisabled:              "Remote workspace disabled",
	KeyRemoteSummaryEnabled:               "Remote workspace",
	KeyRemoteSSHTitle:                     "SSH remote workspace",
	KeyRemoteSSHStatus:                    "SSH readiness",
	KeyRemoteSSHSummaryReady:              "Configuration is complete; pairing can start. Use the TCP probe below to check the port.",
	KeyRemoteSSHSummaryIncomplete:         "Configuration is incomplete; fill in the missing items before pairing.",
	KeyRemoteSSHSummaryDisabled:           "SSH mode is selected, but the remote workspace or pairing is off.",
	KeyRemoteSSHSummaryInactive:           "The current connection is not SSH.",
	KeyRemoteSSHStatusOff:                 "Inactive",
	KeyRemoteSSHReady:                     "Config complete (pairing can start)",
	KeyRemoteSSHIncomplete:                "Config incomplete",
	KeyRemoteSSHDisabled:                  "Workspace or pairing disabled",
	KeyRemoteSSHMissing:                   "Missing",
	KeyRemoteSSHBind:                      "remoteBindAddress",
	KeyRemoteSSHLanTLS:                    "remoteLanTls",
	KeyRemoteSSHRelayToken:                "credentials remoteRelayToken",
	KeyRemoteSSHPairingOff:                "remoteEnabled is off: the pairing daemon will not start even in SSH mode.",
	KeyRemoteSSHProbe:                     "TCP probe",
	KeyRemoteSSHProbeBtn:                  "Probe pairing port",
	KeyChatLoadEarlier:                    "Load earlier messages",
	KeyChatLoadEarlierBusy:                "Loading…",
	KeyChatEdit:                           "Edit",
	KeyChatEditHint:                       "Re-editing this message",
	KeyChatEditRunning:                    "Cannot edit while the conversation is running",
	KeyChatEditUnsupported:                "Only text messages can be edited",
	KeyChatEditStale:                      "The message changed or the conversation is running. Cancel editing and try again.",
	KeyChatCopy:                           "Copy",
	KeyChatQuote:                          "Quote",
	KeyChatReconnect:                      "Reconnect",
	KeyChatDeliverables:                   "Deliverables",
	KeyChatCompaction:                     "Compaction",
	KeyChatImageClose:                     "Close",
	KeyPlanBoardTitle:                     "Plan board",
	KeyPlanBoardExpand:                    "Expand",
	KeyPlanBoardCollapse:                  "Collapse",
	KeyPlanBoardEmpty:                     "No plan / goal / queue / todos",
	KeyPlanBoardPlan:                      "Plan",
	KeyPlanBoardPlanOn:                    "Plan mode: on",
	KeyPlanBoardPlanOff:                   "Plan mode: off",
	KeyPlanBoardGoal:                      "Goal",
	KeyPlanBoardNoGoal:                    "No active goal",
	KeyPlanBoardQueue:                     "Queue",
	KeyPlanBoardQueueEmpty:                "Queue empty",
	KeyPlanBoardQueueEdit:                 "Edit",
	KeyPlanBoardQueueSteer:                "Steer",
	KeyPlanBoardQueueRemove:               "Remove",
	KeyPlanBoardTodos:                     "Todos",
	KeyPlanBoardTodosEmpty:                "No todos",
	KeyPlanBoardTodoDone:                  "Done %d",
	KeyPlanBoardTodoActive:                "Active %d",
	KeyPlanBoardTodoPending:               "Pending %d",
	KeyPlanBoardGoalCreate:                "Create goal",
	KeyPlanBoardGoalEdit:                  "Save goal",
	KeyPlanBoardGoalComplete:              "Complete",
	KeyPlanBoardGoalClear:                 "Clear",
	KeyPlanBoardGoalPause:                 "Pause",
	KeyPlanBoardGoalResume:                "Resume",
	KeyPlanBoardGoalDraftPlaceholder:      "Goal objective…",
	KeyTodoPanelTitle:                     "Tasks",
	KeyChatRunFailed:                      "This turn failed",
	KeyChatTurnDone:                       "Completed",
	KeyChatTurnDoneSec:                    "Completed in %d s",
	KeyChatTurnDoneMin:                    "Completed in %d min",
	KeyChatTurnDoneMinSec:                 "Completed in %d min %d s",
	KeyScheduleOpen:                       "Schedule",
	KeyScheduleTitle:                      "Scheduled tasks",
	KeyScheduleBack:                       "Back",
	KeyScheduleRefresh:                    "Refresh",
	KeyScheduleLoading:                    "Loading…",
	KeyScheduleNote:                       "From schedule/list and schedule/catalog for the selected session. New schedules are created by chat/agents; here you can rename, view history, and delete (schedule/update|history|delete).",
	KeyScheduleEmpty:                      "No schedules yet; pick a session and refresh.",
	KeyScheduleDelete:                     "Delete",
	KeyScheduleHistory:                    "History",
	KeyScheduleCopyHistory:                "Copy history",
	KeyScheduleEditTitle:                  "Title",
	KeyScheduleSaveTitle:                  "Save title",
	KeyScheduleEditPrompt:                 "Prompt",
	KeyScheduleSavePrompt:                 "Save prompt",
	KeyAgentsTitle:                        "Agent team",
	KeyAgentsClose:                        "Close",
	KeyAgentsRootLabel:                    "Root session",
	KeyAgentsRootFallback:                 "Current session",
	KeyAgentsPickSession:                  "Select a session to view sub-agents.",
	KeyAgentsChildren:                     "Sub-agents",
	KeyAgentsEmpty:                        "No sub-agent sessions.",
	KeyAgentsStop:                         "Stop",
	KeyAgentsInterrupt:                    "Interrupt",
	KeyAgentsInterruptAll:                 "Interrupt all",
	KeyAgentsPromptPlaceholder:            "Message a sub-agent…",
	KeyAgentsPromptSend:                   "Send",
	KeyRemoteMobileTitle:                  "Phone remote",
	KeyRemoteMobileNote:                   "Remote configuration and registered devices; this desktop has not integrated the offer v2 pairing runtime.",
	KeyRemoteMobileAppBase:                "remoteAppBaseUrl",
	KeyRemoteMobileDevices:                "Paired devices",
	KeyRemoteMobileNeedToken:              "remoteEnabled is on but remoteToken is missing; phones cannot pair.",
	KeyRemotePhoneTitle:                   "Phone remote",
	KeyRemotePhoneHowTo:                   "This entry opens the configured mobile website only; it does not verify service availability or pairing. A daemon port is not a mobile website.",
	KeyRemotePhoneStatus:                  "Phone remote readiness",
	KeyRemotePhoneReady:                   "Config complete (pairing service unverified)",
	KeyRemotePhoneIncomplete:              "Config incomplete",
	KeyRemotePhoneDisabled:                "Pairing disabled",
	KeyRemotePhoneMissing:                 "Missing",
	KeyRemotePhoneRelayToken:              "credentials remoteRelayToken",
	KeyRemotePhoneLanBind:                 "remoteBindAddress (LAN)",
	KeyRemotePhoneLanTLS:                  "remoteLanTls",
	KeyRemotePhoneConnectURL:              "Mobile website",
	KeyRemotePhoneQRPlaceholder:           "Pairing QR unavailable: offer v2 and relay connection status are not integrated. A website address cannot replace a pairing code.",
	KeyRemotePhoneURLUnavailable:          "No valid mobile website configured. remoteAppBaseUrl must be an HTTP(S) URL without credentials, query parameters, or fragments.",
	KeyRemotePhoneCopyURL:                 "Copy website address",
	KeyRemotePhoneOpenURL:                 "Open URL",
	KeyRemotePhoneRefresh:                 "Refresh status",
	KeyRemotePhoneDeviceList:              "Paired devices",
	KeyRemotePhoneCopied:                  "Connect URL copied to clipboard.",
	KeyRemotePhoneOpened:                  "Opened connect URL in WebView / browser.",
	KeyRemotePhoneOpenFailed:              "Failed to open connect URL",
	KeyRemotePhoneNoURL:                   "No connect URL available.",
	KeyRemotePhoneRefreshed:               "Refreshed pairing status and devices from credentials.",
	KeyRemotePhoneCopyFailed:              "Failed to copy to clipboard",
	KeyIMOpen:                             "IM",
	KeyIMOpenWeb:                          "Open dsh-im Web",
	KeyIMChannels:                         "Channels",
	KeyIMCompose:                          "Compose",
	KeyIMSend:                             "Send",
	KeyRemoteSSHTunnelTitle:               "SSH session",
	KeyRemoteSSHTunnelNote:                "Public-key login + optional local forward (ssh-tunnel.json).",
	KeyRemoteSSHTunnelState:               "State",
	KeyRemoteSSHFieldHost:                 "Host",
	KeyRemoteSSHFieldUser:                 "User",
	KeyRemoteSSHFieldPort:                 "Port",
	KeyRemoteSSHFieldIdentity:             "Identity",
	KeyRemoteSSHFieldForward:              "Forward",
	KeyRemoteSSHTunnelSave:                "Save tunnel config",
	KeyRemoteSSHTunnelConnect:             "Connect",
	KeyRemoteSSHTunnelDisconnect:          "Disconnect",
	KeyRemoteSSHHostUnknownTitle:          "Unknown SSH host key",
	KeyRemoteSSHHostUnknownMessage:        "The host key for %s is not trusted yet.\nType: %s\nFingerprint: %s\n\nAdd this key to known_hosts and continue?",
	KeyRemoteSSHHostAccept:                "Trust and connect",
	KeyRemoteSSHHostDecline:               "Decline",
	KeyRemoteSSHHostDeclined:              "Unknown host key declined; not connected",
	KeyRemoteSSHHostChangedTitle:          "SSH host key changed",
	KeyRemoteSSHHostChangedMessage:        "The host key for %s does not match known_hosts (remote presented %s %s). Connection refused to prevent a possible MITM. Verify the host and update known_hosts manually.",
	KeyRemoteSSHHostChangedStatus:         "Host key changed; connection refused",
	KeyRemoteSSHConnected:                 "SSH connected",
	KeyIMTitle:                            "Instant messaging",
	KeyIMBack:                             "Back",
	KeyIMRefresh:                          "Refresh",
	KeyIMLoading:                          "Loading…",
	KeyIMStubNote:                         "IM conversations appear after connecting to the Host.",
	KeyIMHarnessOverlay:                   "Harness dsh-im overlay",
	KeyIMPluginDisabled:                   "dsh-im in disabledPlugins",
	KeyIMMux:                              "remote.mux WebSocket",
	KeyIMGeneratedRPC:                     "codegen im/* RPC",
	KeyIMOn:                               "yes",
	KeyIMOff:                              "no",
	KeyIMMuxWaiting:                       "Waiting for Harness",
	KeyIMMuxConnected:                     "Connected",
	KeyIMMuxDisconnected:                  "Disconnected",
	KeyIMRPCYes:                           "list RPC registered",
	KeyIMRPCNo:                            "no im/* endpoints",
	KeyIMRPCPartial:                       "im/* without list",
	KeyIMNoRPC:                            "No im/* methods in internal/dsh/api (dsh-im plugin RPC not in codegen).",
	KeyIMConnectionPlaceholder:            "Connection placeholder: full IM UI stays in the dsh-im plugin until im/* RPCs are codegen’d; this panel will call a list RPC when available.",
	KeyIMListVia:                          "List RPC",
	KeyIMListEmpty:                        "No channels (local or list RPC).",
	KeyIMMethods:                          "Registered im/* methods",
	KeyIMPluginOff:                        "dsh-im is disabled via disabledPlugins.",
	KeyIMLocalMode:                        "Local/diagnostic mode: no im/* response; channels/history stay in userData/im/; shows dsh-im inventory diagnostics.",
	KeyIMHistory:                          "Message history",
	KeyIMHistoryEmpty:                     "No messages in this channel yet. Send one, or Refresh to pull server history.",
	KeyIMNoChannel:                        "Select a channel.",
	KeyIMYou:                              "You",
	KeyIMPeer:                             "Peer",
	KeyIMBridge:                           "Sync bridge",
	KeyIMBridgeRPC:                        "im/* RPC",
	KeyIMBridgeCordis:                     "Cordis connection.status",
	KeyIMBridgeInventory:                  "pluginInventory",
	KeyIMBridgeLocal:                      "local only",
	KeyIMBridgeIdle:                       "not refreshed",
	KeyIMPluginDiag:                       "Connection diagnostics",
	KeyIMHistoryVia:                       "History RPC",
	KeyIMHistorySynced:                    "Merged server history into local store.",
	KeyMarketOpen:                         "Market",
	KeyMarketTitle:                        "Plugin market",
	KeyMarketBack:                         "Back",
	KeyMarketRefresh:                      "Refresh",
	KeyMarketLoading:                      "Loading…",
	KeyMarketHarnessOverlay:               "Market overlay",
	KeyMarketPluginDisabled:               "Market plugin disabled",
	KeyMarketMux:                          "Mux",
	KeyMarketGeneratedRPC:                 "pluginManager RPC",
	KeyMarketInstallToolchain:             "Local install toolchain",
	KeyMarketOn:                           "Yes",
	KeyMarketOff:                          "No",
	KeyMarketMuxWaiting:                   "Waiting for Harness",
	KeyMarketMuxConnected:                 "Connected",
	KeyMarketMuxDisconnected:              "Disconnected",
	KeyMarketRPCYes:                       "listPlugins present",
	KeyMarketRPCNo:                        "No listPlugins",
	KeyMarketRPCPartial:                   "Other pluginManager methods only",
	KeyMarketNoRPC:                        "Cannot list plugins when codegen lacks pluginManager/listPlugins.",
	KeyMarketListVia:                      "List RPC",
	KeyMarketListEmpty:                    "(No installed plugins)",
	KeyMarketMethods:                      "pluginManager methods",
	KeyMarketPluginOff:                    "ui-settings-market is disabled via disabledPlugins.",
	KeyMarketInstallHint:                  "Enter github:owner/repo or github:owner/repo#ref to install via local dsh plugin add (no DSH_DESKTOP_INSTALL_URL in UI).",
	KeyMarketInstallAdd:                   "Install from GitHub",
	KeyMarketInstallSpecLabel:             "Install spec",
	KeyMarketInstallPlaceholder:           "github:owner/repo",
	KeyMarketInstallButton:                "Install",
	KeyMarketRemove:                       "Remove",
	KeyMarketAllowBuildsButton:            "Allow builds and retry",
	KeyMarketAllowBuilds:                  "Needs allowBuilds",
	KeyMarketInstallNotReady:              "Requires Node.js 22+ and a built vendored harness (setup-harness).",
	KeyMarketInstallOK:                    "Install succeeded; restarting Harness…",
	KeyMarketInstallRunning:               "Installing…",
	KeyMarketInstallLog:                   "Install log",
	KeyMarketCopyLog:                      "Copy log",
	KeyMarketOpenHarness:                  "Open full market",
	KeyMarketRegistry:                     "Registry",
	KeyMarketExemptions:                   "Version exemptions",
	KeyMarketCancelInstall:                "Cancel install",
	KeyMarketCancelOK:                     "Cancel install requested",
	KeyMarketIncompatible:                 "Incompatible versions",
	KeyMarketAcceptExemption:              "Accept risk & exempt",
	KeyAssistantOpen:                      "Assistant",
	KeyAssistantTitle:                     "Whale assistant & dshbot",
	KeyAssistantBack:                      "Back",
	KeyAssistantStubNote:                  "Whale uses its dedicated assistant. Select a Dshbot Bot or room to use its canonical session. Open the plugin page for full settings and management.",
	KeyAssistantWhaleTitle:                "dsh-whale assistant",
	KeyAssistantDshbotTitle:               "dshbot",
	KeyAssistantConfigEnabled:             "enabled in config",
	KeyAssistantVendorPresent:             "runtime/vendor present",
	KeyAssistantOn:                        "yes",
	KeyAssistantOff:                       "no",
	KeyAssistantToggleOn:                  "Enable",
	KeyAssistantToggleOff:                 "Disable",
	KeyAssistantOpenWhaleWeb:              "Advanced: open whale in WebView2",
	KeyAssistantOpenDshbotWeb:             "Advanced: open dshbot in WebView2",
	KeyAssistantRestartNote:               "Restart Harness after toggling so overlays remount.",
	KeyAssistantStatusTitle:               "Status",
	KeyAssistantLastWebURL:                "Last WebView URL",
	KeyAssistantLastAction:                "Last native action",
	KeyAssistantComposeTitle:              "Quick prompt",
	KeyAssistantComposeHint:               "Whale sends to its dedicated assistant; Dshbot sends to the explicitly selected Bot or room. Failed requests keep the draft.",
	KeyAssistantComposePlaceholder:        "Type a native quick prompt…",
	KeyAssistantSend:                      "Send",
	KeyAssistantNoSession:                 "No session selected — create or select one in the chat sidebar, or use Open session.",
	KeyAssistantNoRPC:                     "Session / plugin RPC not ready — opened the current-mode WebView as fallback.",
	KeyAssistantPromptSent:                "Sent to the current session.",
	KeyAssistantPluginRPCTitle:            "Plugin control plane (HTTP)",
	KeyAssistantPluginRPCHint:             "POST harness /dsh-whale/* and /dshbot/* (client-request envelope); not typert gen-api.",
	KeyAssistantWhaleEnsure:               "Ensure assistant session",
	KeyAssistantWhaleCatalog:              "Whale catalog",
	KeyAssistantWhalePetChat:              "pet/chat (use draft above)",
	KeyAssistantWhalePetState:             "pet/state history",
	KeyAssistantWhalePetLook:              "pet/look (pick image)",
	KeyAssistantDshbotDescribe:            "dshbot describe",
	KeyAssistantDshbotCatalog:             "Bot catalog",
	KeyAssistantDshbotActivity:            "Bot activity",
	KeyAssistantDshbotRosterTitle:         "Bots & group chats",
	KeyAssistantDshbotRosterRefresh:       "Refresh roster",
	KeyAssistantDshbotRosterEmpty:         "No cached roster. Use Refresh to load from dshbot/describe.",
	KeyAssistantDshbotRosterUpdated:       "Updated",
	KeyAssistantDshbotRosterBot:           "Bot",
	KeyAssistantDshbotRosterRoom:          "Group",
	KeyAssistantDshbotRosterOpen:          "Open",
	KeyAssistantModeLabel:                 "Mode",
	KeyAssistantModeTitle:                 "Assistant mode",
	KeyAssistantModeHint:                  "Switch Whale / Dshbot to choose the assistant and its plugin page.",
	KeyAssistantModeWhale:                 "Whale",
	KeyAssistantModeDshbot:                "Dshbot",
	KeyAssistantOpenModeWeb:               "Open mode WebView",
	KeyAssistantOpenSession:               "Open session",
	KeyAssistantBusy:                      "Working…",
	KeyAssistantLastErr:                   "Last error",
	KeyAssistantSessionID:                 "Assistant session ID",
	KeyAssistantHistoryTitle:              "Native history",
	KeyAssistantHistoryEmpty:              "No local native turns yet. Send one to start.",
	KeyAssistantWhaleRPCSent:              "Completed a dsh-whale/pet/chat turn.",
	KeyAssistantWhaleRPCErr:               "Whale RPC failed",
	KeyAssistantDshbotRouted:              "Sent to the selected Bot’s canonical session.",
	KeyAssistantDshbotProbeOK:             "dshbot/describe is available.",
	KeyAssistantDshbotProbeErr:            "dshbot/describe unavailable",
	KeyAssistantEnsuring:                  "Ensuring whale assistant session…",
	KeyAssistantChatting:                  "pet/chat in progress…",
}

// Normalize maps config / BCP-47 tags onto a catalog id.
// "zh", "zh-CN", "zh_CN" → zh-CN; "en", "en-US", … → en; empty → default.
func Normalize(locale string) string {
	s := strings.TrimSpace(locale)
	if s == "" {
		return DefaultLocale
	}
	s = strings.ReplaceAll(s, "_", "-")
	lower := strings.ToLower(s)
	switch {
	case lower == "zh" || strings.HasPrefix(lower, "zh-"):
		return LocaleZH
	case lower == "en" || strings.HasPrefix(lower, "en-"):
		return LocaleEN
	default:
		return DefaultLocale
	}
}

// SetLocale switches the active catalog. Unknown values fall back via Normalize.
func SetLocale(locale string) {
	mu.Lock()
	defer mu.Unlock()
	current = Normalize(locale)
}

// Locale returns the active catalog id (zh-CN or en).
func Locale() string {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// T looks up key in the active catalog, then English, then returns key.
func T(key string) string {
	mu.RLock()
	loc := current
	mu.RUnlock()
	return Lookup(loc, key)
}

// Lookup resolves key for locale without changing the global locale.
// Fallback order: locale catalog → English → key.
func Lookup(locale, key string) string {
	loc := Normalize(locale)
	if cat := catalogs[loc]; cat != nil {
		if s, ok := cat[key]; ok {
			return s
		}
	}
	if loc != LocaleEN {
		if s, ok := en[key]; ok {
			return s
		}
	}
	return key
}

// Statuses the theme editor and the launcher badge carry register themselves,
// so their words live with the key rather than in the shared catalog literals.
const (
	KeyLauncherCompBundled = "launcher.comp.bundled"

	KeyPluginPreviewRetry = "plugin.preview.retry"

	KeyThemeEditorPreview  = "theme.editor.preview"
	KeyThemeEditorRestored = "theme.editor.restored"
	KeyThemeEditorSaved    = "theme.editor.saved"
	KeyThemeEditorSaving   = "theme.editor.saving"
)

func init() {
	Register(KeyLauncherCompBundled, "内置", "Bundled")
	Register(KeyPluginPreviewRetry, "重试", "Retry")
	Register(KeyThemeEditorPreview, "预览中，尚未保存", "Preview, not saved")
	Register(KeyThemeEditorRestored, "已恢复", "Restored")
	Register(KeyThemeEditorSaved, "已保存", "Saved")
	Register(KeyThemeEditorSaving, "正在保存…", "Saving…")
}

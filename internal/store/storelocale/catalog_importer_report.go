package storelocale

// Labels and outcomes for the read-only profile detection report. The report
// names what was found instead of printing raw field names, and every value is
// a whole word so a reader never has to decode a bare key.

const (
	KeyStoreImporterReportConfig      = "store.importer.report.config"
	KeyStoreImporterReportCredentials = "store.importer.report.credentials"
	KeyStoreImporterReportSessions    = "store.importer.report.sessions"
	KeyStoreImporterReportFound       = "store.importer.report.found"
	KeyStoreImporterReportMissing     = "store.importer.report.missing"
)

func init() {
	register(KeyStoreImporterReportConfig, "配置文件：", "Configuration file: ")
	register(KeyStoreImporterReportCredentials, "登录凭据：", "Sign-in credentials: ")
	register(KeyStoreImporterReportSessions, "会话数据：", "Session data: ")
	register(KeyStoreImporterReportFound, "已找到", "found")
	register(KeyStoreImporterReportMissing, "未找到", "not found")
}

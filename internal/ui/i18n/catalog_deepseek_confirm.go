package i18n

// The dialog a plugin tool's external side effect has to pass before anything is
// uploaded or generated. A refusal, a dismissal and a timeout are the same
// answer, and the sentence says so.
const (
	KeyDeepSeekConfirmTitle = "deepseek.confirm.title"
	KeyDeepSeekConfirmAllow = "deepseek.confirm.allow"
	KeyDeepSeekConfirmDeny  = "deepseek.confirm.deny"
	KeyDeepSeekConfirmBody  = "deepseek.confirm.body"
)

func init() {
	Register(KeyDeepSeekConfirmTitle, "确认这次调用", "Allow this call")
	Register(KeyDeepSeekConfirmAllow, "允许", "Allow")
	Register(KeyDeepSeekConfirmDeny, "不允许", "Do not allow")
	Register(KeyDeepSeekConfirmBody, "这是 AI 发起的操作，内容会发送到 DeepSeek 的付费接口。允许之前请确认内容与费用。", "The assistant started this call and the content goes to a paid DeepSeek API. Check the content and the cost before allowing it.")
}

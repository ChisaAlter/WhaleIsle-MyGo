//go:build windows

package harness

// startStdoutLeaker has no Windows implementation: a child started here stays in
// the Host's taskkill /T tree, so the leaked-pipe fixture is a Unix test. The
// bounded waits it guards are covered on every platform by
// TestStopIsBoundedWhenAStatusListenerNeverReturns and TestWaitExitReportsBudgetExpiry.
func startStdoutLeaker() {}

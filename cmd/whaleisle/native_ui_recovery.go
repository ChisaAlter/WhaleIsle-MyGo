package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/processutil"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
)

const nativeRecoveryFlag = "--native-ui-recovery-watchdog"
const nativeUIHangAfter = 20 * time.Second

type nativeRecoveryMessage struct {
	Kind                         string
	PID                          int
	Executable                   string
	CrashDir, Locale, Diagnostic string
}

// The watchdog has its own event loop and no Host/profile/single-instance lock.
// It therefore remains usable when the primary native UI thread cannot paint.
func handleNativeRecoveryProcess() bool {
	if len(os.Args) != 3 || os.Args[1] != nativeRecoveryFlag {
		return false
	}
	pid, err := strconv.Atoi(os.Args[2])
	if err != nil || pid <= 0 {
		return true
	}
	decoder := json.NewDecoder(bufio.NewReader(os.Stdin))
	var hello nativeRecoveryMessage
	if decoder.Decode(&hello) != nil || hello.Kind != "hello" || hello.PID != pid {
		return true
	}
	process, err := openNativeRecoveryProcess(pid, hello.Executable)
	if err != nil {
		return true
	}
	defer process.close()
	mygo.App.SetName("Whale Isle MyGo · Recovery")
	if hello.Locale != "" {
		i18n.SetLocale(hello.Locale)
	}
	var mu sync.Mutex
	lastBeat := time.Now()
	diagnostic, failure := "", ""
	var win *mygo.Window
	recovering, dismissed, restarting, stopped := false, false, false, false
	commands := json.NewEncoder(os.Stdout)
	var sendMu sync.Mutex
	send := func(kind string) {
		sendMu.Lock()
		_ = commands.Encode(nativeRecoveryMessage{Kind: kind})
		sendMu.Unlock()
	}
	closeWindow := func() {
		mu.Lock()
		current := win
		win = nil
		mu.Unlock()
		if current != nil && !current.IsDestroyed() {
			current.Destroy()
		}
		mu.Lock()
		targetStopped := stopped
		mu.Unlock()
		if targetStopped {
			mygo.App.Quit()
		}
	}
	go func() {
		for {
			var message nativeRecoveryMessage
			if decoder.Decode(&message) != nil {
				mu.Lock()
				active := restarting || stopped
				mu.Unlock()
				if !active {
					mygo.App.Quit()
				}
				return
			}
			mu.Lock()
			switch message.Kind {
			case "beat":
				lastBeat = time.Now()
			case "diagnostic":
				diagnostic = message.Diagnostic
			case "shutdown":
				mu.Unlock()
				mygo.App.Quit()
				return
			}
			current := win
			mu.Unlock()
			if current != nil {
				current.Update(func() {})
			}
		}
	}()
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(func() {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if !process.alive() {
					mu.Lock()
					active := restarting || stopped
					mu.Unlock()
					if !active {
						mygo.App.Quit()
					}
					return
				}
				mu.Lock()
				hung := time.Since(lastBeat) > nativeUIHangAfter
				current := win
				if !hung {
					dismissed = false
				}
				show := hung && current == nil && !dismissed && !recovering && !restarting
				mu.Unlock()
				if !hung && current != nil {
					mygo.RunOnMain(closeWindow)
				}
				if !show {
					continue
				}
				// This report does not depend on any thread or mutex in the hung
				// desktop. The cooperative stack report can enrich it afterwards.
				path := WriteCrashReport(hello.CrashDir, CrashReportInput{Time: time.Now(), Source: "main", Phase: "error", Locale: hello.Locale, Error: fmt.Sprintf("Native UI heartbeat timeout (%s). Desktop PID: %d. Recovery PID: %d.", nativeUIHangAfter, pid, os.Getpid())})
				mu.Lock()
				diagnostic = path
				mu.Unlock()
				send("diagnostic")
				mygo.RunOnMain(func() {
					mu.Lock()
					if win != nil || recovering || time.Since(lastBeat) <= nativeUIHangAfter {
						mu.Unlock()
						return
					}
					mu.Unlock()
					created := mygo.NewWindow(mygo.WindowOptions{Title: lifecycleText("界面未响应", "Interface not responding"), Width: 640, Height: 420, MinWidth: 380, MinHeight: 300, AlwaysOnTop: true, TitleBarStyle: mygo.TitleBarHidden, TitleBarHeight: 48, AutoHideMenuBar: true, Content: ui.View(func(c *ui.Context) {
						mu.Lock()
						path, problem, ask, busy, targetStopped := diagnostic, failure, recovering, recovering || restarting, stopped
						mu.Unlock()
						ui.Column(c).Fill().Background(c.Theme().Background).Padding(48, 24, 24, 24).Gap(16).Children(func() {
							ui.Box(c).Absolute().Left(0).Top(0).Right(120).Height(48).DragWindow()
							headline := lifecycleText("桌面界面已超过 20 秒未响应。", "The desktop interface has not responded for over 20 seconds.")
							if targetStopped {
								headline = lifecycleText("原桌面已停止，可以重试启动。", "The original desktop has stopped. You can retry starting it.")
							}
							ui.Text(c, headline).FontSize(18).FontWeight(600)
							ui.Scroll(c).FillWidth().MinHeight(0).Grow(1).Children(func() {
								ui.Column(c).Gap(12).Children(func() {
									ui.Text(c, lifecycleText("诊断窗口独立运行。可以继续等待；重启会强制停止此桌面及其 Host 子进程，正在运行的任务可能中断。未知写入不会自动重放。", "This recovery window runs independently. You can keep waiting. Restart forcibly stops this desktop and its Host children; active tasks may be interrupted. Unknown writes are never replayed."))
									if path != "" {
										ui.Text(c, path).FontSize(12).Selectable()
									}
									if problem != "" {
										ui.Text(c, problem).TextColor(ui.RGBA(220, 38, 38, 1)).Selectable()
									}
								})
							})
							ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
								waitLabel := lifecycleText("继续等待", "Keep waiting")
								if targetStopped {
									waitLabel = lifecycleText("关闭恢复窗口", "Close recovery")
								}
								if ui.Button(c, waitLabel).AutoFocus().Disabled(busy).Clicked() {
									mu.Lock()
									dismissed = true
									mu.Unlock()
									closeWindow()
								}
								if ui.Button(c, lifecycleText("重新收集诊断", "Collect diagnostics")).Disabled(busy || targetStopped).Clicked() {
									send("diagnostic")
								}
								restartLabel := lifecycleText("停止并重启…", "Stop and restart…")
								if targetStopped {
									restartLabel = lifecycleText("重试启动…", "Retry starting…")
								}
								if ui.Button(c, restartLabel).Disabled(busy).Clicked() {
									mu.Lock()
									recovering = true
									mu.Unlock()
								}
							})
						})
						if ask {
							settings.ConfirmDialog(c, settings.ConfirmOptions{Title: lifecycleText("停止任务并重启桌面？", "Stop tasks and restart desktop?"), Body: lifecycleText("将终止这个无响应桌面和它的子进程。未保存界面状态可能丢失；磁盘上的项目和会话不删除。", "This terminates this unresponsive desktop and its children. Unsaved interface state may be lost; project and session files are not deleted."), ConfirmLabel: lifecycleText("停止并重启", "Stop and restart"), CancelLabel: lifecycleText("取消", "Cancel"), Danger: true, OnCancel: func() { mu.Lock(); recovering = false; mu.Unlock() }, OnConfirm: func() {
								mu.Lock()
								if !stopped && time.Since(lastBeat) <= nativeUIHangAfter {
									recovering = false
									mu.Unlock()
									closeWindow()
									return
								}
								alreadyStopped := stopped
								recovering = false
								restarting = true
								mu.Unlock()
								go func() {
									var stopErr error
									if !alreadyStopped {
										stopErr = process.stopTree(os.Getpid())
									}
									if stopErr != nil {
										mu.Lock()
										failure = stopErr.Error()
										recovering = false
										restarting = false
										mu.Unlock()
										mygo.RunOnMain(func() {
											if win != nil {
												win.Invalidate()
											}
										})
										return
									}
									mu.Lock()
									stopped = true
									mu.Unlock()
									// Reopen the same profile through its inherited environment, never
									// replay launch-time deep links or notification activations.
									cmd := exec.Command(hello.Executable)
									processutil.Background(cmd)
									if err := cmd.Start(); err != nil {
										mu.Lock()
										failure = fmt.Sprintf("Restart failed: %v", err)
										recovering = false
										restarting = false
										mu.Unlock()
										mygo.RunOnMain(func() {
											if win != nil {
												win.Invalidate()
											}
										})
										return
									}
									_ = cmd.Process.Release()
									mygo.App.Quit()
								}()
							}})
						}
					})})
					mu.Lock()
					win = created
					mu.Unlock()
					created.OnClosed(func() {
						mu.Lock()
						if win == created {
							win = nil
							dismissed = true
						}
						targetStopped := stopped
						mu.Unlock()
						if targetStopped {
							mygo.App.Quit()
						}
					})
					created.Focus()
				})
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Println("native recovery:", err)
	}
	return true
}

func (s *mainState) startNativeUIRecovery(win *mygo.Window) {
	if win == nil || win.IsDestroyed() {
		return
	}
	executable, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(executable, nativeRecoveryFlag, strconv.Itoa(os.Getpid()))
	processutil.Background(cmd)
	input, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return
	}
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		log.Println("native recovery monitor:", err)
		return
	}
	queue := make(chan nativeRecoveryMessage, 8)
	done := make(chan struct{})
	var once sync.Once
	stop := func() { once.Do(func() { close(done); input.Close() }) }
	win.OnClosed(stop)
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { stop() })
	go func() {
		defer stop()
		encoder := json.NewEncoder(input)
		if encoder.Encode(nativeRecoveryMessage{Kind: "hello", PID: os.Getpid(), Executable: executable, CrashDir: CrashLogDir(), Locale: i18n.Locale()}) != nil {
			return
		}
		for {
			select {
			case <-done:
				return
			case message := <-queue:
				if encoder.Encode(message) != nil {
					return
				}
			}
		}
	}()
	go func() {
		defer output.Close()
		decoder := json.NewDecoder(output)
		for {
			var request nativeRecoveryMessage
			if decoder.Decode(&request) != nil {
				return
			}
			if request.Kind != "diagnostic" {
				continue
			}
			stack := make([]byte, 2<<20)
			n := runtime.Stack(stack, true)
			path := recordShellCrash("main", fmt.Sprintf("Native UI heartbeat exceeded %s\n\n%s", nativeUIHangAfter, stack[:n]), "")
			select {
			case queue <- nativeRecoveryMessage{Kind: "diagnostic", Diagnostic: path}:
			case <-done:
				return
			}
		}
	}()
	go func() {
		defer cmd.Wait()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				win.Update(func() {
					select {
					case queue <- nativeRecoveryMessage{Kind: "beat"}:
					default:
					}
				})
			}
		}
	}()
}

func nativeRecoveryExecutable() string { executable, _ := os.Executable(); return executable }

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	desktopgit "whaleisle/internal/desktop/git"
)

type phonePRResult struct {
	value map[string]any
	err   error
}
type phonePRReview struct {
	ctx                        context.Context
	root, title, body, message string
	draft, busy                bool
	candidate                  *desktopgit.GeneratedPRContent
	plan                       *desktopgit.PullRequestPlan
	win                        *mygo.Window
	finish                     func(phonePRResult)
	authorize                  func() bool
}

// reviewPhonePullRequest is used only for unchanged phone clients that cannot
// supply reviewed PR text. It never interprets commit text as PR approval.
func (s *mainState) reviewPhonePullRequest(ctx context.Context, root string) (map[string]any, error) {
	s.mu.Lock()
	parent := s.win
	s.mu.Unlock()
	if parent == nil || parent.IsDestroyed() {
		return nil, errors.New("Open the desktop to review this PR")
	}
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan phonePRResult, 1)
	var once sync.Once
	q := &phonePRReview{ctx: childCtx, root: root, draft: true, authorize: func() bool { return s.authorizePhoneGit(root, "gitCreateChangeRequest") }}
	q.finish = func(r phonePRResult) { once.Do(func() { done <- r; cancel() }) }
	parent.Update(func() {
		if childCtx.Err() != nil {
			return
		}
		q.win = mygo.NewWindow(mygo.WindowOptions{Title: "手机 PR 审阅 / Phone PR review", Width: 660, Height: 720, MinWidth: 520, MinHeight: 520, Parent: parent, Content: ui.View(q.view)})
		if q.win == nil || q.win.IsDestroyed() {
			q.finish(phonePRResult{err: errors.New("Could not open desktop PR review")})
			return
		}
		q.win.OnClosed(func() {
			q.finish(phonePRResult{err: errors.New("Desktop PR review cancelled; retry remaining action on phone")})
		})
		q.win.Focus()
	})
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case r := <-done:
			parent.Update(func() {
				if q.win != nil && !q.win.IsDestroyed() {
					q.win.Close()
				}
			})
			return r.value, r.err
		case <-childCtx.Done():
			parent.Update(func() {
				if q.win != nil && !q.win.IsDestroyed() {
					q.win.Close()
				}
			})
			select {
			case r := <-done:
				return r.value, r.err
			default:
				return nil, childCtx.Err()
			}
		case <-ticker.C:
			parent.Update(func() {
				if q.win != nil && !q.win.IsDestroyed() {
					q.win.Update(func() {})
				}
			})
		}
	}
}
func (q *phonePRReview) start(work func() func()) {
	q.busy = true
	q.message = "处理中 / Working…"
	go func() {
		apply := work()
		q.win.Update(func() {
			if q.ctx.Err() != nil {
				return
			}
			q.busy = false
			apply()
		})
	}()
}
func (q *phonePRReview) view(c *ui.Context) {
	ui.Scroll(c).Grow(1).Padding(18).Children(func() {
		ui.Column(c).Gap(10).Children(func() {
			ui.Text(c, "手机请求创建 PR；请在本窗口核对并确认 / Phone requested a PR; review and confirm here").Bold()
			ui.Text(c, q.root)
			if deadline, ok := q.ctx.Deadline(); ok {
				ui.Text(c, fmt.Sprintf("剩余 %d 秒 / seconds remaining", max(0, int(time.Until(deadline).Seconds()))))
			}
			ui.Text(c, "手机上的提交/推送可能已完成。取消不会撤销这些操作；手机只重试剩余 PR。 / Commit/push may already be complete. Cancel does not undo them; the phone retains the remaining PR step.")
			if !q.busy {
				beforeTitle, beforeBody, beforeDraft := q.title, q.body, q.draft
				ui.TextInput(c, &q.title).FillWidth().Placeholder("PR 标题 / PR title")
				ui.TextArea(c, &q.body).FillWidth().Lines(4, 10).Placeholder("PR 正文 / PR body")
				ui.Checkbox(c, &q.draft, "创建草稿 PR / Create draft PR")
				if beforeTitle != q.title || beforeBody != q.body || beforeDraft != q.draft {
					q.plan = nil
				}
				if ui.Button(c, "生成候选文案 / Generate candidate").Clicked() {
					q.plan = nil
					q.start(func() func() {
						g, e := desktopgit.GeneratePullRequestContent(q.ctx, q.root)
						return func() {
							if e != nil {
								q.message = e.Error()
							} else {
								q.candidate = &g
								q.message = "候选文案未采用 / Candidate not yet applied"
							}
						}
					})
				}
				if q.candidate != nil {
					ui.Text(c, q.candidate.Title)
					ui.Text(c, q.candidate.Body)
					if ui.Button(c, "采用候选文案 / Apply candidate").Clicked() {
						q.title = q.candidate.Title
						q.body = q.candidate.Body
						q.candidate = nil
						q.plan = nil
					}
				}
				if ui.Button(c, "准备核对目标 / Prepare target review").Disabled(strings.TrimSpace(q.title) == "").Clicked() {
					title, body, draft := q.title, q.body, q.draft
					q.plan = nil
					q.start(func() func() {
						push, e := desktopgit.PreparePush(q.root)
						var p desktopgit.PullRequestPlan
						if e == nil {
							p, e = desktopgit.PreparePullRequestMode(q.ctx, q.root, push, title, body, draft)
						}
						return func() {
							if e != nil {
								q.message = e.Error()
							} else {
								q.plan = &p
								q.message = "请核对下方完整内容 / Review the exact content below"
							}
						}
					})
				}
				if q.plan != nil {
					p := *q.plan
					ui.Text(c, "GitHub: "+p.Repository+" · "+p.Push.Target+" → "+p.Base).Bold()
					ui.Text(c, p.Title)
					ui.Text(c, p.Body)
					ui.Text(c, fmt.Sprintf("Draft: %t", p.Draft))
					if ui.Button(c, "确认创建此 PR / Confirm this PR").Clicked() {
						q.plan = nil
						q.start(func() func() {
							if q.ctx.Err() != nil || !q.authorize() {
								return func() { q.finish(phonePRResult{err: errors.New("Mobile Git desktop session permission expired")}) }
							}
							result, e := desktopgit.CreatePullRequestReviewed(q.ctx, q.root, p, "")
							return func() {
								if e != nil {
									q.message = e.Error()
									if result.Uncertain {
										q.message += "; 结果不确定，请先检查已有 PR / Outcome uncertain; check existing PR before retry"
									}
								} else {
									q.finish(phonePRResult{value: map[string]any{"ok": true, "status": "created", "url": result.URL, "existing": result.Existing}})
								}
							}
						})
					}
				}
			}
			if q.message != "" {
				ui.Text(c, q.message)
			}
			if ui.Button(c, "取消 / Cancel").Clicked() {
				q.finish(phonePRResult{err: errors.New("Desktop PR review cancelled; retry remaining action on phone")})
			}
		})
	})
}

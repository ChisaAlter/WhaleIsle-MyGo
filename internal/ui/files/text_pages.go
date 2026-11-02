package files

import (
	"context"
	"errors"
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/theme"
)

var errTextPageChanged = errors.New("text page version changed")

func beginNextTextPage(m Model) {
	tab := selectedTab(m)
	if tab == nil || m.Tabs == nil || tab.readLoading || tab.readBlocked || tab.textPage.EOF || !tab.textPaged || tab.Dirty() {
		return
	}
	var sourceErr error
	m, sourceErr = deliveryReadModel(m, tab)
	if sourceErr != nil {
		rejectDeliveryRead(m, tab, sourceErr)
		return
	}
	resolve := previewPathResolver(m, tab)
	localSource := tab.deliverySource
	t := m.Tabs
	t.stopTextRead(false, false)
	if t.textReader == nil {
		t.textReader = &textReadQueue{}
	}
	t.textReader.setNotify(m.notify)
	tab.readLoading = true
	tab.pendingLine.resume = false
	t.readTab = tab
	path, remote, previous := tab.Path, tab.Remote && tab.deliverySource == nil, tab.textPage
	reader := m.ReadTextPageContext
	t.textReader.request(textReadOwnerFor(m, tab), func(ctx context.Context) textReadResult {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		result := textReadResult{appendPage: true, remote: remote}
		var page workspacefs.TextPage
		var err error
		if localSource != nil {
			page, err = localSource.readTextPage(ctx, previous.Next)
		} else if remote {
			if reader == nil {
				err = errors.New("Host text paging unavailable")
			} else {
				page, err = reader(ctx, path, previous.Next)
			}
		} else {
			var full string
			full, err = resolve()
			if err == nil {
				page, err = workspacefs.ReadFileTextPage(full, previous.Next)
			}
		}
		versionChanged := err == nil && previous.Version != page.Version
		if versionChanged {
			err = errors.New(label("文件已变化，请重新载入。", "File changed; reload before continuing."))
		}
		if err == nil && !page.EOF && page.Next <= previous.Next {
			err = errors.New("text page did not advance")
		}
		if versionChanged {
			err = errors.Join(errTextPageChanged, err)
		}
		result.page, result.err = &page, err
		return result
	})
}

func applyNextTextPage(m Model, tab *FileTab, result textReadResult) {
	// Never replace edits or the already displayed prefix on a page failure.
	tab.readBlocked = false
	if result.err != nil {
		if errors.Is(result.err, errTextPageChanged) {
			tab.pendingLine.reset()
			tab.revealLine = 0
		}
		tab.Status = result.err.Error()
		restore(m, tab)
		return
	}
	if tab.Dirty() || result.page == nil {
		return
	}
	page := *result.page
	tab.Preview += page.Separator + page.Text
	tab.Draft = tab.Preview
	tab.textPage = page
	tab.Truncated = true // Paged resources never become an editor save baseline.
	tab.Status = ""
	restore(m, tab)
	continueTextLine(m)
}

func paintTextPageControls(c *ui.Context, m Model, muted ui.Color) {
	tab := selectedTab(m)
	if tab == nil || !tab.textPaged {
		return
	}
	ui.Row(c).Gap(8).Padding(0, 12, 4, 12).Children(func() {
		ui.Text(c, fmt.Sprintf(label("已载入 %d 字节", "Loaded %d bytes"), len(tab.Preview))).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		if tab.textPage.EOF {
			ui.Text(c, label("已到文件末尾（只读）", "End of file (read-only)")).TextColor(muted)
		} else if ui.Button(c, label("读取下一页", "Load next page")).Disabled(tab.readLoading || tab.readBlocked || tab.Dirty()).Clicked() {
			beginNextTextPage(m)
		}
	})
}

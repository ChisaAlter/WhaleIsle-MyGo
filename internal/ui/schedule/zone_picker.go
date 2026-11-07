package schedule

import (
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

type scheduleZoneChoice struct {
	ID, Label string
	Aliases   []string
}

type scheduleZonePicker struct {
	open    bool
	query   string
	locale  string
	stored  string
	catalog []scheduleZoneChoice
	active  int
	rows    []*ui.Handle
}

func scheduleZoneLabel(zone, system string, at time.Time) string {
	label := formatCatalogZone(zone, at, i18n.Locale())
	if zone == system {
		label += scheduleText("（系统）", " (system)")
	}
	return label
}

// Build once per opening: every exact IANA identity stays searchable even when
// the original UI groups equivalent visible labels into one row.
func scheduleZoneCatalog(stored, system string, at time.Time) []scheduleZoneChoice {
	type candidate struct {
		id     string
		offset int
	}
	seen := map[string]bool{system: true}
	zones := []candidate{}
	for _, zone := range append(append([]string{"UTC"}, scheduleZoneNames...), stored) {
		if zone == "" || seen[zone] {
			continue
		}
		location, err := time.LoadLocation(zone)
		if err != nil {
			continue
		}
		seen[zone] = true
		_, offset := at.In(location).Zone()
		zones = append(zones, candidate{zone, offset})
	}
	sort.SliceStable(zones, func(i, j int) bool {
		if zones[i].offset != zones[j].offset {
			return zones[i].offset < zones[j].offset
		}
		return zones[i].id < zones[j].id
	})
	zones = append([]candidate{{id: system}}, zones...)
	result := []scheduleZoneChoice{}
	labels := map[string]int{}
	for _, zone := range zones {
		label := scheduleZoneLabel(zone.id, system, at)
		if index, exists := labels[label]; exists {
			result[index].Aliases = append(result[index].Aliases, zone.id)
			if zone.id == stored {
				result[index].ID = stored
			}
			continue
		}
		labels[label] = len(result)
		result = append(result, scheduleZoneChoice{ID: zone.id, Label: label, Aliases: []string{zone.id}})
	}
	return result
}

func matchingScheduleZones(catalog []scheduleZoneChoice, recent []string, query string) []scheduleZoneChoice {
	query = strings.ToLower(strings.TrimSpace(query))
	result := []scheduleZoneChoice{}
	seen := map[string]bool{}
	if query == "" {
		for _, zone := range recent {
			for _, item := range catalog {
				for _, alias := range item.Aliases {
					if alias == zone && !seen[item.Label] {
						item.ID = zone
						result = append(result, item)
						seen[item.Label] = true
					}
				}
			}
		}
	}
	for _, item := range catalog {
		if seen[item.Label] || !strings.Contains(strings.ToLower(strings.Join(item.Aliases, " ")+" "+item.Label), query) {
			continue
		}
		// Exact alias search wins. For partial alias searches preserve the rule's
		// preferred identity only when it is one of the identities that matched.
		if query != "" {
			first, exact, preferred := "", "", false
			for _, alias := range item.Aliases {
				lower := strings.ToLower(alias)
				if lower == query {
					exact = alias
				}
				if strings.Contains(lower, query) {
					if first == "" {
						first = alias
					}
					preferred = preferred || alias == item.ID
				}
			}
			if exact != "" {
				item.ID = exact
			} else if first != "" && !preferred {
				item.ID = first
			}
		}
		result = append(result, item)
	}
	return result
}

func (p *Panel) zonePicker(c *ui.Context, d *taskDraft) {
	system, now := systemScheduleZone(), time.Now()
	if p.recentZones == nil {
		p.recentZones = loadRecentZones(system)
	}
	state := &d.picker.zone
	trigger := schedulePickerTrigger(c, "schedule-zone-trigger", scheduleText("时区", "Time zone"), scheduleZoneLabel(d.Zone, system, now), state.open)
	trigger.BorderWidth(0)
	if trigger.Clicked() {
		state.open = !state.open
		state.query = ""
		if state.open {
			state.catalog = nil
			state.active = -1
		}
	}
	trigger.Expanded(state.open)
	if !state.open {
		return
	}
	if state.catalog == nil || state.locale != i18n.Locale() || state.stored != d.Zone {
		state.catalog = scheduleZoneCatalog(d.Zone, system, now)
		state.locale, state.stored = i18n.Locale(), d.Zone
	}
	ui.PopoverBase(c.Key("schedule-zone-menu"), trigger, &state.open, func(panel ui.Element) {
		panel = chrome.MenuSurface(panel.Width(360).MaxWidthPercent(100).Padding(4).Gap(4), c.Theme().Surface, c.Theme().Border, 16)
		if panel.OverlayShortcut(0, ui.KeyEscape) {
			state.open, state.query = false, ""
			trigger.Focus()
			return
		}
		panel.Children(func() {
			search := ui.TextInputBase(c.Key("schedule-zone-search"), &state.query).Label(scheduleText("搜索 UTC 偏移、IANA 标识或城市", "Search by UTC offset, IANA ID, or city")).Placeholder(scheduleText("搜索时区", "Search time zones")).AutoFocus().FillWidth().MinWidth(0).Padding(8, 10)
			items := matchingScheduleZones(state.catalog, p.recentZones, state.query)
			if len(state.rows) != len(items) {
				state.rows = make([]*ui.Handle, len(items))
				for index := range state.rows {
					state.rows[index] = &ui.Handle{}
				}
				state.active = -1
			}
			if search.Changed() {
				state.active = -1
			}
			if len(items) > 0 {
				next := -1
				if panel.Shortcut(0, ui.KeyDown) {
					next = (state.active + 1) % len(items)
				} else if panel.Shortcut(0, ui.KeyUp) {
					next = (state.active + len(items) - 1) % len(items)
					if state.active < 0 {
						next = len(items) - 1
					}
				} else if !search.Focused() && panel.Shortcut(0, ui.KeyHome) {
					next = 0
				} else if !search.Focused() && panel.Shortcut(0, ui.KeyEnd) {
					next = len(items) - 1
				}
				if next >= 0 {
					state.active = next
					state.rows[next].RequestFocus(c)
				}
			}
			ui.Scroll(c.Key("schedule-zone-options")).FillWidth().MinWidth(0).MaxHeight(300).Children(func() {
				if len(items) == 0 {
					ui.Text(c, scheduleText("没有匹配的时区", "No matching time zones")).Padding(8, 10).TextColor(c.Theme().TextMuted)
				}
				for index, item := range items {
					item := item
					row := ui.Button(c.Key("schedule-zone-"+item.ID), "").Label(item.Label).Tooltip(strings.Join(item.Aliases, ", ")).Bind(state.rows[index]).FillWidth().MinWidth(0).Padding(8, 10).BorderWidth(0).Background(ui.Transparent).Radius(6)
					if row.Focused() {
						state.active = index
					}
					row.Children(func() {
						ui.Text(c, item.Label).Grow(1).MinWidth(0)
						for _, alias := range item.Aliases {
							if alias == d.Zone {
								ui.Text(c, "✓")
								break
							}
						}
					})
					if row.Clicked() {
						if d.Kind == "at" {
							date, clock := d.wallClock()
							if err := d.pickWallClock(date, clock, item.ID); err != nil {
								p.message(scheduleText("该日期和时间在所选时区中不存在，请调整后重试。", "This date and time do not exist in the selected time zone. Adjust them and try again."))
								return
							}
						} else {
							d.Zone = item.ID
						}
						p.recentZones = rememberZone(loadRecentZones(system), item.ID)
						state.open, state.query = false, ""
						trigger.Focus()
					}
				}
			})
		})
	})
}

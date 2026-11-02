package schedule

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Pinned Unicode CLDR48 data, license and upstream provenance live beside the
// formatter. No Node, network, native API or host-global locale is used at run
// time. Go's bundled IANA data owns offsets; CLDR owns localized display names.
//
//go:embed cldr/timeZoneNames-*.json cldr/metaZones.json cldr/timezone-bcp47.json cldr/primaryZones.json cldr/territories-*.json cldr/metaZones-icu78-tz2026c.json cldr/zoneSavings-icu78.json cldr/LICENSE.txt cldr/SOURCE.txt
var catalogZoneResources embed.FS

var catalogZones struct {
	sync.Once
	locales    map[string]catalogZoneLocale
	aliases    map[string]string
	territory  map[string]string
	primary    map[string]string
	periods    map[string][]catalogZonePeriod
	references map[string]map[string]string
	savings    map[string]catalogSavingsZone
}

type catalogSavingsTransition struct {
	At      int64
	Savings int
}
type catalogSavingsZone struct {
	Before      int
	Transitions []catalogSavingsTransition
}

type catalogZonePeriod struct {
	From, To time.Time
	Meta     string
}
type catalogZoneNames struct{ Generic, Standard string }
type catalogZoneLocale struct {
	Region, Fallback string
	Zones            map[string]catalogZoneNames
	Cities           map[string]string
	Meta             map[string]catalogZoneNames
	Territories      map[string]string
}

func catalogJSON(path string, value any) {
	data, err := catalogZoneResources.ReadFile("cldr/" + path)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		panic(err)
	}
}

func initializeCatalogZones() {
	catalogZones.locales = map[string]catalogZoneLocale{}
	catalogZones.aliases = map[string]string{}
	catalogZones.territory = map[string]string{}
	catalogZones.primary = map[string]string{}
	catalogZones.periods = map[string][]catalogZonePeriod{}
	catalogZones.references = map[string]map[string]string{}
	var savings struct{ Zones map[string]catalogSavingsZone }
	catalogJSON("zoneSavings-icu78.json", &savings)
	catalogZones.savings = savings.Zones
	var bcp struct {
		Keyword struct {
			U struct {
				TZ map[string]json.RawMessage `json:"tz"`
			} `json:"u"`
		} `json:"keyword"`
	}
	catalogJSON("timezone-bcp47.json", &bcp)
	counts := map[string]int{}
	for key, raw := range bcp.Keyword.U.TZ {
		if strings.HasPrefix(key, "_") {
			continue
		}
		var entry struct {
			Alias      string `json:"_alias"`
			Deprecated bool   `json:"_deprecated"`
		}
		if json.Unmarshal(raw, &entry) != nil {
			continue
		}
		aliases := strings.Fields(entry.Alias)
		if len(aliases) == 0 {
			continue
		}
		canonical := aliases[0]
		for _, alias := range aliases {
			catalogZones.aliases[strings.ToLower(alias)] = canonical
		}
		// BCP47 zone keys begin with their ISO territory; fixed-offset and
		// unknown keys deliberately have no geographic region.
		if len(key) >= 2 && key != "gmt" && key != "unk" && !strings.HasPrefix(key, "utc") && !entry.Deprecated {
			territory := strings.ToUpper(key[:2])
			catalogZones.territory[canonical] = territory
			counts[territory]++
			catalogZones.primary[territory] = canonical
		}
	}
	for territory, count := range counts {
		if count != 1 {
			delete(catalogZones.primary, territory)
		}
	}
	var primary struct {
		Supplemental struct {
			Primary map[string]string `json:"primaryZones"`
		} `json:"supplemental"`
	}
	catalogJSON("primaryZones.json", &primary)
	for territory, zone := range primary.Supplemental.Primary {
		catalogZones.primary[territory] = zone
	}
	var meta struct {
		Supplemental struct {
			Meta struct {
				Info struct {
					Timezone json.RawMessage `json:"timezone"`
				} `json:"metazoneInfo"`
				Mappings []struct {
					Zone struct {
						Meta      string `json:"_other"`
						Zone      string `json:"_type"`
						Territory string `json:"_territory"`
					} `json:"mapZone"`
				} `json:"metazones"`
			} `json:"metaZones"`
		} `json:"supplemental"`
	}
	catalogJSON("metaZones.json", &meta)
	var walk func(string, json.RawMessage)
	walk = func(path string, raw json.RawMessage) {
		var periods []struct {
			Use struct {
				Meta string `json:"_mzone"`
				From string `json:"_from"`
				To   string `json:"_to"`
			} `json:"usesMetazone"`
		}
		if json.Unmarshal(raw, &periods) == nil {
			for _, p := range periods {
				from, _ := time.Parse("2006-01-02 15:04", p.Use.From)
				to, _ := time.Parse("2006-01-02 15:04", p.Use.To)
				catalogZones.periods[path] = append(catalogZones.periods[path], catalogZonePeriod{from, to, p.Use.Meta})
			}
			return
		}
		var children map[string]json.RawMessage
		if json.Unmarshal(raw, &children) != nil {
			return
		}
		for name, child := range children {
			next := name
			if path != "" {
				next = path + "/" + name
			}
			walk(next, child)
		}
	}
	walk("", meta.Supplemental.Meta.Info.Timezone)
	// ICU78's maintained metazone periods include the TZ2026c transitions
	// published after the base CLDR48 JSON release. Load the complete pinned
	// upstream period table, retaining CLDR's names/reference-zone mappings.
	var maintained struct {
		Supplemental struct {
			Meta struct {
				Info struct {
					Timezone map[string]json.RawMessage `json:"timezone"`
				} `json:"metazoneInfo"`
			} `json:"metaZones"`
		} `json:"supplemental"`
	}
	catalogJSON("metaZones-icu78-tz2026c.json", &maintained)
	for zone, periods := range maintained.Supplemental.Meta.Info.Timezone {
		catalogZones.periods[zone] = nil
		walk(zone, periods)
	}
	for _, m := range meta.Supplemental.Meta.Mappings {
		if catalogZones.references[m.Zone.Meta] == nil {
			catalogZones.references[m.Zone.Meta] = map[string]string{}
		}
		catalogZones.references[m.Zone.Meta][m.Zone.Territory] = m.Zone.Zone
	}
	for _, locale := range []string{"en", "zh"} {
		var names struct {
			Main map[string]struct {
				Dates struct {
					Names struct {
						Region   string          `json:"regionFormat"`
						Fallback string          `json:"fallbackFormat"`
						Zone     json.RawMessage `json:"zone"`
						Meta     map[string]struct {
							Long catalogZoneNames `json:"long"`
						} `json:"metazone"`
					} `json:"timeZoneNames"`
				} `json:"dates"`
			} `json:"main"`
		}
		catalogJSON("timeZoneNames-"+locale+".json", &names)
		n := names.Main[locale].Dates.Names
		data := catalogZoneLocale{Region: n.Region, Fallback: n.Fallback, Zones: map[string]catalogZoneNames{}, Cities: map[string]string{}, Meta: map[string]catalogZoneNames{}}
		for meta, entry := range n.Meta {
			data.Meta[meta] = entry.Long
		}
		var walkNames func(string, json.RawMessage)
		walkNames = func(path string, raw json.RawMessage) {
			var leaf struct {
				Type string           `json:"_type"`
				City string           `json:"exemplarCity"`
				Long catalogZoneNames `json:"long"`
			}
			if json.Unmarshal(raw, &leaf) != nil {
				return
			}
			if leaf.Type == "zone" {
				data.Zones[path], data.Cities[path] = leaf.Long, leaf.City
				return
			}
			var children map[string]json.RawMessage
			if json.Unmarshal(raw, &children) != nil {
				return
			}
			for name, child := range children {
				next := name
				if path != "" {
					next = path + "/" + name
				}
				walkNames(next, child)
			}
		}
		walkNames("", n.Zone)
		var territories struct {
			Main map[string]struct {
				Names struct {
					Territories map[string]string `json:"territories"`
				} `json:"localeDisplayNames"`
			} `json:"main"`
		}
		catalogJSON("territories-"+locale+".json", &territories)
		data.Territories = territories.Main[locale].Names.Territories
		catalogZones.locales[locale] = data
	}
}

func formatCatalogZone(zone string, now time.Time, locale string) string {
	catalogZones.Do(initializeCatalogZones)
	canonical := catalogZones.aliases[strings.ToLower(zone)]
	if canonical == "" {
		canonical = zone
	}
	location, err := time.LoadLocation(canonical)
	if err != nil {
		return zone
	}
	at := now.In(location)
	_, seconds := at.Zone()
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	offset := fmt.Sprintf("UTC%s%02d:%02d", sign, seconds/3600, seconds%3600/60)
	// V8's CanonicalizeTimeZoneID forces these request spellings to the
	// custom UTC object; other BCP47 zero-offset aliases retain the named
	// ICU Etc/GMT object. Canonical resolvedOptions alone loses this detail.
	upper := strings.ToUpper(zone)
	if canonical == "Etc/UTC" || upper == "GMT" || upper == "ETC/GMT" || upper == "GMT0" || upper == "GMT+0" || upper == "GMT-0" {
		return offset
	}
	language := catalogCopy(locale, "zh", "en")
	data := catalogZones.locales[language]
	name := catalogGenericZoneName(canonical, at, language, data)
	// The browser strips localized GMT offset fallback names, retaining its
	// stable UTC offset. Invalid zones retain the exact stored identifier.
	if name == "" || name == "GMT" || strings.HasPrefix(name, "GMT+") || strings.HasPrefix(name, "GMT-") {
		return offset
	}
	return offset + " · " + name
}

func catalogGenericZoneName(zone string, at time.Time, locale string, data catalogZoneLocale) string {
	own := data.Zones[zone]
	if own.Generic != "" {
		return own.Generic
	}
	meta := ""
	for _, period := range catalogZones.periods[zone] {
		if (period.From.IsZero() || !at.Before(period.From)) && (period.To.IsZero() || at.Before(period.To)) {
			meta = period.Meta
			break
		}
	}
	names := data.Meta[meta]
	if catalogStandardNameApplies(at) {
		standard := own.Standard
		if standard == "" {
			standard = names.Standard
		}
		if standard != "" && !strings.EqualFold(standard, names.Generic) {
			return standard
		}
	}
	if names.Generic != "" {
		references := catalogZones.references[meta]
		region := catalogCopy(locale, "CN", "US")
		golden := references[region]
		if golden == "" {
			golden = references["001"]
		}
		if golden != "" && golden != zone {
			if location, err := time.LoadLocation(golden); err == nil {
				// ICU compares the reference zone at this wall clock, including
				// both standard offset and DST savings, not merely at this UTC.
				wall := time.Date(at.Year(), at.Month(), at.Day(), at.Hour(), at.Minute(), at.Second(), at.Nanosecond(), location)
				raw, dst := catalogZoneOffsets(at)
				otherRaw, otherDST := catalogZoneOffsets(wall)
				if raw != otherRaw || dst != otherDST {
					territory := catalogZones.territory[zone]
					place := catalogZoneCity(zone, data)
					if references[territory] == zone && data.Territories[territory] != "" {
						place = data.Territories[territory]
					}
					return strings.NewReplacer("{0}", place, "{1}", names.Generic).Replace(data.Fallback)
				}
			}
		}
		return names.Generic
	}
	territory := catalogZones.territory[zone]
	if territory == "" {
		return ""
	}
	place := catalogZoneCity(zone, data)
	if catalogZones.primary[territory] == zone && data.Territories[territory] != "" {
		place = data.Territories[territory]
	}
	return strings.ReplaceAll(data.Region, "{0}", place)
}

func catalogZoneCity(zone string, data catalogZoneLocale) string {
	if city := data.Cities[zone]; city != "" {
		return city
	}
	if index := strings.LastIndexByte(zone, '/'); index >= 0 {
		return strings.ReplaceAll(zone[index+1:], "_", " ")
	}
	return zone
}

func catalogStandardNameApplies(at time.Time) bool {
	if zone, ok := catalogZones.savings[at.Location().String()]; ok && at.Unix() >= -2147483648 && at.Unix() <= 2147483647 {
		// ICU's raw/savings split can differ from IANA's isDST bit, including
		// negative-DST rules. Use upstream savings at the nearest transitions.
		seconds, current := at.Unix(), zone.Before
		for index, transition := range zone.Transitions {
			if seconds < transition.At {
				return current == 0 && (transition.At-seconds >= 184*86400 || transition.Savings == 0)
			}
			previous := current
			current = transition.Savings
			if index == len(zone.Transitions)-1 || seconds < zone.Transitions[index+1].At {
				if current != 0 || (seconds-transition.At < 184*86400 && previous != 0) {
					return false
				}
				if index+1 < len(zone.Transitions) && zone.Transitions[index+1].At-seconds < 184*86400 && zone.Transitions[index+1].Savings != 0 {
					return false
				}
				return true
			}
		}
	}
	if at.IsDST() {
		return false
	}
	from, to := at.ZoneBounds()
	const nearby = 184 * 24 * time.Hour
	if !from.IsZero() && at.Sub(from) < nearby && from.Add(-time.Nanosecond).In(at.Location()).IsDST() {
		return false
	}
	if !to.IsZero() && to.Sub(at) < nearby && to.In(at.Location()).IsDST() {
		return false
	}
	return true
}

func catalogZoneOffsets(at time.Time) (raw, dst int) {
	_, total := at.Zone()
	if !at.IsDST() {
		return total, 0
	}
	// Find the adjacent standard interval through the actual IANA transition
	// boundaries. DST need not be one hour (Lord Howe uses30 minutes).
	before, after := at, at
	for i := 0; i < 16; i++ {
		from, _ := before.ZoneBounds()
		if !from.IsZero() {
			before = from.Add(-time.Nanosecond).In(at.Location())
			if !before.IsDST() {
				_, standard := before.Zone()
				return standard, total - standard
			}
		}
		_, to := after.ZoneBounds()
		if !to.IsZero() {
			after = to.In(at.Location())
			if !after.IsDST() {
				_, standard := after.Zone()
				return standard, total - standard
			}
		}
		if from.IsZero() && to.IsZero() {
			break
		}
	}
	return total, 0
}

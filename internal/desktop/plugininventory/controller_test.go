package plugininventory

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"whaleisle/internal/dsh/api"
)

// This fixture uses the original pluginInventory/list wire shape. No Host,
// plugin installation, credentials or user configuration is used by this test.
const inventoryFixture = `{"managementAvailable":true,"entries":[
 {"entryId":"include:global/ok","moduleName":"@sample/shared","enabled":true,"fiberPhase":"active","meta":{"title":{"en":"Search","zh":"检索"}}},
 {"entryId":"include:global/broken","moduleName":"@sample/broken","enabled":false,"fiberPhase":"failed","meta":{"error":"manifest unavailable"}}
],"agentPresets":[
 {"id":"writer","name":"Writer","isDefault":false,"rows":[{"entryId":null,"moduleName":"@sample/writing","enabled":true,"fiberPhase":"active"}]},
 {"id":"default","name":"Default","isDefault":true,"rows":[{"entryId":"shared","moduleName":"@sample/shared","enabled":"conditional","condition":"ctx.scope.active","fiberPhase":null}]},
 {"id":"broken","isDefault":false,"broken":"composition missing","rows":[]}
]}`

type fixtureCaller func(context.Context, string, any, any) error

func (f fixtureCaller) Call(ctx context.Context, method string, args, out any) error {
	return f(ctx, method, args, out)
}

func TestPluginInventoryFixture(t *testing.T) {
	load := fixtureCaller(func(_ context.Context, method string, args, out any) error {
		if method != api.PluginInventoryListMethod || args != nil {
			t.Fatalf("unexpected RPC: %s %#v", method, args)
		}
		return json.Unmarshal([]byte(inventoryFixture), out)
	})
	t.Run("preset and global diagnostics remain distinct", func(t *testing.T) {
		var c Controller
		c.Attach(load)
		c.Refresh(context.Background())
		s := c.Snapshot().Value
		v := s.View("zh-CN")
		if !s.Ready || s.Busy || v.SelectedID != "default" || len(v.Presets) != 1 || v.Presets[0].Rows[0].Enabled != "conditional" || v.Presets[0].Rows[0].Phase != "" {
			t.Fatalf("default/conditional state: %#v / %#v", s, v)
		}
		if v.Presets[0].Rows[0].Condition != "ctx.scope.active" || len(v.Global) != 2 || v.Global[0].Phase != "failed" || v.Global[0].MetadataError != "manifest unavailable" {
			t.Fatalf("lost diagnostic: %#v", v)
		}
		if v.Global[1].Title != "检索" || v.Global[1].Key == v.Presets[0].Rows[0].Key {
			t.Fatalf("localized title or scoped row identity lost: %#v", v)
		}
		c.SelectPreset(s.Host, "writer")
		c.SetQuery(s.Host, "shared")
		v = c.Snapshot().Value.View("en")
		if v.SelectedID != "writer" || len(v.Presets) != 2 || len(v.Presets[0].Rows) != 0 || v.Presets[1].ID != "default" || len(v.Presets[1].Rows) != 1 || len(v.Global) != 1 {
			t.Fatalf("search must include other presets without changing selection: %#v", v)
		}
		c.SetQuery(s.Host, "检索")
		if got := c.Snapshot().Value.View("zh"); len(got.Global) != 1 || got.Empty {
			t.Fatalf("localized search: %#v", got)
		}
		c.SetQuery(s.Host, "")
		c.SelectPreset(s.Host, "broken")
		if got := c.Snapshot().Value.View("en"); got.Presets[0].Broken != "composition missing" {
			t.Fatalf("broken preset hidden: %#v", got)
		}
		// Published views are immutable: a later write publishes a new
		// revision and leaves the value an earlier frame captured intact.
		if s.SelectedID != "default" || s.Query != "" || s.Presets[1].Rows[0].Module != "@sample/shared" {
			t.Fatalf("earlier snapshot changed under later writes: %#v", s)
		}
	})
	t.Run("refresh failure preserves same host snapshot and local selection", func(t *testing.T) {
		var c Controller
		fail := false
		calls := 0
		c.Attach(fixtureCaller(func(ctx context.Context, method string, args, out any) error {
			calls++
			if fail {
				return errors.New("inventory temporarily unavailable")
			}
			return load.Call(ctx, method, args, out)
		}))
		c.Refresh(context.Background())
		s := c.Snapshot().Value
		c.SelectPreset(s.Host, "writer")
		c.SetQuery(s.Host, "writing")
		fail = true
		c.Refresh(context.Background())
		s = c.Snapshot().Value
		if !s.Ready || s.Err == nil || s.Busy || s.SelectedID != "writer" || s.Query != "writing" || len(s.Global) != 2 || calls != 2 {
			t.Fatalf("failed refresh destroyed context: %#v, calls=%d", s, calls)
		}
	})
	t.Run("host replacement cancels old read and rejects stale UI events", func(t *testing.T) {
		var c Controller
		started, release, done := make(chan context.Context, 1), make(chan struct{}), make(chan struct{})
		c.Attach(fixtureCaller(func(ctx context.Context, method string, args, out any) error {
			started <- ctx
			<-release
			return load.Call(ctx, method, args, out)
		}))
		oldHost := c.Snapshot().Value.Host
		go func() { c.Refresh(context.Background()); close(done) }()
		oldContext := <-started
		c.Refresh(context.Background()) // duplicate click while busy must not start another read
		c.Attach(nil)
		if oldContext.Err() == nil {
			t.Fatal("old Host read was not cancelled")
		}
		c.SetQuery(oldHost, "stale")
		c.SelectPreset(oldHost, "writer")
		c.ToggleExpanded(oldHost, "old row")
		close(release)
		<-done
		s := c.Snapshot().Value
		if s.Connected || s.Ready || s.Busy || len(s.Global) > 0 || s.Query != "" || s.SelectedID != "" || s.Expanded != "" {
			t.Fatalf("old Host result/action leaked after detach: %#v", s)
		}
		c.Attach(load)
		c.Refresh(context.Background())
		if !c.Snapshot().Value.Ready {
			t.Fatal("replacement Host cannot load")
		}
	})
}

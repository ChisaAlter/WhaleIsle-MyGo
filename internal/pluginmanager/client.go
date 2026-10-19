package pluginmanager

import (
	"context"
	"whaleisle/internal/dsh/api"
)

// Caller is dsh.Client's existing authenticated Typert boundary. Call already
// validates server-response/rpcId, unwraps result.value and reports result.error.
type Caller interface {
	Call(context.Context, string, any, any) error
}
type Client struct{ Caller Caller }

func (c Client) Inventory(ctx context.Context) (api.PluginInventorySnapshot, error) {
	var v api.PluginInventorySnapshot
	e := c.Caller.Call(ctx, "pluginInventory/list", nil, &v)
	return v, e
}
func (c Client) Bundles(ctx context.Context) ([]Bundle, error) {
	var v []Bundle
	e := c.Caller.Call(ctx, "pluginManager/listBundles", nil, &v)
	return v, e
}
func (c Client) Plugins(ctx context.Context) ([]Plugin, error) {
	var v []Plugin
	e := c.Caller.Call(ctx, "pluginManager/listPlugins", nil, &v)
	return v, e
}
func (c Client) Indexes(ctx context.Context) ([]Index, error) {
	var v []Index
	e := c.Caller.Call(ctx, "pluginMarket/listIndexes", nil, &v)
	return v, e
}
func (c Client) Search(ctx context.Context, url string, q Query) (SearchResult, error) {
	var v SearchResult
	e := c.Caller.Call(ctx, "pluginMarket/search", struct {
		URL   string `json:"url"`
		Query Query  `json:"query"`
	}{url, q}, &v)
	return v, e
}
func (c Client) SetIndexes(ctx context.Context, urls []string) error {
	return c.Caller.Call(ctx, "pluginMarket/setIndexes", struct {
		URLs []string `json:"urls"`
	}{urls}, nil)
}
func (c Client) Registries(ctx context.Context) (RegistryInfo, error) {
	var v RegistryInfo
	e := c.Caller.Call(ctx, "pluginManager/registries", nil, &v)
	return v, e
}
func (c Client) Inspect(ctx context.Context, spec string, registry *string) (Inspection, error) {
	var v Inspection
	e := c.Caller.Call(ctx, "pluginManager/inspect", struct {
		Spec    string `json:"spec"`
		Options struct {
			Registry *string `json:"registry,omitempty"`
		} `json:"options"`
	}{spec, struct {
		Registry *string `json:"registry,omitempty"`
	}{registry}}, &v)
	return v, e
}
func (c Client) Install(ctx context.Context, spec string, options InstallOptions) (Change, error) {
	var v Change
	e := c.Caller.Call(ctx, "pluginManager/installBundle", struct {
		Spec    string         `json:"spec"`
		Options InstallOptions `json:"options"`
	}{spec, options}, &v)
	return v, e
}
func (c Client) SetBundleEnabled(ctx context.Context, name string, enabled bool) (Change, error) {
	var v Change
	e := c.Caller.Call(ctx, "pluginManager/setBundleEnabled", struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}{name, enabled}, &v)
	return v, e
}
func (c Client) SetRowEnabled(ctx context.Context, id string, enabled bool) (Change, error) {
	var v Change
	e := c.Caller.Call(ctx, "pluginManager/setPluginEnabled", struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}{id, enabled}, &v)
	return v, e
}
func (c Client) Remove(ctx context.Context, name string) (Change, error) {
	var v Change
	e := c.Caller.Call(ctx, "pluginManager/removeBundle", struct {
		Name string `json:"name"`
	}{name}, &v)
	return v, e
}
func (c Client) CancelInstall(ctx context.Context, id string) (string, error) {
	var v struct {
		Status string `json:"status"`
	}
	e := c.Caller.Call(ctx, "pluginManager/cancelInstall", struct {
		ID string `json:"requestId"`
	}{id}, &v)
	return v.Status, e
}
func (c Client) WaitInstall(ctx context.Context, id string) (*Change, error) {
	var v *Change
	e := c.Caller.Call(ctx, "pluginManager/waitForInstall", struct {
		ID string `json:"requestId"`
	}{id}, &v)
	return v, e
}
func (c Client) Exempt(ctx context.Context, pkg, runtime string) (Change, error) {
	var v Change
	e := c.Caller.Call(ctx, "pluginManager/setVersionExemption", struct {
		Package    string `json:"packageVersion"`
		Runtime    string `json:"runtimeVersion"`
		Enabled    bool   `json:"enabled"`
		AcceptRisk bool   `json:"acceptRisk"`
	}{pkg, runtime, true, true}, &v)
	return v, e
}

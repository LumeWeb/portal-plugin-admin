package admin

import (
	"go.lumeweb.com/portal-plugin-admin/internal"
	"go.lumeweb.com/portal-plugin-admin/internal/api"
	"go.lumeweb.com/portal/core"
	portal_plugin_admin "go.lumeweb.com/web/go/portal-plugin-admin"
)

func GetPluginInfo() core.PluginInfo {
	return core.PluginInfo{
		ID: internal.PLUGIN_NAME,
		API: func() (core.API, []core.ContextBuilderOption, error) {
			return api.NewAPI()
		},
		Depends:    []string{"core"},
		TargetApps: []string{"admin"},
		WebBundles: core.NewWebBundles(core.NewWebBundle(portal_plugin_admin.GetFS())),
	}
}

func init() {
	core.RegisterPlugin(GetPluginInfo())
}

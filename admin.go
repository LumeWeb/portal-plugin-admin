package admin

import (
	"go.lumeweb.com/portal-plugin-admin/internal"
	"go.lumeweb.com/portal-plugin-admin/internal/api"
	"go.lumeweb.com/portal-plugin-admin/internal/service"
	"go.lumeweb.com/portal/core"
)

func init() {
	core.RegisterPlugin(core.PluginInfo{
		ID: internal.PluginName,
		API: func() (core.API, []core.ContextBuilderOption, error) {
			return api.NewAPI()
		},
		Depends: []string{"dashboard"},
		Services: func() ([]core.ServiceInfo, error) {
			return []core.ServiceInfo{
				{
					ID: service.ADMIN_CRON_SERVICE,
					Factory: func() (core.Service, []core.ContextBuilderOption, error) {
						return service.NewAdminCronService()
					},
					Depends: []string{core.CRON_SERVICE},
				},

				{
					ID: service.ADMIN_SETTINGS_SERVICE,
					Factory: func() (core.Service, []core.ContextBuilderOption, error) {
						return service.NewAdminSettingsService()
					},
					Depends: []string{core.CONFIG_SERVICE},
				},
			}, nil
		},
	})
}

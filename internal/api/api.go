package api

import (
	_ "embed"

	"go.lumeweb.com/portal-plugin-admin/internal"
	pluginConfig "go.lumeweb.com/portal-plugin-admin/internal/config"
	router "go.lumeweb.com/portal-router"
	"go.lumeweb.com/portal/config"
	"go.lumeweb.com/portal/core"
	portal_admin "go.lumeweb.com/web/go/portal-admin"
)

var _ core.API = (*API)(nil)

type API struct {
	*core.BaseComponent
}

func (a API) Name() string {
	return internal.PLUGIN_NAME
}

func (a API) ID() string {
	return a.Name()
}

func (a API) Subdomain() string {
	return internal.PLUGIN_NAME
}

func (a API) Configure(r router.Router, _ core.AccessService) error {
	router.MustDefaultStaticSetup(r, router.NewAppFilesystem(portal_admin.GetFS(), a.Config().Config().Core.Domain))
	return nil
}

func (a API) AuthTokenName() string {
	return core.AUTH_TOKEN_NAME
}

func (a API) GetConfig() config.APIConfig {
	return &pluginConfig.APIConfig{}
}

// OpenAPIInfo provides top-level API metadata for Swagger documentation.
// Since this plugin only serves a static app and has no API endpoints,
// this can return a minimal definition.
func (a API) OpenAPIInfo() router.APIInfoDefinition {
	return router.APIInfo().
		Title("Portal Admin API").
		Description("Provides administrative endpoints for managing the Portal.")
}

func NewAPI() (core.API, []core.ContextBuilderOption, error) {
	api := &API{}

	return api, nil, nil
}

package api

import (
	"github.com/gorilla/mux"
	"go.lumeweb.com/portal-plugin-admin/internal"
	"go.lumeweb.com/portal/config"
	"go.lumeweb.com/portal/core"
)

var _ core.API = (*API)(nil)

type API struct {
	ctx core.Context
}

func (A API) Name() string {
	return internal.PluginName
}

func (A API) Subdomain() string {
	return internal.PluginName
}

func (A API) Configure(router *mux.Router) error {
}

func (A API) AuthTokenName() string {
	return core.AUTH_TOKEN_NAME
}

func (A API) Config() config.APIConfig {
	return nil
}

func NewAPI() (core.API, []core.ContextBuilderOption, error) {
	api := &API{}

	opts := core.ContextOptions(
		core.ContextWithStartupFunc(func(ctx core.Context) error {
			api.ctx = ctx
			return nil
		}),
	)

	return &API{}, opts, nil
}

package service

import (
	"github.com/invopop/jsonschema"
	"go.lumeweb.com/portal/core"
)

const ADMIN_SETTINGS_SERVICE = "admin_settings"

var _ core.Service = (*AdminSettingsService)(nil)

type AdminSettingsService struct {
	ctx core.Context
}

func (a *AdminSettingsService) ID() string {
	return ADMIN_SETTINGS_SERVICE
}

func NewAdminSettingsService() (core.Service, []core.ContextBuilderOption, error) {
	adminSettingsService := &AdminSettingsService{}

	opts := core.ContextOptions(
		core.ContextWithStartupFunc(func(ctx core.Context) error {
			adminSettingsService.ctx = ctx
			return nil
		}),
	)

	return adminSettingsService, opts, nil
}

func (a *AdminSettingsService) ListSettings() (*jsonschema.Schema, error) {
	return jsonschema.Reflect(a.ctx.Config().Config()), nil
}

package api

import (
	"fmt"
	"github.com/gorilla/mux"
	"go.lumeweb.com/httputil"
	"go.lumeweb.com/portal-plugin-admin/internal/api/messages"
	"net/http"
)

func (a *API) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	settings := a.settings.GetSchema()

	ctx.Encode(settings)
}

func (a *API) handleListSettings(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	settings := a.settings.GetSettings()

	ctx.Encode(settings)
}

func (a *API) handleGetSetting(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	vars := mux.Vars(r)
	id := vars["id"]

	setting := a.settings.GetSetting(id)
	if setting == nil {
		_ = ctx.Error(fmt.Errorf("Setting not found"), http.StatusNotFound)
		return
	}

	ctx.Encode(setting)
}

func (a *API) handleUpdateSetting(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	vars := mux.Vars(r)
	id := vars["id"]

	setting := a.settings.GetSetting(id)
	if setting == nil {
		_ = ctx.Error(fmt.Errorf("Setting not found"), http.StatusNotFound)
		return
	}

	var data messages.SettingUpdateRequest
	if err := ctx.Decode(&data); err != nil {
		return
	}

	if err := a.settings.UpdateSetting(
		&messages.SettingsItem{
			Key:   setting.Key,
			Value: data.Value,
		}); err != nil {
		_ = ctx.Error(err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

package api

import (
	"fmt"
	"github.com/gorilla/mux"
	"go.lumeweb.com/httputil"
	"go.lumeweb.com/portal-plugin-admin/internal/api/messages"
	"go.lumeweb.com/portal-plugin-admin/internal/internal"
	"net/http"
	"strconv"
	"strings"
)

func (a *API) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	settings := a.settings.GetSchema()

	ctx.Encode(settings)
}

func (a *API) handleListSettings(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)

	// Parse query parameters
	query := r.URL.Query()
	start, _ := strconv.Atoi(query.Get("_start"))
	end, _ := strconv.Atoi(query.Get("_end"))
	keyLike := query.Get("key_like")
	valueLike := query.Get("value_like")

	// Get all settings
	allSettings := a.settings.GetSettings()

	// Filter settings
	filteredSettings := filterSettings(allSettings, keyLike, valueLike)

	// Apply pagination
	totalCount := len(filteredSettings)
	paginatedSettings := paginateSettings(filteredSettings, start, end)

	// Set Content-Range header
	w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))

	ctx.Encode(paginatedSettings)
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

	if !setting.Editable {
		_ = ctx.Error(fmt.Errorf("Setting is not editable"), http.StatusForbidden)
		return
	}

	setting, err := normalizeSetting(setting, data.Value)

	// Verify the data type before updating
	if err != nil {
		_ = ctx.Error(err, http.StatusBadRequest)
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

func normalizeSetting(setting *messages.SettingsItem, newValue any) (*messages.SettingsItem, error) {
	normalized, err := internal.NormalizeSetting(setting.Value, newValue)
	if err != nil {
		return nil, err
	}
	setting.Value = normalized
	return setting, nil
}

func filterSettings(settings []*messages.SettingsItem, keyLike, valueLike string) []*messages.SettingsItem {
	if keyLike == "" && valueLike == "" {
		return settings
	}

	var filtered []*messages.SettingsItem
	for _, setting := range settings {
		if (keyLike == "" || strings.Contains(strings.ToLower(setting.Key), strings.ToLower(keyLike))) &&
			(valueLike == "" || strings.Contains(fmt.Sprintf("%v", setting.Value), valueLike)) {
			filtered = append(filtered, setting)
		}
	}
	return filtered
}

func paginateSettings(settings []*messages.SettingsItem, start, end int) []*messages.SettingsItem {
	if start < 0 {
		start = 0
	}
	if end > len(settings) || end == 0 {
		end = len(settings)
	}
	if start > end {
		return []*messages.SettingsItem{}
	}
	return settings[start:end]
}

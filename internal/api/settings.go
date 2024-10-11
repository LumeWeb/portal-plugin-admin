package api

import (
	"go.lumeweb.com/httputil"
	"net/http"
)

func (a *API) handleListSettings(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	settings, err := a.settings.ListSettings()
	if ctx.Check("Failed to list settings", err) != nil {
		return
	}

	ctx.Encode(settings)
}

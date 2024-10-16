package api

import (
	"go.lumeweb.com/httputil"
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

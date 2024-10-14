package api

import (
	"go.lumeweb.com/httputil"
	"net/http"
)

func (a *API) handleListSettings(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	settings := a.settings.ListSettings()

	ctx.Encode(settings)
}
func (a *API) handleListSettings1(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	settings := a.settings.ListSettings1()

	ctx.Encode(settings)
}

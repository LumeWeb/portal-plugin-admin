package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	pluginConfig "go.lumeweb.com/portal-plugin-admin/internal/config"
)

// PprofSharedSecretAuth validates a request using a shared secret
// provided as a Bearer token in the Authorization header.
// If PprofSecret is empty in config, the middleware is a pass-through.
func PprofSharedSecretAuth(cfg *pluginConfig.APIConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			configuredSecret := cfg.PprofSecret

			if configuredSecret == "" {
				return next(c)
			}

			providedSecret := extractBearerToken(c)

			if providedSecret == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Unauthorized: missing pprof secret",
				})
			}

			if subtle.ConstantTimeCompare([]byte(providedSecret), []byte(configuredSecret)) != 1 {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Unauthorized: invalid pprof secret",
				})
			}

			return next(c)
		}
	}
}

func extractBearerToken(c echo.Context) string {
	authHeader := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

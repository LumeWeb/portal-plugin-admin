package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	pluginConfig "go.lumeweb.com/portal-plugin-admin/internal/config"
)

func TestPprofSharedSecretAuth_ValidBearerToken(t *testing.T) {
	e := echo.New()
	cfg := &pluginConfig.APIConfig{PprofSecret: "test-secret"}

	req := httptest.NewRequest(http.MethodGet, "/api/debug/pprof/", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := PprofSharedSecretAuth(cfg)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestPprofSharedSecretAuth_InvalidSecret(t *testing.T) {
	e := echo.New()
	cfg := &pluginConfig.APIConfig{PprofSecret: "test-secret"}

	req := httptest.NewRequest(http.MethodGet, "/api/debug/pprof/", nil)
	req.Header.Set("Authorization", "Bearer wrong-secret")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := PprofSharedSecretAuth(cfg)(func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	})

	_ = handler(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestPprofSharedSecretAuth_MissingSecret(t *testing.T) {
	e := echo.New()
	cfg := &pluginConfig.APIConfig{PprofSecret: "test-secret"}

	req := httptest.NewRequest(http.MethodGet, "/api/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := PprofSharedSecretAuth(cfg)(func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	})

	_ = handler(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestPprofSharedSecretAuth_EmptyConfigSecret_Disabled(t *testing.T) {
	e := echo.New()
	cfg := &pluginConfig.APIConfig{PprofSecret: ""}

	req := httptest.NewRequest(http.MethodGet, "/api/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := PprofSharedSecretAuth(cfg)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestPprofSharedSecretAuth_NoBearerPrefix(t *testing.T) {
	e := echo.New()
	cfg := &pluginConfig.APIConfig{PprofSecret: "test-secret"}

	req := httptest.NewRequest(http.MethodGet, "/api/debug/pprof/", nil)
	req.Header.Set("Authorization", "test-secret")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := PprofSharedSecretAuth(cfg)(func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	})

	_ = handler(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

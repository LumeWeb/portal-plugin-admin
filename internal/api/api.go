package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"

	"github.com/labstack/echo/v4"
	pprofhttp "net/http/pprof"

	"go.lumeweb.com/httputil"
	"go.lumeweb.com/portal-middleware/middleware"
	"go.lumeweb.com/portal-middleware/auth/jwt"
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
	blockProfileRate atomic.Int32
	mutexFraction    atomic.Int32
}

func (a *API) Name() string {
	return internal.PLUGIN_NAME
}

func (a *API) ID() string {
	return a.Name()
}

func (a *API) Subdomain() string {
	return internal.PLUGIN_NAME
}

func (a *API) Configure(r router.Router, accessSvc core.AccessService) error {
	router.MustDefaultStaticSetup(r, router.NewAppFilesystem(portal_admin.GetFS(), a.Config().Config().Core.Domain))

	authMw := middleware.AuthMiddleware(a.Context(), middleware.WithAuthPurpose(jwt.PurposeLogin))
	accessMw := middleware.AccessMiddleware(a.Context())

	routes := a.buildPprofRoutes(authMw, accessMw)
	if err := router.RegisterRoutes(r, accessSvc, a.Subdomain(), routes); err != nil {
		return fmt.Errorf("failed to register pprof routes: %w", err)
	}

	return nil
}

func (a *API) AuthTokenName() string {
	return core.AUTH_TOKEN_NAME
}

func (a *API) GetConfig() config.APIConfig {
	return &pluginConfig.APIConfig{}
}

func (a *API) OpenAPIInfo() router.APIInfoDefinition {
	return router.APIInfo().
		Title("Portal Admin API").
		Description("Provides administrative endpoints for managing the Portal.")
}

func NewAPI() (core.API, []core.ContextBuilderOption, error) {
	api := &API{}

	return api, nil, nil
}

func (a *API) buildPprofRoutes(authMw, accessMw echo.MiddlewareFunc) []router.Route {
	return []router.Route{
		router.NewRoute(http.MethodGet, "/api/debug/pprof/", a.pprofIndex,
			router.WithSwaggerOptions(
				router.WithSummary("pprof index"),
				router.WithDescription("Returns a page listing available pprof profiles."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "HTML index of available profiles"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/profile", a.pprofProfile,
			router.WithSwaggerOptions(
				router.WithSummary("CPU profile"),
				router.WithDescription("Returns a CPU profile for the specified duration. Supports 'seconds' and 'debug' query parameters."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "CPU profile data"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/trace", a.pprofTrace,
			router.WithSwaggerOptions(
				router.WithSummary("Execution trace"),
				router.WithDescription("Returns an execution trace. Supports 'seconds' query parameter."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "Execution trace data"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/cmdline", a.pprofCmdline,
			router.WithSwaggerOptions(
				router.WithSummary("Command line"),
				router.WithDescription("Returns the command line of the running program."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "Command line output"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/symbol", a.pprofSymbol,
			router.WithSwaggerOptions(
				router.WithSummary("Symbol lookup"),
				router.WithDescription("Looks up program counters and returns function names."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "Symbol lookup results"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/goroutine", a.pprofGoroutine,
			router.WithSwaggerOptions(
				router.WithSummary("Goroutine profile"),
				router.WithDescription("Returns stack traces of all current goroutines."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "Goroutine profile data"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/heap", a.pprofHeap,
			router.WithSwaggerOptions(
				router.WithSummary("Heap profile"),
				router.WithDescription("Returns a sampling of memory allocations."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "Heap profile data"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/threadcreate", a.pprofThreadcreate,
			router.WithSwaggerOptions(
				router.WithSummary("Thread creation profile"),
				router.WithDescription("Returns stack traces that led to the creation of new OS threads."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "Thread creation profile data"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/block", a.pprofBlock,
			router.WithSwaggerOptions(
				router.WithSummary("Block profile"),
				router.WithDescription("Returns stack traces that led to blocking on synchronization primitives."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "Block profile data"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/mutex", a.pprofMutex,
			router.WithSwaggerOptions(
				router.WithSummary("Mutex profile"),
				router.WithDescription("Returns stack traces of holders of contended mutexes."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "Mutex profile data"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodGet, "/api/debug/pprof/status", a.pprofStatus,
			router.WithSwaggerOptions(
				router.WithSummary("Profiling status"),
				router.WithDescription("Returns the current block and mutex profiling rates."),
				router.WithTags("Profiling"),
				router.WithSuccessResponse(http.StatusOK, "Current profiling status",
					router.WithJSONContent(ProfilingStatusResponse{}),
				),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodPut, "/api/debug/pprof/block", a.pprofBlockUpdate,
			router.WithSwaggerOptions(
				router.WithSummary("Set block profile rate"),
				router.WithDescription("Sets the block profiling rate. 0 disables, 1 captures all, higher values sample."),
				router.WithTags("Profiling"),
				router.WithRequestBody(BlockProfileRequest{}, "Block profile rate", true),
				router.WithSuccessResponse(http.StatusNoContent, "Block profile rate updated"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
		router.NewRoute(http.MethodPut, "/api/debug/pprof/mutex", a.pprofMutexUpdate,
			router.WithSwaggerOptions(
				router.WithSummary("Set mutex profile fraction"),
				router.WithDescription("Sets the mutex profiling fraction. 0 disables, 1 captures all, 100 samples 1%."),
				router.WithTags("Profiling"),
				router.WithRequestBody(MutexProfileRequest{}, "Mutex profile fraction", true),
				router.WithSuccessResponse(http.StatusNoContent, "Mutex profile fraction updated"),
			),
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		),
	}
}

func (a *API) pprofIndex(c echo.Context) error {
	pprofhttp.Index(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofProfile(c echo.Context) error {
	pprofhttp.Profile(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofTrace(c echo.Context) error {
	pprofhttp.Trace(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofCmdline(c echo.Context) error {
	pprofhttp.Cmdline(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofSymbol(c echo.Context) error {
	pprofhttp.Symbol(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofGoroutine(c echo.Context) error {
	pprofhttp.Handler("goroutine").ServeHTTP(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofHeap(c echo.Context) error {
	pprofhttp.Handler("heap").ServeHTTP(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofThreadcreate(c echo.Context) error {
	pprofhttp.Handler("threadcreate").ServeHTTP(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofBlock(c echo.Context) error {
	pprofhttp.Handler("block").ServeHTTP(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofMutex(c echo.Context) error {
	pprofhttp.Handler("mutex").ServeHTTP(c.Response().Writer, c.Request())
	return nil
}

func (a *API) pprofStatus(c echo.Context) error {
	ctx := httputil.Context(c)

	model := &ProfilingStatusResponse{
		BlockProfileRate: int(a.blockProfileRate.Load()),
		MutexFraction:   int(a.mutexFraction.Load()),
	}

	var responseDto ProfilingStatusResponse
	if err := responseDto.FromModel(model); err != nil {
		return ctx.Error(err, http.StatusInternalServerError)
	}

	return httputil.EncodeResponse[*ProfilingStatusResponse](ctx, model, &responseDto)
}

func (a *API) pprofBlockUpdate(c echo.Context) error {
	ctx := httputil.Context(c)

	var requestDto BlockProfileRequest
	_, ok := httputil.DecodeAndValidateRequest[*BlockProfileRequest, *BlockProfileRequest](ctx, &requestDto)
	if !ok {
		return nil
	}

	runtime.SetBlockProfileRate(requestDto.Rate)
	a.blockProfileRate.Store(int32(requestDto.Rate))

	return c.NoContent(http.StatusNoContent)
}

func (a *API) pprofMutexUpdate(c echo.Context) error {
	ctx := httputil.Context(c)

	var requestDto MutexProfileRequest
	_, ok := httputil.DecodeAndValidateRequest[*MutexProfileRequest, *MutexProfileRequest](ctx, &requestDto)
	if !ok {
		return nil
	}

	runtime.SetMutexProfileFraction(requestDto.Fraction)
	a.mutexFraction.Store(int32(requestDto.Fraction))

	return c.NoContent(http.StatusNoContent)
}

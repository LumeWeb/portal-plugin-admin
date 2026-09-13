package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"go.lumeweb.com/httputil"
	mcontext "go.lumeweb.com/portal-middleware/context"
	router "go.lumeweb.com/portal-router"
	"go.lumeweb.com/portal/core"
	"go.lumeweb.com/portal/db/models"
	"go.lumeweb.com/queryutil"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// userListEntityName is used for the Content-Range header on the list route.
const userListEntityName = "users"

// buildUserRoutes constructs the admin-protected user-management routes. They
// share the same login-JWT auth middleware, access middleware and
// ACCESS_ADMIN_ROLE requirement as the existing profiling routes.
func (a *API) buildUserRoutes(authMw, accessMw echo.MiddlewareFunc) []router.Route {
	applyAuth := func(swaggerOpts ...router.SwaggerOption) []router.RouteOption {
		return append(
			[]router.RouteOption{router.WithSwaggerOptions(swaggerOpts...)},
			router.WithAccess(core.ACCESS_ADMIN_ROLE),
			router.WithMiddlewares(authMw, accessMw),
		)
	}

	return []router.Route{
		router.NewRoute(http.MethodGet, "/api/users", a.listUsers,
			applyAuth(
				router.WithSummary("List users"),
				router.WithDescription("Returns a filterable, sortable, paginated list of portal users. Authentication- and database-sensitive fields are never returned."),
				router.WithTags("Users"),
				router.WithQueryParam("email", "Filter by exact email address", ""),
				router.WithQueryParam("verified", "Filter by verification state (true/false)", false),
				router.WithQueryParam("_sort", "Comma-separated sort fields (e.g. id,email,created_at)", ""),
				router.WithQueryParam("_order", "Sort order (asc/desc)", ""),
				router.WithPaginationParams(),
				router.WithSuccessResponse(http.StatusOK, "List of users",
					router.WithJSONContent(UserListResponse{}),
					router.WithTotalCountHeader(),
				),
			)...,
		),
		router.NewRoute(http.MethodPost, "/api/users", a.createUser,
			applyAuth(
				router.WithSummary("Create user"),
				router.WithDescription("Creates a portal account with an email and password, plus optional names and verification-email behavior. The password is hashed server-side and never returned."),
				router.WithTags("Users"),
				router.WithRequestBody(&UserCreateRequest{}, "Account details", true),
				router.WithSuccessResponse(http.StatusCreated, "User created", router.WithJSONContent(&UserResponse{})),
			)...,
		),
		router.NewRoute(http.MethodGet, "/api/users/:id", a.getUser,
			applyAuth(
				router.WithSummary("Get user"),
				router.WithDescription("Returns a single portal user by numeric ID."),
				router.WithTags("Users"),
				router.WithPathParam("id", "Numeric ID of the user", ""),
				router.WithSuccessResponse(http.StatusOK, "User details", router.WithJSONContent(&UserResponse{})),
			)...,
		),
		router.NewRoute(http.MethodPatch, "/api/users/:id", a.updateUser,
			applyAuth(
				router.WithSummary("Update user"),
				router.WithDescription("Partially updates a portal user. Fields that are omitted are left unchanged; an explicit false or empty value is applied. A new password is hashed server-side."),
				router.WithTags("Users"),
				router.WithPathParam("id", "Numeric ID of the user", ""),
				router.WithRequestBody(&UserUpdateRequest{}, "Fields to change (omitted fields are left unchanged)", true),
				router.WithSuccessResponse(http.StatusOK, "User updated", router.WithJSONContent(&UserResponse{})),
			)...,
		),
		router.NewRoute(http.MethodDelete, "/api/users/:id", a.deleteUser,
			applyAuth(
				router.WithSummary("Delete user"),
				router.WithDescription("Permanently deletes a portal user. Deleting your own (the authenticated administrator's) account is rejected."),
				router.WithTags("Users"),
				router.WithPathParam("id", "Numeric ID of the user", ""),
				router.WithoutDefaultSuccessResponse(),
				router.WithSuccessResponse(http.StatusNoContent, "User deleted"),
			)...,
		),
	}
}

// parseUserID parses the ":id" path parameter as a positive numeric user ID.
func parseUserID(c echo.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid user id")
	}
	return uint(id), nil
}

// listUsers returns a filterable, sortable, paginated list of portal users.
// The read-only query runs directly against the component DB with request
// context because the pinned portal UserService exposes no list method. It
// uses a separate filtered count so Content-Range and total reflect the full
// filtered set, not the current page.
func (a *API) listUsers(c echo.Context) error {
	ctx := httputil.Context(c)
	reqCtx := c.Request().Context()

	filters, sorts, pagination, err := queryutil.ParseQuery(c.Request().URL.Query())
	if err != nil {
		return ctx.Error(err, http.StatusBadRequest)
	}

	baseQuery := func() *gorm.DB {
		return a.DB().WithContext(reqCtx).Model(&models.User{})
	}

	// Filtered count (no pagination).
	countQuery := queryutil.ApplyFilters(baseQuery(), filters, nil)
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return ctx.Error(err, http.StatusInternalServerError)
	}

	// Filtered, sorted, paginated data query.
	query := queryutil.ApplySort(queryutil.ApplyFilters(baseQuery(), filters, nil), sorts)
	query = queryutil.ApplyPagination(query, pagination)

	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		return ctx.Error(err, http.StatusInternalServerError)
	}

	responses := make([]UserResponse, 0, len(users))
	for i := range users {
		responses = append(responses, userToResponse(&users[i]))
	}

	resultCount := queryutil.GetResultCount(responses)
	c.Response().Header().Set("Content-Range", queryutil.FormatContentRange(userListEntityName, pagination, resultCount, int(total)))
	c.Response().Header().Set("X-Total-Count", strconv.FormatInt(total, 10))

	return c.JSON(http.StatusOK, UserListResponse{
		Data:  responses,
		Total: total,
	})
}

// createUser creates a portal account via UserService.CreateAccount and then
// applies any provided names through an approved service update.
func (a *API) createUser(c echo.Context) error {
	ctx := httputil.Context(c)
	reqCtx := c.Request().Context()

	var req UserCreateRequest
	if _, ok := httputil.DecodeAndValidateRequest[*UserCreateRequest, *UserCreateRequest](ctx, &req); !ok {
		return nil
	}

	userSvc := core.GetService[core.UserService](a.Context(), core.USER_SERVICE)

	user, err := userSvc.CreateAccount(reqCtx, req.Email, req.Password, req.VerifyEmail)
	if err != nil {
		return a.handleUserServiceError(ctx, err)
	}

	if req.FirstName != "" || req.LastName != "" {
		if err := userSvc.UpdateAccountName(reqCtx, user.ID, req.FirstName, req.LastName); err != nil {
			return a.handleUserServiceError(ctx, err)
		}
		// Reflect the applied names by re-fetching the account.
		_, fresh, rerr := userSvc.AccountExists(reqCtx, user.ID)
		if rerr != nil {
			return a.handleUserServiceError(ctx, rerr)
		}
		if fresh == nil {
			return ctx.Error(core.NewAccountError(core.ErrKeyUserNotFound, nil), http.StatusNotFound)
		}
		user = fresh
	}

	return c.JSON(http.StatusCreated, userToResponse(user))
}

// getUser returns a single user by numeric ID.
func (a *API) getUser(c echo.Context) error {
	ctx := httputil.Context(c)
	reqCtx := c.Request().Context()

	userID, err := parseUserID(c)
	if err != nil {
		return ctx.Error(err, http.StatusBadRequest)
	}

	userSvc := core.GetService[core.UserService](a.Context(), core.USER_SERVICE)

	exists, user, err := userSvc.AccountExists(reqCtx, userID)
	if err != nil {
		return a.handleUserServiceError(ctx, err)
	}
	if !exists {
		return ctx.Error(core.NewAccountError(core.ErrKeyUserNotFound, nil), http.StatusNotFound)
	}

	return ctx.JSON(http.StatusOK, userToResponse(user))
}

// updateUser partially updates a user via the dedicated UserUpdateRequest
// patch semantics: empty PATCH rejected, omitted fields preserved, explicit
// false/empty applied, duplicate email detected, and password hashed before
// any update payload is produced.
func (a *API) updateUser(c echo.Context) error {
	ctx := httputil.Context(c)
	reqCtx := c.Request().Context()

	userID, err := parseUserID(c)
	if err != nil {
		return ctx.Error(err, http.StatusBadRequest)
	}

	var req UserUpdateRequest
	if _, ok := httputil.DecodeAndValidateRequest[*UserUpdateRequest, *UserUpdateRequest](ctx, &req); !ok {
		return nil
	}
	if !req.HasUpdates() {
		return ctx.Error(core.NewAccountError(core.ErrKeyAccountUpdateFailed, errNoUserUpdates), http.StatusUnprocessableEntity)
	}

	userSvc := core.GetService[core.UserService](a.Context(), core.USER_SERVICE)

	// The user must already exist.
	exists, _, err := userSvc.AccountExists(reqCtx, userID)
	if err != nil {
		return a.handleUserServiceError(ctx, err)
	}
	if !exists {
		return ctx.Error(core.NewAccountError(core.ErrKeyUserNotFound, nil), http.StatusNotFound)
	}

	// Detect duplicate email before mutating the account.
	if req.Email != nil {
		emailExists, other, eerr := userSvc.EmailExists(reqCtx, *req.Email)
		if eerr != nil {
			return a.handleUserServiceError(ctx, eerr)
		}
		if emailExists && other != nil && other.ID != userID {
			return ctx.Error(core.NewAccountError(core.ErrKeyEmailAlreadyExists, nil), http.StatusConflict)
		}
	}

	// Convert to an allowlisted update map, hashing any password first. The
	// raw password never enters this map nor any log/response.
	updates, err := req.BuildUpdates(userSvc.HashPassword)
	if err != nil {
		return a.handleUserServiceError(ctx, err)
	}

	if err := userSvc.UpdateAccountInfo(reqCtx, userID, updates); err != nil {
		return a.handleUserServiceError(ctx, err)
	}

	// Re-fetch to return a refreshed, safe response.
	_, fresh, err := userSvc.AccountExists(reqCtx, userID)
	if err != nil {
		return a.handleUserServiceError(ctx, err)
	}
	if fresh == nil {
		return ctx.Error(core.NewAccountError(core.ErrKeyUserNotFound, nil), http.StatusNotFound)
	}

	return ctx.JSON(http.StatusOK, userToResponse(fresh))
}

// deleteUser permanently deletes a user via UserService.DeleteAccount. It
// rejects deletion of the calling administrator's own account to prevent an
// immediate lockout.
func (a *API) deleteUser(c echo.Context) error {
	ctx := httputil.Context(c)
	reqCtx := c.Request().Context()

	userID, err := parseUserID(c)
	if err != nil {
		return ctx.Error(err, http.StatusBadRequest)
	}

	// Protect the calling administrator.
	adminID, err := mcontext.GetUserID(c)
	if err != nil {
		return ctx.Error(err, http.StatusUnauthorized)
	}
	if adminID == userID {
		return ctx.Error(core.NewAccountError(core.ErrKeyAccountUpdateFailed, errors.New("cannot delete your own account")), http.StatusConflict)
	}

	userSvc := core.GetService[core.UserService](a.Context(), core.USER_SERVICE)

	if err := userSvc.DeleteAccount(reqCtx, userID); err != nil {
		return a.handleUserServiceError(ctx, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// handleUserServiceError maps errors returned by core.UserService to HTTP
// responses. Account errors carry their own status codes (e.g. duplicate
// email => 409, user not found => 404); unknown account errors and generic
// service failures fall back to 500.
func (a *API) handleUserServiceError(ctx httputil.RequestContext, err error) error {
	if err == nil {
		return nil
	}

	apiErr := core.AsAccountError(err)
	if apiErr != nil {
		return ctx.Error(apiErr, apiErr.HttpStatus())
	}

	a.Logger().Error("user service operation failed", zap.Error(err))
	return ctx.Error(err, http.StatusInternalServerError)
}

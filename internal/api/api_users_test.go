package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.lumeweb.com/portal-plugin-admin/internal"
	"go.lumeweb.com/portal/core"
	coreTesting "go.lumeweb.com/portal/core/testing"
	"go.lumeweb.com/portal/db/models"
	"gorm.io/gorm"
)

// getUserAPITestOptions returns the shared test options for the user-management
// API. It reuses the admin API options and adds a real SQLite database so the
// list endpoint (which queries models.User directly) can be exercised.
func getUserAPITestOptions() coreTesting.TestContextBuilderOption {
	return coreTesting.CombineOptions(
		getAdminAPITestOptions(),
		coreTesting.WithSQLite(),
	)
}

// setupUserAdminAuth establishes an authenticated (login-purpose) admin
// identity. The default access mock grants admin access to every user ID, so
// the returned token identifies the caller for delete self-protection.
// wirePluginDB links the plugin API's underlying gorm handle to the per-test
// SQLite database exposed by the test context. The list endpoint queries
// models.User directly through a.DB(), which is not hooked up by WithSQLite
// alone, so we must point it at the same database used for seeding.
func wirePluginDB(ctx coreTesting.TestContext) {
	core.GetAPI(internal.PLUGIN_NAME).SetDB(ctx.DB())
}

func setupUserAdminAuth(tb testing.TB, ctx coreTesting.TestContext, userID uint) string {
	tb.Helper()
	wirePluginDB(ctx)
	userSvc := coreTesting.GetMockUserService(ctx)
	mockUser := &models.User{Model: gorm.Model{ID: userID}, Email: "admin@example.com", Role: "admin"}
	userSvc.EXPECT().AccountExists(mock.Anything, userID).Return(true, mockUser, nil).Maybe()

	jwtHelper := coreTesting.NewJWTHelper(ctx)
	token, err := jwtHelper.CreateLoginToken(userID)
	require.NoError(tb, err, "failed to create admin login token")
	return token
}

func userRequest(tb testing.TB, ctx coreTesting.TestContext, method, path string, body []byte, token string) *httptest.ResponseRecorder {
	tb.Helper()
	req := ctx.NewAPIRequest(method, path, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	ctx.Router().ServeHTTP(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// Step 2 & 3: DTO behaviour (no DB / no context required)
// ---------------------------------------------------------------------------

func TestUserUpdateRequest_HasUpdates(t *testing.T) {
	cases := []struct {
		name string
		req  UserUpdateRequest
		want bool
	}{
		{"empty patch rejected", UserUpdateRequest{}, false},
		{"only first name", UserUpdateRequest{FirstName: strPtr("Jane")}, true},
		{"only last name", UserUpdateRequest{LastName: strPtr("Doe")}, true},
		{"only email", UserUpdateRequest{Email: strPtr("j@example.com")}, true},
		{"explicit false verified", UserUpdateRequest{Verified: boolPtr(false)}, true},
		{"only password", UserUpdateRequest{Password: strPtr("secret")}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.req.HasUpdates())
		})
	}
}

func TestUserUpdateRequest_BuildUpdates(t *testing.T) {
	t.Run("omitted fields remain absent", func(t *testing.T) {
		req := UserUpdateRequest{FirstName: strPtr("Jane")}
		updates, err := req.BuildUpdates(func(s string) (string, error) { return "hashed_" + s, nil })
		require.NoError(t, err)
		_, hasLast := updates["last_name"]
		_, hasEmail := updates["email"]
		_, hasVerified := updates["verified"]
		_, hasPass := updates["password_hash"]
		assert.Equal(t, map[string]any{"first_name": "Jane"}, updates)
		assert.False(t, hasLast)
		assert.False(t, hasEmail)
		assert.False(t, hasVerified)
		assert.False(t, hasPass)
	})

	t.Run("explicit false retained", func(t *testing.T) {
		req := UserUpdateRequest{Verified: boolPtr(false)}
		updates, err := req.BuildUpdates(func(s string) (string, error) { panic("no password expected") })
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"verified": false}, updates)
	})

	t.Run("optional names explicitly cleared", func(t *testing.T) {
		req := UserUpdateRequest{FirstName: strPtr(""), LastName: strPtr("")}
		updates, err := req.BuildUpdates(func(s string) (string, error) { return "", nil })
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"first_name": "", "last_name": ""}, updates)
	})

	t.Run("password yields only a hash in the allowlist", func(t *testing.T) {
		req := UserUpdateRequest{Password: strPtr("s3cret")}
		updates, err := req.BuildUpdates(func(s string) (string, error) { return "bcrypt_" + s, nil })
		require.NoError(t, err)
		// The raw password must never appear in the produced update map.
		assert.Equal(t, map[string]any{"password_hash": "bcrypt_s3cret"}, updates)
		raw, ok := updates["password"]
		assert.False(t, ok, "raw password must not be placed in updates, got %v", raw)
	})

	t.Run("empty request returns error", func(t *testing.T) {
		req := UserUpdateRequest{}
		_, err := req.BuildUpdates(func(s string) (string, error) { return "", nil })
		assert.Error(t, err)
		assert.ErrorIs(t, err, errNoUserUpdates)
	})
}

func TestUserResponse_DoesNotExposeSensitiveFields(t *testing.T) {
	now := timeNow()
	user := &models.User{
		Model:              gorm.Model{ID: 7},
		FirstName:          "Jane",
		LastName:           "Doe",
		Email:              "jane@example.com",
		PasswordHash:       "bcrypt-secret",
		Role:               "admin",
		LastLogin:          &now,
		LastLoginIP:        "203.0.113.9",
		OTPEnabled:         true,
		OTPVerified:        true,
		OTPSecret:          "top-secret",
		OTPAuthUrl:         "otpauth://totp/...",
		Verified:           true,
		EmailVerifications: nil,
		PasswordResets:     nil,
	}

	resp := userToResponse(user)
	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	body := strings.ToLower(string(raw))

	assert.Contains(t, body, `"email":"jane@example.com"`)
	for _, sensitive := range []string{
		"password_hash", "passwordhash", "otpsecret", "otp_auth", "otpauturl",
		"last_login_ip", "lastloginip", "email_verifications", "password_resets",
		"public_keys",
	} {
		assert.NotContains(t, body, sensitive, "response must not contain %q", sensitive)
	}
	assert.Equal(t, uint(7), resp.ID)
	assert.Equal(t, "Jane", resp.FirstName)
	assert.Equal(t, true, resp.Verified)
}

// ---------------------------------------------------------------------------
// Step 9: Authorization (all five routes reject unauthenticated requests)
// ---------------------------------------------------------------------------

func TestUserRoutes_Unauthenticated(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/users"},
		{http.MethodPost, "/api/users"},
		{http.MethodGet, "/api/users/1"},
		{http.MethodPatch, "/api/users/1"},
		{http.MethodDelete, "/api/users/1"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
				rec := userRequest(tb, ctx, r.method, r.path, nil, "")
				assert.Equal(tb, http.StatusUnauthorized, rec.Code)
			}, getUserAPITestOptions())
		})
	}
}

// ---------------------------------------------------------------------------
// Step 4: List
// ---------------------------------------------------------------------------

func seedUsers(tb testing.TB, ctx coreTesting.TestContext, users []*models.User) {
	tb.Helper()
	require.NoError(tb, ctx.DB().AutoMigrate(&models.User{}))
	for _, u := range users {
		require.NoError(tb, ctx.DB().Create(u).Error)
	}
}

func TestUserList_HappyPath(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		seedUsers(tb, ctx, []*models.User{
			{Model: gorm.Model{ID: 1}, Email: "a@example.com", Role: "user"},
			{Model: gorm.Model{ID: 2}, Email: "b@example.com", Role: "user"},
			{Model: gorm.Model{ID: 3}, Email: "c@example.com", Role: "user"},
		})

		rec := userRequest(tb, ctx, http.MethodGet, "/api/users", nil, token)
		require.Equal(tb, http.StatusOK, rec.Code)

		var envelope UserListResponse
		require.NoError(tb, json.Unmarshal(rec.Body.Bytes(), &envelope))
		assert.Equal(t, int64(3), envelope.Total)
		assert.Len(t, envelope.Data, 3)
		assert.NotEmpty(t, rec.Header().Get("Content-Range"))
		assert.NotEmpty(t, rec.Header().Get("X-Total-Count"))

		// Sensitive fields are never serialized in the list payload.
		raw := strings.ToLower(rec.Body.String())
		assert.NotContains(t, raw, "password_hash")
		assert.NotContains(t, raw, "otp")
	}, getUserAPITestOptions())
}

func TestUserList_FilterAndPagination(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		seedUsers(tb, ctx, []*models.User{
			{Model: gorm.Model{ID: 1}, Email: "a@example.com", Role: "user"},
			{Model: gorm.Model{ID: 2}, Email: "b@example.com", Role: "user"},
			{Model: gorm.Model{ID: 3}, Email: "c@example.com", Role: "user"},
		})

		// Pagination: only the first two rows.
		rec := userRequest(tb, ctx, http.MethodGet, "/api/users?_start=0&_end=2", nil, token)
		require.Equal(tb, http.StatusOK, rec.Code)
		var page UserListResponse
		require.NoError(tb, json.Unmarshal(rec.Body.Bytes(), &page))
		assert.Equal(t, int64(3), page.Total, "total reflects the full filtered set, not the page")
		assert.Len(t, page.Data, 2)
		assert.Contains(t, rec.Header().Get("Content-Range"), "users 0-1/3")

		// Filter by exact email.
		rec = userRequest(tb, ctx, http.MethodGet, "/api/users?email=b@example.com", nil, token)
		require.Equal(tb, http.StatusOK, rec.Code)
		var filtered UserListResponse
		require.NoError(tb, json.Unmarshal(rec.Body.Bytes(), &filtered))
		assert.Equal(t, int64(1), filtered.Total)
		require.Len(t, filtered.Data, 1)
		assert.Equal(t, "b@example.com", filtered.Data[0].Email)
	}, getUserAPITestOptions())
}

// ---------------------------------------------------------------------------
// Step 5: Get single user
// ---------------------------------------------------------------------------

func TestUserGet_HappyPath(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)
		userSvc.SetupUserExistsExpectation(2, true, &models.User{
			Model: gorm.Model{ID: 2}, Email: "target@example.com", FirstName: "T", Role: "user", Verified: true,
		}, nil)

		rec := userRequest(tb, ctx, http.MethodGet, "/api/users/2", nil, token)
		require.Equal(tb, http.StatusOK, rec.Code)
		var resp UserResponse
		require.NoError(tb, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, uint(2), resp.ID)
		assert.Equal(t, "target@example.com", resp.Email)
	}, getUserAPITestOptions())
}

func TestUserGet_NotFound(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)
		userSvc.UserDoesNotExist(99)

		rec := userRequest(tb, ctx, http.MethodGet, "/api/users/99", nil, token)
		assert.Equal(tb, http.StatusNotFound, rec.Code)
	}, getUserAPITestOptions())
}

func TestUserId_ParseFailures(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		for _, path := range []string{"/api/users/abc", "/api/users/0", "/api/users/-1"} {
			rec := userRequest(tb, ctx, http.MethodGet, path, nil, token)
			assert.Equal(tb, http.StatusBadRequest, rec.Code, "path %s should be rejected", path)
		}
	}, getUserAPITestOptions())
}

// ---------------------------------------------------------------------------
// Step 6: Create
// ---------------------------------------------------------------------------

func TestUserCreate_HappyPath(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)

		newUser := &models.User{
			Model: gorm.Model{ID: 42}, Email: "new@example.com", Role: "user", Verified: true,
		}
		userSvc.EXPECT().CreateAccount(mock.Anything, "new@example.com", "s3cret", false).Return(newUser, nil)

		body := []byte(`{"email":"new@example.com","password":"s3cret"}`)
		rec := userRequest(tb, ctx, http.MethodPost, "/api/users", body, token)
		require.Equal(tb, http.StatusCreated, rec.Code, rec.Body.String())

		var resp UserResponse
		require.NoError(tb, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, uint(42), resp.ID)
		assert.Equal(t, "new@example.com", resp.Email)
		assert.NotContains(t, strings.ToLower(rec.Body.String()), "password")
	}, getUserAPITestOptions())
}

func TestUserCreate_WithNames(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)

		newUser := &models.User{
			Model: gorm.Model{ID: 7}, Email: "named@example.com", Role: "user", Verified: true,
		}
		userSvc.EXPECT().CreateAccount(mock.Anything, "named@example.com", "pw", false).Return(newUser, nil)
		userSvc.EXPECT().UpdateAccountName(mock.Anything, uint(7), "Jane", "Doe").Return(nil)
		fresh := &models.User{
			Model: gorm.Model{ID: 7}, Email: "named@example.com", FirstName: "Jane", LastName: "Doe", Role: "user", Verified: true,
		}
		userSvc.EXPECT().AccountExists(mock.Anything, uint(7)).Return(true, fresh, nil)

		body := []byte(`{"email":"named@example.com","password":"pw","first_name":"Jane","last_name":"Doe"}`)
		rec := userRequest(tb, ctx, http.MethodPost, "/api/users", body, token)
		require.Equal(tb, http.StatusCreated, rec.Code, rec.Body.String())

		var resp UserResponse
		require.NoError(tb, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "Jane", resp.FirstName)
		assert.Equal(t, "Doe", resp.LastName)
	}, getUserAPITestOptions())
}

func TestUserCreate_InvalidPayload(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		body := []byte(`{"email":"not-an-email","password":""}`)
		rec := userRequest(tb, ctx, http.MethodPost, "/api/users", body, token)
		assert.Equal(tb, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}, getUserAPITestOptions())
}

// ---------------------------------------------------------------------------
// Step 7: Partial update
// ---------------------------------------------------------------------------

func TestUserUpdate_ApplyNames(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)

		target := &models.User{Model: gorm.Model{ID: 2}, Email: "target@example.com", Role: "user", Verified: true}
		userSvc.EXPECT().AccountExists(mock.Anything, uint(2)).Return(true, target, nil).Twice()
		userSvc.EXPECT().UpdateAccountInfo(mock.Anything, uint(2), map[string]any{"first_name": "Jane", "last_name": "Doe"}).Return(nil)
		userSvc.EXPECT().EmailExists(mock.Anything, mock.Anything).Return(false, nil, nil).Maybe()

		body := []byte(`{"first_name":"Jane","last_name":"Doe"}`)
		rec := userRequest(tb, ctx, http.MethodPatch, "/api/users/2", body, token)
		require.Equal(tb, http.StatusOK, rec.Code, rec.Body.String())

		var resp UserResponse
		require.NoError(tb, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "target@example.com", resp.Email)
	}, getUserAPITestOptions())
}

func TestUserUpdate_PasswordHashedBeforePush(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)

		target := &models.User{Model: gorm.Model{ID: 2}, Email: "target@example.com", Role: "user", Verified: true}
		userSvc.EXPECT().AccountExists(mock.Anything, uint(2)).Return(true, target, nil).Twice()
		// HashPassword here is the mock's auto-implemented helper, which returns
		// "hashed_" + password. The handler must hand the pre-hashed value (not
		// the raw password) to UpdateAccountInfo.
		userSvc.EXPECT().UpdateAccountInfo(mock.Anything, uint(2), map[string]any{"password_hash": "hashed_s3cret"}).Return(nil)

		body := []byte(`{"password":"s3cret"}`)
		rec := userRequest(tb, ctx, http.MethodPatch, "/api/users/2", body, token)
		assert.Equal(tb, http.StatusOK, rec.Code, rec.Body.String())
		assert.NotContains(t, strings.ToLower(rec.Body.String()), "s3cret")

		// The outer MockUserService.HashPassword auto-helper shadows the
		// embedded generated mock and registers a matching expectation on it.
		// The handler's call therefore short-circuits in the helper (returning
		// the pre-hash) and leaves that embedded expectation pending; consume
		// it here so mock assertions pass. The handler-visible side effect is
		// the "hashed_s3cret" value already asserted via UpdateAccountInfo above.
		userSvc.MockUserService.HashPassword("s3cret")
	}, getUserAPITestOptions())
}

func TestUserUpdate_ExplicitFalseVerified(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)

		target := &models.User{Model: gorm.Model{ID: 2}, Email: "target@example.com", Role: "user", Verified: false}
		userSvc.EXPECT().AccountExists(mock.Anything, uint(2)).Return(true, target, nil).Twice()
		userSvc.EXPECT().UpdateAccountInfo(mock.Anything, uint(2), map[string]any{"verified": false}).Return(nil)

		body := []byte(`{"verified":false}`)
		rec := userRequest(tb, ctx, http.MethodPatch, "/api/users/2", body, token)
		assert.Equal(tb, http.StatusOK, rec.Code, rec.Body.String())
	}, getUserAPITestOptions())
}

func TestUserUpdate_EmptyPatchRejected(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		body := []byte(`{}`)
		rec := userRequest(tb, ctx, http.MethodPatch, "/api/users/2", body, token)
		assert.Equal(tb, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}, getUserAPITestOptions())
}

func TestUserUpdate_NotFound(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)
		userSvc.UserDoesNotExist(99)

		body := []byte(`{"first_name":"Jane"}`)
		rec := userRequest(tb, ctx, http.MethodPatch, "/api/users/99", body, token)
		assert.Equal(tb, http.StatusNotFound, rec.Code, rec.Body.String())
	}, getUserAPITestOptions())
}

func TestUserUpdate_DuplicateEmail(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)

		target := &models.User{Model: gorm.Model{ID: 2}, Email: "target@example.com", Role: "user"}
		userSvc.EXPECT().AccountExists(mock.Anything, uint(2)).Return(true, target, nil).Once()

		body := []byte(`{"email":"taken@example.com"}`)
		rec := userRequest(tb, ctx, http.MethodPatch, "/api/users/2", body, token)
		assert.Equal(tb, http.StatusConflict, rec.Code, rec.Body.String())

		// The handler's call to userSvc.EmailExists dispatches to the outer
		// MockUserService auto-helper (which shadows the embedded generated
		// mock), short-circuiting with a different account (ID 1) and
		// registering a matching expectation on the embedded mock. Consume that
		// pending expectation so mock assertions pass; the 409 conflict path has
		// already been exercised above.
		userSvc.MockUserService.EmailExists(context.Background(), "taken@example.com")
	}, getUserAPITestOptions())
}

// ---------------------------------------------------------------------------
// Step 8: Delete
// ---------------------------------------------------------------------------

func TestUserDelete_HappyPath(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 1)
		userSvc := coreTesting.GetMockUserService(ctx)
		userSvc.DeleteUser(2)

		rec := userRequest(tb, ctx, http.MethodDelete, "/api/users/2", nil, token)
		assert.Equal(tb, http.StatusNoContent, rec.Code, rec.Body.String())
	}, getUserAPITestOptions())
}

func TestUserDelete_SelfDeletionRejected(t *testing.T) {
	coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
		token := setupUserAdminAuth(tb, ctx, 7)
		rec := userRequest(tb, ctx, http.MethodDelete, "/api/users/7", nil, token)
		assert.Equal(tb, http.StatusConflict, rec.Code, rec.Body.String())
	}, getUserAPITestOptions())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func timeNow() time.Time { return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) }

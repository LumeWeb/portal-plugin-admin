package api

import (
	"errors"
	"time"

	z "github.com/Oudwins/zog"
	"go.lumeweb.com/httputil"
	"go.lumeweb.com/portal/db/models"
)

// errNoUserUpdates is returned by BuildUpdates when no field was provided. It
// is defensive: handlers reject empty updates via HasUpdates before the
// conversion runs.
var errNoUserUpdates = errors.New("at least one field must be provided")

var (
	_ httputil.DTOValidator                   = (*UserCreateRequest)(nil)
	_ httputil.DTORequest[*UserCreateRequest] = (*UserCreateRequest)(nil)

	_ httputil.DTOValidator                   = (*UserUpdateRequest)(nil)
	_ httputil.DTORequest[*UserUpdateRequest] = (*UserUpdateRequest)(nil)

	_ httputil.DTOResponse[*models.User] = (*UserResponse)(nil)
)

// UserCreateRequest is the admin payload for creating a portal account.
// PasswordHash is never accepted nor returned: the handler hashes the
// password via the UserService before it is persisted.
type UserCreateRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	// VerifyEmail controls whether a verification email is sent for the new
	// account. Defaults to false.
	VerifyEmail bool `json:"verify_email"`
}

func (r *UserCreateRequest) Schema() *z.StructSchema {
	return z.Struct(z.Shape{
		"Email":       z.String().Required().Email(),
		"Password":    z.String().Required().Min(1),
		"FirstName":   z.String(),
		"LastName":    z.String(),
		"VerifyEmail": z.Bool(),
	})
}

func (r *UserCreateRequest) ToModel() (*UserCreateRequest, error) {
	return r, nil
}

// UserUpdateRequest is the admin payload for partially updating a portal
// account. Every field is a pointer so the API can distinguish "omitted"
// (nil => leave the stored value unchanged) from "explicitly set" (non-nil =>
// apply, including explicit false booleans and empty optional names). It
// follows the patch semantics used by the Dashboard SocialProviderUpdateRequest
// and the empty-update rejection used by the IPFS WebsiteUpdateRequest.
//
// Password only ever contains a raw value momentarily; it is hashed before it
// is placed in any update payload. It is never echoed back in responses.
type UserUpdateRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
	Verified  *bool   `json:"verified,omitempty"`
	Password  *string `json:"password,omitempty"`
}

func (r *UserUpdateRequest) Schema() *z.StructSchema {
	return z.Struct(z.Shape{
		"FirstName": z.Ptr(z.String()),
		"LastName":  z.Ptr(z.String()),
		"Email":     z.Ptr(z.String().Email()),
		"Verified":  z.Ptr(z.Bool()),
		"Password":  z.Ptr(z.String().Min(1)),
	})
}

func (r *UserUpdateRequest) ToModel() (*UserUpdateRequest, error) {
	return r, nil
}

// HasUpdates reports whether at least one field is explicitly set. As in the
// IPFS WebsiteUpdateRequest, an empty PATCH is rejected by the handler.
func (r *UserUpdateRequest) HasUpdates() bool {
	return r.FirstName != nil || r.LastName != nil || r.Email != nil || r.Verified != nil || r.Password != nil
}

// BuildUpdates converts the request into an allowlisted set of GORM column
// updates. Only the columns an administrator is permitted to mutate are ever
// produced; every other User model column (password hash aside, OTP, reset
// and verification records, last-login data, role) is intentionally absent.
//
// The supplied hasher must return a bcrypt hash of the raw password. The hash
// (never the raw password) is what ends up in the returned map.
func (r *UserUpdateRequest) BuildUpdates(hasher func(string) (string, error)) (map[string]any, error) {
	updates := make(map[string]any)

	if r.FirstName != nil {
		updates["first_name"] = *r.FirstName
	}
	if r.LastName != nil {
		updates["last_name"] = *r.LastName
	}
	if r.Email != nil {
		updates["email"] = *r.Email
	}
	if r.Verified != nil {
		updates["verified"] = *r.Verified
	}
	if r.Password != nil {
		hash, err := hasher(*r.Password)
		if err != nil {
			return nil, err
		}
		updates["password_hash"] = hash
	}

	// Guarantee a non-nil map so callers can rely on it being valid.
	if len(updates) == 0 {
		return nil, errNoUserUpdates
	}

	return updates, nil
}

// UserResponse is the safe serialization of a User model. Database-only and
// authentication-sensitive fields (PasswordHash, OTP secret/URL, reset and
// verification records) are deliberately not present.
type UserResponse struct {
	ID        uint       `json:"id"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	Verified  bool       `json:"verified"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (r *UserResponse) FromModel(model *models.User) error {
	r.ID = model.ID
	r.FirstName = model.FirstName
	r.LastName = model.LastName
	r.Email = model.Email
	r.Role = model.Role
	r.Verified = model.Verified
	r.LastLogin = model.LastLogin
	r.CreatedAt = model.CreatedAt
	r.UpdatedAt = model.UpdatedAt
	return nil
}

// userToResponse is the single reusable converter used by every handler. It
// explicitly copies only approved fields, so a future drift in the User model
// can never surface sensitive columns in an API response.
func userToResponse(user *models.User) UserResponse {
	var resp UserResponse
	_ = resp.FromModel(user)
	return resp
}

// UserListResponse is the standard paginated list envelope (Refine-compatible
// Simple REST data provider shape): data plus total.
type UserListResponse struct {
	Data  []UserResponse `json:"data"`
	Total int64          `json:"total"`
}

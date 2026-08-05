package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/OderoCeasar/afyalink/backend/internal/httpx"
	"github.com/OderoCeasar/afyalink/backend/pkg/validator"
)

// refreshCookieName is the HttpOnly cookie carrying the refresh token.
//
// Why a cookie rather than a JSON field the client stores: the frontend keeps
// the access token in memory only (per spec §6), so a page reload would
// otherwise log the user out. An HttpOnly cookie survives reload while staying
// unreadable to any injected script.
//
// CSRF: the cookie is SameSite=Strict and scoped to /api/auth, so a browser
// will not attach it to a request initiated by another site. Every other
// state-changing endpoint authenticates with a Bearer header, which browsers
// never attach automatically. Those two facts together are what make a
// separate CSRF token unnecessary here — if the cookie were ever relaxed to
// SameSite=None, one would have to be added.
const refreshCookieName = "afyalink_refresh"

// refreshCookiePath keeps the cookie off every route that does not need it, so
// an XSS on a patient page cannot cause it to be sent anywhere useful.
const refreshCookiePath = "/api/auth"

// Handler exposes the auth endpoints.
type Handler struct {
	service *Service
	secure  bool
}

// NewHandler builds the auth handler. secure should be true in production so
// the refresh cookie is only ever sent over TLS.
func NewHandler(service *Service, secureCookies bool) *Handler {
	return &Handler{service: service, secure: secureCookies}
}

// UserResponse is the client-facing shape of a user. It has no password hash
// field at all, so no future edit can accidentally serialise one.
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Phone string `json:"phone,omitempty"`
	Role  string `json:"role"`
}

func toUserResponse(u User) UserResponse {
	resp := UserResponse{ID: u.ID.String(), Email: u.Email, Role: string(u.Role)}
	if u.Phone != nil {
		resp.Phone = *u.Phone
	}
	return resp
}

// tokenResponse is returned by login and refresh. The refresh token itself is
// deliberately absent from the body — it travels only in the HttpOnly cookie.
type tokenResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresIn   int          `json:"expires_in"`
	User        UserResponse `json:"user"`
}

type registerRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Phone    string `json:"phone" validate:"omitempty,max=32"`
	Password string `json:"password" validate:"required,min=12,max=128"`
	Role     string `json:"role" validate:"required,oneof=patient caregiver"`
}

// Register creates a patient or caregiver account.
//
// The route is mounted behind RequireRole(clinician). Creating another
// clinician is deliberately not possible here: clinician accounts are
// provisioned out of band (the seed command), so a single compromised
// clinician session cannot mint itself more privileged peers.
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := validator.BindJSON(c, &req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	var phone *string
	if req.Phone != "" {
		phone = &req.Phone
	}

	user, err := h.service.Register(c.Request.Context(), req.Email, phone, req.Password, Role(req.Role))
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			httpx.Conflict(c, "a user with that email already exists")
			return
		}
		httpx.Internal(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": toUserResponse(user)})
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,max=128"`
}

// Login exchanges credentials for an access token plus a refresh cookie.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := validator.BindJSON(c, &req); err != nil {
		httpx.BadRequest(c, err)
		return
	}

	pair, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Unauthorized(c, "invalid email or password")
			return
		}
		httpx.Internal(c, err)
		return
	}

	h.setRefreshCookie(c, pair)
	c.JSON(http.StatusOK, tokenResponse{
		AccessToken: pair.AccessToken,
		ExpiresIn:   int(pair.AccessExpiresIn.Seconds()),
		User:        toUserResponse(pair.User),
	})
}

// Refresh rotates the refresh cookie and issues a new access token.
func (h *Handler) Refresh(c *gin.Context) {
	token, err := c.Cookie(refreshCookieName)
	if err != nil || token == "" {
		httpx.Unauthorized(c, "refresh token missing")
		return
	}

	pair, err := h.service.Refresh(c.Request.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenReplay):
			// Every session for this user has just been revoked. Clear the
			// cookie so the client stops replaying a token that is now dead.
			h.clearRefreshCookie(c)
			httpx.Unauthorized(c, "session ended for security reasons, please sign in again")
		case errors.Is(err, ErrInvalidRefreshToken):
			h.clearRefreshCookie(c)
			httpx.Unauthorized(c, "invalid or expired session")
		default:
			httpx.Internal(c, err)
		}
		return
	}

	h.setRefreshCookie(c, pair)
	c.JSON(http.StatusOK, tokenResponse{
		AccessToken: pair.AccessToken,
		ExpiresIn:   int(pair.AccessExpiresIn.Seconds()),
		User:        toUserResponse(pair.User),
	})
}

// Logout revokes the refresh token and clears the cookie. It always reports
// success so it cannot be used to probe token validity.
func (h *Handler) Logout(c *gin.Context) {
	token, err := c.Cookie(refreshCookieName)
	if err == nil && token != "" {
		if err := h.service.Logout(c.Request.Context(), token); err != nil {
			httpx.Internal(c, err)
			return
		}
	}

	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"status": "signed out"})
}

// Me returns the authenticated caller, letting the frontend restore its
// session state after a reload without decoding the JWT client-side.
func (h *Handler) Me(c *gin.Context) {
	principal := MustPrincipal(c)

	user, err := h.service.store.GetUserByID(c.Request.Context(), principal.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": toUserResponse(user)})
}

func (h *Handler) setRefreshCookie(c *gin.Context, pair TokenPair) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    pair.RefreshToken,
		Path:     refreshCookiePath,
		Expires:  pair.RefreshExpires,
		MaxAge:   int(time.Until(pair.RefreshExpires).Seconds()),
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

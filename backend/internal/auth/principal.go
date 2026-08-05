package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Role enumerates the three roles this MVP supports. There is deliberately no
// admin, nurse, or CHW role — adding one is a schema and policy change, not a
// string literal.
type Role string

const (
	RolePatient   Role = "patient"
	RoleCaregiver Role = "caregiver"
	RoleClinician Role = "clinician"
)

// Valid reports whether r is one of the three known roles. Anything else,
// including a role smuggled in via a forged token payload, is rejected.
func (r Role) Valid() bool {
	switch r {
	case RolePatient, RoleCaregiver, RoleClinician:
		return true
	default:
		return false
	}
}

func (r Role) String() string { return string(r) }

// Principal is the authenticated caller, derived solely from a verified access
// token. Handlers must take identity from here and never from a request body
// or query parameter.
type Principal struct {
	UserID uuid.UUID
	Role   Role
}

// principalContextKey is unexported so no other package can plant a Principal
// into the Gin context and impersonate a caller.
const principalContextKey = "afyalink.principal"

// SetPrincipal stores the authenticated caller on the request context. Only
// the auth middleware should call this.
func SetPrincipal(c *gin.Context, p Principal) {
	c.Set(principalContextKey, p)
}

// PrincipalFrom returns the authenticated caller, or ok=false when the route
// was reached without authentication.
func PrincipalFrom(c *gin.Context) (Principal, bool) {
	value, exists := c.Get(principalContextKey)
	if !exists {
		return Principal{}, false
	}
	p, ok := value.(Principal)
	return p, ok
}

// MustPrincipal returns the authenticated caller and panics if absent.
//
// The panic is intentional and is recovered by the recovery middleware into a
// 500. It can only fire if a route was registered behind no auth middleware,
// which is a wiring bug that must fail loudly in tests rather than quietly
// serving patient data to an anonymous caller.
func MustPrincipal(c *gin.Context) Principal {
	p, ok := PrincipalFrom(c)
	if !ok {
		panic("auth: route reached without authentication middleware")
	}
	return p
}

package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestHashPasswordProducesVerifiablePHCString(t *testing.T) {
	const password = "AfyaLinkDemo2026!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	// The PHC prefix carries the algorithm and cost, so an old hash stays
	// verifiable after the parameters are raised.
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("hash does not carry the expected OWASP parameters: %q", hash)
	}

	if err := VerifyPassword(password, hash); err != nil {
		t.Fatalf("VerifyPassword on the correct password: %v", err)
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if err := VerifyPassword("wrong password", hash); err != ErrInvalidPassword {
		t.Fatalf("got %v, want ErrInvalidPassword", err)
	}
}

// Two hashes of the same password must differ, proving the salt is random.
func TestHashPasswordIsSalted(t *testing.T) {
	const password = "AfyaLinkDemo2026!"

	first, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	second, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if first == second {
		t.Fatal("hashing the same password twice produced identical output; the salt is not random")
	}
	if err := VerifyPassword(password, second); err != nil {
		t.Fatalf("second hash failed to verify: %v", err)
	}
}

// A corrupt or truncated stored hash must be a verification failure, never a
// panic and never an accidental success.
func TestVerifyPasswordRejectsMalformedHashes(t *testing.T) {
	malformed := []string{
		"",
		"not-a-hash",
		"$argon2id$v=19$m=19456,t=2,p=1$onlysalt",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",  // wrong variant
		"$argon2id$v=16$m=19456,t=2,p=1$c2FsdA$aGFzaA", // wrong version
		"$argon2id$v=19$m=0,t=0,p=0$c2FsdA$aGFzaA",     // zero cost
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$aGFzaA",    // bad base64 salt
	}

	for _, hash := range malformed {
		if err := VerifyPassword("anything", hash); err != ErrInvalidPassword {
			t.Errorf("hash %q: got %v, want ErrInvalidPassword", hash, err)
		}
	}
}

func TestJWTIssueAndVerify(t *testing.T) {
	manager := NewJWTManager([]byte(strings.Repeat("k", 32)), 15*time.Minute)
	userID := uuid.New()

	token, err := manager.Issue(userID, RoleClinician)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	principal, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if principal.UserID != userID {
		t.Fatalf("user id: got %s, want %s", principal.UserID, userID)
	}
	if principal.Role != RoleClinician {
		t.Fatalf("role: got %s, want clinician", principal.Role)
	}
}

func TestJWTRejectsTokenSignedWithAnotherKey(t *testing.T) {
	issuer := NewJWTManager([]byte(strings.Repeat("a", 32)), 15*time.Minute)
	verifier := NewJWTManager([]byte(strings.Repeat("b", 32)), 15*time.Minute)

	token, err := issuer.Issue(uuid.New(), RolePatient)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := verifier.Verify(token); err != ErrInvalidToken {
		t.Fatalf("got %v, want ErrInvalidToken", err)
	}
}

func TestJWTRejectsExpiredToken(t *testing.T) {
	manager := NewJWTManager([]byte(strings.Repeat("k", 32)), -time.Minute)

	token, err := manager.Issue(uuid.New(), RolePatient)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := manager.Verify(token); err != ErrInvalidToken {
		t.Fatalf("expired token: got %v, want ErrInvalidToken", err)
	}
}

// The classic JWT algorithm-confusion attack: a token with alg "none" must not
// be accepted just because it parses.
func TestJWTRejectsNoneAlgorithm(t *testing.T) {
	manager := NewJWTManager([]byte(strings.Repeat("k", 32)), 15*time.Minute)

	claims := Claims{
		Role: RoleClinician,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.NewString(),
			Issuer:    tokenIssuer,
			Audience:  jwt.ClaimStrings{tokenAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("building alg=none token: %v", err)
	}

	if _, err := manager.Verify(unsigned); err != ErrInvalidToken {
		t.Fatalf("alg=none token: got %v, want ErrInvalidToken", err)
	}
}

// A token carrying a role the system does not define must be refused rather
// than defaulted into something permissive.
func TestJWTRejectsUnknownRole(t *testing.T) {
	secret := []byte(strings.Repeat("k", 32))
	manager := NewJWTManager(secret, 15*time.Minute)

	claims := Claims{
		Role: Role("admin"),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.NewString(),
			Issuer:    tokenIssuer,
			Audience:  jwt.ClaimStrings{tokenAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("signing forged token: %v", err)
	}

	if _, err := manager.Verify(forged); err != ErrInvalidToken {
		t.Fatalf("unknown role: got %v, want ErrInvalidToken", err)
	}
}

func TestIssueRefusesUnknownRole(t *testing.T) {
	manager := NewJWTManager([]byte(strings.Repeat("k", 32)), 15*time.Minute)

	if _, err := manager.Issue(uuid.New(), Role("superuser")); err == nil {
		t.Fatal("Issue accepted an unknown role")
	}
}

// Refresh tokens are stored only as hashes, and the hash must be stable and
// not equal to the token itself.
func TestRefreshTokenHashing(t *testing.T) {
	token, hash, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("generateRefreshToken: %v", err)
	}

	if token == "" || hash == "" {
		t.Fatal("generateRefreshToken returned an empty value")
	}
	if token == hash {
		t.Fatal("the stored hash equals the token; the database would hold usable credentials")
	}
	if hashRefreshToken(token) != hash {
		t.Fatal("hashRefreshToken is not stable for the same token")
	}

	otherToken, _, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("generateRefreshToken: %v", err)
	}
	if otherToken == token {
		t.Fatal("two generated refresh tokens collided")
	}
}

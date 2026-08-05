// Package api wires the HTTP route table.
//
// The route table is the single place where "who may call what" is declared,
// so it doubles as the authorisation spec. Every patient-scoped route carries
// both a RequireRole guard (coarse) and, inside its handler, a row-level
// ownership check (fine). Neither is sufficient alone: the role guard cannot
// tell one caregiver from another, and the ownership check cannot stop a
// caregiver from reaching a clinician-only action.
package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/OderoCeasar/afyalink/backend/internal/alerts"
	"github.com/OderoCeasar/afyalink/backend/internal/audit"
	"github.com/OderoCeasar/afyalink/backend/internal/auth"
	"github.com/OderoCeasar/afyalink/backend/internal/checkins"
	"github.com/OderoCeasar/afyalink/backend/internal/config"
	"github.com/OderoCeasar/afyalink/backend/internal/crypto"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
	"github.com/OderoCeasar/afyalink/backend/internal/httpx"
	"github.com/OderoCeasar/afyalink/backend/internal/medications"
	"github.com/OderoCeasar/afyalink/backend/internal/middleware"
	"github.com/OderoCeasar/afyalink/backend/internal/patients"
)

// Deps are everything the router needs to build its handlers.
type Deps struct {
	Config *config.Config
	Pool   *db.Pool
	Logger *slog.Logger
	Cipher *crypto.Cipher
}

// New builds the configured Gin engine.
func New(deps Deps) (*gin.Engine, error) {
	if deps.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// Trust only the configured proxies. Gin's default trusts everything,
	// which would let any client forge X-Forwarded-For and defeat the per-IP
	// rate limit and the audit log's ip_address column.
	if err := engine.SetTrustedProxies(deps.Config.TrustedProxies); err != nil {
		return nil, err
	}

	// Stores and services.
	authStore := auth.NewStore(deps.Pool)
	auditStore := audit.NewStore(deps.Pool)
	patientStore := patients.NewStore(deps.Pool)
	patientAccess := patients.NewAccess(deps.Pool)
	medicationStore := medications.NewStore(deps.Pool)
	checkinStore := checkins.NewStore(deps.Pool)
	alertStore := alerts.NewStore(deps.Pool)

	jwtManager := auth.NewJWTManager(deps.Config.JWTSecret, deps.Config.AccessTokenTTL)
	authService := auth.NewService(authStore, jwtManager, deps.Config.RefreshTokenTTL)
	patientService := patients.NewService(patientStore, patientAccess, deps.Cipher)
	checkinService := checkins.NewService(checkinStore, patientAccess)

	// Handlers.
	authHandler := auth.NewHandler(authService, deps.Config.IsProduction())
	patientHandler := patients.NewHandler(patientService)
	medicationHandler := medications.NewHandler(medicationStore, patientAccess)
	checkinHandler := checkins.NewHandler(checkinService)
	alertHandler := alerts.NewHandler(alertStore, patientAccess)

	// Global middleware, outermost first.
	engine.Use(
		middleware.RequestID(),
		middleware.Recovery(deps.Logger),
		middleware.SecurityHeaders(deps.Config.IsProduction()),
		middleware.CORS(deps.Config.CORSOrigins),
		middleware.RequestLogger(deps.Logger),
		middleware.Audit(auditStore, deps.Logger),
	)

	engine.NoRoute(func(c *gin.Context) { httpx.NotFound(c) })

	// Liveness probe. No auth, and it reveals nothing beyond "the process is
	// up" — deliberately not a database health check, which would let an
	// unauthenticated caller probe infrastructure state.
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	requireAuth := middleware.RequireAuth(jwtManager)
	authRateLimit := middleware.NewRateLimiter(deps.Config.LoginRateLimit, deps.Config.LoginRateBurst).Middleware()

	api := engine.Group("/api")

	// ---- Authentication ----
	authGroup := api.Group("/auth")
	{
		// Rate limited: these two are the credential-guessing surface.
		authGroup.POST("/login", authRateLimit, authHandler.Login)
		authGroup.POST("/refresh", authRateLimit, authHandler.Refresh)

		authGroup.POST("/logout", authHandler.Logout)

		// Registration is clinician-only — there is no public self-signup in
		// this MVP. Clinicians bootstrap patient and caregiver accounts.
		authGroup.POST("/register", requireAuth, middleware.RequireRole(auth.RoleClinician), authHandler.Register)

		authGroup.GET("/me", requireAuth, authHandler.Me)
	}

	// ---- Patients ----
	patientGroup := api.Group("/patients", requireAuth)
	{
		allRoles := middleware.RequireRole(auth.RoleClinician, auth.RoleCaregiver, auth.RolePatient)
		clinicianOnly := middleware.RequireRole(auth.RoleClinician)

		patientGroup.GET("", allRoles, patientHandler.List)
		patientGroup.POST("", clinicianOnly, patientHandler.Create)
		patientGroup.GET("/:id", allRoles, patientHandler.Get)

		patientGroup.POST("/:id/discharge", clinicianOnly, patientHandler.CreateDischarge)
		patientGroup.GET("/:id/discharge", allRoles, patientHandler.ListDischarges)

		patientGroup.POST("/:id/medications", clinicianOnly, medicationHandler.Create)
		patientGroup.GET("/:id/medications", allRoles, medicationHandler.List)

		patientGroup.GET("/:id/checkins", allRoles, checkinHandler.List)

		// Clinical notes are clinician-only on both verbs. Spec §3 denies
		// caregivers notes outright, and patients are not granted them either.
		patientGroup.POST("/:id/notes", clinicianOnly, patientHandler.CreateNote)
		patientGroup.GET("/:id/notes", clinicianOnly, patientHandler.ListNotes)
	}

	// ---- Medication dose logs ----
	medicationGroup := api.Group("/medications", requireAuth)
	{
		// Only the patient records their own doses. A clinician marking a dose
		// taken on a patient's behalf would make adherence data meaningless.
		medicationGroup.POST("/:id/logs/:logId/taken",
			middleware.RequireRole(auth.RolePatient), medicationHandler.MarkTaken)
	}

	// ---- Check-in responses ----
	checkinGroup := api.Group("/checkins", requireAuth)
	{
		checkinGroup.POST("/:id/respond",
			middleware.RequireRole(auth.RolePatient), checkinHandler.Respond)
	}

	// ---- Alerts ----
	alertGroup := api.Group("/alerts", requireAuth)
	{
		alertGroup.GET("",
			middleware.RequireRole(auth.RoleClinician, auth.RoleCaregiver), alertHandler.List)

		alertGroup.PATCH("/:id/resolve",
			middleware.RequireRole(auth.RoleClinician), alertHandler.Resolve)
	}

	return engine, nil
}

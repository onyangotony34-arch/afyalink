// Package audit records an append-only trail of access to patient-scoped data.
//
// Entries carry only identifiers — who, what action, which resource — and
// never field values. An audit trail that quotes a diagnosis is itself a PHI
// store with a second copy of the data to protect.
package audit

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/OderoCeasar/afyalink/backend/internal/db"
)

// Entry is one audited access.
type Entry struct {
	UserID     *uuid.UUID
	Action     string
	Resource   string
	ResourceID *uuid.UUID
	IPAddress  string
}

// Actions performed against patient-scoped data. Read actions are audited
// alongside writes: for health data, who looked at a record matters as much as
// who changed it.
const (
	ActionCreate  = "create"
	ActionRead    = "read"
	ActionList    = "list"
	ActionUpdate  = "update"
	ActionRespond = "respond"
	ActionResolve = "resolve"
)

// Resource names, one per audited table.
const (
	ResourcePatient       = "patient"
	ResourceDischarge     = "discharge_record"
	ResourceMedication    = "medication"
	ResourceMedicationLog = "medication_log"
	ResourceCheckin       = "checkin"
	ResourceAlert         = "alert"
	ResourceClinicalNote  = "clinical_note"
)

// Store persists audit entries.
type Store struct {
	pool *db.Pool
}

// NewStore builds an audit store.
func NewStore(pool *db.Pool) *Store { return &Store{pool: pool} }

// Write inserts one audit entry.
func (s *Store) Write(ctx context.Context, e Entry) error {
	const query = `
		INSERT INTO audit_logs (user_id, action, resource, resource_id, ip_address)
		VALUES ($1, $2, $3, $4, $5)`

	if _, err := s.pool.Exec(ctx, query, e.UserID, e.Action, e.Resource, e.ResourceID, e.IPAddress); err != nil {
		return fmt.Errorf("audit: writing entry: %w", err)
	}
	return nil
}

// pendingKey holds entries staged by handlers until the middleware flushes
// them. It is unexported so nothing outside this package can forge entries.
const pendingKey = "afyalink.audit.pending"

// Record stages an audit entry for the current request.
//
// Handlers call this on the success path only. Staging rather than writing
// immediately means the entry shares the request's IP and user resolution, and
// keeps a failed handler from logging an access that never happened.
func Record(c *gin.Context, action, resource string, resourceID *uuid.UUID) {
	entry := Entry{Action: action, Resource: resource, ResourceID: resourceID}

	existing, _ := c.Get(pendingKey)
	entries, _ := existing.([]Entry)
	c.Set(pendingKey, append(entries, entry))
}

// Pending returns the entries staged during this request.
func Pending(c *gin.Context) []Entry {
	value, ok := c.Get(pendingKey)
	if !ok {
		return nil
	}
	entries, _ := value.([]Entry)
	return entries
}

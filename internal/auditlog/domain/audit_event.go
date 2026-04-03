package domain

import (
	"time"

	"github.com/google/uuid"
)

type ActorType string

const (
	ActorTypeUser    ActorType = "user"
	ActorTypeService ActorType = "service"
	ActorTypeSystem  ActorType = "system"
)

type Action string

const (
	ActionCreated     Action = "CREATED"
	ActionUpdated     Action = "UPDATED"
	ActionDeleted     Action = "DELETED"
	ActionCompensated Action = "COMPENSATED"
)

type Outcome string

const (
	OutcomeSuccess Outcome = "SUCCESS"
	OutcomeFailure Outcome = "FAILURE"
	OutcomePartial Outcome = "PARTIAL"
)

type AuditEvent struct {
	ID            uuid.UUID
	TenantID      string
	ActorID       string
	ActorType     ActorType
	EntityType    string
	EntityID      string
	Action        Action
	Outcome       Outcome
	ServiceName   string
	SourceIP      string
	SessionID     string
	CorrelationID string
	TraceID       string
	Timestamp     time.Time
	CompensatesID *uuid.UUID
	Before        map[string]any
	After         map[string]any
	Diff          map[string]any
	Metadata      map[string]any
	Reason        string
	Tags          []string
}

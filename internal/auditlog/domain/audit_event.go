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

type ID string

func (id ID) String() string {
	return string(id)
}

type AuditEvent struct {
	ID            uuid.UUID
	TenantID      ID
	Namespace     string
	ActorID       ID
	ActorType     ActorType
	EntityType    string
	EntityID      ID
	Action        Action
	Outcome       Outcome
	ServiceName   string
	SourceIP      string
	SessionID     ID
	CorrelationID ID
	TraceID       string
	OccurredAt    *time.Time // when the event happened in the source system
	Timestamp     time.Time  // when the event was received by this service
	CompensatesID *uuid.UUID
	Before        map[string]any
	After         map[string]any
	Diff          map[string]any
	Metadata      map[string]any
	Reason        string
	Tags          []string
}

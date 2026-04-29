package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"

	"audit-log/internal/auditlog/domain"
)

type WriteEventOptions struct {
	TenantID      string
	Namespace     string
	OccurredAt    *time.Time
	ActorID       string
	ActorType     domain.ActorType
	EntityType    string
	EntityID      string
	Action        domain.Action
	Outcome       domain.Outcome
	ServiceName   string
	SourceIP      string
	SessionID     string
	CorrelationID string
	TraceID       string
	Before        map[string]any
	After         map[string]any
	Metadata      map[string]any
	Reason        string
	Tags          []string
}

type WriteCompensationOptions struct {
	WriteEventOptions
	CompensatesID uuid.UUID
}

type QueryEventsResult struct {
	Events        []*domain.AuditEvent
	NextPageToken *uuid.UUID
}

type AuditService interface {
	WriteEvent(ctx context.Context, opts WriteEventOptions) (*domain.AuditEvent, error)
	WriteCompensation(ctx context.Context, opts WriteCompensationOptions) (*domain.AuditEvent, error)
	QueryEvents(ctx context.Context, opts QueryEventsOptions) (*QueryEventsResult, error)
	GetEvent(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error)
	ListNamespaces(ctx context.Context) ([]string, error)
}

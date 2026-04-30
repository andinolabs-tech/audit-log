package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"

	"audit-log/internal/auditlog/domain"
)

//go:generate mockgen -destination=../../../test/unit/doubles/auditlog/eventstore_mock.go -package=auditlogdoubles audit-log/internal/auditlog/usecases EventStore

type QueryEventsOptions struct {
	TenantID      *string
	Namespaces    []string
	ActorID       *string
	ActorType     *domain.ActorType
	EntityType    *string
	EntityID      *string
	Action        *domain.Action
	Outcome       *domain.Outcome
	ServiceName   *string
	SourceIP      *string
	SessionID     *string
	CorrelationID *string
	TraceID       *string
	TimestampFrom *time.Time
	TimestampTo   *time.Time
	PageToken     *uuid.UUID
	PageSize      int
}

type EventStore interface {
	Save(ctx context.Context, event *domain.AuditEvent) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error)
	Query(ctx context.Context, opts QueryEventsOptions) ([]*domain.AuditEvent, error)
	QueryNamespaces(ctx context.Context) ([]string, error)
}

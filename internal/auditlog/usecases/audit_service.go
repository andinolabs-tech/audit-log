package usecases

import (
	"context"

	"github.com/google/uuid"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/infra/jsonpatch"
)

type SimpleAuditService struct {
	store EventStore
}

func NewSimpleAuditService(store EventStore) *SimpleAuditService {
	return &SimpleAuditService{store: store}
}

func (s *SimpleAuditService) WriteEvent(ctx context.Context, opts WriteEventOptions) (*domain.AuditEvent, error) {
	return s.writeAndSave(ctx, opts, nil)
}

func (s *SimpleAuditService) WriteCompensation(ctx context.Context, opts WriteCompensationOptions) (*domain.AuditEvent, error) {
	ref, err := s.store.FindByID(ctx, opts.CompensatesID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, ErrReferencedEventNotFound
	}
	if ref.TenantID != opts.TenantID {
		return nil, ErrTenantMismatch
	}
	return s.writeAndSave(ctx, opts.WriteEventOptions, &opts.CompensatesID)
}

func (s *SimpleAuditService) writeAndSave(ctx context.Context, opts WriteEventOptions, compensates *uuid.UUID) (*domain.AuditEvent, error) {
	b := domain.NewAuditEventBuilder().
		WithTenantID(opts.TenantID).
		WithActorID(opts.ActorID).
		WithActorType(opts.ActorType).
		WithEntityType(opts.EntityType).
		WithEntityID(opts.EntityID).
		WithOutcome(opts.Outcome).
		WithServiceName(opts.ServiceName).
		WithSourceIP(opts.SourceIP).
		WithSessionID(opts.SessionID).
		WithCorrelationID(opts.CorrelationID).
		WithTraceID(opts.TraceID).
		WithBefore(opts.Before).
		WithAfter(opts.After).
		WithMetadata(opts.Metadata).
		WithReason(opts.Reason).
		WithTags(opts.Tags)

	if compensates != nil {
		b = b.WithAction(domain.ActionCompensated).WithCompensatesID(*compensates)
	} else {
		b = b.WithAction(opts.Action)
	}

	event, err := b.Build()
	if err != nil {
		return nil, err
	}

	if opts.Before != nil || opts.After != nil {
		before := opts.Before
		after := opts.After
		if before == nil {
			before = map[string]any{}
		}
		if after == nil {
			after = map[string]any{}
		}
		diff, derr := jsonpatch.DiffMaps(before, after)
		if derr != nil {
			return nil, derr
		}
		event.Diff = diff
	}

	if err := s.store.Save(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *SimpleAuditService) QueryEvents(ctx context.Context, opts QueryEventsOptions) (*QueryEventsResult, error) {
	if opts.PageSize < 1 || opts.PageSize > 500 {
		return nil, ErrInvalidPageSize
	}
	q := opts
	q.PageSize = opts.PageSize + 1
	events, err := s.store.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	var next *uuid.UUID
	if len(events) > opts.PageSize {
		last := events[opts.PageSize-1].ID
		next = &last
		events = events[:opts.PageSize]
	}
	return &QueryEventsResult{Events: events, NextPageToken: next}, nil
}

func (s *SimpleAuditService) GetEvent(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error) {
	return s.store.FindByID(ctx, id)
}

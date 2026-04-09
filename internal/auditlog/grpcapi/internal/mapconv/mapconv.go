package mapconv

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/auditlog/usecases"
	auditlogv1 "audit-log/proto/auditlogv1"
)

func WriteEventRequestToOpts(req *auditlogv1.WriteEventRequest) (usecases.WriteEventOptions, error) {
	before, err := StructToMap(req.GetBefore())
	if err != nil {
		return usecases.WriteEventOptions{}, err
	}
	after, err := StructToMap(req.GetAfter())
	if err != nil {
		return usecases.WriteEventOptions{}, err
	}
	meta, err := StructToMap(req.GetMetadata())
	if err != nil {
		return usecases.WriteEventOptions{}, err
	}
	opts := usecases.WriteEventOptions{
		TenantID:      req.GetTenantId(),
		Namespace:     req.GetNamespace(),
		ActorID:       req.GetActorId(),
		ActorType:     domain.ActorType(req.GetActorType()),
		EntityType:    req.GetEntityType(),
		EntityID:      req.GetEntityId(),
		Action:        domain.Action(req.GetAction()),
		Outcome:       domain.Outcome(req.GetOutcome()),
		ServiceName:   req.GetServiceName(),
		SourceIP:      req.GetSourceIp(),
		SessionID:     req.GetSessionId(),
		CorrelationID: req.GetCorrelationId(),
		TraceID:       req.GetTraceId(),
		Before:        before,
		After:         after,
		Metadata:      meta,
		Reason:        req.GetReason(),
		Tags:          append([]string(nil), req.GetTags()...),
	}
	if t := req.GetOccurredAt(); t != nil {
		if err := t.CheckValid(); err != nil {
			return usecases.WriteEventOptions{}, err
		}
		tt := t.AsTime().UTC()
		opts.OccurredAt = &tt
	}
	return opts, nil
}

func QueryEventsRequestToOpts(req *auditlogv1.QueryEventsRequest) (usecases.QueryEventsOptions, error) {
	opts := usecases.QueryEventsOptions{PageSize: int(req.GetPageSize())}
	if req.TenantId != nil {
		v := req.GetTenantId()
		opts.TenantID = &v
	}
	if req.Namespace != nil {
		v := *req.Namespace
		opts.Namespace = &v
	}
	if req.ActorId != nil {
		v := req.GetActorId()
		opts.ActorID = &v
	}
	if req.ActorType != nil {
		v := domain.ActorType(req.GetActorType())
		opts.ActorType = &v
	}
	if req.EntityType != nil {
		v := req.GetEntityType()
		opts.EntityType = &v
	}
	if req.EntityId != nil {
		v := req.GetEntityId()
		opts.EntityID = &v
	}
	if req.Action != nil {
		v := domain.Action(req.GetAction())
		opts.Action = &v
	}
	if req.Outcome != nil {
		v := domain.Outcome(req.GetOutcome())
		opts.Outcome = &v
	}
	if req.ServiceName != nil {
		v := req.GetServiceName()
		opts.ServiceName = &v
	}
	if req.SourceIp != nil {
		v := req.GetSourceIp()
		opts.SourceIP = &v
	}
	if req.SessionId != nil {
		v := req.GetSessionId()
		opts.SessionID = &v
	}
	if req.CorrelationId != nil {
		v := req.GetCorrelationId()
		opts.CorrelationID = &v
	}
	if req.TraceId != nil {
		v := req.GetTraceId()
		opts.TraceID = &v
	}
	if req.GetPageToken() != "" {
		tok, err := uuid.Parse(req.GetPageToken())
		if err != nil {
			return usecases.QueryEventsOptions{}, err
		}
		opts.PageToken = &tok
	}
	return opts, nil
}

func StructToMap(s *structpb.Struct) (map[string]any, error) {
	if s == nil {
		return nil, nil
	}
	return s.AsMap(), nil
}

func DomainEventToProto(e *domain.AuditEvent) (*auditlogv1.AuditEvent, error) {
	before, err := MapToStruct(e.Before)
	if err != nil {
		return nil, err
	}
	after, err := MapToStruct(e.After)
	if err != nil {
		return nil, err
	}
	meta, err := MapToStruct(e.Metadata)
	if err != nil {
		return nil, err
	}
	var diff *structpb.Value
	if e.Diff != nil {
		diff, err = structpb.NewValue(e.Diff)
		if err != nil {
			return nil, err
		}
	}
	out := &auditlogv1.AuditEvent{
		Id:            e.ID.String(),
		TenantId:      e.TenantID,
		Namespace:     e.Namespace,
		ActorId:       e.ActorID,
		ActorType:     string(e.ActorType),
		EntityType:    e.EntityType,
		EntityId:      e.EntityID,
		Action:        string(e.Action),
		Outcome:       string(e.Outcome),
		ServiceName:   e.ServiceName,
		SourceIp:      e.SourceIP,
		SessionId:     e.SessionID,
		CorrelationId: e.CorrelationID,
		TraceId:       e.TraceID,
		Timestamp:     timestamppb.New(e.Timestamp),
		Before:        before,
		After:         after,
		Diff:          diff,
		Metadata:      meta,
		Reason:        e.Reason,
		Tags:          append([]string(nil), e.Tags...),
	}
	if e.OccurredAt != nil {
		out.OccurredAt = timestamppb.New(*e.OccurredAt)
	}
	if e.CompensatesID != nil {
		s := e.CompensatesID.String()
		out.CompensatesId = &s
	}
	return out, nil
}

func MapToStruct(m map[string]any) (*structpb.Struct, error) {
	if m == nil {
		return nil, nil
	}
	return structpb.NewStruct(m)
}

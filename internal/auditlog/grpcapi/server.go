package grpcapi

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"audit-log/internal/auditlog/grpcapi/internal/mapconv"
	"audit-log/internal/auditlog/usecases"
	auditlogv1 "audit-log/proto/auditlogv1"
)

type Server struct {
	auditlogv1.UnimplementedAuditLogServer
	svc usecases.AuditService
}

func NewServer(svc usecases.AuditService) *Server {
	return &Server{svc: svc}
}

func (s *Server) WriteEvent(ctx context.Context, req *auditlogv1.WriteEventRequest) (*auditlogv1.WriteEventResponse, error) {
	opts, err := mapconv.WriteEventRequestToOpts(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	ev, err := s.svc.WriteEvent(ctx, opts)
	if err != nil {
		return nil, mapUsecaseError(err)
	}
	pb, err := mapconv.DomainEventToProto(ev)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditlogv1.WriteEventResponse{Event: pb}, nil
}

func (s *Server) WriteCompensation(ctx context.Context, req *auditlogv1.WriteCompensationRequest) (*auditlogv1.WriteCompensationResponse, error) {
	if req.GetBase() == nil {
		return nil, status.Error(codes.InvalidArgument, "base is required")
	}
	compID, err := uuid.Parse(req.GetCompensatesId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid compensates_id")
	}
	baseOpts, err := mapconv.WriteEventRequestToOpts(req.GetBase())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	ev, err := s.svc.WriteCompensation(ctx, usecases.WriteCompensationOptions{
		WriteEventOptions: baseOpts,
		CompensatesID:     compID,
	})
	if err != nil {
		return nil, mapUsecaseError(err)
	}
	pb, err := mapconv.DomainEventToProto(ev)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditlogv1.WriteCompensationResponse{Event: pb}, nil
}

func (s *Server) QueryEvents(ctx context.Context, req *auditlogv1.QueryEventsRequest) (*auditlogv1.QueryEventsResponse, error) {
	opts, err := mapconv.QueryEventsRequestToOpts(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	res, err := s.svc.QueryEvents(ctx, opts)
	if err != nil {
		return nil, mapUsecaseError(err)
	}
	events := make([]*auditlogv1.AuditEvent, 0, len(res.Events))
	for _, e := range res.Events {
		pb, err := mapconv.DomainEventToProto(e)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		events = append(events, pb)
	}
	out := &auditlogv1.QueryEventsResponse{Events: events}
	if res.NextPageToken != nil {
		out.NextPageToken = res.NextPageToken.String()
	}
	return out, nil
}

func (s *Server) GetEvent(ctx context.Context, req *auditlogv1.GetEventRequest) (*auditlogv1.GetEventResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	ev, err := s.svc.GetEvent(ctx, id)
	if err != nil {
		return nil, mapUsecaseError(err)
	}
	if ev == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	pb, err := mapconv.DomainEventToProto(ev)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditlogv1.GetEventResponse{Event: pb}, nil
}

func mapUsecaseError(err error) error {
	switch {
	case errors.Is(err, usecases.ErrReferencedEventNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, usecases.ErrTenantMismatch):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, usecases.ErrInvalidPageSize):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

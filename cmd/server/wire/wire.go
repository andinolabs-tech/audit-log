//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	"audit-log/internal/auditlog/grpcapi"
	"audit-log/internal/auditlog/persistence"
	"audit-log/internal/auditlog/usecases"
	"audit-log/internal/infra/grpcserver"
	auditlogv1 "audit-log/proto/auditlogv1"
)

func registerAuditLogGRPC(api *grpcapi.Server) *grpc.Server {
	srv := grpcserver.New()
	auditlogv1.RegisterAuditLogServer(srv, api)
	return srv
}

func InitializeGRPC(db *gorm.DB) (*grpc.Server, error) {
	wire.Build(
		persistence.NewEventRepository,
		wire.Bind(new(usecases.EventStore), new(*persistence.EventRepository)),
		usecases.NewSimpleAuditService,
		wire.Bind(new(usecases.AuditService), new(*usecases.SimpleAuditService)),
		grpcapi.NewServer,
		registerAuditLogGRPC,
	)
	return nil, nil
}

func InitializeService(db *gorm.DB) (usecases.AuditService, error) {
	wire.Build(
		persistence.NewEventRepository,
		wire.Bind(new(usecases.EventStore), new(*persistence.EventRepository)),
		usecases.NewSimpleAuditService,
		wire.Bind(new(usecases.AuditService), new(*usecases.SimpleAuditService)),
	)
	return nil, nil
}

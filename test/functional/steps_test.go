package functional_test

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	auditlogv1 "audit-log/proto/auditlogv1"
)

var (
	testGRPCAddr string
	lastEventID  string
	apiClient    auditlogv1.AuditLogClient
	grpcConn     *grpc.ClientConn
	lastTenant   string
)

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		lastEventID = ""
		lastTenant = ""
		dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		conn, err := grpc.DialContext(dialCtx, testGRPCAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return ctx, err
		}
		grpcConn = conn
		apiClient = auditlogv1.NewAuditLogClient(conn)
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if grpcConn != nil {
			_ = grpcConn.Close()
			grpcConn = nil
		}
		return ctx, nil
	})

	ctx.Step(`^I write a minimal audit event$`, iWriteMinimalEvent)
	ctx.Step(`^I write a minimal audit event for tenant "([^"]*)"$`, iWriteMinimalEventForTenant)
	ctx.Step(`^the write response should include a generated id$`, assertGeneratedID)
	ctx.Step(`^I get that event by id$`, iGetEventByLastID)
	ctx.Step(`^the event tenant should be "([^"]*)"$`, assertEventTenant)
	ctx.Step(`^I query events for tenant "([^"]*)" with page size (\d+)$`, iQueryEvents)
	ctx.Step(`^I should receive at least one event$`, assertAtLeastOneEvent)
}

func iWriteMinimalEvent() error {
	return writeEvent("func-test-tenant")
}

func iWriteMinimalEventForTenant(tenant string) error {
	return writeEvent(tenant)
}

func writeEvent(tenant string) error {
	lastTenant = tenant
	before, _ := structpb.NewStruct(map[string]any{"k": "v"})
	resp, err := apiClient.WriteEvent(context.Background(), &auditlogv1.WriteEventRequest{
		TenantId:    tenant,
		ActorId:     "actor-1",
		ActorType:   "user",
		EntityType:  "Thing",
		EntityId:    "e1",
		Action:      "CREATED",
		Outcome:     "SUCCESS",
		ServiceName: "functional-test",
		Before:      before,
	})
	if err != nil {
		return err
	}
	if resp.GetEvent() == nil || resp.GetEvent().GetId() == "" {
		return fmt.Errorf("missing event in response")
	}
	lastEventID = resp.GetEvent().GetId()
	return nil
}

func assertGeneratedID() error {
	if lastEventID == "" {
		return fmt.Errorf("expected lastEventID set")
	}
	return nil
}

func iGetEventByLastID() error {
	_, err := apiClient.GetEvent(context.Background(), &auditlogv1.GetEventRequest{Id: lastEventID})
	return err
}

func assertEventTenant(want string) error {
	resp, err := apiClient.GetEvent(context.Background(), &auditlogv1.GetEventRequest{Id: lastEventID})
	if err != nil {
		return err
	}
	if resp.GetEvent().GetTenantId() != want {
		return fmt.Errorf("tenant: got %q want %q", resp.GetEvent().GetTenantId(), want)
	}
	return nil
}

var lastQueryCount int

func iQueryEvents(tenant string, pageSize int) error {
	ps := int32(pageSize)
	tid := tenant
	resp, err := apiClient.QueryEvents(context.Background(), &auditlogv1.QueryEventsRequest{
		TenantId: &tid,
		PageSize: ps,
	})
	if err != nil {
		return err
	}
	lastQueryCount = len(resp.GetEvents())
	return nil
}

func assertAtLeastOneEvent() error {
	if lastQueryCount < 1 {
		return fmt.Errorf("expected at least one event, got %d", lastQueryCount)
	}
	return nil
}

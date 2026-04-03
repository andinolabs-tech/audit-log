package functional_test

import (
	"net"
	"testing"

	"github.com/cucumber/godog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"audit-log/cmd/server/wire"
	"audit-log/internal/auditlog/persistence"
)

func TestFunctional(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional suite in -short mode")
	}

	cleanup := startTestServer(t)
	defer cleanup()

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("functional scenarios failed")
	}
}

func startTestServer(t *testing.T) func() {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := persistence.AutoMigrateModel(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	srv, err := wire.InitializeGRPC(db)
	if err != nil {
		t.Fatalf("wire: %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	testGRPCAddr = lis.Addr().String()

	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Logf("grpc serve: %v", err)
		}
	}()

	return func() {
		srv.GracefulStop()
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

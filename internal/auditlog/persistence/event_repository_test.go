package persistence_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/auditlog/persistence"
	"audit-log/internal/auditlog/usecases"
)

var _ = Describe("EventRepository", func() {
	var (
		db   *gorm.DB
		repo *persistence.EventRepository
		ctx  context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(persistence.AutoMigrateModel(db)).To(Succeed())
		repo = persistence.NewEventRepository(db)
	})

	It("round-trips an event through Save and FindByID", func() {
		id := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
		ev := &domain.AuditEvent{
			ID:          id,
			TenantID:    "t1",
			ActorID:     "a1",
			ActorType:   domain.ActorTypeUser,
			EntityType:  "E",
			EntityID:    "e1",
			Action:      domain.ActionCreated,
			Outcome:     domain.OutcomeSuccess,
			ServiceName: "svc",
			Timestamp:   time.Now().UTC(),
			Before:      map[string]any{"x": float64(1)},
			After:       map[string]any{"x": float64(2)},
			Diff:        map[string]any{"operations": []any{}},
			Metadata:    map[string]any{"k": "v"},
			Tags:        []string{"t"},
			Reason:      "r",
		}
		Expect(repo.Save(ctx, ev)).To(Succeed())

		got, err := repo.FindByID(ctx, id)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).NotTo(BeNil())
		Expect(got.TenantID).To(Equal("t1"))
		Expect(got.Before).To(Equal(map[string]any{"x": float64(1)}))
		Expect(got.Tags).To(Equal([]string{"t"}))
	})

	It("returns nil when id is missing", func() {
		id := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
		got, err := repo.FindByID(ctx, id)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(BeNil())
	})

	It("applies cursor pagination in Query", func() {
		ids := []uuid.UUID{
			uuid.MustParse("018f0000-0000-7000-8000-000000000001"),
			uuid.MustParse("018f0000-0000-7000-8000-000000000002"),
			uuid.MustParse("018f0000-0000-7000-8000-000000000003"),
		}
		for _, id := range ids {
			ev := &domain.AuditEvent{
				ID:          id,
				TenantID:    "t1",
				ActorID:     "a",
				ActorType:   domain.ActorTypeUser,
				EntityType:  "E",
				EntityID:    "e",
				Action:      domain.ActionCreated,
				Outcome:     domain.OutcomeSuccess,
				ServiceName: "svc",
				Timestamp:   time.Now().UTC(),
			}
			Expect(repo.Save(ctx, ev)).To(Succeed())
		}
		firstPage, err := repo.Query(ctx, usecases.QueryEventsOptions{PageSize: 2})
		Expect(err).NotTo(HaveOccurred())
		Expect(firstPage).To(HaveLen(2))
		tok := firstPage[1].ID
		secondPage, err := repo.Query(ctx, usecases.QueryEventsOptions{PageToken: &tok, PageSize: 10})
		Expect(err).NotTo(HaveOccurred())
		Expect(secondPage).To(HaveLen(1))
	})
})

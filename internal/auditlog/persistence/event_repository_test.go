package persistence_test

import (
	"context"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
			Namespace:   "ns1",
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
		Expect(got.TenantID).To(Equal(domain.ID("t1")))
		Expect(got.Namespace).To(Equal("ns1"))
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
				Namespace:   "ns1",
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

	It("filters events by multiple namespaces using IN", func() {
		for _, ns := range []string{"ns1", "ns2", "ns3"} {
			Expect(repo.Save(ctx, &domain.AuditEvent{
				ID:          uuid.New(),
				TenantID:    "t1",
				Namespace:   ns,
				ActorID:     "a1",
				ActorType:   domain.ActorTypeUser,
				EntityType:  "E",
				EntityID:    "e1",
				Action:      domain.ActionCreated,
				Outcome:     domain.OutcomeSuccess,
				ServiceName: "svc",
				Timestamp:   time.Now().UTC(),
			})).To(Succeed())
		}

		results, err := repo.Query(ctx, usecases.QueryEventsOptions{
			Namespaces: []string{"ns1", "ns2"},
			PageSize:   10,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(2))
		namespaces := []string{results[0].Namespace, results[1].Namespace}
		Expect(namespaces).To(ConsistOf("ns1", "ns2"))
	})

	It("filters events by an inclusive timestamp range", func() {
		from := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 4, 12, 23, 59, 59, 0, time.UTC)
		events := []struct {
			entityID  string
			timestamp time.Time
		}{
			{"before", from.Add(-time.Nanosecond)},
			{"start-boundary", from},
			{"inside", from.Add(24 * time.Hour)},
			{"end-boundary", to},
			{"after", to.Add(time.Nanosecond)},
		}

		for _, ev := range events {
			Expect(repo.Save(ctx, &domain.AuditEvent{
				ID:          uuid.New(),
				TenantID:    "t1",
				Namespace:   "ns1",
				ActorID:     "a1",
				ActorType:   domain.ActorTypeUser,
				EntityType:  "E",
				EntityID:    domain.ID(ev.entityID),
				Action:      domain.ActionCreated,
				Outcome:     domain.OutcomeSuccess,
				ServiceName: "svc",
				Timestamp:   ev.timestamp,
			})).To(Succeed())
		}

		results, err := repo.Query(ctx, usecases.QueryEventsOptions{
			TimestampFrom: &from,
			TimestampTo:   &to,
			PageSize:      10,
		})

		Expect(err).NotTo(HaveOccurred())
		entityIDs := make([]string, 0, len(results))
		for _, result := range results {
			entityIDs = append(entityIDs, string(result.EntityID))
		}
		Expect(entityIDs).To(ConsistOf("start-boundary", "inside", "end-boundary"))
	})

	It("returns distinct namespaces ordered alphabetically", func() {
		for _, ev := range []struct {
			id string
			ns string
		}{
			{"018f0000-0000-7000-8000-000000000010", "billing"},
			{"018f0000-0000-7000-8000-000000000011", "auth"},
			{"018f0000-0000-7000-8000-000000000012", "billing"},
		} {
			Expect(repo.Save(ctx, &domain.AuditEvent{
				ID:          uuid.MustParse(ev.id),
				TenantID:    "t1",
				Namespace:   ev.ns,
				ActorID:     "a1",
				ActorType:   domain.ActorTypeUser,
				EntityType:  "E",
				EntityID:    "e1",
				Action:      domain.ActionCreated,
				Outcome:     domain.OutcomeSuccess,
				ServiceName: "svc",
				Timestamp:   time.Now().UTC(),
			})).To(Succeed())
		}

		ns, err := repo.QueryNamespaces(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(ns).To(Equal([]string{"auth", "billing"}))
	})

	It("returns empty namespace slice when no events exist", func() {
		ns, err := repo.QueryNamespaces(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(ns).To(BeEmpty())
	})
})

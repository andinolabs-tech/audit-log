package usecases_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/auditlog/usecases"
	auditlogdoubles "audit-log/test/unit/doubles/auditlog"
)

var _ = Describe("SimpleAuditService", func() {
	var (
		ctrl  *gomock.Controller
		store *auditlogdoubles.MockEventStore
		svc   *usecases.SimpleAuditService
		ctx   context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		ctrl = gomock.NewController(GinkgoT())
		store = auditlogdoubles.NewMockEventStore(ctrl)
		svc = usecases.NewSimpleAuditService(store)
	})

	baseWriteOpts := func() usecases.WriteEventOptions {
		return usecases.WriteEventOptions{
			TenantID:    "t1",
			Namespace:   "ns1",
			ActorID:     "a1",
			ActorType:   domain.ActorTypeUser,
			EntityType:  "Order",
			EntityID:    "o1",
			Action:      domain.ActionCreated,
			Outcome:     domain.OutcomeSuccess,
			ServiceName: "svc",
		}
	}

	Context("WriteEvent", func() {
		When("before and after differ", func() {
			It("computes RFC 6902 diff and saves the event", func() {
				store.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(&domain.AuditEvent{})).
					DoAndReturn(func(_ context.Context, e *domain.AuditEvent) error {
						Expect(e.Diff).NotTo(BeNil())
						ops, ok := e.Diff["operations"].([]any)
						Expect(ok).To(BeTrue())
						Expect(ops).NotTo(BeEmpty())
						return nil
					})
				opts := baseWriteOpts()
				opts.Before = map[string]any{"n": "old"}
				opts.After = map[string]any{"n": "new"}
				_, err := svc.WriteEvent(ctx, opts)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("before and after are both nil", func() {
			It("leaves diff nil", func() {
				store.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(&domain.AuditEvent{})).
					DoAndReturn(func(_ context.Context, e *domain.AuditEvent) error {
						Expect(e.Diff).To(BeNil())
						return nil
					})
				_, err := svc.WriteEvent(ctx, baseWriteOpts())
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("WriteCompensation", func() {
		When("referenced event does not exist", func() {
			It("returns ErrReferencedEventNotFound", func() {
				missing := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
				store.EXPECT().FindByID(gomock.Any(), missing).Return(nil, nil)

				_, err := svc.WriteCompensation(ctx, usecases.WriteCompensationOptions{
					WriteEventOptions: baseWriteOpts(),
					CompensatesID:     missing,
				})
				Expect(err).To(MatchError(usecases.ErrReferencedEventNotFound))
			})
		})

		When("referenced event belongs to another tenant", func() {
			It("returns ErrTenantMismatch", func() {
				refID := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
				store.EXPECT().FindByID(gomock.Any(), refID).Return(&domain.AuditEvent{
					ID:       refID,
					TenantID: "other",
				}, nil)

				_, err := svc.WriteCompensation(ctx, usecases.WriteCompensationOptions{
					WriteEventOptions: baseWriteOpts(),
					CompensatesID:     refID,
				})
				Expect(err).To(MatchError(usecases.ErrTenantMismatch))
			})
		})

		When("referenced event exists for same tenant", func() {
			It("stores action COMPENSATED and saves", func() {
				refID := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
				store.EXPECT().FindByID(gomock.Any(), refID).Return(&domain.AuditEvent{
					ID:       refID,
					TenantID: "t1",
				}, nil)
				store.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(&domain.AuditEvent{})).
					DoAndReturn(func(_ context.Context, e *domain.AuditEvent) error {
						Expect(e.Action).To(Equal(domain.ActionCompensated))
						Expect(e.CompensatesID).NotTo(BeNil())
						Expect(*e.CompensatesID).To(Equal(refID))
						return nil
					})

				_, err := svc.WriteCompensation(ctx, usecases.WriteCompensationOptions{
					WriteEventOptions: baseWriteOpts(),
					CompensatesID:     refID,
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("QueryEvents", func() {
		When("page size is invalid", func() {
			It("returns ErrInvalidPageSize for zero", func() {
				_, err := svc.QueryEvents(ctx, usecases.QueryEventsOptions{PageSize: 0})
				Expect(err).To(MatchError(usecases.ErrInvalidPageSize))
			})
		})

		When("more rows exist than page size", func() {
			It("returns next_page_token as the last returned event id", func() {
				id1 := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
				id2 := uuid.MustParse("018f1235-5678-7abc-8def-123456789abc")
				id3 := uuid.MustParse("018f1236-5678-7abc-8def-123456789abc")
				store.EXPECT().Query(gomock.Any(), gomock.Any()).Return([]*domain.AuditEvent{
					{ID: id1}, {ID: id2}, {ID: id3},
				}, nil)

				res, err := svc.QueryEvents(ctx, usecases.QueryEventsOptions{PageSize: 2})
				Expect(err).NotTo(HaveOccurred())
				Expect(res.Events).To(HaveLen(2))
				Expect(res.Events[0].ID).To(Equal(id1))
				Expect(res.Events[1].ID).To(Equal(id2))
				Expect(res.NextPageToken).NotTo(BeNil())
				Expect(*res.NextPageToken).To(Equal(id2))
			})
		})
	})

	Context("GetEvent", func() {
		It("returns the event from the store", func() {
			id := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
			ev := &domain.AuditEvent{ID: id, TenantID: "t1"}
			store.EXPECT().FindByID(gomock.Any(), id).Return(ev, nil)

			got, err := svc.GetEvent(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(ev))
		})
	})
})

package mapconv_test

import (
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/auditlog/httpapi/internal/mapconv"
	"audit-log/internal/auditlog/usecases"
)

var _ = Describe("QueryParamsToOpts", func() {
	It("parses repeated namespace params into a Namespaces slice", func() {
		r := httptest.NewRequest("GET", "/api/events?namespace=auth&namespace=billing&page_size=50", nil)

		opts, err := mapconv.QueryParamsToOpts(r)

		Expect(err).NotTo(HaveOccurred())
		Expect(opts.Namespaces).To(ConsistOf("auth", "billing"))
		Expect(opts.PageSize).To(Equal(50))
	})

	It("defaults page_size to 20 when not provided", func() {
		r := httptest.NewRequest("GET", "/api/events", nil)

		opts, err := mapconv.QueryParamsToOpts(r)

		Expect(err).NotTo(HaveOccurred())
		Expect(opts.PageSize).To(Equal(20))
	})

	It("returns error for non-integer page_size", func() {
		r := httptest.NewRequest("GET", "/api/events?page_size=bad", nil)

		_, err := mapconv.QueryParamsToOpts(r)

		Expect(err).To(MatchError(ContainSubstring("page_size")))
	})

	It("returns error for invalid page_token UUID", func() {
		r := httptest.NewRequest("GET", "/api/events?page_token=notauuid", nil)

		_, err := mapconv.QueryParamsToOpts(r)

		Expect(err).To(MatchError(ContainSubstring("page_token")))
	})

	It("parses page_token as UUID", func() {
		id := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
		r := httptest.NewRequest("GET", "/api/events?page_token="+id.String(), nil)

		opts, err := mapconv.QueryParamsToOpts(r)

		Expect(err).NotTo(HaveOccurred())
		Expect(opts.PageToken).NotTo(BeNil())
		Expect(*opts.PageToken).To(Equal(id))
	})

	It("parses optional scalar filters", func() {
		r := httptest.NewRequest("GET", "/api/events?tenant_id=t1&actor_id=a1&action=CREATED&outcome=SUCCESS&source_ip=127.0.0.1&session_id=s1&correlation_id=c1&trace_id=tr1", nil)

		opts, err := mapconv.QueryParamsToOpts(r)

		Expect(err).NotTo(HaveOccurred())
		Expect(opts.TenantID).To(HaveValue(Equal("t1")))
		Expect(opts.ActorID).To(HaveValue(Equal("a1")))
		Expect(opts.Action).To(HaveValue(Equal(domain.ActionCreated)))
		Expect(opts.Outcome).To(HaveValue(Equal(domain.OutcomeSuccess)))
		Expect(opts.SourceIP).To(HaveValue(Equal("127.0.0.1")))
		Expect(opts.SessionID).To(HaveValue(Equal("s1")))
		Expect(opts.CorrelationID).To(HaveValue(Equal("c1")))
		Expect(opts.TraceID).To(HaveValue(Equal("tr1")))
	})
})

var _ = Describe("DomainEventToResponse", func() {
	It("maps all standard fields", func() {
		ts := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		ev := &domain.AuditEvent{
			ID:          uuid.MustParse("018f1234-5678-7abc-8def-123456789abc"),
			TenantID:    "t1",
			Namespace:   "auth",
			ActorID:     "user-1",
			ActorType:   domain.ActorTypeUser,
			EntityType:  "Order",
			EntityID:    "order-1",
			Action:      domain.ActionCreated,
			Outcome:     domain.OutcomeSuccess,
			ServiceName: "orders",
			Timestamp:   ts,
			Tags:        []string{"important"},
		}

		r := mapconv.DomainEventToResponse(ev)

		Expect(r.ID).To(Equal(ev.ID.String()))
		Expect(r.TenantID).To(Equal("t1"))
		Expect(r.Namespace).To(Equal("auth"))
		Expect(r.ActorID).To(Equal("user-1"))
		Expect(r.ActorType).To(Equal("user"))
		Expect(r.EntityType).To(Equal("Order"))
		Expect(r.EntityID).To(Equal("order-1"))
		Expect(r.Action).To(Equal("CREATED"))
		Expect(r.Outcome).To(Equal("SUCCESS"))
		Expect(r.ServiceName).To(Equal("orders"))
		Expect(r.Timestamp).To(Equal("2024-01-15T10:00:00Z"))
		Expect(r.Tags).To(Equal([]string{"important"}))
		Expect(r.CompensatesID).To(BeNil())
		Expect(r.OccurredAt).To(BeNil())
	})

	It("maps CompensatesID and OccurredAt when present", func() {
		compID := uuid.MustParse("018f0000-0000-7000-8000-000000000099")
		occurredAt := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
		ev := &domain.AuditEvent{
			ID:            uuid.New(),
			Timestamp:     time.Now(),
			OccurredAt:    &occurredAt,
			CompensatesID: &compID,
		}

		r := mapconv.DomainEventToResponse(ev)

		Expect(r.CompensatesID).NotTo(BeNil())
		Expect(*r.CompensatesID).To(Equal(compID.String()))
		Expect(r.OccurredAt).NotTo(BeNil())
		Expect(*r.OccurredAt).To(Equal("2024-01-01T09:00:00Z"))
	})

	It("returns empty Tags slice when event has no tags", func() {
		ev := &domain.AuditEvent{ID: uuid.New(), Timestamp: time.Now()}

		r := mapconv.DomainEventToResponse(ev)

		Expect(r.Tags).NotTo(BeNil())
		Expect(r.Tags).To(BeEmpty())
	})
})

var _ = Describe("QueryResultToResponse", func() {
	It("sets next_page_token when present", func() {
		tok := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
		res := &usecases.QueryEventsResult{
			Events:        []*domain.AuditEvent{{ID: uuid.New(), Timestamp: time.Now()}},
			NextPageToken: &tok,
		}

		resp := mapconv.QueryResultToResponse(res)

		Expect(resp.NextPageToken).To(Equal(tok.String()))
		Expect(resp.Events).To(HaveLen(1))
	})

	It("leaves next_page_token empty string when nil", func() {
		res := &usecases.QueryEventsResult{Events: []*domain.AuditEvent{}}

		resp := mapconv.QueryResultToResponse(res)

		Expect(resp.NextPageToken).To(BeEmpty())
	})
})

var _ = Describe("NamespacesToResponse", func() {
	It("wraps namespaces in response struct", func() {
		r := mapconv.NamespacesToResponse([]string{"auth", "billing"})

		Expect(r.Namespaces).To(Equal([]string{"auth", "billing"}))
	})

	It("returns empty slice for nil input", func() {
		r := mapconv.NamespacesToResponse(nil)

		Expect(r.Namespaces).NotTo(BeNil())
		Expect(r.Namespaces).To(BeEmpty())
	})
})

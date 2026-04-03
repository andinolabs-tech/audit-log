package domain_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/google/uuid"

	"audit-log/internal/auditlog/domain"
)

var _ = Describe("AuditEventBuilder", func() {
	base := func() *domain.AuditEventBuilder {
		return domain.NewAuditEventBuilder().
			WithTenantID("tenant-1").
			WithActorID("actor-1").
			WithActorType(domain.ActorTypeUser).
			WithEntityType("Order").
			WithEntityID("ord-1").
			WithAction(domain.ActionCreated).
			WithOutcome(domain.OutcomeSuccess).
			WithServiceName("orders-api")
	}

	It("generates UUID version 7 for the event id", func() {
		event, err := base().Build()
		Expect(err).NotTo(HaveOccurred())
		Expect(event.ID.Version()).To(Equal(uuid.Version(7)))
	})

	It("sets timestamp to current UTC time at build", func() {
		before := time.Now().UTC()
		event, err := base().Build()
		after := time.Now().UTC()
		Expect(err).NotTo(HaveOccurred())
		Expect(event.Timestamp.Location()).To(Equal(time.UTC))
		Expect(event.Timestamp).To(BeTemporally(">=", before))
		Expect(event.Timestamp).To(BeTemporally("<=", after))
	})

	It("does not populate Diff; caller computes it in use case layer", func() {
		event, err := base().
			WithBefore(map[string]any{"a": 1}).
			WithAfter(map[string]any{"a": 2}).
			Build()
		Expect(err).NotTo(HaveOccurred())
		Expect(event.Diff).To(BeNil())
	})

	It("returns validation error when action is Compensated and CompensatesID is missing", func() {
		_, err := base().
			WithAction(domain.ActionCompensated).
			Build()
		Expect(err).To(MatchError(domain.ErrCompensatesIDRequired))
	})

	It("accepts compensation event when CompensatesID is set", func() {
		ref := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
		event, err := base().
			WithAction(domain.ActionCompensated).
			WithCompensatesID(ref).
			Build()
		Expect(err).NotTo(HaveOccurred())
		Expect(event.Action).To(Equal(domain.ActionCompensated))
		Expect(event.CompensatesID).NotTo(BeNil())
		Expect(*event.CompensatesID).To(Equal(ref))
	})

	It("stores optional static and dynamic fields", func() {
		event, err := base().
			WithSourceIP("10.0.0.1").
			WithSessionID("sess-1").
			WithCorrelationID("corr-1").
			WithTraceID("trace-1").
			WithBefore(map[string]any{"x": true}).
			WithAfter(map[string]any{"x": false}).
			WithMetadata(map[string]any{"k": "v"}).
			WithReason("manual fix").
			WithTags([]string{"a", "b"}).
			Build()
		Expect(err).NotTo(HaveOccurred())
		Expect(event.SourceIP).To(Equal("10.0.0.1"))
		Expect(event.SessionID).To(Equal("sess-1"))
		Expect(event.CorrelationID).To(Equal("corr-1"))
		Expect(event.TraceID).To(Equal("trace-1"))
		Expect(event.Before).To(Equal(map[string]any{"x": true}))
		Expect(event.After).To(Equal(map[string]any{"x": false}))
		Expect(event.Metadata).To(Equal(map[string]any{"k": "v"}))
		Expect(event.Reason).To(Equal("manual fix"))
		Expect(event.Tags).To(Equal([]string{"a", "b"}))
	})
})

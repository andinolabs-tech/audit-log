package mapconv_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/types/known/timestamppb"

	"audit-log/internal/auditlog/grpcapi/internal/mapconv"
	auditlogv1 "audit-log/proto/auditlogv1"
)

var _ = Describe("QueryEventsRequestToOpts", func() {
	It("maps the timestamp range into UTC bounds", func() {
		from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)
		req := &auditlogv1.QueryEventsRequest{
			TimestampFrom: timestamppb.New(from),
			TimestampTo:   timestamppb.New(to),
		}

		opts, err := mapconv.QueryEventsRequestToOpts(req)

		Expect(err).NotTo(HaveOccurred())
		Expect(opts.TimestampFrom).To(HaveValue(Equal(from)))
		Expect(opts.TimestampTo).To(HaveValue(Equal(to)))
	})

	It("normalizes non-UTC timestamp bounds to UTC", func() {
		zone := time.FixedZone("UTC-4", -4*60*60)
		from := time.Date(2026, 4, 1, 20, 0, 0, 0, zone)

		opts, err := mapconv.QueryEventsRequestToOpts(&auditlogv1.QueryEventsRequest{
			TimestampFrom: timestamppb.New(from),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(opts.TimestampFrom).To(HaveValue(Equal(time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC))))
	})

	It("leaves the timestamp range unset when not provided", func() {
		opts, err := mapconv.QueryEventsRequestToOpts(&auditlogv1.QueryEventsRequest{})

		Expect(err).NotTo(HaveOccurred())
		Expect(opts.TimestampFrom).To(BeNil())
		Expect(opts.TimestampTo).To(BeNil())
	})

	It("returns an error for an invalid timestamp_from", func() {
		_, err := mapconv.QueryEventsRequestToOpts(&auditlogv1.QueryEventsRequest{
			TimestampFrom: &timestamppb.Timestamp{Seconds: -62135596801},
		})

		Expect(err).To(HaveOccurred())
	})

	It("returns an error for an invalid timestamp_to", func() {
		_, err := mapconv.QueryEventsRequestToOpts(&auditlogv1.QueryEventsRequest{
			TimestampTo: &timestamppb.Timestamp{Nanos: -1},
		})

		Expect(err).To(HaveOccurred())
	})
})

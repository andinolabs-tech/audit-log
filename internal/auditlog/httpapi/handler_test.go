package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/auditlog/httpapi"
	"audit-log/internal/auditlog/usecases"
)

type stubSvc struct {
	queryOpts     usecases.QueryEventsOptions
	queryResult   *usecases.QueryEventsResult
	queryErr      error
	namespaces    []string
	namespacesErr error
}

func (s *stubSvc) WriteEvent(_ context.Context, _ usecases.WriteEventOptions) (*domain.AuditEvent, error) {
	return nil, nil
}

func (s *stubSvc) WriteCompensation(_ context.Context, _ usecases.WriteCompensationOptions) (*domain.AuditEvent, error) {
	return nil, nil
}

func (s *stubSvc) QueryEvents(_ context.Context, opts usecases.QueryEventsOptions) (*usecases.QueryEventsResult, error) {
	s.queryOpts = opts
	return s.queryResult, s.queryErr
}

func (s *stubSvc) GetEvent(_ context.Context, _ uuid.UUID) (*domain.AuditEvent, error) {
	return nil, nil
}

func (s *stubSvc) ListNamespaces(_ context.Context) ([]string, error) {
	return s.namespaces, s.namespacesErr
}

var _ = Describe("Handler", func() {
	var (
		svc *stubSvc
		mux *http.ServeMux
	)

	BeforeEach(func() {
		svc = &stubSvc{}
		mux = http.NewServeMux()
		httpapi.NewHandler(svc).RegisterRoutes(mux)
	})

	Describe("GET /api/events", func() {
		Context("when service returns events", func() {
			BeforeEach(func() {
				tok := uuid.MustParse("018f1234-5678-7abc-8def-123456789abc")
				svc.queryResult = &usecases.QueryEventsResult{
					Events: []*domain.AuditEvent{
						{
							ID:          uuid.MustParse("018f0000-0000-7000-8000-000000000001"),
							TenantID:    "t1",
							Namespace:   "auth",
							ActorID:     "user-1",
							ActorType:   domain.ActorTypeUser,
							EntityType:  "Session",
							EntityID:    "s1",
							Action:      domain.ActionCreated,
							Outcome:     domain.OutcomeSuccess,
							ServiceName: "auth-svc",
							Timestamp:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
							Tags:        []string{"login"},
						},
					},
					NextPageToken: &tok,
				}
			})

			It("returns 200 with events and next_page_token", func() {
				req := httptest.NewRequest("GET", "/api/events?namespace=auth&namespace=billing&page_size=10", nil)
				w := httptest.NewRecorder()

				mux.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
				Expect(svc.queryOpts.Namespaces).To(Equal([]string{"auth", "billing"}))
				var body map[string]any
				Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
				events := body["events"].([]any)
				Expect(events).To(HaveLen(1))
				first := events[0].(map[string]any)
				Expect(first["namespace"]).To(Equal("auth"))
				Expect(first["action"]).To(Equal("CREATED"))
				Expect(body["next_page_token"]).To(Equal("018f1234-5678-7abc-8def-123456789abc"))
			})
		})

		Context("when page_size is invalid", func() {
			It("returns 400", func() {
				req := httptest.NewRequest("GET", "/api/events?page_size=notanumber", nil)
				w := httptest.NewRecorder()

				mux.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
				var body map[string]string
				Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
				Expect(body["error"]).To(ContainSubstring("page_size"))
			})
		})

		Context("when service returns ErrInvalidPageSize", func() {
			It("returns 400", func() {
				svc.queryErr = usecases.ErrInvalidPageSize
				req := httptest.NewRequest("GET", "/api/events?page_size=0", nil)
				w := httptest.NewRecorder()

				mux.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("when service returns an unexpected error", func() {
			It("returns 500", func() {
				svc.queryErr = errors.New("db down")
				req := httptest.NewRequest("GET", "/api/events", nil)
				w := httptest.NewRecorder()

				mux.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	Describe("GET /api/namespaces", func() {
		It("returns 200 with namespace list", func() {
			svc.namespaces = []string{"auth", "billing"}
			req := httptest.NewRequest("GET", "/api/namespaces", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			var body map[string]any
			Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
			ns := body["namespaces"].([]any)
			Expect(ns).To(HaveLen(2))
			Expect(ns[0]).To(Equal("auth"))
			Expect(ns[1]).To(Equal("billing"))
		})

		It("returns empty array when no namespaces exist", func() {
			svc.namespaces = []string{}
			req := httptest.NewRequest("GET", "/api/namespaces", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			var body map[string]any
			Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
			ns := body["namespaces"].([]any)
			Expect(ns).To(BeEmpty())
		})

		It("returns 500 when service errors", func() {
			svc.namespacesErr = errors.New("db error")
			req := httptest.NewRequest("GET", "/api/namespaces", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})

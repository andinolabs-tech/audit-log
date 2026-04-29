# Embedded SPA + HTTP Query API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an HTTP server to the audit-log binary that serves an embedded React SPA and exposes `GET /api/events` (multi-namespace query) and `GET /api/namespaces` endpoints, mirroring the existing gRPC `QueryEvents` use case.

**Architecture:** New `internal/auditlog/httpapi` package mirrors `grpcapi` — thin HTTP handlers delegate to `usecases.AuditService` via `httpapi/internal/mapconv`. The React SPA is built with Vite into `internal/web/dist/` and embedded via `//go:embed all:dist`. Both gRPC (port 50051) and HTTP (port 8080) servers start concurrently from the same `cmd/server` binary.

**Tech Stack:** Go 1.26 stdlib `net/http`, `//go:embed`, Vite 6 + React 18 + TypeScript + Tailwind CSS 3 + React Router v6, Ginkgo/Gomega tests, gomock.

---

## File Map

### Created
- `internal/auditlog/httpapi/handler.go` — HTTP handlers for /api/events and /api/namespaces
- `internal/auditlog/httpapi/httpapi_suite_test.go` — Ginkgo suite runner
- `internal/auditlog/httpapi/handler_test.go` — handler tests with stub AuditService
- `internal/auditlog/httpapi/internal/mapconv/mapconv.go` — URL params → options; domain → JSON structs
- `internal/auditlog/httpapi/internal/mapconv/mapconv_suite_test.go` — Ginkgo suite runner
- `internal/auditlog/httpapi/internal/mapconv/mapconv_test.go` — mapconv unit tests
- `internal/web/embed.go` — `//go:embed all:dist` + `Handler()` with SPA fallback
- `internal/web/dist/.gitkeep` — keeps dist/ tracked so embed compiles on a fresh clone
- `web/package.json`, `web/vite.config.ts`, `web/tsconfig.json`, `web/tailwind.config.ts`, `web/postcss.config.js`, `web/index.html`
- `web/src/main.tsx`, `web/src/index.css`, `web/src/App.tsx`
- `web/src/pages/QueryPage.tsx` — namespace multi-select + results table
- `web/src/types/event.ts` — TypeScript AuditEvent / response types

### Modified
- `internal/auditlog/usecases/repository_port.go` — `Namespace *string` → `Namespaces []string`; add `QueryNamespaces`
- `internal/auditlog/usecases/api.go` — add `ListNamespaces` to `AuditService`
- `internal/auditlog/usecases/audit_service.go` — implement `ListNamespaces`
- `internal/auditlog/usecases/audit_service_test.go` — add `ListNamespaces` test
- `internal/auditlog/persistence/event_repository.go` — `IN` filter for `Namespaces`; add `QueryNamespaces`
- `internal/auditlog/persistence/event_repository_test.go` — add multi-namespace and `QueryNamespaces` tests
- `internal/auditlog/grpcapi/internal/mapconv/mapconv.go` — map single proto `namespace` → `Namespaces` slice
- `test/unit/doubles/auditlog/eventstore_mock.go` — regenerated to include `QueryNamespaces`
- `internal/infra/config/config.go` — add `HTTPPort` (default 8080)
- `cmd/server/wire/wire.go` — add `InitializeService`
- `cmd/server/wire/wire_gen.go` — add generated `InitializeService`
- `cmd/server/main.go` — start HTTP server alongside gRPC
- `arch-go.yml` — add httpapi dependency rule
- `justfile` — add `web-build`, `web-dev`; make `build` depend on `web-build`
- `.gitignore` — exclude `internal/web/dist/*` (keep `.gitkeep`)

---

## Task 1: Extend `QueryEventsOptions.Namespaces` and update persistence + grpcapi

**Files:**
- Modify: `internal/auditlog/usecases/repository_port.go`
- Modify: `internal/auditlog/persistence/event_repository.go`
- Modify: `internal/auditlog/grpcapi/internal/mapconv/mapconv.go`
- Modify: `internal/auditlog/persistence/event_repository_test.go`

- [ ] **Step 1: Write failing persistence test for multi-namespace IN filter**

Add to the `Describe("EventRepository")` block in `internal/auditlog/persistence/event_repository_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/auditlog/persistence/... -run "TestPersistence" -v 2>&1 | tail -20
```

Expected: compile error (`Namespaces` field does not exist yet) or test failure.

- [ ] **Step 3: Change `Namespace *string` to `Namespaces []string` in `repository_port.go`**

Replace in `internal/auditlog/usecases/repository_port.go`:

```go
type QueryEventsOptions struct {
	TenantID      *string
	Namespaces    []string
	ActorID       *string
	ActorType     *domain.ActorType
	EntityType    *string
	EntityID      *string
	Action        *domain.Action
	Outcome       *domain.Outcome
	ServiceName   *string
	SourceIP      *string
	SessionID     *string
	CorrelationID *string
	TraceID       *string
	PageToken     *uuid.UUID
	PageSize      int
}
```

- [ ] **Step 4: Update `applyQueryFilters` in `event_repository.go`**

Replace the `Namespace` filter block in `applyQueryFilters`:

```go
// old:
// if opts.Namespace != nil {
//     q = q.Where("namespace = ?", *opts.Namespace)
// }

// new:
if len(opts.Namespaces) > 0 {
    q = q.Where("namespace IN ?", opts.Namespaces)
}
```

- [ ] **Step 5: Update gRPC mapconv for backward compatibility**

In `internal/auditlog/grpcapi/internal/mapconv/mapconv.go`, replace:

```go
// old:
// if req.Namespace != nil {
//     v := *req.Namespace
//     opts.Namespace = &v
// }

// new:
if req.Namespace != nil && *req.Namespace != "" {
    opts.Namespaces = []string{*req.Namespace}
}
```

- [ ] **Step 6: Regenerate the EventStore mock**

```bash
just mock
```

Expected: `test/unit/doubles/auditlog/eventstore_mock.go` regenerated without errors.

- [ ] **Step 7: Run all unit tests**

```bash
go test ./... -race -count=1 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/auditlog/usecases/repository_port.go \
        internal/auditlog/persistence/event_repository.go \
        internal/auditlog/persistence/event_repository_test.go \
        internal/auditlog/grpcapi/internal/mapconv/mapconv.go \
        test/unit/doubles/auditlog/eventstore_mock.go
git commit -m "refactor: change QueryEventsOptions.Namespace to Namespaces slice for multi-namespace filtering"
```

---

## Task 2: Add `ListNamespaces` operation

**Files:**
- Modify: `internal/auditlog/usecases/repository_port.go`
- Modify: `internal/auditlog/usecases/api.go`
- Modify: `internal/auditlog/usecases/audit_service.go`
- Modify: `internal/auditlog/usecases/audit_service_test.go`
- Modify: `internal/auditlog/persistence/event_repository.go`
- Modify: `internal/auditlog/persistence/event_repository_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/auditlog/persistence/event_repository_test.go` inside `Describe("EventRepository")`:

```go
It("returns distinct namespaces ordered alphabetically", func() {
    for _, ev := range []struct{ id, ns string }{
        {"018f0000-0000-7000-8000-000000000010", "billing"},
        {"018f0000-0000-7000-8000-000000000011", "auth"},
        {"018f0000-0000-7000-8000-000000000012", "billing"}, // duplicate
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

It("returns empty slice when no events exist", func() {
    ns, err := repo.QueryNamespaces(ctx)
    Expect(err).NotTo(HaveOccurred())
    Expect(ns).To(BeEmpty())
})
```

Add to `internal/auditlog/usecases/audit_service_test.go` inside `Describe("SimpleAuditService")`:

```go
Context("ListNamespaces", func() {
    It("delegates to the store and returns namespaces", func() {
        store.EXPECT().QueryNamespaces(gomock.Any()).Return([]string{"auth", "billing"}, nil)
        ns, err := svc.ListNamespaces(ctx)
        Expect(err).NotTo(HaveOccurred())
        Expect(ns).To(Equal([]string{"auth", "billing"}))
    })

    It("propagates store errors", func() {
        store.EXPECT().QueryNamespaces(gomock.Any()).Return(nil, errors.New("db error"))
        _, err := svc.ListNamespaces(ctx)
        Expect(err).To(MatchError("db error"))
    })
})
```

Also add `"errors"` to the import in `audit_service_test.go` if not already present.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/auditlog/... -race -count=1 2>&1 | tail -20
```

Expected: compile errors — `QueryNamespaces` and `ListNamespaces` not defined yet.

- [ ] **Step 3: Add `QueryNamespaces` to `EventStore` interface**

In `internal/auditlog/usecases/repository_port.go`, extend the interface:

```go
//go:generate mockgen -destination=../../../test/unit/doubles/auditlog/eventstore_mock.go -package=auditlogdoubles audit-log/internal/auditlog/usecases EventStore

type EventStore interface {
	Save(ctx context.Context, event *domain.AuditEvent) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error)
	Query(ctx context.Context, opts QueryEventsOptions) ([]*domain.AuditEvent, error)
	QueryNamespaces(ctx context.Context) ([]string, error)
}
```

- [ ] **Step 4: Add `ListNamespaces` to `AuditService` interface**

In `internal/auditlog/usecases/api.go`, extend:

```go
type AuditService interface {
	WriteEvent(ctx context.Context, opts WriteEventOptions) (*domain.AuditEvent, error)
	WriteCompensation(ctx context.Context, opts WriteCompensationOptions) (*domain.AuditEvent, error)
	QueryEvents(ctx context.Context, opts QueryEventsOptions) (*QueryEventsResult, error)
	GetEvent(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error)
	ListNamespaces(ctx context.Context) ([]string, error)
}
```

- [ ] **Step 5: Implement `ListNamespaces` in `SimpleAuditService`**

Add to `internal/auditlog/usecases/audit_service.go`:

```go
func (s *SimpleAuditService) ListNamespaces(ctx context.Context) ([]string, error) {
	return s.store.QueryNamespaces(ctx)
}
```

- [ ] **Step 6: Implement `QueryNamespaces` in `EventRepository`**

Add to `internal/auditlog/persistence/event_repository.go`:

```go
func (r *EventRepository) QueryNamespaces(ctx context.Context) ([]string, error) {
	var namespaces []string
	err := r.db.WithContext(ctx).
		Model(&internal.AuditEventRecord{}).
		Distinct("namespace").
		Where("namespace IS NOT NULL AND namespace != ''").
		Order("namespace ASC").
		Pluck("namespace", &namespaces).Error
	if err != nil {
		return nil, err
	}
	if namespaces == nil {
		namespaces = []string{}
	}
	return namespaces, nil
}
```

- [ ] **Step 7: Regenerate the EventStore mock**

```bash
just mock
```

Expected: `test/unit/doubles/auditlog/eventstore_mock.go` updated with `QueryNamespaces` method.

- [ ] **Step 8: Run all unit tests**

```bash
go test ./... -race -count=1 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 9: Commit**

```bash
git add internal/auditlog/usecases/repository_port.go \
        internal/auditlog/usecases/api.go \
        internal/auditlog/usecases/audit_service.go \
        internal/auditlog/usecases/audit_service_test.go \
        internal/auditlog/persistence/event_repository.go \
        internal/auditlog/persistence/event_repository_test.go \
        test/unit/doubles/auditlog/eventstore_mock.go
git commit -m "feat: add ListNamespaces operation to AuditService and EventStore"
```

---

## Task 3: httpapi mapconv

**Files:**
- Create: `internal/auditlog/httpapi/internal/mapconv/mapconv.go`
- Create: `internal/auditlog/httpapi/internal/mapconv/mapconv_suite_test.go`
- Create: `internal/auditlog/httpapi/internal/mapconv/mapconv_test.go`

- [ ] **Step 1: Create the Ginkgo suite runner**

Create `internal/auditlog/httpapi/internal/mapconv/mapconv_suite_test.go`:

```go
package mapconv_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMapconv(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTPApi Mapconv Suite")
}
```

- [ ] **Step 2: Write failing tests**

Create `internal/auditlog/httpapi/internal/mapconv/mapconv_test.go`:

```go
package mapconv_test

import (
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/google/uuid"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/auditlog/httpapi/internal/mapconv"
	"audit-log/internal/auditlog/usecases"
)

var _ = Describe("QueryParamsToOpts", func() {
	It("parses multiple namespace params into Namespaces slice", func() {
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
		r := httptest.NewRequest("GET", "/api/events?tenant_id=t1&actor_id=a1&action=CREATED&outcome=SUCCESS", nil)
		opts, err := mapconv.QueryParamsToOpts(r)
		Expect(err).NotTo(HaveOccurred())
		Expect(opts.TenantID).To(HaveValue(Equal("t1")))
		Expect(opts.ActorID).To(HaveValue(Equal("a1")))
		Expect(opts.Action).To(HaveValue(Equal(domain.ActionCreated)))
		Expect(opts.Outcome).To(HaveValue(Equal(domain.OutcomeSuccess)))
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
		Expect(r.Namespace).To(Equal("auth"))
		Expect(r.Action).To(Equal("CREATED"))
		Expect(r.Outcome).To(Equal("SUCCESS"))
		Expect(r.Timestamp).To(Equal("2024-01-15T10:00:00Z"))
		Expect(r.Tags).To(Equal([]string{"important"}))
		Expect(r.CompensatesID).To(BeNil())
		Expect(r.OccurredAt).To(BeNil())
	})

	It("maps CompensatesID when present", func() {
		compID := uuid.MustParse("018f0000-0000-7000-8000-000000000099")
		ev := &domain.AuditEvent{ID: uuid.New(), Timestamp: time.Now(), CompensatesID: &compID}
		r := mapconv.DomainEventToResponse(ev)
		Expect(r.CompensatesID).NotTo(BeNil())
		Expect(*r.CompensatesID).To(Equal(compID.String()))
	})

	It("returns empty Tags slice (never nil) when event has no tags", func() {
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

	It("returns empty slice (never nil) for nil input", func() {
		r := mapconv.NamespacesToResponse(nil)
		Expect(r.Namespaces).NotTo(BeNil())
		Expect(r.Namespaces).To(BeEmpty())
	})
})
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/auditlog/httpapi/... 2>&1 | tail -10
```

Expected: compile error — package `mapconv` not found.

- [ ] **Step 4: Create the mapconv implementation**

Create `internal/auditlog/httpapi/internal/mapconv/mapconv.go`:

```go
package mapconv

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/auditlog/usecases"
)

type EventResponse struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id"`
	Namespace     string         `json:"namespace"`
	ActorID       string         `json:"actor_id"`
	ActorType     string         `json:"actor_type"`
	EntityType    string         `json:"entity_type"`
	EntityID      string         `json:"entity_id"`
	Action        string         `json:"action"`
	Outcome       string         `json:"outcome"`
	ServiceName   string         `json:"service_name"`
	SourceIP      string         `json:"source_ip,omitempty"`
	SessionID     string         `json:"session_id,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	TraceID       string         `json:"trace_id,omitempty"`
	Timestamp     string         `json:"timestamp"`
	OccurredAt    *string        `json:"occurred_at,omitempty"`
	CompensatesID *string        `json:"compensates_id,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	Tags          []string       `json:"tags"`
	Before        map[string]any `json:"before,omitempty"`
	After         map[string]any `json:"after,omitempty"`
	Diff          map[string]any `json:"diff,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type QueryEventsResponse struct {
	Events        []EventResponse `json:"events"`
	NextPageToken string          `json:"next_page_token"`
}

type NamespacesResponse struct {
	Namespaces []string `json:"namespaces"`
}

func QueryParamsToOpts(r *http.Request) (usecases.QueryEventsOptions, error) {
	q := r.URL.Query()
	opts := usecases.QueryEventsOptions{PageSize: 20}

	if v := q.Get("tenant_id"); v != "" {
		opts.TenantID = &v
	}
	if ns := q["namespace"]; len(ns) > 0 {
		opts.Namespaces = ns
	}
	if v := q.Get("actor_id"); v != "" {
		opts.ActorID = &v
	}
	if v := q.Get("actor_type"); v != "" {
		t := domain.ActorType(v)
		opts.ActorType = &t
	}
	if v := q.Get("entity_type"); v != "" {
		opts.EntityType = &v
	}
	if v := q.Get("entity_id"); v != "" {
		opts.EntityID = &v
	}
	if v := q.Get("action"); v != "" {
		a := domain.Action(v)
		opts.Action = &a
	}
	if v := q.Get("outcome"); v != "" {
		o := domain.Outcome(v)
		opts.Outcome = &o
	}
	if v := q.Get("service_name"); v != "" {
		opts.ServiceName = &v
	}
	if v := q.Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return usecases.QueryEventsOptions{}, fmt.Errorf("invalid page_size: must be an integer")
		}
		opts.PageSize = n
	}
	if v := q.Get("page_token"); v != "" {
		tok, err := uuid.Parse(v)
		if err != nil {
			return usecases.QueryEventsOptions{}, fmt.Errorf("invalid page_token: must be a UUID")
		}
		opts.PageToken = &tok
	}
	return opts, nil
}

func DomainEventToResponse(e *domain.AuditEvent) EventResponse {
	r := EventResponse{
		ID:            e.ID.String(),
		TenantID:      e.TenantID.String(),
		Namespace:     e.Namespace,
		ActorID:       e.ActorID.String(),
		ActorType:     string(e.ActorType),
		EntityType:    e.EntityType,
		EntityID:      e.EntityID.String(),
		Action:        string(e.Action),
		Outcome:       string(e.Outcome),
		ServiceName:   e.ServiceName,
		SourceIP:      e.SourceIP,
		SessionID:     e.SessionID.String(),
		CorrelationID: e.CorrelationID.String(),
		TraceID:       e.TraceID,
		Timestamp:     e.Timestamp.UTC().Format(time.RFC3339),
		Reason:        e.Reason,
		Tags:          e.Tags,
		Before:        e.Before,
		After:         e.After,
		Diff:          e.Diff,
		Metadata:      e.Metadata,
	}
	if e.OccurredAt != nil {
		s := e.OccurredAt.UTC().Format(time.RFC3339)
		r.OccurredAt = &s
	}
	if e.CompensatesID != nil {
		s := e.CompensatesID.String()
		r.CompensatesID = &s
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}
	return r
}

func QueryResultToResponse(res *usecases.QueryEventsResult) QueryEventsResponse {
	events := make([]EventResponse, 0, len(res.Events))
	for _, e := range res.Events {
		events = append(events, DomainEventToResponse(e))
	}
	resp := QueryEventsResponse{Events: events}
	if res.NextPageToken != nil {
		resp.NextPageToken = res.NextPageToken.String()
	}
	return resp
}

func NamespacesToResponse(ns []string) NamespacesResponse {
	if ns == nil {
		ns = []string{}
	}
	return NamespacesResponse{Namespaces: ns}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/auditlog/httpapi/internal/mapconv/... -v 2>&1 | tail -20
```

Expected: all specs pass.

- [ ] **Step 6: Commit**

```bash
git add internal/auditlog/httpapi/
git commit -m "feat: add httpapi mapconv for URL params → QueryEventsOptions and domain → JSON"
```

---

## Task 4: httpapi handler

**Files:**
- Create: `internal/auditlog/httpapi/httpapi_suite_test.go`
- Create: `internal/auditlog/httpapi/handler_test.go`
- Create: `internal/auditlog/httpapi/handler.go`

- [ ] **Step 1: Create the Ginkgo suite runner**

Create `internal/auditlog/httpapi/httpapi_suite_test.go`:

```go
package httpapi_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHttpapi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTPApi Suite")
}
```

- [ ] **Step 2: Write failing handler tests**

Create `internal/auditlog/httpapi/handler_test.go`:

```go
package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/google/uuid"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/auditlog/httpapi"
	"audit-log/internal/auditlog/usecases"
)

// stubSvc is a hand-rolled test double for usecases.AuditService.
type stubSvc struct {
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
func (s *stubSvc) QueryEvents(_ context.Context, _ usecases.QueryEventsOptions) (*usecases.QueryEventsResult, error) {
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
				req := httptest.NewRequest("GET", "/api/events?namespace=auth&page_size=10", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
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
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/auditlog/httpapi/... 2>&1 | tail -10
```

Expected: compile error — `httpapi.NewHandler` not found.

- [ ] **Step 4: Create the handler implementation**

Create `internal/auditlog/httpapi/handler.go`:

```go
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"audit-log/internal/auditlog/httpapi/internal/mapconv"
	"audit-log/internal/auditlog/usecases"
)

type Handler struct {
	svc usecases.AuditService
}

func NewHandler(svc usecases.AuditService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events", h.queryEvents)
	mux.HandleFunc("GET /api/namespaces", h.listNamespaces)
}

func (h *Handler) queryEvents(w http.ResponseWriter, r *http.Request) {
	opts, err := mapconv.QueryParamsToOpts(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.QueryEvents(r.Context(), opts)
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidPageSize) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, mapconv.QueryResultToResponse(res))
}

func (h *Handler) listNamespaces(w http.ResponseWriter, r *http.Request) {
	ns, err := h.svc.ListNamespaces(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, mapconv.NamespacesToResponse(ns))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/auditlog/httpapi/... -v 2>&1 | tail -30
```

Expected: all specs pass.

- [ ] **Step 6: Run full unit suite**

```bash
go test ./... -race -count=1 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/auditlog/httpapi/
git commit -m "feat: add httpapi handler for GET /api/events and GET /api/namespaces"
```

---

## Task 5: `internal/web` embed package

**Files:**
- Create: `internal/web/embed.go`
- Create: `internal/web/dist/.gitkeep`
- Modify: `.gitignore`

- [ ] **Step 1: Create `internal/web/dist/.gitkeep`**

```bash
mkdir -p internal/web/dist
touch internal/web/dist/.gitkeep
```

- [ ] **Step 2: Add dist and node_modules to `.gitignore`**

Add to `.gitignore` (after the existing `# Build output` block):

```
# Frontend build output (populated by `just web-build`)
/internal/web/dist/*
!/internal/web/dist/.gitkeep

# Node
web/node_modules/
```

- [ ] **Step 3: Create `internal/web/embed.go`**

```go
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var dist embed.FS

func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if _, err := fs.Stat(sub, path); err != nil {
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./internal/web/... 2>&1
```

Expected: no errors (dist/ exists with .gitkeep, satisfying the embed directive).

- [ ] **Step 5: Commit**

```bash
git add internal/web/ .gitignore
git commit -m "feat: add embedded SPA handler with SPA fallback and cache headers"
```

---

## Task 6: Config, Wire, main.go, and arch-go

**Files:**
- Modify: `internal/infra/config/config.go`
- Modify: `cmd/server/wire/wire.go`
- Modify: `cmd/server/wire/wire_gen.go`
- Modify: `cmd/server/main.go`
- Modify: `arch-go.yml`

- [ ] **Step 1: Add `HTTPPort` to config**

In `internal/infra/config/config.go`:

Add `HTTPPort int` to the `Config` struct:
```go
type Config struct {
	ServerPort      int
	HTTPPort        int
	PprofPort       int
	DBDSN           string
	DBAdminDSN      string
	OTelEnabled     bool
	OTelEndpoint    string
	OTelServiceName    string
	OTelServiceVersion string
	OTelEnvironment    string
	OTelSampleRate     float64
	GeneralLogLevel string
}
```

Add default and binding in `load()`:
```go
v.SetDefault("http_port", 8080)
```

Add to Config initialization:
```go
HTTPPort: v.GetInt("http_port"),
```

- [ ] **Step 2: Add `InitializeService` to wire.go**

In `cmd/server/wire/wire.go`, add after `InitializeGRPC`:

```go
func InitializeService(db *gorm.DB) (usecases.AuditService, error) {
	wire.Build(
		persistence.NewEventRepository,
		wire.Bind(new(usecases.EventStore), new(*persistence.EventRepository)),
		usecases.NewSimpleAuditService,
		wire.Bind(new(usecases.AuditService), new(*usecases.SimpleAuditService)),
	)
	return nil, nil
}
```

- [ ] **Step 3: Add generated `InitializeService` to wire_gen.go**

In `cmd/server/wire/wire_gen.go`, add the generated function (append after the existing `InitializeGRPC`):

```go
func InitializeService(db *gorm.DB) (usecases.AuditService, error) {
	eventRepository := persistence.NewEventRepository(db)
	simpleAuditService := usecases.NewSimpleAuditService(eventRepository)
	return simpleAuditService, nil
}
```

Also add `"audit-log/internal/auditlog/usecases"` to the imports in `wire_gen.go` if not already present (it already is).

- [ ] **Step 4: Update `main.go` to start the HTTP server**

Replace the entire `run()` function in `cmd/server/main.go` with:

```go
func run() error {
	cfg, err := config.Get()
	if err != nil {
		return err
	}
	initLogging(cfg)

	shutdownTelemetry, err := telemetry.Install(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			slog.Error("telemetry shutdown", "err", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var db *gorm.DB
	if cfg.DBDSN == "" {
		slog.Warn("AUDIT_LOG_DB_DSN not set — using in-memory SQLite (data will not persist)")
		db, err = database.OpenInMemory()
		if err != nil {
			return err
		}
		if err := persistence.AutoMigrateModel(db); err != nil {
			return err
		}
	} else {
		if cfg.DBAdminDSN != "" {
			admin, err := database.OpenGORM(cfg.DBAdminDSN)
			if err != nil {
				return err
			}
			sqlAdmin, err := admin.DB()
			if err != nil {
				return err
			}
			defer sqlAdmin.Close()
			if err := persistence.AutoMigrateModel(admin); err != nil {
				return err
			}
			if err := persistence.BootstrapSQL(admin); err != nil {
				slog.Warn("database bootstrap SQL failed (expected on non-Postgres or missing privileges)", "err", err)
			}
		}
		db, err = database.OpenGORM(cfg.DBDSN)
		if err != nil {
			return err
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	grpcSrv, err := wire.InitializeGRPC(db)
	if err != nil {
		return err
	}

	svc, err := wire.InitializeService(db)
	if err != nil {
		return err
	}

	go func() {
		addr := ":" + strconv.Itoa(cfg.PprofPort)
		slog.Info("pprof listening", "addr", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			slog.Error("pprof server", "err", err)
		}
	}()

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.ServerPort))
	if err != nil {
		return err
	}
	slog.Info("gRPC listening", "addr", lis.Addr().String())
	reflection.Register(grpcSrv)

	httpMux := http.NewServeMux()
	httpapi.NewHandler(svc).RegisterRoutes(httpMux)
	httpMux.Handle("/", webstatic.Handler())
	httpSrv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.HTTPPort),
		Handler: httpMux,
	}

	errCh := make(chan error, 2)
	go func() {
		if serveErr := grpcSrv.Serve(lis); serveErr != nil {
			errCh <- serveErr
		}
	}()
	go func() {
		slog.Info("HTTP listening", "addr", httpSrv.Addr)
		if serveErr := httpSrv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal")

		httpCtx, httpCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer httpCancel()
		if shutdownErr := httpSrv.Shutdown(httpCtx); shutdownErr != nil {
			slog.Error("HTTP shutdown", "err", shutdownErr)
		}

		stopped := make(chan struct{})
		go func() {
			grpcSrv.GracefulStop()
			close(stopped)
		}()
		select {
		case <-time.After(15 * time.Second):
			grpcSrv.Stop()
		case <-stopped:
		}
		return context.Canceled
	case err := <-errCh:
		return err
	}
}
```

Also update the imports block in `main.go` to include the new packages:

```go
import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"

	"audit-log/cmd/server/wire"
	"audit-log/internal/auditlog/httpapi"
	"audit-log/internal/auditlog/persistence"
	"audit-log/internal/infra/config"
	"audit-log/internal/infra/database"
	"audit-log/internal/infra/telemetry"
	webstatic "audit-log/internal/web"
)
```

- [ ] **Step 5: Add httpapi rule to `arch-go.yml`**

Add after the existing grpcapi rule in `arch-go.yml`:

```yaml
  - package: "audit-log/internal/auditlog/httpapi"
    shouldNotDependsOn:
      internal:
        - "audit-log/internal/auditlog/persistence"
```

- [ ] **Step 6: Build to verify compilation**

```bash
go build ./cmd/server/... 2>&1
```

Expected: no errors.

- [ ] **Step 7: Run arch check**

```bash
just arch 2>&1 | tail -10
```

Expected: 100% compliance.

- [ ] **Step 8: Commit**

```bash
git add internal/infra/config/config.go \
        cmd/server/wire/wire.go \
        cmd/server/wire/wire_gen.go \
        cmd/server/main.go \
        arch-go.yml
git commit -m "feat: start HTTP server alongside gRPC; add HTTPPort config; wire InitializeService"
```

---

## Task 7: Frontend scaffold

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/tailwind.config.ts`
- Create: `web/postcss.config.js`
- Create: `web/index.html`

- [ ] **Step 1: Create `web/package.json`**

```json
{
  "name": "audit-log-ui",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.30.0"
  },
  "devDependencies": {
    "@types/react": "^18.3.21",
    "@types/react-dom": "^18.3.6",
    "@vitejs/plugin-react": "^4.3.4",
    "autoprefixer": "^10.4.21",
    "postcss": "^8.5.3",
    "tailwindcss": "^3.4.17",
    "typescript": "^5.7.3",
    "vite": "^6.3.4"
  }
}
```

- [ ] **Step 2: Create `web/vite.config.ts`**

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../internal/web/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
```

- [ ] **Step 3: Create `web/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src"]
}
```

- [ ] **Step 4: Create `web/tailwind.config.ts`**

```ts
import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {},
  },
  plugins: [],
} satisfies Config
```

- [ ] **Step 5: Create `web/postcss.config.js`**

```js
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
```

- [ ] **Step 6: Create `web/index.html`**

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Audit Log</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 7: Install npm dependencies**

```bash
cd web && npm install
```

Expected: `node_modules/` populated, `package-lock.json` created.

- [ ] **Step 8: Verify TypeScript compiles**

```bash
cd web && npx tsc --noEmit 2>&1 | head -20
```

Expected: no errors (src/ is empty so there's nothing to check yet — this may report no input files, which is fine).

- [ ] **Step 9: Commit scaffold**

```bash
git add web/package.json web/vite.config.ts web/tsconfig.json web/tailwind.config.ts web/postcss.config.js web/index.html web/package-lock.json
git commit -m "chore: add frontend scaffold (Vite + React + TypeScript + Tailwind)"
```

---

## Task 8: Frontend app

**Files:**
- Create: `web/src/types/event.ts`
- Create: `web/src/main.tsx`
- Create: `web/src/index.css`
- Create: `web/src/App.tsx`
- Create: `web/src/pages/QueryPage.tsx`

- [ ] **Step 1: Create `web/src/types/event.ts`**

```ts
export interface AuditEvent {
  id: string
  tenant_id: string
  namespace: string
  actor_id: string
  actor_type: string
  entity_type: string
  entity_id: string
  action: string
  outcome: string
  service_name: string
  source_ip?: string
  session_id?: string
  correlation_id?: string
  trace_id?: string
  timestamp: string
  occurred_at?: string
  compensates_id?: string
  reason?: string
  tags: string[]
  before?: Record<string, unknown>
  after?: Record<string, unknown>
  diff?: Record<string, unknown>
  metadata?: Record<string, unknown>
}

export interface QueryEventsResponse {
  events: AuditEvent[]
  next_page_token: string
}

export interface NamespacesResponse {
  namespaces: string[]
}
```

- [ ] **Step 2: Create `web/src/index.css`**

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

- [ ] **Step 3: Create `web/src/main.tsx`**

```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
```

- [ ] **Step 4: Create `web/src/App.tsx`**

```tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import QueryPage from './pages/QueryPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<QueryPage />} />
        <Route path="/events" element={<QueryPage />} />
      </Routes>
    </BrowserRouter>
  )
}
```

- [ ] **Step 5: Create `web/src/pages/QueryPage.tsx`**

```tsx
import { useEffect, useState } from 'react'
import type { AuditEvent, NamespacesResponse, QueryEventsResponse } from '../types/event'

export default function QueryPage() {
  const [namespaces, setNamespaces] = useState<string[]>([])
  const [selected, setSelected] = useState<string[]>([])
  const [pageSize, setPageSize] = useState(20)
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [nextPageToken, setNextPageToken] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [searched, setSearched] = useState(false)

  useEffect(() => {
    fetch('/api/namespaces')
      .then((r) => r.json() as Promise<NamespacesResponse>)
      .then((data) => setNamespaces(data.namespaces))
      .catch(() => {})
  }, [])

  const buildUrl = (pageToken?: string) => {
    const params = new URLSearchParams()
    selected.forEach((ns) => params.append('namespace', ns))
    params.set('page_size', String(pageSize))
    if (pageToken) params.set('page_token', pageToken)
    return `/api/events?${params.toString()}`
  }

  const search = async (append = false, pageToken?: string) => {
    setLoading(true)
    setError('')
    try {
      const res = await fetch(buildUrl(pageToken))
      if (!res.ok) {
        const body = (await res.json()) as { error?: string }
        throw new Error(body.error ?? 'Request failed')
      }
      const data = (await res.json()) as QueryEventsResponse
      setEvents((prev) => (append ? [...prev, ...data.events] : data.events))
      setNextPageToken(data.next_page_token)
      setSearched(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  const toggle = (ns: string) =>
    setSelected((prev) =>
      prev.includes(ns) ? prev.filter((n) => n !== ns) : [...prev, ns],
    )

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-6xl mx-auto">
        <h1 className="text-2xl font-semibold text-gray-900 mb-6">Audit Log</h1>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 mb-6">
          <div className="flex flex-wrap gap-4 items-end">
            <div className="flex-1 min-w-48">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Namespaces
              </label>
              {namespaces.length === 0 ? (
                <p className="text-sm text-gray-400 italic">No namespaces available</p>
              ) : (
                <div className="flex flex-wrap gap-2 max-h-32 overflow-y-auto border border-gray-300 rounded-md p-2">
                  {namespaces.map((ns) => (
                    <label key={ns} className="flex items-center gap-1.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selected.includes(ns)}
                        onChange={() => toggle(ns)}
                        className="rounded border-gray-300 text-blue-600"
                      />
                      <span className="text-sm text-gray-700">{ns}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Page size
              </label>
              <input
                type="number"
                min={1}
                max={500}
                value={pageSize}
                onChange={(e) => setPageSize(Number(e.target.value))}
                className="w-24 border border-gray-300 rounded-md px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <button
              onClick={() => search(false)}
              disabled={loading}
              className="px-4 py-1.5 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {loading ? 'Loading…' : 'Search'}
            </button>
          </div>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
            {error}
          </div>
        )}

        {searched && events.length === 0 && !loading && (
          <p className="text-center text-gray-500 py-12">No events found.</p>
        )}

        {events.length > 0 && (
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    {['Timestamp', 'Namespace', 'Action', 'Actor', 'Entity', 'Outcome'].map(
                      (h) => (
                        <th
                          key={h}
                          className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                        >
                          {h}
                        </th>
                      ),
                    )}
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {events.map((ev) => (
                    <tr key={ev.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-gray-600 whitespace-nowrap text-xs">
                        {new Date(ev.timestamp).toLocaleString()}
                      </td>
                      <td className="px-4 py-3 text-gray-900">{ev.namespace}</td>
                      <td className="px-4 py-3">
                        <span className="inline-flex px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                          {ev.action}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-600 font-mono text-xs truncate max-w-32">
                        {ev.actor_id}
                      </td>
                      <td className="px-4 py-3 text-gray-600 text-xs">
                        {ev.entity_type}/{ev.entity_id}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${
                            ev.outcome === 'SUCCESS'
                              ? 'bg-green-100 text-green-800'
                              : ev.outcome === 'FAILURE'
                                ? 'bg-red-100 text-red-800'
                                : 'bg-yellow-100 text-yellow-800'
                          }`}
                        >
                          {ev.outcome}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {nextPageToken && (
              <div className="px-4 py-3 border-t border-gray-200 text-center">
                <button
                  onClick={() => search(true, nextPageToken)}
                  disabled={loading}
                  className="px-4 py-1.5 text-sm text-blue-600 hover:text-blue-800 font-medium disabled:opacity-50"
                >
                  {loading ? 'Loading…' : 'Load more'}
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
```

- [ ] **Step 6: Verify TypeScript and build**

```bash
cd web && npm run build 2>&1 | tail -20
```

Expected: build succeeds, `internal/web/dist/` populated with `index.html` and `assets/`.

- [ ] **Step 7: Commit**

```bash
git add web/src/
git commit -m "feat: add React SPA with namespace multi-select and audit event results table"
```

---

## Task 9: justfile + end-to-end build verification

**Files:**
- Modify: `justfile`

- [ ] **Step 1: Update `justfile`**

Replace the existing `build` recipe and add `web-build` and `web-dev`:

```just
web-build:
    cd web && npm ci && npm run build

web-dev:
    cd web && npm run dev

build: web-build
    go build -o bin/audit-log ./cmd/server
```

The existing `run` recipe depends on `build`, so it will automatically include `web-build`.

- [ ] **Step 2: Run `just web-build` to verify frontend build**

```bash
just web-build 2>&1 | tail -10
```

Expected: Vite build completes, `internal/web/dist/index.html` exists.

- [ ] **Step 3: Run `just build` to verify the full binary compiles**

```bash
just build 2>&1 | tail -10
```

Expected: `bin/audit-log` produced without errors.

- [ ] **Step 4: Smoke-test the binary**

```bash
./bin/audit-log &
SERVER_PID=$!
sleep 1

# Check namespaces endpoint
curl -s http://localhost:8080/api/namespaces | python3 -m json.tool

# Check events endpoint
curl -s "http://localhost:8080/api/events?page_size=5" | python3 -m json.tool

# Check SPA is served at /
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/

# Check SPA fallback works for /events route
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/events

kill $SERVER_PID
```

Expected:
- `GET /api/namespaces` → `{"namespaces":[]}` (200)
- `GET /api/events` → `{"events":[],"next_page_token":""}` (200)
- `GET /` → 200
- `GET /events` → 200 (SPA fallback serves index.html)

- [ ] **Step 5: Run full test suite one final time**

```bash
go test ./... -race -count=1 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add justfile
git commit -m "chore: add web-build and web-dev justfile targets; make build depend on web-build"
```

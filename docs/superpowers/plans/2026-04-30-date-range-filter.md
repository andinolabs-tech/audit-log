# Date Range Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a timestamp date range filter to the audit log UI and backend query pipeline.

**Architecture:** The frontend owns local date boundary calculation and sends UTC RFC3339 query params. The HTTP map conversion parses those bounds into `QueryEventsOptions`, and the persistence layer applies inclusive timestamp filters.

**Tech Stack:** React, TypeScript, Vitest, Go, Ginkgo/Gomega, Gorm.

---

### Task 1: Date Range Computation

**Files:**
- Create: `web/src/dateRange.ts`
- Test: `web/src/dateRange.test.ts`

- [ ] Write tests for preset and custom range resolution using fake timers.
- [ ] Verify the new tests fail because `dateRange.ts` does not exist.
- [ ] Implement `DatePreset`, `DateRangeValue`, and `resolveDateRange`.
- [ ] Run `npm --prefix web test -- src/dateRange.test.ts`.

### Task 2: Date Range UI and URL Params

**Files:**
- Create: `web/src/components/DateRangeFilter.tsx`
- Modify: `web/src/pages/QueryPage.tsx`
- Test: `web/src/pages/QueryPage.test.tsx`

- [ ] Write tests that selecting a preset sends `timestamp_from` and `timestamp_to`.
- [ ] Write tests that incomplete custom ranges omit timestamp params.
- [ ] Verify the tests fail before implementation.
- [ ] Add the date range component and wire `QueryPage` state, layout, and URL generation.
- [ ] Run `npm --prefix web test -- src/pages/QueryPage.test.tsx src/dateRange.test.ts`.

### Task 3: Backend Query Params and Repository Filters

**Files:**
- Modify: `internal/auditlog/usecases/repository_port.go`
- Modify: `internal/auditlog/httpapi/internal/mapconv/mapconv.go`
- Modify: `internal/auditlog/persistence/event_repository.go`
- Test: `internal/auditlog/httpapi/internal/mapconv/mapconv_test.go`
- Test: `internal/auditlog/persistence/event_repository_test.go`

- [ ] Write tests for valid and invalid RFC3339 timestamp query params.
- [ ] Write a repository test proving timestamp range filters include boundary events.
- [ ] Verify the Go tests fail before implementation.
- [ ] Add timestamp fields, parse query params, and apply inclusive database filters.
- [ ] Run `go test ./internal/auditlog/httpapi/internal/mapconv ./internal/auditlog/persistence -count=1`.

### Task 4: Final Verification

- [ ] Run focused frontend and backend tests.
- [ ] Run lints/diagnostics for edited files.
- [ ] Summarize behavior and any remaining risk.

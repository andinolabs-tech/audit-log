# Date Range Filter — Design Spec

**Date:** 2026-04-30  
**Status:** Approved

## Overview

Add a date range filter to the audit log query UI that lets users narrow events by timestamp using quick presets (Today, Yesterday, This week, Last week, This month, Last month) or a custom from/to date pair. The backend query pipeline must also be extended to accept and apply timestamp bounds.

---

## 1. Filter Bar Layout

The current 2-column filter grid at the `lg` breakpoint:

```
[16rem  minmax(0,1fr)]
Namespace | Search
```

Becomes a 3-column grid:

```
[16rem  16rem  minmax(0,1fr)]
Namespace | Date Range | Search
```

Below `lg` all three controls stack vertically (existing single-column collapse behaviour is unchanged). When Custom is selected the two date inputs appear below the date dropdown in the same column, expanding the card height on demand.

---

## 2. Frontend — `DateRangeFilter` Component

**File:** `web/src/components/DateRangeFilter.tsx`

### Types (`web/src/dateRange.ts`)

```ts
export type DatePreset =
  | 'today'
  | 'yesterday'
  | 'this_week'
  | 'last_week'
  | 'this_month'
  | 'last_month'
  | 'custom'

export type DateRangeValue =
  | { preset: Exclude<DatePreset, 'custom'> }
  | { preset: 'custom'; from: string; to: string } // YYYY-MM-DD strings
```

### Component props

```ts
interface DateRangeFilterProps {
  value: DateRangeValue | null
  onChange: (value: DateRangeValue | null) => void
}
```

### Behaviour

- Renders a labeled dropdown button (`Date range` label, matching the `Namespaces` label style).
- Button shows the active preset label or "Any date" when value is null.
- Clicking the button toggles the dropdown menu.
- The menu lists all 7 options; the active selection shows a checkmark.
- Selecting a preset (non-custom) closes the menu and calls `onChange`.
- Selecting Custom closes the menu and reveals two `<input type="date">` fields below the button: "From" and "To". Both fields must be non-empty for the custom range to be sent to the API; if either is blank the date filter is omitted from the request entirely.
- A click-outside handler closes the menu (same pattern as the namespace picker).

### QueryPage state

```ts
const [dateRange, setDateRange] = useState<DateRangeValue | null>(null)
```

`buildUrl` is extended to append `timestamp_from` and `timestamp_to` (RFC3339) when `dateRange` is non-null (resolved via `resolveDateRange` at call time).

---

## 3. Date Range Computation (`web/src/dateRange.ts`)

Pure function `resolveDateRange(value: DateRangeValue): { from: Date; to: Date }` computed at search time (not at selection time) so presets always reflect the current moment.

| Preset | `from` | `to` |
|---|---|---|
| Today | start of today 00:00:00 local | end of today 23:59:59 local |
| Yesterday | start of yesterday 00:00:00 local | end of yesterday 23:59:59 local |
| This week | preceding Monday 00:00:00 local | end of today 23:59:59 local |
| Last week | Monday of last week 00:00:00 local | Sunday of last week 23:59:59 local |
| This month | 1st of current month 00:00:00 local | end of today 23:59:59 local |
| Last month | 1st of last month 00:00:00 local | last day of last month 23:59:59 local |
| Custom | `from` date 00:00:00 local | `to` date 23:59:59 local |

Week boundaries: Monday = day 1 (ISO week). Boundary times are computed in the **browser's local timezone** (e.g. "start of today" = local midnight), then serialized via `Date.toISOString()` which outputs UTC RFC3339. The server compares stored UTC timestamps against these UTC bounds, so the effective filter window correctly reflects the user's local day.

---

## 4. Backend Changes

### 4a. `internal/auditlog/usecases/repository_port.go`

Add to `QueryEventsOptions`:

```go
TimestampFrom *time.Time
TimestampTo   *time.Time
```

### 4b. `internal/auditlog/httpapi/internal/mapconv/mapconv.go`

In `QueryParamsToOpts`, parse the new params:

```go
if v := q.Get("timestamp_from"); v != "" {
    t, err := time.Parse(time.RFC3339, v)
    if err != nil {
        return usecases.QueryEventsOptions{}, fmt.Errorf("invalid timestamp_from: must be RFC3339")
    }
    opts.TimestampFrom = &t
}
if v := q.Get("timestamp_to"); v != "" {
    t, err := time.Parse(time.RFC3339, v)
    if err != nil {
        return usecases.QueryEventsOptions{}, fmt.Errorf("invalid timestamp_to: must be RFC3339")
    }
    opts.TimestampTo = &t
}
```

### 4c. `internal/auditlog/persistence/event_repository.go`

In `applyQueryFilters`, append timestamp conditions:

```go
if opts.TimestampFrom != nil {
    q = q.Where("timestamp >= ?", *opts.TimestampFrom)
}
if opts.TimestampTo != nil {
    q = q.Where("timestamp <= ?", *opts.TimestampTo)
}
```

The `timestamp` column already has an index (`idx_audit_events_timestamp_partial`), so range queries are efficient.

---

## 5. Files Changed

| File | Change |
|---|---|
| `web/src/dateRange.ts` | New — `DatePreset`, `DateRangeValue` types, `resolveDateRange` function |
| `web/src/components/DateRangeFilter.tsx` | New — dropdown + custom date inputs component |
| `web/src/pages/QueryPage.tsx` | Add `dateRange` state, extend grid to 3 cols, wire `buildUrl` |
| `internal/auditlog/usecases/repository_port.go` | Add `TimestampFrom`, `TimestampTo` to `QueryEventsOptions` |
| `internal/auditlog/httpapi/internal/mapconv/mapconv.go` | Parse `timestamp_from` / `timestamp_to` query params |
| `internal/auditlog/persistence/event_repository.go` | Add timestamp range conditions to `applyQueryFilters` |

---

## 6. Out of Scope

- Server-side timezone handling (client sends UTC-adjusted RFC3339; server stores/compares in UTC).
- `This month` / `Last month` do not require backend changes beyond what is already specified.
- No changes to the gRPC API or proto definitions.

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
		opts.Namespaces = append([]string(nil), ns...)
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
	if v := q.Get("source_ip"); v != "" {
		opts.SourceIP = &v
	}
	if v := q.Get("session_id"); v != "" {
		opts.SessionID = &v
	}
	if v := q.Get("correlation_id"); v != "" {
		opts.CorrelationID = &v
	}
	if v := q.Get("trace_id"); v != "" {
		opts.TraceID = &v
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
		Tags:          append([]string(nil), e.Tags...),
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

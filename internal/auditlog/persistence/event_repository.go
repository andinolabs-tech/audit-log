package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"audit-log/internal/auditlog/domain"
	"audit-log/internal/auditlog/persistence/internal"
	"audit-log/internal/auditlog/usecases"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

func marshalJSONMap(m map[string]any) (datatypes.JSON, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func unmarshalJSONMap(j datatypes.JSON) (map[string]any, error) {
	if len(j) == 0 {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func toRecord(e *domain.AuditEvent) (*internal.AuditEventRecord, error) {
	r := &internal.AuditEventRecord{
		ID:             e.ID,
		TenantID:       e.TenantID,
		Namespace:      e.Namespace,
		ActorID:        e.ActorID,
		ActorType:      string(e.ActorType),
		EntityType:     e.EntityType,
		EntityID:       e.EntityID,
		Action:         string(e.Action),
		Outcome:        string(e.Outcome),
		ServiceName:    e.ServiceName,
		SourceIP:       e.SourceIP,
		SessionID:      e.SessionID,
		CorrelationID:  e.CorrelationID,
		TraceID:        e.TraceID,
		OccurredAt:     e.OccurredAt,
		Timestamp:      e.Timestamp,
		CompensatesID:  e.CompensatesID,
		Reason:         e.Reason,
	}
	var err error
	if r.Before, err = marshalJSONMap(e.Before); err != nil {
		return nil, err
	}
	if r.After, err = marshalJSONMap(e.After); err != nil {
		return nil, err
	}
	if r.Diff, err = marshalJSONMap(e.Diff); err != nil {
		return nil, err
	}
	if r.Metadata, err = marshalJSONMap(e.Metadata); err != nil {
		return nil, err
	}
	if e.Tags != nil {
		b, err := json.Marshal(e.Tags)
		if err != nil {
			return nil, err
		}
		r.Tags = b
	}
	return r, nil
}

func toDomain(r *internal.AuditEventRecord) (*domain.AuditEvent, error) {
	e := &domain.AuditEvent{
		ID:             r.ID,
		TenantID:       r.TenantID,
		Namespace:      r.Namespace,
		ActorID:        r.ActorID,
		ActorType:      domain.ActorType(r.ActorType),
		EntityType:     r.EntityType,
		EntityID:       r.EntityID,
		Action:         domain.Action(r.Action),
		Outcome:        domain.Outcome(r.Outcome),
		ServiceName:    r.ServiceName,
		SourceIP:       r.SourceIP,
		SessionID:      r.SessionID,
		CorrelationID:  r.CorrelationID,
		TraceID:        r.TraceID,
		OccurredAt:     r.OccurredAt,
		Timestamp:      r.Timestamp,
		CompensatesID:  r.CompensatesID,
		Reason:         r.Reason,
	}
	var err error
	if e.Before, err = unmarshalJSONMap(r.Before); err != nil {
		return nil, err
	}
	if e.After, err = unmarshalJSONMap(r.After); err != nil {
		return nil, err
	}
	if e.Diff, err = unmarshalJSONMap(r.Diff); err != nil {
		return nil, err
	}
	if e.Metadata, err = unmarshalJSONMap(r.Metadata); err != nil {
		return nil, err
	}
	if len(r.Tags) > 0 {
		if err := json.Unmarshal(r.Tags, &e.Tags); err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (r *EventRepository) Save(ctx context.Context, event *domain.AuditEvent) error {
	rec, err := toRecord(event)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *EventRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error) {
	var rec internal.AuditEventRecord
	tx := r.db.WithContext(ctx).Where("id = ?", id).First(&rec)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		return nil, tx.Error
	}
	return toDomain(&rec)
}

func (r *EventRepository) Query(ctx context.Context, opts usecases.QueryEventsOptions) ([]*domain.AuditEvent, error) {
	q := r.db.WithContext(ctx).Model(&internal.AuditEventRecord{}).Order("id ASC")
	q = applyQueryFilters(q, opts)
	if opts.PageToken != nil {
		q = q.Where("id > ?", *opts.PageToken)
	}
	if opts.PageSize > 0 {
		q = q.Limit(opts.PageSize)
	}
	var rows []internal.AuditEventRecord
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.AuditEvent, 0, len(rows))
	for i := range rows {
		ev, err := toDomain(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

func applyQueryFilters(q *gorm.DB, opts usecases.QueryEventsOptions) *gorm.DB {
	if opts.TenantID != nil {
		q = q.Where("tenant_id = ?", *opts.TenantID)
	}
	if opts.Namespace != nil {
		q = q.Where("namespace = ?", *opts.Namespace)
	}
	if opts.ActorID != nil {
		q = q.Where("actor_id = ?", *opts.ActorID)
	}
	if opts.ActorType != nil {
		q = q.Where("actor_type = ?", string(*opts.ActorType))
	}
	if opts.EntityType != nil {
		q = q.Where("entity_type = ?", *opts.EntityType)
	}
	if opts.EntityID != nil {
		q = q.Where("entity_id = ?", *opts.EntityID)
	}
	if opts.Action != nil {
		q = q.Where("action = ?", string(*opts.Action))
	}
	if opts.Outcome != nil {
		q = q.Where("outcome = ?", string(*opts.Outcome))
	}
	if opts.ServiceName != nil {
		q = q.Where("service_name = ?", *opts.ServiceName)
	}
	if opts.SourceIP != nil {
		q = q.Where("source_ip = ?", *opts.SourceIP)
	}
	if opts.SessionID != nil {
		q = q.Where("session_id = ?", *opts.SessionID)
	}
	if opts.CorrelationID != nil {
		q = q.Where("correlation_id = ?", *opts.CorrelationID)
	}
	if opts.TraceID != nil {
		q = q.Where("trace_id = ?", *opts.TraceID)
	}
	return q
}

// AutoMigrateModel runs schema migration for audit events (admin connection).
func AutoMigrateModel(db *gorm.DB) error {
	return db.AutoMigrate(&internal.AuditEventRecord{})
}

// BootstrapSQL runs role and index DDL (admin connection). Idempotent guards are in SQL.
func BootstrapSQL(db *gorm.DB) error {
	stmts := []string{
		`REVOKE UPDATE, DELETE ON audit_events FROM PUBLIC`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_before_gin ON audit_events USING GIN ((before::jsonb))`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_after_gin ON audit_events USING GIN ((after::jsonb))`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_diff_gin ON audit_events USING GIN ((diff::jsonb))`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_metadata_gin ON audit_events USING GIN ((metadata::jsonb))`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_partial ON audit_events (tenant_id) WHERE tenant_id IS NOT NULL AND tenant_id <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_actor_partial ON audit_events (actor_id) WHERE actor_id IS NOT NULL AND actor_id <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_entity_type_partial ON audit_events (entity_type) WHERE entity_type IS NOT NULL AND entity_type <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_namespace_partial ON audit_events (namespace) WHERE namespace IS NOT NULL AND namespace <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_action_partial ON audit_events (action) WHERE action IS NOT NULL AND action <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp_partial ON audit_events ("timestamp")`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return fmt.Errorf("bootstrap: %q: %w", s, err)
		}
	}
	return nil
}

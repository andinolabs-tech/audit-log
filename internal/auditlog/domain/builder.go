package domain

import (
	"time"

	"github.com/google/uuid"
)

type auditEventBuilderHandler func(*AuditEvent) error

type AuditEventBuilder struct {
	actions []auditEventBuilderHandler
}

func NewAuditEventBuilder() *AuditEventBuilder {
	return &AuditEventBuilder{}
}

func (b *AuditEventBuilder) WithTenantID(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.TenantID = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithNamespace(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.Namespace = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithActorID(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.ActorID = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithActorType(v ActorType) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.ActorType = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithEntityType(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.EntityType = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithEntityID(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.EntityID = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithAction(v Action) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.Action = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithOutcome(v Outcome) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.Outcome = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithServiceName(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.ServiceName = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithSourceIP(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.SourceIP = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithSessionID(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.SessionID = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithCorrelationID(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.CorrelationID = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithTraceID(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.TraceID = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithOccurredAt(v time.Time) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		t := v
		e.OccurredAt = &t
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithCompensatesID(v uuid.UUID) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		id := v
		e.CompensatesID = &id
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithBefore(v map[string]any) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.Before = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithAfter(v map[string]any) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.After = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithMetadata(v map[string]any) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.Metadata = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithReason(v string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.Reason = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) WithTags(v []string) *AuditEventBuilder {
	b.actions = append(b.actions, func(e *AuditEvent) error {
		e.Tags = v
		return nil
	})
	return b
}

func (b *AuditEventBuilder) Build() (*AuditEvent, error) {
	var e AuditEvent
	for _, fn := range b.actions {
		if err := fn(&e); err != nil {
			return nil, err
		}
	}
	if e.TenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if e.Namespace == "" {
		return nil, ErrNamespaceRequired
	}
	if e.EntityType == "" {
		return nil, ErrEntityTypeRequired
	}
	if e.EntityID == "" {
		return nil, ErrEntityIDRequired
	}
	if e.Action == ActionCompensated && e.CompensatesID == nil {
		return nil, ErrCompensatesIDRequired
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	e.ID = id
	e.Timestamp = time.Now().UTC()
	return &e, nil
}

package internal

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type AuditEventRecord struct {
	ID             uuid.UUID `gorm:"primaryKey"`
	TenantID       string    `gorm:"not null"`
	ActorID        string    `gorm:"not null"`
	ActorType      string    `gorm:"not null"`
	EntityType     string    `gorm:"not null"`
	EntityID       string    `gorm:"not null"`
	Action         string    `gorm:"not null"`
	Outcome        string    `gorm:"not null"`
	ServiceName    string    `gorm:"not null"`
	SourceIP       string
	SessionID      string
	CorrelationID  string
	TraceID        string
	Timestamp      time.Time `gorm:"not null"`
	CompensatesID  *uuid.UUID
	Before         datatypes.JSON
	After          datatypes.JSON
	Diff           datatypes.JSON
	Metadata       datatypes.JSON
	Reason         string
	Tags           datatypes.JSON `gorm:"type:jsonb"`
}

func (AuditEventRecord) TableName() string {
	return "audit_events"
}

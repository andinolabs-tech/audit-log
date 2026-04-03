package domain

import "errors"

var (
	ErrCompensatesIDRequired = errors.New("compensates_id is required when action is COMPENSATED")
	ErrTenantIDRequired      = errors.New("tenant_id is required")
	ErrNamespaceRequired     = errors.New("namespace is required")
	ErrEntityTypeRequired    = errors.New("entity_type is required")
	ErrEntityIDRequired      = errors.New("entity_id is required")
)

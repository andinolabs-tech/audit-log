package usecases

import "errors"

var (
	ErrReferencedEventNotFound = errors.New("referenced event not found")
	ErrTenantMismatch          = errors.New("compensated event tenant does not match referenced event")
	ErrInvalidPageSize         = errors.New("page size must be between 1 and 500")
)

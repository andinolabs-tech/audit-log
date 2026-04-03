package domain

import "errors"

var ErrCompensatesIDRequired = errors.New("compensates_id is required when action is COMPENSATED")

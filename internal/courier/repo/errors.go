package repo

import "errors"

// ErrNotFound indicates missing entities in the courier repositories.
var ErrNotFound = errors.New("courier: not found")

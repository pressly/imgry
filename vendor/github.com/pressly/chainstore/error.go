package chainstore

import (
	"errors"
)

// Public error messages.
var (
	ErrInvalidKey    = errors.New("Invalid key")
	ErrMissingStores = errors.New("No stores provided")
	ErrNoSuchKey     = errors.New("No such key")
	ErrTimeout       = errors.New("Timed out")
)

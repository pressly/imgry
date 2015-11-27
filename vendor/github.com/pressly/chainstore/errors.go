package chainstore

import (
	"errors"
)

// Public error messages.
var (
	ErrInvalidKey    = errors.New("Invalid key")
	ErrMissingStores = errors.New("No stores provided")
	ErrNoSuchKey     = errors.New("No such key")
)

type fewerrors []error

func (es fewerrors) Error() string {
	var msg string
	if len(es) > 0 {
		for i, e := range es {
			msg += e.Error()
			if i+1 < len(es) {
				msg += ", "
			}
		}
	}
	return msg
}

package caches

import (
	"errors"
	"fmt"
)

type ValidationError struct {
	// Err    error
	Reason string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("creation of cache failed for reason : %s ", ve.Reason)

}

// TODO : This should be a more specific error
var ErrValidation = errors.New("validation error")

package application

import (
	"context"
	"errors"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
	"github.com/google/uuid"
)

var ErrAttemptPolicyChanged = errors.New("identity: attempt policy changed")
var ErrAttemptStorage = errors.New("identity: attempt storage failed")

// AttemptKey identifies one budget. Only trusted server code may construct
// the opaque digest, never a request parameter. Unknown identities need
// their own bounded admission policy before this port is exposed.
type AttemptKey struct {
	OrganizationID uuid.UUID
	Kind           string
	Digest         [32]byte
}

type AttemptStore interface {
	// ReserveAttempt returns true only after the reservation commits.
	ReserveAttempt(context.Context, AttemptKey, domain.AttemptPolicy) (bool, error)
}

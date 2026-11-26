package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrCredentialConflict means a verifier was replaced since it was read.
var ErrCredentialConflict = errors.New("identity: credential changed")
var ErrCredentialStorage = errors.New("identity: credential storage failed")

// PasswordStore is an internal persistence port. Callers must authorize
// enrollment/rotation and validate cryptographic envelopes before writing.
// Never expose its values through an API or log them.
type PasswordStore interface {
	LoadPassword(ctx context.Context, org, user uuid.UUID) (string, error)
	InsertPassword(ctx context.Context, org, user uuid.UUID, verifier string) error
	ReplacePassword(ctx context.Context, org, user uuid.UUID, expected, replacement string) error
}

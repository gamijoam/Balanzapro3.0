// Package secretfile loads an operator-provisioned, dedicated binary key.
// It never creates, logs, replaces or rotates a secret.
package secretfile

import "errors"

var ErrUnavailable = errors.New("server key unavailable or invalid")

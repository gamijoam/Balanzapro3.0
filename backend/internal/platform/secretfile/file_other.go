//go:build !linux

package secretfile

// Fail closed on platforms whose file access controls have not been audited.
func ReadKey(string) ([32]byte, error) { return [32]byte{}, ErrUnavailable }

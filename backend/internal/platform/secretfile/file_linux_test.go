//go:build linux

package secretfile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/secretfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadDedicatedKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture")
	key := [32]byte{17, 24}
	require.NoError(t, os.WriteFile(path, key[:], 0600))
	got, err := secretfile.ReadKey(path)
	require.NoError(t, err)
	assert.Equal(t, key, got)
	require.NoError(t, os.Chmod(path, 0400))
	got, err = secretfile.ReadKey(path)
	require.NoError(t, err)
	assert.Equal(t, key, got)
	require.NoError(t, os.Chmod(path, 0644))
	got, err = secretfile.ReadKey(path)
	assert.ErrorIs(t, err, secretfile.ErrUnavailable)
	assert.Equal(t, [32]byte{}, got)
	require.NoError(t, os.Chmod(path, 0600))
	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink(path, link))
	for _, invalid := range []string{link, dir, filepath.Join(dir, "absent"), "relative"} {
		got, err = secretfile.ReadKey(invalid)
		assert.ErrorIs(t, err, secretfile.ErrUnavailable)
		assert.Equal(t, [32]byte{}, got)
	}
	for _, bad := range [][]byte{nil, make([]byte, 31), make([]byte, 32), make([]byte, 33)} {
		require.NoError(t, os.WriteFile(path, bad, 0600))
		got, err = secretfile.ReadKey(path)
		assert.ErrorIs(t, err, secretfile.ErrUnavailable)
		assert.Equal(t, [32]byte{}, got)
	}
}

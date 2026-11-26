package argon2id

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestPasswordVerifier(t *testing.T) {
	p := []byte("synthetic phrase for unit tests")
	a, err := Hash(p)
	require.NoError(t, err)
	b, err := Hash(p)
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	assert.True(t, Verify(p, a))
	assert.True(t, Verify(p, b))
	assert.False(t, Verify([]byte("wrong"), a))
	assert.NotContains(t, a, string(p))
	_, err = Hash(nil)
	assert.Error(t, err)
	_, err = Hash(make([]byte, 1025))
	assert.Error(t, err)
	assert.False(t, Verify(make([]byte, 1025), a))
	for _, bad := range []string{"", a + "$", strings.Replace(a, "m=65536", "m=4294967295", 1), strings.Replace(a, "t=3", "t=0", 1), strings.Replace(a, "p=1", "p=255", 1), strings.Replace(a, "v=19", "v=16", 1), strings.Replace(a, "argon2id", "argon2i", 1), strings.Repeat("x", 513), strings.Replace(a, "m=65536", "m=065536", 1)} {
		assert.False(t, Verify(p, bad))
	}
}

func BenchmarkVerify(b *testing.B) {
	p := []byte("synthetic benchmark phrase")
	h, err := Hash(p)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !Verify(p, h) {
			b.Fatal("verify failed")
		}
	}
}

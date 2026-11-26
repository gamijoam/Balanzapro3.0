//go:build linux

package secretfile

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// ReadKey requires an absolute path in an operator-controlled directory.
// The last component cannot be a symlink; the open descriptor is validated
// before reading. Parent directory trust is a deployment requirement.
func ReadKey(path string) ([32]byte, error) {
	var key [32]byte
	if !filepath.IsAbs(path) {
		return key, ErrUnavailable
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return key, ErrUnavailable
	}
	f := os.NewFile(uintptr(fd), "server key")
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return key, ErrUnavailable
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || st.Uid != uint32(os.Geteuid()) ||
		st.Nlink != 1 || (info.Mode().Perm() != 0400 && info.Mode().Perm() != 0600) ||
		info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || info.Size() != 32 {
		return key, ErrUnavailable
	}
	var buf [33]byte
	defer clear(buf[:])
	n, err := io.ReadFull(f, buf[:])
	if n != 32 || err != io.ErrUnexpectedEOF {
		return key, ErrUnavailable
	}
	copy(key[:], buf[:32])
	if key == ([32]byte{}) {
		return [32]byte{}, ErrUnavailable
	}
	return key, nil
}

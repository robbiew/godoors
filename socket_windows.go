//go:build windows

package godoors

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ioctlSocket = windows.NewLazySystemDLL("ws2_32.dll").NewProc("ioctlsocket")

// This deliberately implements io.ReadWriteCloser rather than promising
// net.Conn deadlines. Probe deadlines are provided by the shared input pump.
// Nonblocking operations let Close cancel pending I/O without shutdown(),
// which would disconnect the socket still owned by the parent BBS.
type windowsSocket struct {
	writeMu sync.Mutex
	mu      sync.Mutex
	handle  windows.Handle
	closed  bool
	done    chan struct{}
}

func platformConnection(info DropFile) (io.ReadWriteCloser, error) {
	if info.CommType == 1 {
		return nil, fmt.Errorf("godoors: Windows serial doors are not supported")
	}
	if info.CommType != 2 {
		return nil, nil
	}
	if info.Handle == 0 || info.Handle == ^uintptr(0) {
		return nil, fmt.Errorf("godoors: invalid socket handle")
	}
	var data windows.WSAData
	if err := windows.WSAStartup(0x202, &data); err != nil {
		return nil, err
	}
	h := windows.Handle(info.Handle)
	if _, err := windows.Getpeername(h); err != nil {
		windows.WSACleanup()
		return nil, fmt.Errorf("godoors: inherited socket: %w", err)
	}
	mode := uint32(1)
	ret, _, err := ioctlSocket.Call(uintptr(h), 0x8004667e, uintptr(unsafe.Pointer(&mode)))
	if ret != 0 {
		windows.WSACleanup()
		return nil, fmt.Errorf("godoors: set nonblocking socket: %w", err)
	}
	return &windowsSocket{handle: h, done: make(chan struct{})}, nil
}

func (s *windowsSocket) transfer(p []byte, write bool) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	// WSABuf lengths are uint32; bound each operation on 64-bit hosts as well.
	if len(p) > 1<<30 {
		p = p[:1<<30]
	}
	for {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return 0, net.ErrClosed
		}
		buf := windows.WSABuf{Len: uint32(len(p)), Buf: &p[0]}
		var n, flags uint32
		var err error
		if write {
			err = windows.WSASend(s.handle, &buf, 1, &n, 0, nil, nil)
		} else {
			err = windows.WSARecv(s.handle, &buf, 1, &n, &flags, nil, nil)
		}
		s.mu.Unlock()
		if err == windows.WSAEWOULDBLOCK {
			select {
			case <-s.done:
				return 0, net.ErrClosed
			case <-time.After(5 * time.Millisecond):
				continue
			}
		}
		if err != nil {
			return int(n), err
		}
		if n == 0 {
			if write {
				return 0, io.ErrNoProgress
			}
			return 0, io.EOF
		}
		return int(n), nil
	}
}
func (s *windowsSocket) Read(p []byte) (int, error) { return s.transfer(p, false) }
func (s *windowsSocket) Write(p []byte) (int, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	n := 0
	for n < len(p) {
		k, err := s.transfer(p[n:], true)
		n += k
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
func (s *windowsSocket) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.done)
	err := windows.Closesocket(s.handle)
	cleanupErr := windows.WSACleanup()
	if err != nil {
		return err
	}
	return cleanupErr
}

func sessionDropFile(path string) (DropFile, error) { return ReadDropFile(path) }

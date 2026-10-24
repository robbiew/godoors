//go:build windows

package godoors

import (
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var duplicateSocket = windows.NewLazySystemDLL("ws2_32.dll").NewProc("WSADuplicateSocketW")

// Use a private overlapped descriptor without changing shared socket mode or
// event registrations. Only our operations are cancelled; never shutdown().
type windowsSocket struct {
	writeMu    sync.Mutex
	closeMu    sync.Mutex
	mu         sync.Mutex
	operations sync.WaitGroup
	handle     windows.Handle
	inherited  windows.Handle
	closing    bool
	closed     bool
	done       windows.Handle
	// Per-instance seam for exercising retryable closes without process globals.
	closeHandle func(windows.Handle) error
}

func platformConnection(info DropFile) (io.ReadWriteCloser, error) {
	if info.CommType == 1 {
		return nil, fmt.Errorf("godoors: Windows serial doors are not supported")
	}
	if info.CommType != 2 || info.Handle == 0 {
		return nil, nil
	}
	if info.Handle == ^uintptr(0) {
		return nil, fmt.Errorf("godoors: invalid socket handle")
	}
	var data windows.WSAData
	if err := windows.WSAStartup(0x202, &data); err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			windows.WSACleanup()
		}
	}()
	h := windows.Handle(info.Handle)
	if _, err := windows.Getpeername(h); err != nil {
		if err == windows.WSAENOTSOCK {
			return nil, nil
		}
		return nil, fmt.Errorf("godoors: inherited socket: %w", err)
	}
	// An existing local pump cannot be cancelled safely without closing the
	// caller-owned stdin. Keep its buffered keys and outstanding read intact.
	if stdinPump != nil && stdinPump.reqs != nil {
		return nil, fmt.Errorf("godoors: initialize socket session before reading or probing stdin")
	}
	var protocol windows.WSAProtocolInfo
	ret, _, err := duplicateSocket.Call(uintptr(h), uintptr(windows.GetCurrentProcessId()), uintptr(unsafe.Pointer(&protocol)))
	if ret != 0 {
		return nil, fmt.Errorf("godoors: duplicate socket: %w", err)
	}
	private, err := windows.WSASocket(-1, -1, -1, &protocol, 0, windows.WSA_FLAG_OVERLAPPED|windows.WSA_FLAG_NO_HANDLE_INHERIT)
	if err != nil {
		return nil, err
	}
	done, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		windows.Closesocket(private)
		return nil, err
	}
	success = true
	return &windowsSocket{handle: private, inherited: h, done: done, closeHandle: windows.Closesocket}, nil
}

func (s *windowsSocket) transfer(p []byte, write bool) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(p) > 1<<30 {
		p = p[:1<<30]
	}
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(event)
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return 0, net.ErrClosed
	}
	s.operations.Add(1)
	s.mu.Unlock()
	defer s.operations.Done()
	// Winsock cancels pending operations when the issuing OS thread exits.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ov := windows.Overlapped{HEvent: event}
	buf := windows.WSABuf{Len: uint32(len(p)), Buf: &p[0]}
	var n, flags uint32
	if write {
		err = windows.WSASend(s.handle, &buf, 1, &n, 0, &ov, nil)
	} else {
		err = windows.WSARecv(s.handle, &buf, 1, &n, &flags, &ov, nil)
	}
	if err == windows.ERROR_IO_PENDING {
		var waitErr error
		result, e := windows.WaitForMultipleObjects([]windows.Handle{s.done, event}, false, windows.INFINITE)
		if e != nil {
			waitErr = e
		}
		if e != nil || result == windows.WAIT_OBJECT_0 {
			windows.CancelIoEx(s.handle, &ov)
		}
		// Even after cancellation, keep the buffer and OVERLAPPED alive until the
		// kernel has finished with them. CancelIoEx alone does not guarantee that.
		err = windows.WSAGetOverlappedResult(s.handle, &ov, &n, true, &flags)
		if waitErr != nil && err == nil {
			err = waitErr
		}
	}
	runtime.KeepAlive(p)
	if err != nil {
		s.mu.Lock()
		closing := s.closing
		s.mu.Unlock()
		if closing {
			return int(n), net.ErrClosed
		}
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
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	if !s.closing {
		if err := windows.SetEvent(s.done); err != nil {
			s.mu.Unlock()
			return err
		}
		s.closing = true
	}
	s.mu.Unlock()
	s.operations.Wait()
	// Retain descriptors and the Winsock reference on failure so Close can be
	// retried. New I/O stays cancelled during that retry interval.
	if s.handle != windows.InvalidHandle {
		if err := s.closeHandle(s.handle); err != nil {
			return err
		}
		s.handle = windows.InvalidHandle
	}
	if s.inherited != windows.InvalidHandle {
		if err := s.closeHandle(s.inherited); err != nil {
			return err
		}
		s.inherited = windows.InvalidHandle
	}
	if err := windows.WSACleanup(); err != nil {
		return err
	}
	windows.CloseHandle(s.done)
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return nil
}
func sessionDropFile(path string) (DropFile, error) { return ReadDropFile(path) }

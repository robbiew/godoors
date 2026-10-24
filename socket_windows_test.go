//go:build windows

package godoors

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Exercise an actual Winsock handle, independently of Go's net.Conn poller.
func socketPair(t *testing.T) (io.ReadWriteCloser, net.Conn) {
	t.Helper()
	var data windows.WSAData
	if err := windows.WSAStartup(0x202, &data); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { windows.WSACleanup() })
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	h, err := windows.Socket(windows.AF_INET, windows.SOCK_STREAM, windows.IPPROTO_TCP)
	if err != nil {
		t.Fatal(err)
	}
	owned := true
	defer func() {
		if owned {
			windows.Closesocket(h)
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := windows.Connect(h, &windows.SockaddrInet4{Port: port, Addr: [4]byte{127, 0, 0, 1}}); err != nil {
		t.Fatal(err)
	}
	peer, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	peer.SetDeadline(time.Now().Add(5 * time.Second))
	t.Cleanup(func() { peer.Close() })
	conn, err := platformConnection(DropFile{CommType: 2, Handle: uintptr(h)})
	if err != nil {
		t.Fatal(err)
	}
	owned = false
	t.Cleanup(func() { conn.Close() })
	return conn, peer
}

func TestWindowsSocketRoundTripAndEOF(t *testing.T) {
	conn, peer := socketPair(t)
	payload := bytes.Repeat([]byte{0xdb, 0xdc, 0x1b, '[', '0', 'm'}, 1<<18)
	received := make(chan error, 1)
	go func() {
		b := make([]byte, len(payload))
		_, err := io.ReadFull(peer, b)
		if err == nil && !bytes.Equal(b, payload) {
			err = errors.New("corrupted output")
		}
		received <- err
	}()
	n, err := conn.Write(payload)
	if err != nil || n != len(payload) {
		t.Fatalf("write %d: %v", n, err)
	}
	if err := <-received; err != nil {
		t.Fatal(err)
	}
	if _, err := peer.Write([]byte("\x1b[Aq")); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 4)
	if _, err := io.ReadFull(conn, b); err != nil || string(b) != "\x1b[Aq" {
		t.Fatalf("input %q: %v", b, err)
	}
	peer.Close()
	if _, err := conn.Read(b); err != io.EOF {
		t.Fatalf("want EOF, got %v", err)
	}
}

func TestWindowsSocketCloseCancelsRead(t *testing.T) {
	conn, _ := socketPair(t)
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() { close(started); _, err := conn.Read(make([]byte, 1)); done <- err }()
	<-started
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("close did not cancel read")
	}
	if _, err := conn.Write([]byte("x")); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("closed write: %v", err)
	}
}

func TestWindowsSocketRejectsInvalidTransport(t *testing.T) {
	for _, info := range []DropFile{{CommType: 1}, {CommType: 2, Handle: ^uintptr(0)}} {
		if _, err := platformConnection(info); err == nil {
			t.Fatalf("accepted %+v", info)
		}
	}
}

// The child receives the same inherited socket mechanism used by a BBS.
func TestWindowsInheritedSocketChild(t *testing.T) {
	raw := os.Getenv("GODOORS_TEST_SOCKET")
	if raw == "" {
		t.Skip("subprocess helper")
	}
	handle, err := strconv.ParseUint(raw, 10, strconv.IntSize)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	content := fmt.Sprintf("2\r\n%d\r\n57600\r\nTest BBS\r\n1\r\nReal Name\r\nCaller\r\n255\r\n30\r\n1\r\n1\r\n", handle)
	if err := os.WriteFile(filepath.Join(dir, "DOOR32.SYS"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	probeTimeout = 20 * time.Millisecond
	u, err := Initialize(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer CloseSession()
	if u.Alias != "Caller" {
		t.Fatalf("user: %+v", u)
	}
	ch, _, err := GetKey()
	if err != nil || ch != 'q' {
		t.Fatalf("key %q: %v", ch, err)
	}
	if _, err := fmt.Fprint(Output(), "child"); err != nil {
		t.Fatal(err)
	}
	if err := CloseSession(); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsInheritedSocketReturnsToParent(t *testing.T) {
	conn, peer := socketPair(t)
	h := conn.(*windowsSocket).inherited
	if err := windows.SetHandleInformation(h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestWindowsInheritedSocketChild$", "-test.timeout=10s")
	cmd.Env = append(os.Environ(), "GODOORS_TEST_SOCKET="+strconv.FormatUint(uint64(h), 10))
	cmd.SysProcAttr = &syscall.SysProcAttr{AdditionalInheritedHandles: []syscall.Handle{syscall.Handle(h)}}
	if _, err := peer.Write([]byte("q")); err != nil {
		t.Fatal(err)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child: %v: %s", err, out)
	}
	want := "\x1b[999;999f\x1b[6n\x1b[cchild"
	b := make([]byte, len(want))
	if _, err := io.ReadFull(peer, b); err != nil || string(b) != want {
		t.Fatalf("child output %q: %v", b, err)
	}
	if _, err := conn.Write([]byte("parent")); err != nil {
		t.Fatal(err)
	}
	b = make([]byte, 6)
	if _, err := io.ReadFull(peer, b); err != nil || string(b) != "parent" {
		t.Fatalf("parent output %q: %v", b, err)
	}
	assertBlockingRead(t, h, peer)
}

func TestWindowsSocketCloseCancelsBlockedWrite(t *testing.T) {
	conn, peer := socketPair(t)
	if err := peer.(*net.TCPConn).SetReadBuffer(1024); err != nil {
		t.Fatal(err)
	}
	h := conn.(*windowsSocket).handle
	// Exercise cancellation with a socket that the host already made nonblocking.
	// This test-only setup must not be needed or performed by the adapter.
	mode := uint32(1)
	ioctl := windows.NewLazySystemDLL("ws2_32.dll").NewProc("ioctlsocket")
	if ret, _, err := ioctl.Call(uintptr(h), 0x8004667e, uintptr(unsafe.Pointer(&mode))); ret != 0 {
		t.Fatal(err)
	}
	if err := windows.SetsockoptInt(h, windows.SOL_SOCKET, windows.SO_SNDBUF, 1024); err != nil {
		t.Fatal(err)
	}

	// Windows loopback can accept megabytes in one send despite SO_SNDBUF.
	// Fill the actual connection until Winsock reports backpressure instead of
	// assuming that a fixed payload size must block on every runner.
	payload := make([]byte, 64<<10)
	deadline := time.Now().Add(5 * time.Second)
	for {
		buf := windows.WSABuf{Len: uint32(len(payload)), Buf: &payload[0]}
		var n uint32
		err := windows.WSASend(h, &buf, 1, &n, 0, nil, nil)
		if err == windows.WSAEWOULDBLOCK {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("socket did not reach backpressure")
		}
	}
	done := make(chan error, 1)
	go func() {
		// Keep writing until cancellation, even if Windows frees some buffer
		// space between the saturation check and this goroutine starting.
		for {
			if _, err := conn.Write(payload); err != nil {
				done <- err
				return
			}
		}
	}()
	select {
	case err := <-done:
		t.Fatalf("write ended before close: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("close did not cancel write")
	}
}

func TestWindowsStdioFallback(t *testing.T) {
	conn, err := platformConnection(DropFile{CommType: 2})
	if err != nil || conn != nil {
		t.Fatalf("zero handle fallback: %v %v", conn, err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	conn, err = platformConnection(DropFile{CommType: 2, Handle: r.Fd()})
	if err != nil || conn != nil {
		t.Fatalf("pipe handle fallback: %v %v", conn, err)
	}
}

func TestWindowsSocketCloseRetry(t *testing.T) {
	conn, _ := socketPair(t)
	s := conn.(*windowsSocket)
	attempts := 0
	s.closeHandle = func(h windows.Handle) error {
		attempts++
		if attempts == 1 {
			return windows.WSAEWOULDBLOCK
		}
		return windows.Closesocket(h)
	}
	if err := s.Close(); err != windows.WSAEWOULDBLOCK {
		t.Fatalf("first close: %v", err)
	}
	if s.closed || s.handle == windows.InvalidHandle {
		t.Fatal("lost retryable descriptor")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("close calls: %d", attempts)
	}
}

func TestWindowsSocketPreservesBlockingMode(t *testing.T) {
	conn, peer := socketPair(t)
	assertBlockingRead(t, conn.(*windowsSocket).inherited, peer)
}

func assertBlockingRead(t *testing.T, h windows.Handle, peer net.Conn) {
	t.Helper()

	// Raw non-overlapped recv must wait for input instead of WSAEWOULDBLOCK.
	done := make(chan error, 1)
	go func() {
		b := []byte{0}
		buf := windows.WSABuf{Len: 1, Buf: &b[0]}
		var n, flags uint32
		err := windows.WSARecv(h, &buf, 1, &n, &flags, nil, nil)
		if err == nil && (n != 1 || b[0] != 'x') {
			err = fmt.Errorf("bad raw read: %d %q", n, b)
		}
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("blocking read returned before input: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	peer.Write([]byte("x"))
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("raw read stalled")
	}
}

func TestWindowsSocketInitializationPreservesExistingInput(t *testing.T) {
	conn, _ := socketPair(t)
	pw := fakeStdin(t)
	go pw.Write([]byte("ab"))
	ch, _, err := GetKey()
	if err != nil || ch != 'a' {
		t.Fatalf("first key: %q %v", ch, err)
	}
	next, err := platformConnection(DropFile{CommType: 2, Handle: uintptr(conn.(*windowsSocket).inherited)})
	if err == nil || next != nil {
		t.Fatalf("expected guarded transition, got %v %v", next, err)
	}
	ch, _, err = GetKey()
	if err != nil || ch != 'b' {
		t.Fatalf("lost buffered key: %q %v", ch, err)
	}
}

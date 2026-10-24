package godoors

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type testConnection struct {
	io.Reader
	bytes.Buffer
	closes int
}

func (c *testConnection) Read(p []byte) (int, error) { return c.Reader.Read(p) }
func (c *testConnection) Close() error               { c.closes++; return nil }

func TestSessionRoutesDrawingInputAndProbes(t *testing.T) {
	oldPump, oldReader := stdinPump, stdinReader
	oldKnown, oldCTerm := ctermKnown, ctermDetected
	c := &testConnection{Reader: strings.NewReader("q\x1b[30;100R\x1b[=67;84;101;114;109;1cZ")}
	activeConnection = c
	resetSessionInput()
	t.Cleanup(func() {
		CloseSession()
		stdinPump, stdinReader = oldPump, oldReader
		ctermKnown, ctermDetected = oldKnown, oldCTerm
	})
	h, w := GetTermSize()
	if h != 30 || w != 100 || !IsCTerm() {
		t.Fatalf("remote probe: %d x %d, CTerm=%v", h, w, IsCTerm())
	}
	for _, want := range []rune{'q', 'Z'} {
		ch, _, err := GetKey()
		if err != nil || ch != want {
			t.Fatalf("input got %q, %v; want %q", ch, err, want)
		}
	}
	local := captureStdout(t, func() {
		PrintStringLoc("remote-text", 1, 1)
		PrintAnsi("\xdb\xdc", 0, 25)
		CursorUp(2)
		PlayMusic("C4")
		fmt.Fprint(Output(), "application-text")
	})
	if local != "" {
		t.Fatalf("remote output leaked to stdout: %q", local)
	}
	for _, want := range []string{"\x1b[6n", "remote-text", "\xdb\xdc", "\x1b[2A", MusicSequence("C4"), "application-text"} {
		if !strings.Contains(c.String(), want) {
			t.Errorf("missing output %q", want)
		}
	}
	if err := CloseSession(); err != nil {
		t.Fatal(err)
	}
	CloseSession()
	if c.closes != 1 {
		t.Fatalf("closed %d times", c.closes)
	}
	if Output() != os.Stdout {
		t.Fatal("stdout not restored")
	}
}

func TestCloseSessionCancelsOutstandingProbeRead(t *testing.T) {
	oldPump, oldReader := stdinPump, stdinReader
	oldKnown, oldCTerm := ctermKnown, ctermDetected
	r, w := io.Pipe()
	activeConnection = &pipeConnection{r, w}
	resetSessionInput()
	t.Cleanup(func() {
		CloseSession()
		stdinPump, stdinReader = oldPump, oldReader
		ctermKnown, ctermDetected = oldKnown, oldCTerm
	})
	// Initialize the shared reader, then force a probe timeout with a read pending.
	inputReader()
	pump := stdinPump
	pump.setDeadline(time.Now().Add(10 * time.Millisecond))
	if _, err := pump.Read(make([]byte, 1)); err != errProbeTimeout {
		t.Fatalf("got %v", err)
	}
	CloseSession()
	pump.setDeadline(time.Time{})
	done := make(chan error, 1)
	go func() { _, err := pump.Read(make([]byte, 1)); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected closed read")
		}
	case <-time.After(time.Second):
		t.Fatal("read did not unblock")
	}
}

type pipeConnection struct {
	*io.PipeReader
	writer *io.PipeWriter
}

func (c *pipeConnection) Write(p []byte) (int, error) { return len(p), nil }
func (c *pipeConnection) Close() error                { c.writer.Close(); return c.PipeReader.Close() }

func TestReadDropFileTransport(t *testing.T) {
	for _, handle := range []string{"0", "1234", "-1", "18446744073709551616", "bad"} {
		t.Run(handle, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "DOOR32.SYS"), []byte("2\r\n"+handle+"\r\n57600\r\nMystic\r\n1\r\nReal Name\r\nAlias\r\n255\r\n58\r\n1\r\n2\r\n"), 0600)
			info, err := ReadDropFile(dir)
			valid := handle == "0" || handle == "1234"
			if (err == nil) != valid {
				t.Fatalf("handle %q: %v", handle, err)
			}
			if valid && (info.CommType != 2 || info.Alias != "Alias" || info.Node != 2) {
				t.Fatalf("got %+v", info)
			}
			// Metadata-only legacy callers need not interpret communication fields.
			if _, _, _, _, err := DropFileData(dir); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type retryCloseConnection struct{ attempts int }

func (c *retryCloseConnection) Read(p []byte) (int, error)  { return 0, io.EOF }
func (c *retryCloseConnection) Write(p []byte) (int, error) { return len(p), nil }
func (c *retryCloseConnection) Close() error {
	c.attempts++
	if c.attempts == 1 {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func TestCloseSessionRetainsFailedConnection(t *testing.T) {
	c := &retryCloseConnection{}
	activeConnection = c
	t.Cleanup(func() { CloseSession() })
	if err := CloseSession(); err != io.ErrUnexpectedEOF {
		t.Fatalf("got %v", err)
	}
	if Output() != c {
		t.Fatal("failed close lost connection")
	}
	if err := CloseSession(); err != nil {
		t.Fatal(err)
	}
	if c.attempts != 2 || remoteSession() {
		t.Fatal("close was not retried")
	}
}

func TestOutputConcurrentWithClose(t *testing.T) {
	// Use an EOF-only connection whose close has no shared test state.
	activeConnection = &concurrentTestConnection{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10000; i++ {
			_ = Output()
			_ = remoteSession()
			_ = sessionInput()
		}
	}()
	if err := CloseSession(); err != nil {
		t.Fatal(err)
	}
	<-done
}

type concurrentTestConnection struct{}

func (*concurrentTestConnection) Read(p []byte) (int, error)  { return 0, io.EOF }
func (*concurrentTestConnection) Write(p []byte) (int, error) { return len(p), nil }
func (*concurrentTestConnection) Close() error                { return nil }

type stalledOutput struct {
	done chan struct{}
	once sync.Once
}

func (c *stalledOutput) Read(p []byte) (int, error)  { <-c.done; return 0, io.EOF }
func (c *stalledOutput) Write(p []byte) (int, error) { <-c.done; return 0, io.ErrClosedPipe }
func (c *stalledOutput) Close() error                { c.once.Do(func() { close(c.done) }); return nil }
func TestIdleExitDoesNotWaitForStalledOutput(t *testing.T) {
	c := &stalledOutput{done: make(chan struct{})}
	activeConnection = c
	t.Cleanup(func() { CloseSession() })
	done := make(chan struct{})
	go idleExit(func(code int) { close(done) })
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("idle exit stalled")
	}
	if remoteSession() {
		t.Fatal("idle exit did not close socket")
	}
}

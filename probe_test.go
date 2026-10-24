package godoors

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeStdin swaps the shared input for a pipe and returns its writer.
func fakeStdin(t *testing.T) *io.PipeWriter {
	t.Helper()
	origPump, origReader := stdinPump, stdinReader
	originalEscapeReader, originalEscape := pendingEscapeReader, pendingEscape
	pendingEscapeReader, pendingEscape = nil, nil
	pr, pw := io.Pipe()
	stdinPump = newPump(pr)
	stdinReader = bufio.NewReader(stdinPump)
	probeMu.Lock()
	ctermKnown, ctermDetected = false, false
	probeMu.Unlock()
	t.Cleanup(func() {
		pw.Close()
		stdinPump, stdinReader = origPump, origReader
		pendingEscapeReader, pendingEscape = originalEscapeReader, originalEscape
	})
	return pw
}

func TestPumpDeadlineThenLaterInput(t *testing.T) {
	pr, pw := io.Pipe()
	p := newPump(pr)
	p.setDeadline(time.Now().Add(20 * time.Millisecond))
	buf := make([]byte, 8)
	if _, err := p.Read(buf); err != errProbeTimeout {
		t.Fatalf("want errProbeTimeout, got %v", err)
	}
	p.setDeadline(time.Time{})
	go pw.Write([]byte("k"))
	if n, err := p.Read(buf); err != nil || string(buf[:n]) != "k" {
		t.Fatalf("input after a timeout must still arrive: %q %v", buf[:n], err)
	}
	pw.Close()
	if _, err := p.Read(buf); err != io.EOF {
		t.Fatalf("want EOF after close, got %v", err)
	}
}

func TestProbeDetectsCTermAndKeepsKeys(t *testing.T) {
	pw := fakeStdin(t)
	go func() {
		pw.Write([]byte("q"))                              // a key typed during startup
		pw.Write([]byte("\x1b[24;80R"))                    // size reply
		pw.Write([]byte("\x1b[=67;84;101;114;109;1;316c")) // SyncTERM device attributes
		pw.Write([]byte("z"))
	}()
	var h, w int
	var ok bool
	out := captureStdout(t, func() { h, w, ok = probeTerminal(true) })
	if !strings.Contains(out, "\x1b[6n") || !strings.Contains(out, "\x1b[c") {
		t.Fatalf("probe queries not sent: %q", out)
	}
	if !ok || h != 24 || w != 80 {
		t.Fatalf("size = %d,%d ok=%v, want 24,80", h, w, ok)
	}
	if !IsCTerm() {
		t.Fatal("SyncTERM's device attributes must mark the terminal as CTerm")
	}
	for _, want := range "qz" {
		ch, _, err := GetKey()
		if err != nil || ch != want {
			t.Fatalf("GetKey = %q %v, want %q", ch, err, want)
		}
	}
}

func TestProbeTimesOutWithoutReply(t *testing.T) {
	old := probeTimeout
	probeTimeout = 50 * time.Millisecond
	defer func() { probeTimeout = old }()
	fakeStdin(t)
	start := time.Now()
	var ok bool
	captureStdout(t, func() { _, _, ok = probeTerminal(true) })
	if ok {
		t.Fatal("no reply must not produce a size")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("probe took %v; it must give up at the timeout", time.Since(start))
	}
	if IsCTerm() {
		t.Fatal("a silent terminal is not CTerm")
	}
}

func TestPlainVT100IsNotCTerm(t *testing.T) {
	if isCTermDA("\x1b[?1;2c") || !isCTermDA("\x1b[=67;84;101;114;109;1;316c") {
		t.Fatal("isCTermDA misclassified a reply")
	}
}

// Review: CTerm ignores a tune with any character outside its list, and
// the SO terminator must be followed by SI.
func TestMusicSequence(t *testing.T) {
	got := MusicSequence("T120 L8 O4 CDE\x0e\x0fG! xyz")
	if got != "\x1b[|T120 L8 O4 CDEG \x0e\x0f" {
		t.Fatalf("got %q", got)
	}
}

// Review: the pump must not read stdin ahead of demand, or keys typed
// after the door stops reading would be taken from whoever reads next.
func TestPumpDoesNotReadAhead(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	p := newPump(pr)
	go pw.Write([]byte("a"))
	buf := make([]byte, 8)
	if n, err := p.Read(buf); err != nil || string(buf[:n]) != "a" {
		t.Fatalf("first read: %q %v", buf[:n], err)
	}
	wrote := make(chan struct{})
	go func() { pw.Write([]byte("b")); close(wrote) }()
	select {
	case <-wrote:
		t.Fatal("the pump consumed input nobody asked for")
	case <-time.After(50 * time.Millisecond):
	}
	if n, err := p.Read(buf); err != nil || string(buf[:n]) != "b" {
		t.Fatalf("second read: %q %v", buf[:n], err)
	}
}

// Review: an arrow key split across two stdin reads (the first ending at
// ESC) must still decode as one key, not Esc + "[" + "A".
func TestEscSequenceSplitAcrossReads(t *testing.T) {
	pw := fakeStdin(t)
	go func() {
		pw.Write([]byte(strings.Repeat("x", 511) + "\x1b"))
		time.Sleep(5 * time.Millisecond)
		pw.Write([]byte("[A"))
	}()
	for i := 0; i < 511; i++ {
		if ch, _, err := GetKey(); err != nil || ch != 'x' {
			t.Fatalf("key %d = %q %v", i, ch, err)
		}
	}
	if ch, k, err := GetKey(); err != nil || k != KeyUp {
		t.Fatalf("GetKey = %q %v %v, want KeyUp", ch, k, err)
	}
}

func TestLoneEscStillArrives(t *testing.T) {
	pw := fakeStdin(t)
	go pw.Write([]byte("\x1b"))
	if _, k, err := GetKey(); err != nil || k != KeyEsc {
		t.Fatalf("GetKey = %v %v, want KeyEsc", k, err)
	}
	go pw.Write([]byte("k"))
	if ch, _, err := GetKey(); err != nil || ch != 'k' {
		t.Fatalf("GetKey = %q %v, want k", ch, err)
	}
}

// Review: a device-attributes reply that arrives after its probe timed
// out must be swallowed whole, not leak "67;84;...c" as keystrokes.
func TestLateReplyAfterTimeoutIsSkipped(t *testing.T) {
	old := probeTimeout
	probeTimeout = 30 * time.Millisecond
	defer func() { probeTimeout = old }()
	pw := fakeStdin(t)
	captureStdout(t, func() { probeTerminal(false) })
	go pw.Write([]byte("\x1b[=67;84;101;114;109;1;316ck"))
	if ch, k, err := GetKey(); err != nil || ch != 'k' || k != KeyNone {
		t.Fatalf("GetKey = %q %v %v, want the k typed after the late reply", ch, k, err)
	}
}

// Review: concurrent IsCTerm calls must share one probe, not race on the
// shared input with duplicate queries.
func TestIsCTermProbesOnceConcurrently(t *testing.T) {
	pw := fakeStdin(t)
	go pw.Write([]byte("\x1b[=67;84;101;114;109;1;316c"))
	results := make(chan bool, 2)
	out := captureStdout(t, func() {
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); results <- IsCTerm() }()
		}
		wg.Wait()
	})
	if n := strings.Count(out, "\x1b[c"); n != 1 {
		t.Fatalf("sent %d device-attributes queries, want 1", n)
	}
	if !<-results || !<-results {
		t.Fatal("both callers must see the CTerm answer")
	}
}

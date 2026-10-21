package godoors

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"time"
)

// fakeStdin swaps the shared input for a pipe and returns its writer.
func fakeStdin(t *testing.T) *io.PipeWriter {
	t.Helper()
	origPump, origReader := stdinPump, stdinReader
	pr, pw := io.Pipe()
	stdinPump = newPump(pr)
	stdinReader = bufio.NewReader(stdinPump)
	ctermMu.Lock()
	ctermKnown, ctermDetected = false, false
	ctermMu.Unlock()
	t.Cleanup(func() {
		pw.Close()
		stdinPump, stdinReader = origPump, origReader
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

func TestMusicSequence(t *testing.T) {
	got := MusicSequence("T120 L8 O4 CDE\x0e\x0fG")
	if got != "\x1b[|T120 L8 O4 CDEG\x0e" {
		t.Fatalf("got %q", got)
	}
}

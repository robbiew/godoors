package godoors

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"time"
)

func TestPipeColorPreservesLiteralPrefix(t *testing.T) {
	for _, text := range []string{"FACE", "1234", "hello", "|ZZ", "x|"} {
		out := captureStdout(t, func() { PrintPipeColor(text, White) })
		if strings.ReplaceAll(strings.ReplaceAll(out, White, ""), Reset, "") != text {
			t.Errorf("%q: got %q", text, out)
		}
	}
	var tint string
	out := captureStdout(t, func() { tint = PrintPipeColor("FACE|0Ftext", White) })
	if out != White+"FACE"+Reset+BgBlack+WhiteHi+"text"+Reset || tint != BgBlack+WhiteHi {
		t.Fatalf("delimited color: %q, tint %q", out, tint)
	}
}

func TestPartialEscapeSequenceTimesOut(t *testing.T) {
	for _, prefix := range []string{"\x1b[", "\x1bO", "\x1b[12;"} {
		t.Run(prefix, func(t *testing.T) {
			pw := fakeStdin(t)
			go pw.Write([]byte(prefix))
			type result struct {
				key Key
				err error
			}
			done := make(chan result, 1)
			go func() { _, key, err := GetKey(); done <- result{key, err} }()
			select {
			case got := <-done:
				if got.key != KeyEsc || got.err != nil {
					t.Fatalf("got %+v", got)
				}
			case <-time.After(time.Second):
				pw.Close()
				<-done
				t.Fatal("partial sequence did not time out")
			}
			go pw.Write([]byte("q"))
			if ch, _, err := GetKey(); ch != 'q' || err != nil {
				t.Fatalf("key after timeout: %q, %v", ch, err)
			}
		})
	}
}

func TestArrowSequenceSplitAfterIntroducer(t *testing.T) {
	pw := fakeStdin(t)
	go func() { pw.Write([]byte("\x1b[")); time.Sleep(5 * time.Millisecond); pw.Write([]byte("A")) }()
	if _, key, err := GetKey(); key != KeyUp || err != nil {
		t.Fatalf("got %v, %v", key, err)
	}
}

func TestKeyLineEndings(t *testing.T) {
	for _, input := range []string{"\r\nn", "\rn", "\nn"} {
		r := bufio.NewReader(strings.NewReader(input))
		if _, key, err := decodeKey(r); key != KeyEnter || err != nil {
			t.Fatalf("%q: %v, %v", input, key, err)
		}
		if ch, _, err := decodeKey(r); ch != 'n' || err != nil {
			t.Errorf("%q: next key %q, %v", input, ch, err)
		}
	}
	r := bufio.NewReader(strings.NewReader("\r\r"))
	for i := 0; i < 2; i++ {
		if _, key, err := decodeKey(r); key != KeyEnter || err != nil {
			t.Fatalf("consecutive CR: %v, %v", key, err)
		}
	}
}

func TestCRDoesNotWaitForLF(t *testing.T) {
	pw := fakeStdin(t)
	go pw.Write([]byte("\r"))
	done := make(chan Key, 1)
	go func() { _, key, _ := GetKey(); done <- key }()
	select {
	case key := <-done:
		if key != KeyEnter {
			t.Fatal(key)
		}
	case <-time.After(time.Second):
		pw.Close()
		<-done
		t.Fatal("CR waited for a following byte")
	}
	go func() { pw.Write([]byte("\n")); pw.Write([]byte("n")) }()
	if ch, _, err := GetKey(); ch != 'n' || err != nil {
		t.Fatalf("split CRLF: %q, %v", ch, err)
	}
}

func TestColoredArtWidth(t *testing.T) {
	colored := Red + "ABC" + Reset
	if w, h := artDimensions(colored); w != 3 || h != 1 {
		t.Errorf("dimensions %d x %d", w, h)
	}
	u := User{ModalW: 80, ModalH: 24}
	out := captureStdout(t, func() { u.AbsCenterArt(colored) })
	if out != "\x1b[12;39f"+colored {
		t.Errorf("centered art %q", out)
	}
	out = captureStdout(t, func() { CenterText(colored, 9) })
	if out != "   "+colored+"   " {
		t.Errorf("centered text %q", out)
	}
}

func TestLongArtLines(t *testing.T) {
	line := strings.Repeat("X", 70*1024)
	if w, h := artDimensions(line); w != len(line) || h != 1 {
		t.Errorf("unterminated long line: %d x %d", w, h)
	}
	art := line + "\r\nend\r\n"
	if w, h := artDimensions(art); w != len(line) || h != 2 {
		t.Errorf("dimensions %d x %d", w, h)
	}
	if out := captureStdout(t, func() { PrintAnsiLoc(art, 1, 1) }); out != "\x1b[1;1f"+line+"\x1b[2;1fend" {
		t.Error("PrintAnsiLoc lost long line")
	}
	if out := captureStdout(t, func() { PrintAnsi(art, 0, 25) }); out != line+"\r\nend\r\n" {
		t.Error("PrintAnsi lost long line")
	}
	u := User{ModalW: 80, ModalH: 24}
	if out := captureStdout(t, func() { u.AbsCenterArt(art) }); out != "\x1b[12;1f"+line+"\x1b[13;1fend" {
		t.Error("AbsCenterArt lost long line")
	}
}

func TestSizeReplyRequiresPositiveExactGeometry(t *testing.T) {
	for _, reply := range []string{"\x1b[0;80R", "\x1b[24;0R", "prefix24;80", "\x1b[?24;80R", "\x1b[24;80;1R", "\x1b[999999999999999999999;80R"} {
		if _, _, ok := parseTermSizeReply(reply); ok {
			t.Errorf("accepted %q", reply)
		}
	}
	if h, w, ok := parseTermSizeReply("\x1b[24;80R"); !ok || h != 24 || w != 80 {
		t.Fatalf("valid reply %d %d %v", h, w, ok)
	}
}

func TestEscapeEOFIsReported(t *testing.T) {
	pw := fakeStdin(t)
	go func() { pw.Write([]byte("\x1b[")); pw.Close() }()
	if _, _, err := GetKey(); err != io.EOF {
		t.Fatalf("want EOF, got %v", err)
	}
}

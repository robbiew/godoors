package godoors

import (
	"bufio"
	"strings"
	"testing"
)

func TestCursorToIsOneBasedAndClamps(t *testing.T) {
	cases := []struct {
		x, y int
		want string
	}{
		{1, 1, "\x1b[1;1f"},
		{80, 24, "\x1b[24;80f"},
		{0, 0, "\x1b[1;1f"},
		{-3, 5, "\x1b[5;1f"},
	}
	for _, c := range cases {
		if got := cursorTo(c.x, c.y); got != c.want {
			t.Errorf("cursorTo(%d,%d) = %q, want %q", c.x, c.y, got, c.want)
		}
	}
}

// Art drawn from row 0 used to put its first two lines on terminal row 1
// (ESC[0;..f and ESC[1;..f are the same row). Clamping starts it at row 1
// and every following line gets its own row.
func TestPrintAnsiLocRowsNeverCollide(t *testing.T) {
	out := captureStdout(t, func() {
		PrintAnsiLoc("a\nb\nc", 0, 0)
	})
	want := "\x1b[1;1fa\x1b[2;1fb\x1b[3;1fc"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestPrintStringLocClamps(t *testing.T) {
	out := captureStdout(t, func() { PrintStringLoc("x", 0, -1) })
	if out != "\x1b[1;1fx" {
		t.Fatalf("got %q", out)
	}
}

func TestReadCPRKeepsTypedKeys(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("ab\x1b[24;80Rcd"))
	reply, typed, err := readCPR(r)
	if err != nil || reply != "\x1b[24;80R" || string(typed) != "ab" {
		t.Fatalf("got reply=%q typed=%q err=%v", reply, typed, err)
	}
	rest, _ := r.ReadString(0)
	if rest != "cd" {
		t.Fatalf("bytes after the reply must stay buffered, got %q", rest)
	}
	if h, w, ok := parseTermSizeReply(reply); !ok || h != 24 || w != 80 {
		t.Fatalf("reply did not parse: %d,%d,%v", h, w, ok)
	}
}

func TestReadCPRIgnoresStrayR(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("R\x1b[25;132R"))
	reply, typed, err := readCPR(r)
	if err != nil || reply != "\x1b[25;132R" || string(typed) != "R" {
		t.Fatalf("got reply=%q typed=%q err=%v", reply, typed, err)
	}
}

func TestReadCPRNoReply(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("xyz"))
	if _, typed, err := readCPR(r); err == nil || string(typed) != "xyz" {
		t.Fatalf("want an error and the typed bytes back, got typed=%q err=%v", typed, err)
	}
}

func TestUnreadInputReplaysBeforeStdin(t *testing.T) {
	orig := stdinReader
	defer func() { stdinReader = orig }()
	stdinReader = bufio.NewReader(strings.NewReader("cd"))
	unreadInput([]byte("ab"))
	var got []rune
	for i := 0; i < 4; i++ {
		ch, _, err := GetKey()
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, ch)
	}
	if string(got) != "abcd" {
		t.Fatalf("got %q, want typed keys first: abcd", string(got))
	}
}

// Copilot review: an ESC typed before the size reply and "[A" after it
// must still decode as one arrow key once the ESC is replayed.
func TestReplayedEscJoinsBufferedSequence(t *testing.T) {
	orig := stdinReader
	defer func() { stdinReader = orig }()
	stdinReader = bufio.NewReader(strings.NewReader("\x1b\x1b[24;80R[A"))
	reply, typed, err := readCPR(stdinReader)
	if err != nil || reply != "\x1b[24;80R" || string(typed) != "\x1b" {
		t.Fatalf("got reply=%q typed=%q err=%v", reply, typed, err)
	}
	unreadInput(typed)
	if ch, k, err := GetKey(); err != nil || k != KeyUp {
		t.Fatalf("GetKey = %q %v %v, want KeyUp", ch, k, err)
	}
}

// Copilot review: full-height centered art must not end with a newline,
// which would scroll the screen and drop its first line.
func TestFullHeightCenteredArtDoesNotScroll(t *testing.T) {
	u := User{ModalH: 3, ModalW: 10}
	out := captureStdout(t, func() { u.AbsCenterArt("aaa\nbbb\nccc") })
	if strings.ContainsAny(out, "\r\n") {
		t.Fatalf("centered art emitted a line break: %q", out)
	}
	want := "\x1b[1;4faaa\x1b[2;4fbbb\x1b[3;4fccc"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

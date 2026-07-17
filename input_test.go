package godoors

import (
	"bufio"
	"strings"
	"testing"
)

func decodeString(s string) (rune, Key, error) {
	return decodeKey(bufio.NewReader(strings.NewReader(s)))
}

func TestDecodeKeyBasics(t *testing.T) {
	cases := []struct {
		in   string
		ch   rune
		key  Key
	}{
		{"a", 'a', KeyNone},
		{"Q", 'Q', KeyNone},
		{" ", ' ', KeyNone},
		{"3", '3', KeyNone},
		{"\r", 0, KeyEnter},
		{"\n", 0, KeyEnter},
		{"\x1b", 0, KeyEsc},          // lone ESC
		{"\x1b[A", 0, KeyUp},         // CSI arrows
		{"\x1b[B", 0, KeyDown},
		{"\x1b[C", 0, KeyRight},
		{"\x1b[D", 0, KeyLeft},
		{"\x1bOA", 0, KeyUp},         // SS3 arrows (application mode)
	}
	for _, c := range cases {
		ch, key, err := decodeString(c.in)
		if err != nil {
			t.Errorf("%q: unexpected error %v", c.in, err)
			continue
		}
		if ch != c.ch || key != c.key {
			t.Errorf("%q: got (%q, %v), want (%q, %v)", c.in, ch, key, c.ch, c.key)
		}
	}
}

func TestDecodeKeySequentialReads(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("q\r\x1b[Ax"))
	expect := []struct {
		ch  rune
		key Key
	}{{'q', KeyNone}, {0, KeyEnter}, {0, KeyUp}, {'x', KeyNone}}
	for i, e := range expect {
		ch, key, err := decodeKey(r)
		if err != nil || ch != e.ch || key != e.key {
			t.Fatalf("read %d: got (%q, %v, %v), want (%q, %v)", i, ch, key, err, e.ch, e.key)
		}
	}
}

func TestDecodeKeyEOF(t *testing.T) {
	if _, _, err := decodeString(""); err == nil {
		t.Fatal("expected error at EOF (dropped carrier)")
	}
}

func TestDecodeKeyUnknownCSIConsumed(t *testing.T) {
	// An unrecognized CSI sequence is swallowed, then the next key reads fine.
	r := bufio.NewReader(strings.NewReader("\x1b[15~z"))
	ch, key, err := decodeKey(r)
	if err != nil || key != KeyNone || ch != 0 {
		t.Fatalf("unknown CSI: got (%q, %v, %v), want swallowed (0, KeyNone)", ch, key, err)
	}
	ch, _, err = decodeKey(r)
	if err != nil || ch != 'z' {
		t.Fatalf("after unknown CSI: got (%q, %v), want 'z'", ch, err)
	}
}

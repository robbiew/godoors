package godoors

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExtendedKeysAndLegacyDecoder(t *testing.T) {
	cases := map[string]Key{
		"\x1b[5~": KeyPageUp, "\x1b[6~": KeyPageDown, "\x1b[H": KeyHome, "\x1bOH": KeyHome, "\x1b[1~": KeyHome,
		"\x1b[F": KeyEnd, "\x1bOF": KeyEnd, "\x1b[4~": KeyEnd, "\x1b[2~": KeyInsert, "\x1b[3~": KeyDelete,
		"\x1bOP": KeyF1, "\x1bOQ": KeyF2, "\x1bOR": KeyF3, "\x1bOS": KeyF4,
		"\x1b[11~": KeyF1, "\x1b[12~": KeyF2, "\x1b[13~": KeyF3, "\x1b[14~": KeyF4,
		"\x1b[15~": KeyF5, "\x1b[17~": KeyF6, "\x1b[18~": KeyF7, "\x1b[19~": KeyF8,
		"\x1b[20~": KeyF9, "\x1b[21~": KeyF10, "\x1b[23~": KeyF11, "\x1b[24~": KeyF12,
	}
	for sequence, want := range cases {
		t.Run(sequence, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(sequence + "z"))
			_, key, err := decodeKeyEvent(r, true)
			if err != nil || key != want {
				t.Fatalf("got %v %v, want %v", key, err, want)
			}
			ch, _, err := decodeKeyEvent(r, true)
			if err != nil || ch != 'z' {
				t.Fatalf("lost following key: %q %v", ch, err)
			}
			r = bufio.NewReader(strings.NewReader(sequence))
			ch, key, err = decodeKey(r)
			if err != nil || ch != 0 || key != KeyNone {
				t.Fatalf("legacy decoder changed: %q %v %v", ch, key, err)
			}
		})
	}
}

func TestReadKeySharesProbeInput(t *testing.T) {
	pw := fakeStdin(t)
	go pw.Write([]byte("\x1b[5~q"))
	_, key, err := ReadKey()
	if err != nil || key != KeyPageUp {
		t.Fatalf("got %v %v", key, err)
	}
	ch, _, err := GetKey()
	if err != nil || ch != 'q' {
		t.Fatalf("lost key: %q %v", ch, err)
	}
}

func TestOpenSessionDoesNotProbe(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "door32.sys"), []byte("0\n1234\n57600\nTest\n1\nCaller\nAlias\n255\n30\n1\n2\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		info, err := OpenSession(dir)
		if err != nil {
			t.Fatal(err)
		}
		if info.Alias != "Alias" || info.Node != 2 || info.Handle != 1234 {
			t.Fatalf("metadata: %+v", info)
		}
	})
	if out != "" {
		t.Fatalf("unexpected probe/output: %q", out)
	}
	if err := CloseSession(); err != nil {
		t.Fatal(err)
	}
}

func TestReadKeyTimeoutRetainsLaterInput(t *testing.T) {
	pw := fakeStdin(t)
	if _, _, err := ReadKeyTimeout(10 * time.Millisecond); !errors.Is(err, ErrInputTimeout) {
		t.Fatalf("got %v", err)
	}
	go pw.Write([]byte("\x1b[6~q"))
	_, key, err := ReadKeyTimeout(time.Second)
	if err != nil || key != KeyPageDown {
		t.Fatalf("got %v %v", key, err)
	}
	ch, _, err := ReadKey()
	if err != nil || ch != 'q' {
		t.Fatalf("lost key: %q %v", ch, err)
	}
}

func TestReadKeyTimeoutSurvivesUnknownEscape(t *testing.T) {
	pw := fakeStdin(t)
	go pw.Write([]byte("\x1b[99~"))
	done := make(chan error, 1)
	go func() { _, _, err := ReadKeyTimeout(20 * time.Millisecond); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, ErrInputTimeout) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(time.Second):
		pw.Close()
		t.Fatal("escape removed outer deadline")
	}
}

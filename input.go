package godoors

// Door keyboard input, read from STDIN — not /dev/tty. A BBS wires the
// caller's socket to the door's stdin/stdout, so the player's keystrokes
// arrive on stdin; reading the controlling terminal only works for local
// runs and leaves remote users' keys unanswered. When stdin IS a local
// terminal, OpenInput switches it to raw mode so single keypresses arrive
// without Enter; when stdin is a socket or pipe it is left untouched.

import (
	"bufio"
	"os"
	"os/exec"
)

// Key identifies non-printable keys returned by GetKey.
type Key int

const (
	KeyNone Key = iota
	KeyEnter
	KeyEsc
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
)

var (
	stdinReader *bufio.Reader
	stdinIsTTY  bool
)

// OpenInput prepares stdin for single-key reads. Safe to call once at door
// startup; pair with CloseInput (defer) to restore the terminal locally.
func OpenInput() error {
	stdinReader = bufio.NewReader(os.Stdin)
	if st, err := os.Stdin.Stat(); err == nil && st.Mode()&os.ModeCharDevice != 0 {
		stdinIsTTY = true
		raw := exec.Command("/bin/stty", "raw", "-echo")
		raw.Stdin = os.Stdin
		return raw.Run()
	}
	return nil
}

// CloseInput restores the local terminal if OpenInput put it in raw mode.
func CloseInput() {
	if stdinIsTTY {
		cooked := exec.Command("/bin/stty", "-raw", "echo")
		cooked.Stdin = os.Stdin
		_ = cooked.Run()
	}
}

// GetKey blocks for one keypress from stdin. Printable characters come back
// as the rune with KeyNone; Enter, Esc and arrows come back as a Key with
// rune 0. An error means stdin is gone (dropped carrier / closed socket).
func GetKey() (rune, Key, error) {
	if stdinReader == nil {
		stdinReader = bufio.NewReader(os.Stdin)
	}
	return decodeKey(stdinReader)
}

// decodeKey reads one logical keypress from r. A lone ESC (no bytes
// buffered behind it) is the Esc key; ESC followed by a CSI/SS3 sequence
// decodes to arrows, and unrecognized sequences are swallowed so stray
// terminal reports never leak into the game as keys.
func decodeKey(r *bufio.Reader) (rune, Key, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, KeyNone, err
	}
	switch b {
	case 0x0D, 0x0A:
		return 0, KeyEnter, nil
	case 0x1B:
		if r.Buffered() == 0 {
			return 0, KeyEsc, nil
		}
		return decodeEscSequence(r)
	default:
		return rune(b), KeyNone, nil
	}
}

func decodeEscSequence(r *bufio.Reader) (rune, Key, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, KeyEsc, nil
	}
	if b != '[' && b != 'O' {
		// Not a recognized sequence intro; treat as Esc and put the
		// byte back for the next read.
		_ = r.UnreadByte()
		return 0, KeyEsc, nil
	}
	// Read parameter bytes (digits, ';') until the final byte.
	for {
		fb, err := r.ReadByte()
		if err != nil {
			return 0, KeyEsc, nil
		}
		switch fb {
		case 'A':
			return 0, KeyUp, nil
		case 'B':
			return 0, KeyDown, nil
		case 'C':
			return 0, KeyRight, nil
		case 'D':
			return 0, KeyLeft, nil
		}
		if (fb < '0' || fb > '9') && fb != ';' {
			// Final byte of a sequence we don't map: swallow it.
			return 0, KeyNone, nil
		}
	}
}

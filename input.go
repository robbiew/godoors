package godoors

// Door keyboard input, read from STDIN — not /dev/tty. A BBS wires the
// caller's socket to the door's stdin/stdout, so the player's keystrokes
// arrive on stdin; reading the controlling terminal only works for local
// runs and leaves remote users' keys unanswered. When stdin IS a local
// terminal (Linux, macOS, or a Windows console), OpenInput switches it to
// raw mode so single keypresses arrive without Enter; when stdin is a
// socket or pipe it is left untouched.

import (
	"bufio"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
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
	stdinPump   *pumpReader
	stdinReader *bufio.Reader
	rawState    *term.State   // non-nil while OpenInput holds a local terminal in raw mode
	rawMu       sync.Mutex    // also guards restoration by the idle callback
	crReader    *bufio.Reader // suppress one LF after CR on the same input stream
)

// inputReader returns the one buffered reader over stdin. Everything that
// reads the caller's input (GetKey, GetTermSize) shares it, so bytes one
// reader buffered are never lost to another.
func inputReader() *bufio.Reader {
	if stdinReader == nil {
		stdinPump = newPump(sessionInput())
		stdinReader = bufio.NewReader(stdinPump)
	}
	return stdinReader
}

// unreadInput puts bytes back in front of stdin, for the next GetKey.
func unreadInput(b []byte) {
	if len(b) == 0 {
		return
	}
	stdinReader = bufio.NewReader(&replayReader{pending: b, under: inputReader()})
}

// replayReader serves replayed bytes, then the reader underneath. The read
// that finishes the replay also takes whatever the underlying reader
// already has buffered, so an escape sequence split across the boundary
// (ESC replayed, "[A" buffered) arrives in one fill and decodeKey still
// sees it as one key rather than a lone Esc.
type replayReader struct {
	pending []byte
	under   *bufio.Reader
}

func (r *replayReader) Read(b []byte) (int, error) {
	if len(r.pending) == 0 {
		return r.under.Read(b)
	}
	n := copy(b, r.pending)
	r.pending = r.pending[n:]
	if len(r.pending) == 0 && n < len(b) {
		if m := r.under.Buffered(); m > 0 {
			k, _ := r.under.Read(b[n:min(len(b), n+m)]) // buffered: never blocks
			n += k
		}
	}
	return n, nil
}

// OpenInput prepares stdin for single-key reads. Safe to call once at door
// startup; pair with CloseInput (defer) to restore the terminal locally.
// Input already buffered (e.g. keys typed while Initialize probed the
// terminal size) is kept.
func OpenInput() error {
	rawMu.Lock()
	defer rawMu.Unlock()
	prepareConsole()
	inputReader()
	fd := int(os.Stdin.Fd())
	if remoteSession() || rawState != nil || !term.IsTerminal(fd) {
		return nil
	}
	st, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	rawState = st
	return nil
}

// CloseInput restores the local terminal if OpenInput put it in raw mode.
func CloseInput() {
	rawMu.Lock()
	defer rawMu.Unlock()
	if rawState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), rawState)
		rawState = nil
	}
}

// withRawInput runs fn with a local terminal in raw mode (no line
// buffering, no echo), so replies to terminal queries arrive at once and
// don't print. Pipes, sockets and an already-raw terminal are left as is.
func withRawInput(fn func()) {
	rawMu.Lock()
	defer rawMu.Unlock()
	fd := int(os.Stdin.Fd())
	if !remoteSession() && rawState == nil && term.IsTerminal(fd) {
		if st, err := term.MakeRaw(fd); err == nil {
			defer term.Restore(fd, st)
		}
	}
	fn()
}

// GetKey blocks for one keypress from stdin. Printable characters come back
// as the rune with KeyNone; Enter, Esc and arrows come back as a Key with
// rune 0. An error means stdin is gone (dropped carrier / closed socket).
// Escape sequences that aren't keys (such as a terminal reply arriving
// after its probe timed out) are skipped.
func GetKey() (rune, Key, error) {
	for {
		ch, k, err := decodeKey(inputReader())
		if err != nil || ch != 0 || k != KeyNone {
			return ch, k, err
		}
	}
}

// escWait bounds the entire continuation of an ESC sequence, including
// sequences split after the CSI/SS3 introducer. An incomplete sequence is Esc.
var escWait = 30 * time.Millisecond

// decodeKey reads one logical keypress from r. A lone ESC is the Esc key;
// ESC followed by a CSI/SS3 sequence decodes to arrows, and unrecognized
// sequences come back as (0, KeyNone) so stray terminal reports never leak
// into the game as keys.
func decodeKey(r *bufio.Reader) (rune, Key, error) {
	b, err := r.ReadByte()
	if crReader == r && err == nil && b == '\n' {
		b, err = r.ReadByte()
	}
	crReader = nil
	if err != nil {
		return 0, KeyNone, err
	}
	switch b {
	case 0x0D:
		crReader = r
		return 0, KeyEnter, nil
	case 0x0A:
		return 0, KeyEnter, nil
	case 0x1B:
		if stdinPump != nil {
			stdinPump.setDeadline(time.Now().Add(escWait))
			defer stdinPump.setDeadline(time.Time{})
		} else if r.Buffered() == 0 {
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
	// Consume the whole sequence: parameter bytes (0x30-0x3F, which
	// includes the '?' and '=' of terminal replies) and intermediate bytes
	// (0x20-0x2F) up to the final byte (0x40-0x7E).
	for n := 0; n < 128; n++ {
		fb, err := r.ReadByte()
		if err != nil {
			if err == errProbeTimeout {
				return 0, KeyEsc, nil
			}
			return 0, KeyNone, err
		}
		if fb >= 0x40 && fb <= 0x7E {
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
			return 0, KeyNone, nil // a sequence we don't map: swallowed
		}
		if fb < 0x20 || fb > 0x3F {
			return 0, KeyNone, nil // malformed: stop here
		}
	}
	return 0, KeyEsc, nil // oversized control sequence
}

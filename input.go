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
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd
	KeyInsert
	KeyDelete
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
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

// ReadKey reads a logical key including PageUp/PageDown, Home/End, Insert/Delete,
// and F1-F12. GetKey keeps its original arrows-only behavior and skips these
// extended escape sequences. Both share the terminal probe's buffered input.
func ReadKey() (rune, Key, error) {
	for {
		ch, k, err := decodeKeyEvent(inputReader(), true)
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
	return decodeKeyEvent(r, false)
}

func decodeKeyEvent(r *bufio.Reader, extended bool) (rune, Key, error) {
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
			previous := stdinPump.getDeadline()
			deadline := time.Now().Add(escWait)
			if !previous.IsZero() && previous.Before(deadline) {
				deadline = previous
			}
			stdinPump.setDeadline(deadline)
			defer stdinPump.setDeadline(previous)
		} else if r.Buffered() == 0 {
			return 0, KeyEsc, nil
		}
		return decodeEscSequence(r, extended)
	default:
		return rune(b), KeyNone, nil
	}
}

func decodeEscSequence(r *bufio.Reader, extended bool) (rune, Key, error) {
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
	var sequence []byte
	for n := 0; n < 128; n++ {
		fb, err := r.ReadByte()
		if err != nil {
			if err == errProbeTimeout {
				return 0, KeyEsc, nil
			}
			return 0, KeyNone, err
		}
		sequence = append(sequence, fb)
		if fb >= 0x40 && fb <= 0x7E {
			if extended {
				if key := extendedKey(b, string(sequence)); key != KeyNone {
					return 0, key, nil
				}
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
			return 0, KeyNone, nil // a sequence we don't map: swallowed
		}
		if fb < 0x20 || fb > 0x3F {
			return 0, KeyNone, nil // malformed: stop here
		}
	}
	return 0, KeyEsc, nil // oversized control sequence
}

func extendedKey(intro byte, sequence string) Key {
	if intro == 'O' {
		switch sequence {
		case "P":
			return KeyF1
		case "Q":
			return KeyF2
		case "R":
			return KeyF3
		case "S":
			return KeyF4
		case "H":
			return KeyHome
		case "F":
			return KeyEnd
		}
		return KeyNone
	}
	switch sequence {
	case "H", "1~":
		return KeyHome
	case "F", "4~":
		return KeyEnd
	case "2~":
		return KeyInsert
	case "3~":
		return KeyDelete
	case "5~":
		return KeyPageUp
	case "6~":
		return KeyPageDown
	case "11~":
		return KeyF1
	case "12~":
		return KeyF2
	case "13~":
		return KeyF3
	case "14~":
		return KeyF4
	case "15~":
		return KeyF5
	case "17~":
		return KeyF6
	case "18~":
		return KeyF7
	case "19~":
		return KeyF8
	case "20~":
		return KeyF9
	case "21~":
		return KeyF10
	case "23~":
		return KeyF11
	case "24~":
		return KeyF12
	}
	return KeyNone
}

// ReadKeyTimeout is ReadKey with a bounded wait. Buffered keys are retained and
// an outstanding transport read remains available for the next call on timeout.
// Like ReadKey, it must not run concurrently with another input reader/probe.
func ReadKeyTimeout(timeout time.Duration) (rune, Key, error) {
	if timeout <= 0 {
		return 0, KeyNone, ErrInputTimeout
	}
	inputReader()
	previous := stdinPump.getDeadline()
	stdinPump.setDeadline(time.Now().Add(timeout))
	defer stdinPump.setDeadline(previous)
	return ReadKey()
}

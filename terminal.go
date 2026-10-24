package godoors

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

var sizeReply = regexp.MustCompile(`^\x1b\[([0-9]+);([0-9]+)R$`)

func parseTermSizeReply(text string) (int, int, bool) {
	parts := sizeReply.FindStringSubmatch(text)
	if parts == nil {
		return 0, 0, false
	}

	h, herr := strconv.Atoi(parts[1])
	w, werr := strconv.Atoi(parts[2])
	if herr != nil || werr != nil || h < 1 || w < 1 {
		return 0, 0, false
	}

	return h, w, true
}

// GetTermSize detects the connected terminal's size and returns height
// (rows) and width (columns). A local terminal is asked directly; a remote
// caller's terminal is probed by parking the cursor far past any real
// screen and reading back the clamped cursor-position report. If the
// terminal doesn't answer within a moment, or the reply can't be parsed,
// it falls back to the classic 25x80. Keys typed during the probe are kept
// for GetKey.
func GetTermSize() (int, int) {
	const (
		defaultH = 25
		defaultW = 80
	)
	prepareConsole()
	if !remoteSession() {
		if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 && h > 0 {
			return h, w
		}
	}
	h, w, ok := probeTerminal(true)
	if !ok {
		// couldn't detect, so fall back to the classic 80x25 terminal:
		// 25 rows (height) by 80 columns (width).
		return defaultH, defaultW
	}
	return h, w
}

// IsCTerm reports whether the caller's terminal identifies as CTerm,
// i.e. SyncTERM or another CTerm-compatible client, which play ANSI music
// (see PlayMusic). The terminal is asked once; the answer is cached, and
// Initialize fills in User.CTerm with it.
func IsCTerm() bool {
	probeMu.Lock()
	defer probeMu.Unlock()
	if !ctermKnown {
		probeTerminalLocked(false)
	}
	return ctermDetected
}

var (
	// probeMu serializes probes (only one query may be in flight on the
	// shared input) and guards the cached CTerm answer.
	probeMu       sync.Mutex
	ctermKnown    bool
	ctermDetected bool
)

// probeTimeout bounds how long a probe waits for the terminal to answer;
// terminals that ignore a query would otherwise hang the door.
var probeTimeout = 1500 * time.Millisecond

// probeTerminal sends the device-attributes query (ESC[c), plus a
// cursor-position probe when askSize is set, and reads the replies through
// the shared input reader with a deadline. It records whether the terminal
// is CTerm and returns the probed size.
func probeTerminal(askSize bool) (h, w int, sizeOK bool) {
	probeMu.Lock()
	defer probeMu.Unlock()
	return probeTerminalLocked(askSize)
}

// probeTerminalLocked is probeTerminal for callers holding probeMu.
func probeTerminalLocked(askSize bool) (h, w int, sizeOK bool) {
	prepareConsole() // queries are escape sequences: a Windows console must process them
	query, want := "\x1b[c", 1
	if askSize {
		query, want = "\x1b[999;999f\x1b[6n\x1b[c", 2
	}
	var replies []string
	withRawInput(func() {
		fmt.Fprint(Output(), query)
		r := inputReader()
		if stdinPump != nil {
			stdinPump.setDeadline(time.Now().Add(probeTimeout))
			defer stdinPump.setDeadline(time.Time{})
		}
		var typed []byte
		replies, typed, _ = readReplies(r, want)
		unreadInput(typed)
	})
	cterm := false
	for _, rep := range replies {
		switch {
		case strings.HasSuffix(rep, "R"):
			h, w, sizeOK = parseTermSizeReply(rep)
		case strings.HasSuffix(rep, "c"):
			cterm = isCTermDA(rep)
		}
	}
	ctermKnown, ctermDetected = true, cterm
	return h, w, sizeOK
}

// termReply matches the end of a reply to one of our queries: a
// cursor-position report (ESC [ row ; col R) or device attributes
// (ESC [ ? ... c, or ESC [ = ... c from CTerm).
var termReply = regexp.MustCompile(`\x1b\[[?=]?[0-9;]*[Rc]$`)

// readReplies reads from r until n terminal replies have arrived or a read
// fails (such as the probe deadline). Every other byte is a key the caller
// typed while waiting and is returned in typed, in order; bytes after the
// last reply stay buffered in r. A stray 'R' or 'c' keypress doesn't end a
// reply early.
func readReplies(r *bufio.Reader, n int) (replies []string, typed []byte, err error) {
	var buf []byte
	for len(replies) < n {
		b, err := r.ReadByte()
		if err != nil {
			return replies, append(typed, buf...), err
		}
		buf = append(buf, b)
		if b != 'R' && b != 'c' {
			continue
		}
		if loc := termReply.FindIndex(buf); loc != nil {
			replies = append(replies, string(buf[loc[0]:]))
			typed = append(typed, buf[:loc[0]]...)
			buf = nil
		}
	}
	return replies, typed, nil
}

// isCTermDA reports whether a device-attributes reply came from CTerm,
// which answers ESC[c with ESC[=67;84;101;114;109;<version>c ("CTerm" in
// decimal ASCII).
func isCTermDA(reply string) bool {
	return strings.HasPrefix(reply, "\x1b[=67;84;101;114;109")
}

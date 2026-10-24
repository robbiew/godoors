package godoors

import (
	"fmt"
	"time"
)

// User holds the per-session state parsed from the drop file plus the detected
// terminal geometry. ModalH/ModalW are the height and width rounded down to an
// even number, used to center modal content. CTerm is true when the caller's
// terminal identified as SyncTERM/CTerm, which plays ANSI music.
type User struct {
	Alias     string
	TimeLeft  int
	Emulation int
	NodeNum   int
	H         int
	W         int
	ModalH    int
	ModalW    int
	CTerm     bool
}

// evenDown rounds n down to the nearest even number so modal content can be
// split into equal halves.
func evenDown(n int) int {
	if n%2 == 0 {
		return n
	}
	return n - 1
}

// Initialize reads the drop file at path and probes the terminal (size and
// whether it is CTerm), returning a populated User. It returns an error if
// the drop file can't be read or parsed, or the Windows socket cannot be opened.
// Pair a successful call with CloseSession at door exit.
func Initialize(path string) (User, error) {
	info, err := sessionDropFile(path)
	if err != nil {
		return User{}, err
	}
	if err := connectSession(info); err != nil {
		return User{}, err
	}
	alias, timeLeft, emulation, nodeNum := info.Alias, info.TimeLeft, info.Emulation, info.Node

	h, w := GetTermSize()

	return User{
		Alias:     alias,
		TimeLeft:  timeLeft,
		Emulation: emulation,
		NodeNum:   nodeNum,
		H:         h,
		W:         w,
		ModalH:    evenDown(h),
		ModalW:    evenDown(w),
		CTerm:     IsCTerm(),
	}, nil
}

func artDimensions(art string) (int, int) {
	s := artScanner(art)

	width := 0
	height := 0
	for s.Scan() {
		height++
		if lineWidth := textWidth(s.Text()); lineWidth > width {
			width = lineWidth
		}
	}

	return width, height
}

// Modal displays background ANSI art (contents, not a path) centered on screen
// with text and a "Continue? Y/n" prompt, sized to the user's terminal.
func (u User) Modal(art string, text string) {
	artW, artH := artDimensions(art)
	u.absCenterArt(art, artW, artH)
	u.AbsCenterText(text, BgCyan)
}

// AbsCenterText draws s followed by a "Continue? Y/n" prompt, horizontally and
// vertically centered on the user's terminal with background color c, then
// blocks on the prompt (see Continue) before returning. The centered width is
// measured from the rendered text, so no length argument is needed.
func (u User) AbsCenterText(s string, c string) {
	prompt := s + " Continue? Y/n"
	centerY := u.ModalH / 2
	centerX := (u.ModalW-textWidth(prompt))/2 + 1 // 1-based column
	MoveCursor(centerX, centerY)
	fmt.Fprint(Output(), WhiteHi+c+prompt+Reset)
	if Continue() {
		fmt.Fprint(Output(), BgCyan+CyanHi+" Yes"+Reset)
	} else {
		fmt.Fprint(Output(), BgCyan+CyanHi+" No"+Reset)
	}
	time.Sleep(1 * time.Second)
}

// AbsCenterArt prints ANSI art (contents, not a path, SAUCE stripped) centered
// on the user's terminal. Its position is derived from the art's own measured
// width and height.
func (u User) AbsCenterArt(art string) {
	artW, artH := artDimensions(art)
	u.absCenterArt(art, artW, artH)
}

func (u User) absCenterArt(art string, artW int, artH int) {
	// 1-based: art as tall as the screen starts on row 1, not row 0 (which
	// terminals treat as row 1, stacking the first two lines).
	artY := (u.ModalH-artH)/2 + 1
	artX := (u.ModalW-artW)/2 + 1

	s := artScanner(art)

	// Every line is positioned explicitly, so no trailing newline: one after
	// the bottom row of full-height art would scroll its first line away.
	for s.Scan() {
		fmt.Fprint(Output(), cursorTo(artX, artY)+s.Text())
		artY++
	}
}

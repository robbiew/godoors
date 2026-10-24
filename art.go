package godoors

import (
	"bufio"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

func TruncateText(s string, max int) string {
	if max <= 0 {
		return ""
	}

	runeCount := utf8.RuneCountInString(s)
	if runeCount <= max {
		return s
	}

	runes := []rune(s)
	if max <= 2 {
		return string(runes[:max])
	}

	return string(runes[:max-3]) + "..."
}

func PrintAnsi(artfile string, delay int, height int) {
	noSauce := TrimStringFromSauce(artfile) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	i := 1

	for s.Scan() {
		fmt.Fprint(Output(), s.Text())
		time.Sleep(time.Duration(delay) * time.Millisecond)
		if i < height {
			fmt.Fprint(Output(), "\r\n")
		} else {
			MoveCursor(1, 1)
			break
		}
		i++
	}
}

// PrintAnsiLoc prints art (contents, SAUCE stripped) with its top-left at
// column x, row y, moving down one row per line. Coordinates are 1-based
// ((1, 1) is the top-left cell); values below 1 are clamped to 1.
func PrintAnsiLoc(artfile string, x int, y int) {
	yLoc := clampCoord(y)

	noSauce := TrimStringFromSauce(artfile) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {
		fmt.Fprint(Output(), cursorTo(x, yLoc)+s.Text())
		yLoc++
	}
}

// PrintStringLoc prints text at column x, row y. Coordinates are 1-based
// ((1, 1) is the top-left cell); values below 1 are clamped to 1.
func PrintStringLoc(text string, x int, y int) {
	fmt.Fprint(Output(), cursorTo(x, y)+text)
}

// Horizontally center some text.
func CenterText(s string, w int) {
	fmt.Fprintf(Output(), "%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
}

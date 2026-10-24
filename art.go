package godoors

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Art is already in memory, so allow a token as large as the entire body
// plus the byte Scanner needs to detect EOF. Keep ScanLines semantics.
func artScanner(art string) *bufio.Scanner {
	art = TrimStringFromSauce(art)
	s := bufio.NewScanner(strings.NewReader(art))
	s.Buffer(make([]byte, 4096), max(bufio.MaxScanTokenSize, len(art)+1))
	return s
}

var sgrSequence = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

// textWidth ignores SGR color/style controls. Like the existing text helpers,
// it counts runes, not wide/combining cells or arbitrary ANSI cursor motion.
func textWidth(s string) int {
	return utf8.RuneCountInString(sgrSequence.ReplaceAllString(s, ""))
}

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
	s := artScanner(artfile)

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

	s := artScanner(artfile)

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
	width := textWidth(s)
	left := max(0, (w-width)/2)
	right := max(0, w-width-left)
	fmt.Fprint(Output(), strings.Repeat(" ", left), s, strings.Repeat(" ", right))
}

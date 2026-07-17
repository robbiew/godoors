package godoors

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
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
		fmt.Fprint(os.Stdout, s.Text())
		time.Sleep(time.Duration(delay) * time.Millisecond)
		if i < height {
			fmt.Fprint(os.Stdout, "\r\n")
		} else {
			MoveCursor(0, 0)
			break
		}
		i++
	}
}

func PrintAnsiLoc(artfile string, x int, y int) {
	yLoc := y

	noSauce := TrimStringFromSauce(artfile) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {
		fmt.Fprint(os.Stdout, Esc+strconv.Itoa(yLoc)+";"+strconv.Itoa(x)+"f"+s.Text())
		yLoc++
	}
}

// Print text at an X, Y location
func PrintStringLoc(text string, x int, y int) {
	fmt.Fprint(os.Stdout, Esc+strconv.Itoa(y)+";"+strconv.Itoa(x)+"f"+text)

}

// Horizontally center some text.
func CenterText(s string, w int) {
	fmt.Fprintf(os.Stdout, "%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
}

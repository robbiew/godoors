package godoors

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func parseTermSizeReply(text string) (int, int, bool) {
	re := regexp.MustCompile(`\d+;\d+`)
	line := re.FindString(text)
	if line == "" {
		return 0, 0, false
	}

	parts := strings.Split(line, ";")
	if len(parts) != 2 {
		return 0, 0, false
	}

	h, herr := strconv.Atoi(parts[0])
	w, werr := strconv.Atoi(parts[1])
	if herr != nil || werr != nil {
		return 0, 0, false
	}

	return h, w, true
}

// GetTermSize detects the connected terminal's size by parking the cursor far
// past any real screen and reading back the clamped cursor-position report.
// It returns height (rows) and width (columns). If the terminal can't be
// probed or the reply can't be parsed, it falls back to the classic 25x80.
func GetTermSize() (int, int) {
	const (
		defaultH = 25
		defaultW = 80
	)
	// Set the terminal to raw mode so we aren't waiting for CRLF from user (to be undone with `-raw`)
	rawMode := exec.Command("/bin/stty", "raw")
	rawMode.Stdin = os.Stdin
	_ = rawMode.Run()

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stdout, "\033[999;999f") // larger than any known term size
	fmt.Fprintf(os.Stdout, "\033[6n")       // ansi escape code for reporting cursor location
	text, _ := reader.ReadString('R')

	// Set the terminal back from raw mode to 'cooked'
	rawModeOff := exec.Command("/bin/stty", "-raw")
	rawModeOff.Stdin = os.Stdin
	_ = rawModeOff.Run() // Run already waits for the command to finish

	h, w, ok := parseTermSizeReply(string(text))
	if !ok {
		// couldn't detect, so fall back to the classic 80x25 terminal:
		// 25 rows (height) by 80 columns (width).
		return defaultH, defaultW
	}

	return h, w

}

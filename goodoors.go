package godoors

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eiannone/keyboard"
)

// CREDIT TO https://github.com/k0kubun/go-ansi for some of these sequences.

// Common fragments of escape sequences
const (
	Esc = "\u001B["
	Osc = "\u001B]"
	Bel = "\u0007"
)

// Common fonts, supported by SyncTerm
const (
	Mosoul          = Esc + "0;38 D"
	Potnoodle       = Esc + "0;37 D"
	Microknight     = Esc + "0;41 D"
	Microknightplus = Esc + "0;39 D"
	Topaz           = Esc + "0;42 D"
	Topazplus       = Esc + "0;40 D"
	Ibm             = Esc + "0;0 D"
	Ibmthin         = Esc + "0;26 D"
)

// Symbols
var (
	Heart        = string([]rune{'\u0003'})
	ArrowUpDown  = string([]rune{'\u0017'})
	ArrowUp      = string([]rune{'\u0018'})
	ArrowDown    = string([]rune{'\u0019'})
	ArrowDownFat = string([]rune{'\u001F'})
	ArrowRight   = string([]rune{'\u0010'})
	ArrowLeft    = string([]rune{'\u0011'})
	Block        = string([]rune{'\u0219'})
)

// Common ANSI escapes sequences. These should be used when the desired action
// is only needed once; otherwise, use the functions (e.g. moving a cursor
// several lines/columns). See: https://docs.microsoft.com/en-us/windows/console/console-virtual-terminal-sequences
const (
	CursorBackward = Esc + "D"
	CursorPrevLine = Esc + "F"
	CursorLeft     = Esc + "G"
	CursorTop      = Esc + "d"
	CursorTopLeft  = Esc + "H"

	CursorBlinkEnable  = Esc + "?12h"
	CursorBlinkDisable = Esc + "?12I"

	ScrollUp   = Esc + "S"
	ScrollDown = Esc + "T"

	TextInsertChar = Esc + "@"
	TextDeleteChar = Esc + "P"
	TextEraseChar  = Esc + "X"
	TextInsertLine = Esc + "L"
	TextDeleteLine = Esc + "M"

	EraseRight  = Esc + "K"
	EraseLeft   = Esc + "1K"
	EraseLine   = Esc + "2K"
	EraseDown   = Esc + "J"
	EraseUp     = Esc + "1J"
	EraseScreen = Esc + "2J"

	TextColorBlack         = Esc + "30m"
	TextColorRed           = Esc + "31m"
	TextColorGreen         = Esc + "32m"
	TextColorYellow        = Esc + "33m"
	TextColorBlue          = Esc + "34m"
	TextColorMagenta       = Esc + "35m"
	TextColorCyan          = Esc + "36m"
	TextColorWhite         = Esc + "37m"
	TextColorBrightBlack   = Esc + "30;1m"
	TextColorBrightRed     = Esc + "31;1m"
	TextColorBrightGreen   = Esc + "32;1m"
	TextColorBrightYellow  = Esc + "33;1m"
	TextColorBrightBlue    = Esc + "34;1m"
	TextColorBrightMagenta = Esc + "35;1m"
	TextColorBrightCyan    = Esc + "36;1m"
	TextColorBrightWhite   = Esc + "37;1m"

	BackgroundColorBlack         = Esc + "40m"
	BackgroundColorRed           = Esc + "41m"
	BackgroundColorGreen         = Esc + "42m"
	BackgroundColorYellow        = Esc + "43m"
	BackgroundColorBlue          = Esc + "44m"
	BackgroundColorMagenta       = Esc + "45m"
	BackgroundColorCyan          = Esc + "46m"
	BackgroundColorWhite         = Esc + "47m"
	BackgroundColorBrightBlack   = Esc + "40;1m"
	BackgroundColorBrightRed     = Esc + "41;1m"
	BackgroundColorBrightGreen   = Esc + "42;1m"
	BackgroundColorBrightYellow  = Esc + "43;1m"
	BackgroundColorBrightBlue    = Esc + "44;1m"
	BackgroundColorBrightMagenta = Esc + "45;1m"
	BackgroundColorBrightCyan    = Esc + "46;1m"
	BackgroundColorBrightWhite   = Esc + "47;1m"

	ColorReset = Esc + "0m"
)

// Continue Y/N
func Continue() bool {
	char, key, err := keyboard.GetKey()
	if err != nil {
		panic(err)
	}
	var x bool
	if string(char) == "Y" || string(char) == "y" || key == keyboard.KeyEnter {
		x = true
	}
	if string(char) == "N" || string(char) == "n" || key == keyboard.KeyEsc {
		x = false
	}
	return x
}

func Modal(text string, l int, w int, h int) {
	AbsCenterArt("modalBg.ans", 33, w, h)
	AbsCenterText(text, l, w, h, BackgroundColorCyan)
}

func TruncateText(s string, max int) string {
	if len(s) > max {
		r := 0
		for i := range s {
			r++
			if r > max-3 {
				return s[:i] + "..."
			}
		}
	}
	return s
}

// Wait for a key press
func Pause() {
	fmt.Fprintf(os.Stdout, "\r\nPrEsS a KeY")
	_, _, err := keyboard.GetKey()
	if err != nil {
		panic(err)
	}
}

// Move cursor to X, Y location
func MoveCursor(x int, y int) {
	fmt.Printf(Esc+"%d;%df", y, x)
}

// Erase the screen
func ClearScreen() {
	fmt.Println(EraseScreen)
	MoveCursor(0, 0)
}

// Move the cursor n cells to up.
func CursorUp(n int) {
	fmt.Printf(Esc+"%dA", n)
}

// Move the cursor n cells to down.
func CursorDown(n int) {
	fmt.Printf(Esc+"%dB", n)
}

// Move the cursor n cells to right.
func CursorForward(n int) {
	fmt.Printf(Esc+"%dC", n)
}

// Move the cursor n cells to left.
func CursorBack(n int) {
	fmt.Printf(Esc+"%dD", n)
}

// Move cursor to beginning of the line n lines down.
func CursorNextLine(n int) {
	fmt.Printf(Esc+"%dE", n)
}

// Move cursor to beginning of the line n lines up.
func CursorPreviousLine(n int) {
	fmt.Printf(Esc+"%dF", n)
}

// Move cursor horizontally to x.
func CursorHorizontalAbsolute(x int) {
	fmt.Printf(Esc+"%dG", x)
}

// Show the cursor.
func CursorShow() {
	fmt.Print(Esc + "?25h")
}

// Hide the cursor.
func CursorHide() {
	fmt.Print(Esc + "?25l")
}

// Save the screen.
func SaveScreen() {
	fmt.Print(Esc + "?47h")
}

// Restore the saved screen.
func RestoreScreen() {
	fmt.Print(Esc + "?47l")
}

func DropFileData(path string) (string, int, int, int) {
	// path needs to include trailing slash!
	var dropAlias string
	var dropTimeLeft string
	var dropEmulation string
	var nodeNum string

	file, err := os.Open(strings.ToLower(path + "door32.sys"))
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var text []string

	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	file.Close()

	count := 0
	for _, line := range text {
		if count == 6 {
			dropAlias = line
		}
		if count == 8 {
			dropTimeLeft = line
		}
		if count == 9 {
			dropEmulation = line
		}
		if count == 10 {
			nodeNum = line
		}
		if count == 11 {
			break
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	timeInt, err := strconv.Atoi(dropTimeLeft) // return as int
	if err != nil {
		log.Fatal(err)
	}

	emuInt, err := strconv.Atoi(dropEmulation) // return as int
	if err != nil {
		log.Fatal(err)
	}
	nodeInt, err := strconv.Atoi(nodeNum) // return as int
	if err != nil {
		log.Fatal(err)
	}

	return dropAlias, timeInt, emuInt, nodeInt
}

/*
	Get the terminal size
	- Send a cursor position that we know is way too large
	- Terminal sends back the largest row + col size
	- Read in the result
*/
func GetTermSize() (int, int) {
	// Set the terminal to raw mode so we aren't waiting for CLRF rom user (to be undone with `-raw`)
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
	_ = rawModeOff.Run()
	rawModeOff.Wait()

	// check for the desired output
	if strings.Contains(string(text), ";") {
		re := regexp.MustCompile(`\d+;\d+`)
		line := re.FindString(string(text))

		s := strings.Split(line, ";")
		sh, sw := s[0], s[1]

		ih, err := strconv.Atoi(sh)
		if err != nil {
			// handle error
			fmt.Println(err)
			os.Exit(2)
		}

		iw, err := strconv.Atoi(sw)
		if err != nil {
			// handle error
			fmt.Println(err)
			os.Exit(2)
		}
		h := ih
		w := iw

		return h, w

	} else {
		// couldn't detect, so let's just set 80 x 25 to be safe
		h := 80
		w := 25

		return h, w
	}

}

func PrintAnsi(file string, delay int) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	noSauce := TrimStringFromSauce(string(content)) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {
		fmt.Println(s.Text())
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

}

func TrimStringFromSauce(s string) string {
	if idx := strings.Index(s, "COMNT"); idx != -1 {
		string := s
		delimiter := "COMNT"
		leftOfDelimiter := strings.Split(string, delimiter)[0]
		trim := TrimLastChar(leftOfDelimiter)
		return trim
	}
	if idx := strings.Index(s, "SAUCE00"); idx != -1 {
		string := s
		delimiter := "SAUCE00"
		leftOfDelimiter := strings.Split(string, delimiter)[0]
		trim := TrimLastChar(leftOfDelimiter)
		return trim
	}
	return s
}

func TrimLastChar(s string) string {
	r, size := utf8.DecodeLastRuneInString(s)
	if r == utf8.RuneError && (size == 0 || size == 1) {
		size = 0
	}
	return s[:len(s)-size]
}

func PrintAnsiLoc(file string, x int, y int) {
	yLoc := y
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	noSauce := TrimStringFromSauce(string(content)) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {
		fmt.Println("\x1b[" + strconv.Itoa(yLoc) + ";" + strconv.Itoa(x) + "f")
		fmt.Println(s.Text())
		yLoc++
	}
}

// Print text at an X, Y location
func PrintStringLoc(text string, x int, y int) {
	fmt.Println("\x1b[" + strconv.Itoa(y) + ";" + strconv.Itoa(x) + "f")
	fmt.Println(text)
}

// Horizontally center some text.
func CenterText(s string, w int) {
	fmt.Fprintf(os.Stdout, (fmt.Sprintf("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))))
}

// Horizontally and Vertically center some text.
func AbsCenterText(s string, l int, w int, h int, c string) {
	centerY := h / 2
	halfLen := l / 2
	centerX := (w - w/2) - halfLen
	MoveCursor(centerX, centerY)
	fmt.Fprintf(os.Stdout, TextColorBrightWhite+c+s+ColorReset)
	result := Continue()
	if result {
		fmt.Println(BackgroundColorCyan + TextColorBrightCyan + " Yes" + ColorReset)
		time.Sleep(1 * time.Second)
	}
	if !result {
		fmt.Println(BackgroundColorCyan + TextColorBrightCyan + " No" + ColorReset)
		time.Sleep(1 * time.Second)
	}
}

func AbsCenterArt(file string, l int, w int, h int) {
	artY := (h / 2) - 2
	artLen := l / 2
	artX := (w - w/2) - artLen

	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	noSauce := TrimStringFromSauce(string(content)) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {
		fmt.Fprintf(os.Stdout, Esc+strconv.Itoa(artY)+";"+strconv.Itoa(artX)+"f")
		fmt.Println(s.Text())
		artY++
	}
}

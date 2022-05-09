package godoors

import (
	"bufio"
	"fmt"
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

	modalH int // in case height is odd
	modalW int // in case width is odd
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

	Black     = Esc + "30m"
	Red       = Esc + "31m"
	Green     = Esc + "32m"
	Yellow    = Esc + "33m"
	Blue      = Esc + "34m"
	Magenta   = Esc + "35m"
	Cyan      = Esc + "36m"
	White     = Esc + "37m"
	BlackHi   = Esc + "30;1m"
	RedHi     = Esc + "31;1m"
	GreenHi   = Esc + "32;1m"
	YellowHi  = Esc + "33;1m"
	BlueHi    = Esc + "34;1m"
	MagentaHi = Esc + "35;1m"
	CyanHi    = Esc + "36;1m"
	WhiteHi   = Esc + "37;1m"

	BgBlack     = Esc + "40m"
	BgRed       = Esc + "41m"
	BgGreen     = Esc + "42m"
	BgYellow    = Esc + "43m"
	BgBlue      = Esc + "44m"
	BgMagenta   = Esc + "45m"
	BgCyan      = Esc + "46m"
	BgWhite     = Esc + "47m"
	BgBlackHi   = Esc + "40;1m"
	BgRedHi     = Esc + "41;1m"
	BgGreenHi   = Esc + "42;1m"
	BgYellowHi  = Esc + "43;1m"
	BgBlueHi    = Esc + "44;1m"
	BgMagentaHi = Esc + "45;1m"
	BgCyanHi    = Esc + "46;1m"
	BgWhiteHi   = Esc + "47;1m"

	Reset = Esc + "0m"
)

var Idle int

type User struct {
	Alias     string
	TimeLeft  int
	Emulation int
	NodeNum   int
	H         int
	W         int
	ModalH    int
	ModalW    int
}

// Get info from the Drop File, h, w
func Initialize(path string) User {

	alias, timeLeft, emulation, nodeNum := DropFileData(path)
	h, w := GetTermSize()

	if h%2 == 0 {
		modalH = h
	} else {
		modalH = h - 1
	}

	if w%2 == 0 {
		modalW = w
	} else {
		modalW = w - 1
	}

	u := User{
		Alias:     alias,
		TimeLeft:  timeLeft,
		Emulation: emulation,
		NodeNum:   nodeNum,
		H:         h,
		W:         w,
		ModalH:    modalH,
		ModalW:    modalW,
	}
	return u
}

// Continue Y/N
func Continue() bool {
	shortTimer := NewTimer(Idle, func() {
		fmt.Println("\r\nYou've been idle for too long... exiting!")
		time.Sleep(1 * time.Second)
		os.Exit(0)
	})
	defer shortTimer.Stop()

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

func Modal(artPath string, text string, l int) {
	AbsCenterArt(artPath, 33)
	AbsCenterText(text, l, BgCyan)
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

// NewTimer boots a user after being idle too long
func NewTimer(seconds int, action func()) *time.Timer {
	timer := time.NewTimer(time.Second * time.Duration(seconds))

	go func() {
		<-timer.C
		action()
	}()
	return timer
}

// Wait for a key press
func Pause() {

	shortTimer := NewTimer(Idle, func() {
		fmt.Println("\r\nYou've been idle for too long... exiting!")
		time.Sleep(1 * time.Second)
		os.Exit(0)
	})
	defer shortTimer.Stop()

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

		ClearScreen()

		return h, w

	} else {
		// couldn't detect, so let's just set 80 x 25 to be safe
		h := 80
		w := 25

		return h, w
	}

}

func PrintAnsi(artfile string, delay int) {
	noSauce := TrimStringFromSauce(artfile) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {

		fmt.Fprintf(os.Stdout, s.Text())
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

func PrintAnsiLoc(artfile string, x int, y int) {
	yLoc := y

	noSauce := TrimStringFromSauce(artfile) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {
		fmt.Fprintf(os.Stdout, Esc+strconv.Itoa(yLoc)+";"+strconv.Itoa(x)+"f"+s.Text())
		yLoc++
	}
}

// Print text at an X, Y location
func PrintStringLoc(text string, x int, y int) {
	fmt.Fprintf(os.Stdout, Esc+strconv.Itoa(y)+";"+strconv.Itoa(x)+"f"+text)

}

// Horizontally center some text.
func CenterText(s string, w int) {
	fmt.Fprintf(os.Stdout, (fmt.Sprintf("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))))
}

// Horizontally and Vertically center some text.
func AbsCenterText(s string, l int, c string) {
	centerY := modalH / 2
	halfLen := l / 2
	centerX := (modalW - modalW/2) - halfLen
	MoveCursor(centerX, centerY)
	fmt.Fprintf(os.Stdout, WhiteHi+c+s+Reset)
	result := Continue()
	if result {
		fmt.Fprintf(os.Stdout, BgCyan+CyanHi+" Yes"+Reset)
		time.Sleep(1 * time.Second)
	}
	if !result {
		fmt.Fprintf(os.Stdout, BgCyan+CyanHi+" No"+Reset)
		time.Sleep(1 * time.Second)
	}
}

func AbsCenterArt(artfile string, l int) {
	artY := (modalH / 2) - 2
	artLen := l / 2
	artX := (modalW - modalW/2) - artLen

	noSauce := TrimStringFromSauce(artfile) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {
		fmt.Fprintf(os.Stdout, Esc+strconv.Itoa(artY)+";"+strconv.Itoa(artX)+"f")
		fmt.Println(s.Text())
		artY++
	}
}

// Credit to @richorr
func PipeColorToEscCode(ansiColor string) (string, bool) {

	log.Println("checking string == " + ansiColor)

	if len(strings.Trim(ansiColor, " ")) < 2 {
		log.Println("finding no string")
		return "", false
	}
	tint := ""
	isColor := true

	log.Println("finding a string")
	switch ansiColor[0] {
	case '0':
		tint = BgBlack
	case '1':
		tint = BgBlue
	case '2':
		tint = BgGreen
	case '3':
		tint = BgCyan
	case '4':
		tint = BgRed
	case '5':
		tint = BgMagenta
	case '6':
		tint = BgYellow
	case '7':
		tint = BgWhite
	case '8':
		tint = BgBlackHi
	case '9':
		tint = BgBlueHi
	case 'A':
		tint = BgGreenHi
	case 'B':
		tint = BgCyanHi
	case 'C':
		tint = BgRedHi
	case 'D':
		tint = BgMagentaHi
	case 'E':
		tint = BgYellowHi
	case 'F':
		tint = BgWhiteHi
	default:
		isColor = false
	}

	if isColor {
		switch ansiColor[1] {
		case '0':
			tint += Black
		case '1':
			tint += Blue
		case '2':
			tint += Green
		case '3':
			tint += Cyan
		case '4':
			tint += Red
		case '5':
			tint += Magenta
		case '6':
			tint += Yellow
		case '7':
			tint += White
		case '8':
			tint += BlackHi
		case '9':
			tint += BlueHi
		case 'A':
			tint += GreenHi
		case 'B':
			tint += CyanHi
		case 'C':
			tint += RedHi
		case 'D':
			tint += MagentaHi
		case 'E':
			tint += YellowHi
		case 'F':
			tint += WhiteHi
		default:
			isColor = false
		}
	}
	return tint, isColor
}

func PrintPipeColor(text string, defaultTint string) string {
	tint := defaultTint
	// log.Println("Text: ", text)

	coloredSections := strings.Split(text, "|")
	for i, ansiBlock := range coloredSections {
		log.Println("Ansi Block @", strconv.Itoa(i), ": [", ansiBlock, "]")
		if len(strings.Trim(ansiBlock, " ")) < 2 {
			if i == 0 {
				fmt.Print(tint, ansiBlock, Reset)
			} else {
				fmt.Print(tint, "|", ansiBlock, Reset)
			}
		} else {
			newTint, isColor := PipeColorToEscCode(ansiBlock[0:2])

			if isColor {
				// Change the tint and then print the following block
				tint = newTint
				fmt.Print(tint, ansiBlock[2:], Reset)
			} else {
				// Print the pipe if a color was not found
				if i == 0 {
					fmt.Print(tint, ansiBlock, Reset)
				} else {
					fmt.Print(tint, "|", ansiBlock, Reset)
				}
			}
		}
	}
	return tint
}

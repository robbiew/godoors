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
	"unicode/utf8"
)

func DropFileData(nodeNum string, path string) (string, int, int) {

	var dropAlias string
	var dropTimeLeft string
	var dropEmulation string

	file, err := os.Open(strings.ToLower(path + "/" + nodeNum + "/" + "door32.sys"))
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
		if count == 7 {
			dropAlias = line
		}
		if count == 9 {
			dropTimeLeft = line
		}
		if count == 10 {
			dropEmulation = line
		}

		if count == 51 {
			break
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	timeInt, err := strconv.Atoi(dropTimeLeft) // return as int
	if err != nil {
		println(err)
	}

	emuInt, err := strconv.Atoi(dropEmulation) // return as int
	if err != nil {
		println(err)
	}

	return dropAlias, timeInt, emuInt

}

func GetTermSize() (int, int) {

	/*
		Get the terminal size
		- Send a cursor position that we know is way too large
		- Terminal sends back the largest row + col size
		- Read in the result
	*/

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

func PrintAnsi(file string, h int, w int) {

	rowCount := 0

	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	noSauce := TrimStringFromSauce(string(content)) // strip off the SAUCE metadata
	s := bufio.NewScanner(strings.NewReader(string(noSauce)))

	for s.Scan() {

		fmt.Println(s.Text())
		rowCount++
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

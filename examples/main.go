package main

import (
	"flag"
	"fmt"
	"os"

	gd "github.com/robbiew/godoors"
)

func main() {

	h, w := gd.GetTermSize()

	gd.ClearScreen()

	pathPtr := flag.String("path", "", "path to door32.sys file")

	required := []string{"path"}
	flag.Parse()

	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			// or possibly use `log.Fatalf` instead of:
			fmt.Fprintf(os.Stderr, "missing path to door32.sys directory: -%s \n", req)
			os.Exit(2) // the same exit code flag.Parse uses

		}
	}

	path := *pathPtr

	dropAlias, timeInt, emuInt := gd.DropFileData(path)

	var emuName string

	if emuInt == 1 {
		emuName = "ANSI"
	} else if emuInt == 0 {
		emuName = "ASCII"
	}

	gd.MoveCursor(0, 0)
	fmt.Println("Screen Cleared!")
	fmt.Println("Cursor moved to 0,0")

	fmt.Println("\r\nDROP FILE:")
	fmt.Println("Alias: " + dropAlias)
	fmt.Fprintf(os.Stdout, "Emulation: %v\r\n", emuName)
	fmt.Fprintf(os.Stdout, "Time Left: %v\r\n", timeInt)

	fmt.Println("\r\nHEIGHT,WiDTH DETECT:")
	fmt.Fprintf(os.Stdout, "Height: %v\r\n", h)
	fmt.Fprintf(os.Stdout, "Width: %v\r\n", w)

	fmt.Println("\r\nCOLOR TEST:")
	fmt.Println(gd.BackgroundColorBlue + gd.TextColorWhite + " White Text on Blue " + gd.ColorReset)
	fmt.Println(gd.BackgroundColorRed + gd.TextColorBrightRed + " Red Text on Bright Red " + gd.ColorReset)

	fmt.Println("\r\nFONT TEST (SyncTerm):")
	fmt.Println(gd.Topaz + "Topaz")
	fmt.Println(gd.Mosoul + "Mosoul")
	fmt.Println(gd.Ibm + "IBM CP437")
}

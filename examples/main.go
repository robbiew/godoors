package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eiannone/keyboard"
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

	if err := keyboard.Open(); err != nil {
		fmt.Println(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	for {
		fmt.Fprintf(os.Stdout, "\r\n")
		// menu
		fmt.Println("-------MENU-------")
		fmt.Println("[A] ANSI Art Test")
		fmt.Println("[C] Color Test")
		fmt.Println("[D] Drop File Test")
		fmt.Println("[F] Font Test")
		fmt.Println("[T] Term Size Test")
		fmt.Println("[Q] Quit")
		fmt.Fprintf(os.Stdout, "\r\nCommand? ")

		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		gd.ClearScreen()

		if string(char) == "q" || string(char) == "Q" || key == keyboard.KeyEsc {
			break
		}
		if string(char) == "a" || string(char) == "A" {
			fmt.Println("\r\nART TEST:")
			gd.PrintAnsi("mx-sm.ans", 40)
			gd.Pause()
		}
		if string(char) == "c" || string(char) == "C" {

			fmt.Println("\r\nCOLOR TEST:")
			fmt.Println(gd.BackgroundColorBlue + gd.TextColorWhite + " White Text on Blue " + gd.ColorReset)
			fmt.Println(gd.BackgroundColorRed + gd.TextColorBrightRed + " Red Text on Bright Red " + gd.ColorReset)
		}
		if string(char) == "d" || string(char) == "D" {

			fmt.Println("\r\nDROP FILE:")
			fmt.Println("Alias: " + dropAlias)
			fmt.Fprintf(os.Stdout, "Emulation: %v\r\n", emuName)
			fmt.Fprintf(os.Stdout, "Time Left: %v\r\n", timeInt)
		}
		if string(char) == "f" || string(char) == "F" {

			fmt.Println("\r\nFONT TEST (SyncTerm):")
			fmt.Println(gd.Topaz + "Topaz")
			fmt.Println(gd.Mosoul + "Mosoul")
			fmt.Println(gd.Ibm + "IBM CP437")
		}
		if string(char) == "t" || string(char) == "T" {

			fmt.Println("\r\nTERMINAL SIZE DETECT:")
			fmt.Fprintf(os.Stdout, "Height: %v\r\n", h)
			fmt.Fprintf(os.Stdout, "Width: %v\r\n", w)
		}
		continue
	}

}

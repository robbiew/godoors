package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eiannone/keyboard"
	gd "github.com/robbiew/godoors"
)

func main() {

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

	h, w := gd.GetTermSize()
	gd.ClearScreen()

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
		fmt.Println("[M] Modal Test")
		fmt.Println("[T] Term Size Test")
		fmt.Println("[Q] Quit")
		fmt.Fprintf(os.Stdout, "\r\nCommand? ")

		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		if string(char) == "q" || string(char) == "Q" || key == keyboard.KeyEsc {
			break
		}
		if string(char) == "a" || string(char) == "A" {
			gd.ClearScreen()
			fmt.Println("\r\nART TEST:")
			gd.PrintAnsi("mx-sm.ans", 40)
			gd.Pause()
		}
		if string(char) == "c" || string(char) == "C" {
			fmt.Println("\r\nCOLOR TEST:")
			gd.ClearScreen()
			fmt.Println(gd.BackgroundColorBlue + gd.TextColorWhite + " White Text on Blue " + gd.ColorReset)
			fmt.Println(gd.BackgroundColorRed + gd.TextColorBrightRed + " Red Text on Bright Red " + gd.ColorReset)
			gd.Pause()
		}
		if string(char) == "d" || string(char) == "D" {
			gd.ClearScreen()
			fmt.Println("\r\nDROP FILE:")
			fmt.Println("Alias: " + dropAlias)
			fmt.Fprintf(os.Stdout, "Emulation: %v\r\n", emuName)
			fmt.Fprintf(os.Stdout, "Time Left: %v\r\n", timeInt)
			gd.Pause()
		}
		if string(char) == "f" || string(char) == "F" {
			gd.ClearScreen()
			fmt.Println("\r\nFONT TEST (SyncTerm):")
			fmt.Println(gd.Topaz + "Topaz")
			fmt.Println(gd.Mosoul + "Mosoul")
			fmt.Println(gd.Ibm + "IBM CP437")
			gd.Pause()
		}
		if string(char) == "m" || string(char) == "M" {
			mText := "Continue? Y/n"
			mLen := len(mText)
			gd.Modal(mText, mLen, w, h)

		}
		if string(char) == "t" || string(char) == "T" {
			gd.ClearScreen()
			fmt.Println("\r\nTERMINAL SIZE DETECT:")
			fmt.Fprintf(os.Stdout, "Height: %v\r\n", h)
			fmt.Fprintf(os.Stdout, "Width: %v\r\n", w)
			gd.Pause()
		}
		gd.ClearScreen()
		continue
	}
}

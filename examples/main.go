package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/eiannone/keyboard"
	gd "github.com/robbiew/godoors"
)

func main() {

	// Use FLAG to get command line paramenters
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

	// Get info from the Drop File
	dropAlias, timeInt, emuInt, nodeInt := gd.DropFileData(path)

	// Try and detect terminal size
	h, w := gd.GetTermSize()

	gd.ClearScreen()

	var emuName string

	if emuInt == 1 {
		emuName = "ANSI"
	} else if emuInt == 0 {
		emuName = "ASCII"
	}

	gd.MoveCursor(0, 0)

	// Exit if no ANSI capabilities (sorry!)
	if emuInt != 1 {
		fmt.Println("Sorry, ANSI is required to use this...")
		time.Sleep(time.Duration(2) * time.Second)
		os.Exit(0)
	}

	// A reliable keyboard library to detec key presses
	if err := keyboard.Open(); err != nil {
		fmt.Println(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	for {
		fmt.Fprintf(os.Stdout, "\r\n")

		// A Test Menu
		fmt.Println(gd.TextColorBrightCyan + gd.ArrowRight + gd.ColorReset + gd.TextColorCyan + " GODOORS TEST MENU" + gd.ColorReset)
		fmt.Println(gd.TextColorCyan + "\r\n[" + gd.TextColorBrightYellow + "A" + gd.TextColorCyan + "] " + gd.ColorReset + gd.TextColorMagenta + "Art Test")
		fmt.Println(gd.TextColorCyan + "[" + gd.TextColorBrightYellow + "C" + gd.TextColorCyan + "] " + gd.ColorReset + gd.TextColorMagenta + "Color Test")
		fmt.Println(gd.TextColorCyan + "[" + gd.TextColorBrightYellow + "D" + gd.TextColorCyan + "] " + gd.ColorReset + gd.TextColorMagenta + "Drop File Test")
		fmt.Println(gd.TextColorCyan + "[" + gd.TextColorBrightYellow + "F" + gd.TextColorCyan + "] " + gd.ColorReset + gd.TextColorMagenta + "Font Test")
		fmt.Println(gd.TextColorCyan + "[" + gd.TextColorBrightYellow + "M" + gd.TextColorCyan + "] " + gd.ColorReset + gd.TextColorMagenta + "Modal Test")
		fmt.Println(gd.TextColorCyan + "[" + gd.TextColorBrightYellow + "T" + gd.TextColorCyan + "] " + gd.ColorReset + gd.TextColorMagenta + "Term Size Test")
		fmt.Println(gd.TextColorCyan + "[" + gd.TextColorBrightYellow + "Q" + gd.TextColorCyan + "] " + gd.ColorReset + gd.TextColorMagenta + "Quit")
		fmt.Fprintf(os.Stdout, gd.ColorReset+"\r\nCommand? ")

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
			fmt.Fprintf(os.Stdout, "Node: %v\r\n", nodeInt)
			fmt.Fprintf(os.Stdout, "Emulation: %v\r\n", emuName)
			fmt.Fprintf(os.Stdout, "Time Left: %v\r\n", timeInt)
			gd.Pause()
		}
		if string(char) == "f" || string(char) == "F" {
			gd.ClearScreen()
			fmt.Println("\r\nFONT TEST (SyncTerm):")
			fmt.Println(gd.Topaz + "\r\nTopaz")
			fmt.Println(gd.Topazplus + "Topaz+")
			fmt.Println(gd.Microknight + "Microknight")
			fmt.Println(gd.Microknightplus + "Microknight+")
			fmt.Println(gd.Mosoul + "mO'sOul")
			fmt.Println(gd.Ibm + "IBM CP437")
			fmt.Println(gd.Ibmthin + "IBM CP437 Thin")
			gd.Pause()
		}
		// Modal test
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

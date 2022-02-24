package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/eiannone/keyboard"
	gd "github.com/robbiew/godoors"
)

var idle int = 180

func main() {

	// Start the idle timer
	shortTimer := gd.NewTimer(idle, func() {
		fmt.Println("\r\nYou've been idle for too long... exiting!")
		time.Sleep(1 * time.Second)
		os.Exit(0)
	})
	defer shortTimer.Stop()

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

		// Stop the idle timer after key press, then re-start it
		shortTimer.Stop()
		shortTimer = gd.NewTimer(idle, func() {
			fmt.Println("\r\nYou've been idle for too long... exiting!")
			time.Sleep(1 * time.Second)
			os.Exit(0)
		})

		fmt.Fprintf(os.Stdout, "\r\n")

		// A Test Menu
		fmt.Println(gd.CyanHi + gd.ArrowRight + gd.Reset + gd.Cyan + " GODOORS TEST MENU" + gd.Reset)
		fmt.Println(gd.Cyan + "\r\n[" + gd.YellowHi + "A" + gd.Cyan + "] " + gd.Reset + gd.Magenta + "Art Test")
		fmt.Println(gd.Cyan + "[" + gd.YellowHi + "C" + gd.Cyan + "] " + gd.Reset + gd.Magenta + "Color Test")
		fmt.Println(gd.Cyan + "[" + gd.YellowHi + "D" + gd.Cyan + "] " + gd.Reset + gd.Magenta + "Drop File Test")
		fmt.Println(gd.Cyan + "[" + gd.YellowHi + "F" + gd.Cyan + "] " + gd.Reset + gd.Magenta + "Font Test")
		fmt.Println(gd.Cyan + "[" + gd.YellowHi + "M" + gd.Cyan + "] " + gd.Reset + gd.Magenta + "Modal Test")
		fmt.Println(gd.Cyan + "[" + gd.YellowHi + "T" + gd.Cyan + "] " + gd.Reset + gd.Magenta + "Term Size Test")
		fmt.Println(gd.Cyan + "[" + gd.YellowHi + "Q" + gd.Cyan + "] " + gd.Reset + gd.Magenta + "Quit")
		fmt.Fprintf(os.Stdout, gd.Reset+"\r\nCommand? ")

		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		if string(char) == "q" || string(char) == "Q" || key == keyboard.KeyEsc {
			break
		}
		if string(char) == "a" || string(char) == "A" {
			shortTimer.Stop()
			gd.ClearScreen()
			fmt.Println("\r\nART TEST:")
			gd.PrintAnsi("mx-sm.ans", 40)
			gd.Pause()
		}
		if string(char) == "c" || string(char) == "C" {
			shortTimer.Stop()
			fmt.Println("\r\nCOLOR TEST:")
			gd.ClearScreen()
			fmt.Println(gd.BgBlue + gd.White + " White Text on Blue " + gd.Reset)
			fmt.Println(gd.BgRed + gd.RedHi + " Red Text on Bright Red " + gd.Reset)
			gd.Pause()
		}
		if string(char) == "d" || string(char) == "D" {
			shortTimer.Stop()
			gd.ClearScreen()
			fmt.Println("\r\nDROP FILE:")
			fmt.Println("Alias: " + dropAlias)
			fmt.Fprintf(os.Stdout, "Node: %v\r\n", nodeInt)
			fmt.Fprintf(os.Stdout, "Emulation: %v\r\n", emuName)
			fmt.Fprintf(os.Stdout, "Time Left: %v\r\n", timeInt)
			gd.Pause()
		}
		if string(char) == "f" || string(char) == "F" {
			shortTimer.Stop()
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
			shortTimer.Stop()
			mText := "Continue? Y/n"
			mLen := len(mText)
			gd.Modal(mText, mLen, w, h)
		}
		if string(char) == "t" || string(char) == "T" {
			shortTimer.Stop()
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

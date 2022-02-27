package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/eiannone/keyboard"
)

var (
	Idle     int = 120 // seconds without keyboard activity, send program
	DropPath string
)

func init() {
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
	DropPath = *pathPtr
}

func main() {

	// Get door32.sys, h, w as user object
	u := gd.Initialize(DropPath)

	// Start the idle timer
	shortTimer := NewTimer(Idle, func() {
		fmt.Println("\r\nYou've been idle for too long... exiting!")
		time.Sleep(1 * time.Second)
		os.Exit(0)
	})
	defer shortTimer.Stop()

	ClearScreen()
	MoveCursor(0, 0)

	// Exit if no ANSI capabilities (sorry!)
	if u.Emulation != 1 {
		fmt.Println("Sorry, ANSI is required to use this...")
		time.Sleep(time.Duration(2) * time.Second)
		os.Exit(0)
	}

	// A reliable keyboard library to detect key presses
	if err := keyboard.Open(); err != nil {
		fmt.Println(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	for {
		// Stop the idle timer after key press, then re-start it
		shortTimer.Stop()
		shortTimer = NewTimer(Idle, func() {
			fmt.Println("\r\nYou've been idle for too long... exiting!")
			time.Sleep(1 * time.Second)
			os.Exit(0)
		})

		fmt.Fprintf(os.Stdout, "\r\n")

		// A Test Menu
		fmt.Fprintf(os.Stdout, CyanHi+ArrowRight+Reset+Cyan+" GODOORS TEST MENU\r\n"+Reset)
		fmt.Fprintf(os.Stdout, Cyan+"\r\n["+YellowHi+"A"+Cyan+"] "+Reset+Magenta+"Art Test\r\n")
		fmt.Fprintf(os.Stdout, Cyan+"["+YellowHi+"C"+Cyan+"] "+Reset+Magenta+"Color Test\r\n")
		fmt.Fprintf(os.Stdout, Cyan+"["+YellowHi+"D"+Cyan+"] "+Reset+Magenta+"Drop File Test\r\n")
		fmt.Fprintf(os.Stdout, Cyan+"["+YellowHi+"F"+Cyan+"] "+Reset+Magenta+"Font Test\r\n")
		fmt.Fprintf(os.Stdout, Cyan+"["+YellowHi+"M"+Cyan+"] "+Reset+Magenta+"Modal Test\r\n")
		fmt.Fprintf(os.Stdout, Cyan+"["+YellowHi+"T"+Cyan+"] "+Reset+Magenta+"Term Size Test\r\n")
		fmt.Fprintf(os.Stdout, Cyan+"["+YellowHi+"Q"+Cyan+"] "+Reset+Magenta+"Quit\r\n")
		fmt.Fprintf(os.Stdout, Reset+"\r\nCommand? ")

		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		if string(char) == "q" || string(char) == "Q" || key == keyboard.KeyEsc {
			break
		}
		if string(char) == "a" || string(char) == "A" {
			shortTimer.Stop()
			ClearScreen()
			fmt.Println("\r\nART TEST:")
			PrintAnsi("mx-sm.ans", 40)
			Pause()
		}
		if string(char) == "c" || string(char) == "C" {
			shortTimer.Stop()
			fmt.Println("\r\nCOLOR TEST:")
			ClearScreen()
			fmt.Println(BgBlue + White + " White Text on Blue " + Reset)
			fmt.Println(BgRed + RedHi + " Red Text on Bright Red " + Reset)
			Pause()
		}

		if string(char) == "d" || string(char) == "D" {
			shortTimer.Stop()
			ClearScreen()
			fmt.Println("\r\nDROP FILE:")
			fmt.Fprintf(os.Stdout, "Alias: %v\r\n", u.Alias)
			fmt.Fprintf(os.Stdout, "Node: %v\r\n", u.NodeNum)
			fmt.Fprintf(os.Stdout, "Emulation: %v\r\n", u.Emulation)
			fmt.Fprintf(os.Stdout, "Time Left: %v\r\n", u.TimeLeft)
			Pause()
		}
		if string(char) == "f" || string(char) == "F" {
			shortTimer.Stop()
			ClearScreen()
			fmt.Println("\r\nFONT TEST (SyncTerm):")
			fmt.Println(Topaz + "\r\nTopaz")
			fmt.Println(Topazplus + "Topaz+")
			fmt.Println(Microknight + "Microknight")
			fmt.Println(Microknightplus + "Microknight+")
			fmt.Println(Mosoul + "mO'sOul")
			fmt.Println(Ibm + "IBM CP437")
			fmt.Println(Ibmthin + "IBM CP437 Thin")
			Pause()
		}
		// Modal test
		if string(char) == "m" || string(char) == "M" {
			shortTimer.Stop()
			mText := "Continue? Y/n"
			mLen := len(mText)
			Modal(mText, mLen, u.H, u.W)
		}
		if string(char) == "t" || string(char) == "T" {
			shortTimer.Stop()
			ClearScreen()
			fmt.Println("\r\nTERMINAL SIZE DETECT:")
			fmt.Fprintf(os.Stdout, "Height: %v\r\n", u.H)
			fmt.Fprintf(os.Stdout, "Width: %v\r\n", u.W)
			Pause()
		}
		ClearScreen()
		continue
	}
}
